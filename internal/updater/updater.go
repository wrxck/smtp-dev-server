package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	repoOwner = "wrxck"
	repoName  = "smtp-dev-server"
	apiURL    = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
)

type Release struct {
	TagName string  `json:"tag_name"`
	HTMLURL string  `json:"html_url"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int    `json:"size"`
}

// HTTPClient allows injection for testing.
var HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
} = &http.Client{Timeout: 5 * time.Second}

// CheckForUpdate checks GitHub for a newer release. Returns the release info
// if an update is available, or nil if already up to date.
func CheckForUpdate(currentVersion string) (*Release, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "smtp-dev-server/"+currentVersion)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentClean := strings.TrimPrefix(currentVersion, "v")

	if latestVersion == currentClean || currentClean == "dev" {
		return nil, nil
	}

	return &release, nil
}

// AssetName returns the expected asset filename for the current platform.
func AssetName(version string) string {
	return fmt.Sprintf("smtp-dev-server-%s-%s-%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
}

// FindAsset finds the download asset for the current platform in a release.
func FindAsset(release *Release) *Asset {
	name := AssetName(release.TagName)
	for i := range release.Assets {
		if release.Assets[i].Name == name {
			return &release.Assets[i]
		}
	}
	return nil
}

// DownloadAndInstall downloads the release asset and replaces the current binary.
func DownloadAndInstall(asset *Asset) error {
	req, err := http.NewRequest("GET", asset.BrowserDownloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "smtp-dev-server")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "smtp-dev-server-update-*.tar.gz")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	tmpDir, err := os.MkdirTemp("", "smtp-dev-server-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("tar", "xzf", tmpFile.Name(), "-C", tmpDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("extract failed: %w: %s", err, out)
	}

	binaryName := fmt.Sprintf("smtp-dev-server-%s-%s", runtime.GOOS, runtime.GOARCH)
	newBinary := tmpDir + "/" + binaryName

	if _, err := os.Stat(newBinary); err != nil {
		return fmt.Errorf("expected binary %s not found in archive", binaryName)
	}

	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine current binary path: %w", err)
	}

	// Resolve symlinks
	resolved, err := resolveExecutable(currentBinary)
	if err != nil {
		return err
	}

	if err := os.Rename(newBinary, resolved); err != nil {
		// Cross-device rename, fall back to copy
		return copyFile(newBinary, resolved)
	}

	return nil
}

func resolveExecutable(path string) (string, error) {
	resolved, err := os.Readlink(path)
	if err != nil {
		return path, nil // not a symlink
	}
	if !strings.HasPrefix(resolved, "/") {
		dir := path[:strings.LastIndex(path, "/")+1]
		resolved = dir + resolved
	}
	return resolved, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
