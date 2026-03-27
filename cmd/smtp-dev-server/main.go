package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wrxck/smtp-dev-server/internal/smtp"
	"github.com/wrxck/smtp-dev-server/internal/store"
	"github.com/wrxck/smtp-dev-server/internal/web"
)

var version = "dev"

const banner = `
┌────────────.
|\          / \    smtp-dev-server %s
| \        /   \   A fake SMTP server for macOS
|  smtp-dev     /  Web UI: http://%s
|             /
└────────────'
`

func main() {
	smtpAddr := flag.String("smtp", "127.0.0.1:2525", "SMTP server listen address")
	httpAddr := flag.String("http", "127.0.0.1:5050", "Web UI / API listen address")
	maxMessages := flag.Int("max-messages", 500, "Maximum number of messages to keep")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("smtp-dev-server %s\n", version)
		os.Exit(0)
	}

	fmt.Printf(banner, version, *httpAddr)

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
