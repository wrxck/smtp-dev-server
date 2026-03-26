# smtp-dev-server

[![CI](https://github.com/wrxck/smtp-dev-server/actions/workflows/ci.yml/badge.svg)](https://github.com/wrxck/smtp-dev-server/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/wrxck/smtp-dev-server)](https://goreportcard.com/report/github.com/wrxck/smtp-dev-server)
[![GitHub release](https://img.shields.io/github/v/release/wrxck/smtp-dev-server)](https://github.com/wrxck/smtp-dev-server/releases)
[![License](https://img.shields.io/github/license/wrxck/smtp-dev-server)](LICENSE.md)
[![Platform](https://img.shields.io/badge/platform-macOS-blue)](https://github.com/wrxck/smtp-dev-server)

**A fake SMTP server for macOS development and testing.** Catches all outgoing emails and displays them in a slick dark-mode web UI — no runtime dependencies, single binary.

Inspired by [smtp4dev](https://github.com/rnwood/smtp4dev), rewritten from scratch in Go for macOS.

---

## Why?

When building apps that send email (signup flows, password resets, notifications, transactional emails), you need a way to test them without sending real emails. smtp-dev-server captures every email your app sends and lets you inspect it in a web browser.

**No .NET runtime. No Docker. No Node.js.** Just a single ~7MB binary.

---

## Install

### Homebrew (recommended)

```bash
brew tap wrxck/tap
brew install smtp-dev-server
```

### Download binary

Grab the latest release for your Mac from [GitHub Releases](https://github.com/wrxck/smtp-dev-server/releases):

- **Apple Silicon (M1/M2/M3/M4):** `smtp-dev-server-*-darwin-arm64.tar.gz`
- **Intel:** `smtp-dev-server-*-darwin-amd64.tar.gz`

```bash
tar xzf smtp-dev-server-*.tar.gz
chmod +x smtp-dev-server-darwin-*
sudo mv smtp-dev-server-darwin-* /usr/local/bin/smtp-dev-server
```

### Build from source

```bash
go install github.com/wrxck/smtp-dev-server/cmd/smtp-dev-server@latest
```

---

## Quick Start

```bash
smtp-dev-server
```

```
┌────────────.
|\          / \    smtp-dev-server v1.0.0
| \        /   \   A fake SMTP server for macOS
|  smtp-dev     /  Web UI: http://127.0.0.1:5050
|             /
└────────────'
```

This starts two servers:

| Service | Default Address | Purpose |
|---------|----------------|---------|
| **SMTP** | `127.0.0.1:2525` | Receives emails from your application |
| **Web UI** | `http://127.0.0.1:5050` | View and inspect captured emails |

Point your application's SMTP config to `localhost:2525` and open `http://localhost:5050` in your browser.

### Example: Configure your app

**Rails:**
```ruby
# config/environments/development.rb
config.action_mailer.smtp_settings = { address: '127.0.0.1', port: 2525 }
```

**Django:**
```python
# settings.py
EMAIL_HOST = '127.0.0.1'
EMAIL_PORT = 2525
```

**Node.js (Nodemailer):**
```javascript
const transport = nodemailer.createTransport({ host: '127.0.0.1', port: 2525 });
```

**Laravel:**
```env
MAIL_HOST=127.0.0.1
MAIL_PORT=2525
```

**Go:**
```go
smtp.SendMail("127.0.0.1:2525", nil, from, to, msg)
```

**Python:**
```python
import smtplib
server = smtplib.SMTP('127.0.0.1', 2525)
server.sendmail(from_addr, to_addrs, msg)
```

---

## Command Line Options

```
Usage: smtp-dev-server [options]

Options:
  -smtp string        SMTP server listen address (default "127.0.0.1:2525")
  -http string        Web UI / API listen address (default "127.0.0.1:5050")
  -max-messages int   Maximum number of messages to retain (default 500)
  -version            Show version and exit
```

### Examples

```bash
# Use custom ports
smtp-dev-server -smtp 127.0.0.1:1025 -http 127.0.0.1:8080

# Keep more messages in memory
smtp-dev-server -max-messages 2000

# Bind to all interfaces (accessible from other machines/containers)
smtp-dev-server -smtp 0.0.0.0:2525 -http 0.0.0.0:5050
```

---

## Features

### SMTP Server
- Full **ESMTP** support (EHLO, 8BITMIME, SMTPUTF8)
- Accepts any **AUTH** credentials (dev mode — always succeeds)
- Multiple recipients per message
- Large message support (up to 50MB)
- Detailed **session logging** (see every SMTP command exchanged)

### Web UI
- **Real-time updates** — new emails appear instantly via Server-Sent Events (SSE)
- **HTML rendering** — view emails exactly as recipients would see them (sandboxed iframe)
- **Plain text view** — for text-only emails or multipart alternatives
- **Headers inspector** — see all email headers in a clean table
- **Raw source view** — full RFC 822 source for debugging
- **Attachment downloads** — download any attachment from captured emails
- **Unread indicators** — blue dot for unread messages, badge counter in title bar
- **Dark mode** — easy on the eyes during late-night debugging
- **Responsive** — works on any screen size

### REST API
Full API for automation, CI/CD integration, and scripting:

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/messages` | List all captured messages |
| `GET` | `/api/messages/{id}` | Get message metadata (from, to, subject, etc.) |
| `GET` | `/api/messages/{id}/html` | Get rendered HTML body |
| `GET` | `/api/messages/{id}/text` | Get plain text body |
| `GET` | `/api/messages/{id}/raw` | Get raw RFC 822 source |
| `GET` | `/api/messages/{id}/headers` | Get parsed headers as JSON |
| `GET` | `/api/messages/{id}/attachments` | List attachments with metadata |
| `GET` | `/api/messages/{id}/attachments/{i}` | Download attachment by index |
| `POST` | `/api/messages/{id}/read` | Mark message as read |
| `DELETE` | `/api/messages/{id}` | Delete a single message |
| `DELETE` | `/api/messages` | Delete all messages |
| `GET` | `/api/sessions` | List SMTP sessions |
| `GET` | `/api/sessions/{id}` | Get session detail with full SMTP log |
| `GET` | `/api/events` | SSE stream — real-time message notifications |

### API Examples

```bash
# List all messages
curl http://localhost:5050/api/messages | jq

# Get the latest message's HTML body
ID=$(curl -s http://localhost:5050/api/messages | jq -r '.[0].id')
curl http://localhost:5050/api/messages/$ID/html

# Wait for a message in CI (poll until count > 0)
while [ "$(curl -s http://localhost:5050/api/messages | jq length)" = "0" ]; do sleep 1; done

# Delete all messages between test runs
curl -X DELETE http://localhost:5050/api/messages
```

---

## Architecture

```
┌─────────────┐    SMTP (port 2525)    ┌──────────────────┐
│  Your App   │ ────────────────────── │                  │
│  (Rails,    │                        │  smtp-dev-server │
│   Django,   │    HTTP (port 5050)    │                  │
│   Node...)  │                        │  ┌────────────┐  │
└─────────────┘                        │  │ In-memory  │  │
                                       │  │ store      │  │
┌─────────────┐    GET /api/messages   │  └────────────┘  │
│  Browser    │ ◄───────────────────── │                  │
│  (Web UI)   │    SSE /api/events     │                  │
└─────────────┘ ◄───────────────────── └──────────────────┘
```

smtp-dev-server is a single Go binary with three components:
- **SMTP server** — accepts incoming emails on a TCP socket
- **In-memory store** — holds messages and sessions (configurable max)
- **HTTP server** — serves the web UI and REST API, plus SSE for live updates

All state is in-memory. Restarting the server clears all messages.

---

## License

BSD-3-Clause — see [LICENSE.md](LICENSE.md)
