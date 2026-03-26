# smtp-dev-server

A fake SMTP server for macOS development and testing. Catches all outgoing emails and displays them in a web UI.

Based on the concept of [smtp4dev](https://github.com/rnwood/smtp4dev), rewritten as a single native binary for macOS — no runtime dependencies.

## Install

```bash
brew tap wrxck/tap
brew install smtp-dev-server
```

Or download the latest release from [GitHub Releases](https://github.com/wrxck/smtp-dev-server/releases).

## Usage

```bash
smtp-dev-server
```

This starts:
- **SMTP server** on `127.0.0.1:2525`
- **Web UI** at `http://127.0.0.1:5000`

Point your application's SMTP settings to `localhost:2525` and all emails will be captured and displayed in the web UI.

### Options

```
-smtp     SMTP listen address (default: 127.0.0.1:2525)
-http     Web UI listen address (default: 127.0.0.1:5000)
-max-messages  Maximum messages to retain (default: 500)
-version  Show version
```

### Examples

```bash
# Use custom ports
smtp-dev-server -smtp 127.0.0.1:1025 -http 127.0.0.1:8080

# Keep more messages
smtp-dev-server -max-messages 2000
```

## Features

- SMTP server with ESMTP support (EHLO, AUTH, 8BITMIME, SMTPUTF8)
- Web UI with real-time updates (SSE)
- View HTML and plain text email bodies
- Inspect email headers and raw source
- Download attachments
- SMTP session logging
- REST API for automation
- Single binary, no dependencies
- Dark mode UI

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/messages` | List all messages |
| GET | `/api/messages/{id}` | Get message metadata |
| GET | `/api/messages/{id}/html` | Get HTML body |
| GET | `/api/messages/{id}/text` | Get plain text body |
| GET | `/api/messages/{id}/raw` | Get raw RFC822 source |
| GET | `/api/messages/{id}/headers` | Get parsed headers |
| GET | `/api/messages/{id}/attachments` | List attachments |
| GET | `/api/messages/{id}/attachments/{index}` | Download attachment |
| POST | `/api/messages/{id}/read` | Mark as read |
| DELETE | `/api/messages/{id}` | Delete a message |
| DELETE | `/api/messages` | Delete all messages |
| GET | `/api/sessions` | List SMTP sessions |
| GET | `/api/sessions/{id}` | Get session with log |
| GET | `/api/events` | SSE stream for real-time updates |

## License

BSD-3-Clause — see [LICENSE.md](LICENSE.md)
