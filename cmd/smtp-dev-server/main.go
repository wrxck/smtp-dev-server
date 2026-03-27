package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wrxck/smtp-dev-server/internal/smtp"
	"github.com/wrxck/smtp-dev-server/internal/store"
	"github.com/wrxck/smtp-dev-server/internal/updater"
	"github.com/wrxck/smtp-dev-server/internal/web"
)

var version = "dev"

const banner = `
┌────────────.
|\          / \    smtp-dev-server %s
| \        /   \   A fake SMTP server for development
|  smtp-dev     /  Web UI: http://%s
|             /
└────────────'
`

func main() {
	smtpAddr := flag.String("smtp", "127.0.0.1:2525", "SMTP server listen address")
	httpAddr := flag.String("http", "127.0.0.1:5050", "Web UI / API listen address")
	maxMessages := flag.Int("max-messages", 500, "Maximum number of messages to keep")
	showVersion := flag.Bool("version", false, "Show version and exit")
	doUpdate := flag.Bool("update", false, "Check for updates and install if available")
	flag.Parse()

	if *showVersion {
		fmt.Printf("smtp-dev-server %s\n", version)
		os.Exit(0)
	}

	if *doUpdate {
		runUpdate()
		return
	}

	fmt.Printf(banner, version, *httpAddr)

	// Check for updates in the background on startup
	go checkUpdateOnStartup()

	s := store.New(*maxMessages)

	smtpServer := smtp.NewServer(*smtpAddr, s)
	if err := smtpServer.Start(); err != nil {
		log.Fatalf("Failed to start SMTP server: %v", err)
	}

	webServer := web.NewServer(*httpAddr, s)
	if err := webServer.Start(); err != nil {
		log.Fatalf("Failed to start web server: %v", err)
	}

	log.Printf("Ready. SMTP on %s, Web UI on http://%s", *smtpAddr, *httpAddr)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down...")
	smtpServer.Stop()
	webServer.Stop()
}

func checkUpdateOnStartup() {
	release, err := updater.CheckForUpdate(version)
	if err != nil || release == nil {
		return
	}

	asset := updater.FindAsset(release)
	if asset == nil {
		return
	}

	fmt.Printf("\n  Update available: %s -> %s\n", version, release.TagName)
	fmt.Printf("  Run with --update to install, or download from:\n")
	fmt.Printf("  %s\n\n", release.HTMLURL)
}

func runUpdate() {
	fmt.Printf("smtp-dev-server %s — checking for updates...\n", version)

	release, err := updater.CheckForUpdate(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	if release == nil {
		fmt.Println("Already up to date.")
		return
	}

	asset := updater.FindAsset(release)
	if asset == nil {
		fmt.Fprintf(os.Stderr, "No binary available for your platform. Download manually:\n%s\n", release.HTMLURL)
		os.Exit(1)
	}

	fmt.Printf("Update available: %s -> %s (%s, %.1f MB)\n",
		version, release.TagName, asset.Name, float64(asset.Size)/(1024*1024))
	fmt.Print("Press 'u' to install update, or any other key to cancel: ")

	reader := bufio.NewReader(os.Stdin)
	ch, _, err := reader.ReadRune()
	if err != nil || (ch != 'u' && ch != 'U') {
		fmt.Println("\nUpdate cancelled.")
		return
	}

	fmt.Println("\nDownloading and installing...")

	if err := updater.DownloadAndInstall(asset); err != nil {
		fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated to %s. Restart smtp-dev-server to use the new version.\n", release.TagName)
}
