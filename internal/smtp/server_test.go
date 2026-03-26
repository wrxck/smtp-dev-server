package smtp

import (
	"bytes"
	"fmt"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/wrxck/smtp-dev-server/internal/store"
)

func readLine(conn net.Conn) string {
	var buf bytes.Buffer
	one := make([]byte, 1)
	for {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, err := conn.Read(one)
		if err != nil {
			break
		}
		if one[0] == '\n' {
			return strings.TrimRight(buf.String(), "\r")
		}
		buf.WriteByte(one[0])
	}
	return buf.String()
}

func readMultiLine(conn net.Conn) string {
	var lines []string
	for {
		line := readLine(conn)
		lines = append(lines, line)
		if len(line) < 4 || line[3] != '-' {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func writeLine(conn net.Conn, line string) {
	conn.Write([]byte(line + "\r\n"))
}

func startTestServer(t *testing.T) (*Server, *store.Store, string) {
	t.Helper()
	s := store.New(100)
	srv := NewServer("127.0.0.1:0", s)
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start smtp server: %v", err)
	}
	return srv, s, srv.Addr()
}

func TestServerStartStop(t *testing.T) {
	srv, _, _ := startTestServer(t)
	defer srv.Stop()
}

func TestServerAddr(t *testing.T) {
	s := store.New(10)
	srv := NewServer("127.0.0.1:0", s)
	// Before start, Addr returns the configured addr
	if srv.Addr() != "127.0.0.1:0" {
		t.Fatalf("expected 127.0.0.1:0, got %s", srv.Addr())
	}

	srv.Start()
	defer srv.Stop()
	// After start, Addr returns the actual address
	addr := srv.Addr()
	if addr == "127.0.0.1:0" {
		t.Fatal("expected actual bound address")
	}
}

func TestServerListenError(t *testing.T) {
	s := store.New(10)
	// Bind to a port first
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	// Try to bind the same port
	srv := NewServer(addr, s)
	if err := srv.Start(); err == nil {
		srv.Stop()
		t.Fatal("expected error binding to occupied port")
	}
}

func TestSMTPConversation(t *testing.T) {
	srv, st, addr := startTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Read greeting
	greeting := readLine(conn)
	if !strings.HasPrefix(greeting, "220") {
		t.Fatalf("expected 220 greeting, got: %s", greeting)
	}

	// EHLO
	writeLine(conn, "EHLO test")
	ehlo := readMultiLine(conn)
	if !strings.Contains(ehlo, "250") {
		t.Fatalf("expected 250 response to EHLO, got: %s", ehlo)
	}

	// MAIL FROM
	writeLine(conn, "MAIL FROM:<sender@test.com>")
	resp := readLine(conn)
	if !strings.HasPrefix(resp, "250") {
		t.Fatalf("expected 250 for MAIL FROM, got: %s", resp)
	}

	// RCPT TO
	writeLine(conn, "RCPT TO:<rcpt@test.com>")
	resp = readLine(conn)
	if !strings.HasPrefix(resp, "250") {
		t.Fatalf("expected 250 for RCPT TO, got: %s", resp)
	}

	// DATA
	writeLine(conn, "DATA")
	resp = readLine(conn)
	if !strings.HasPrefix(resp, "354") {
		t.Fatalf("expected 354 for DATA, got: %s", resp)
	}

	// Send message
	writeLine(conn, "From: sender@test.com")
	writeLine(conn, "To: rcpt@test.com")
	writeLine(conn, "Subject: Test Subject")
	writeLine(conn, "Content-Type: text/plain")
	writeLine(conn, "")
	writeLine(conn, "Hello, World!")
	writeLine(conn, ".")
	resp = readLine(conn)
	if !strings.HasPrefix(resp, "250") {
		t.Fatalf("expected 250 after DATA, got: %s", resp)
	}

	// QUIT
	writeLine(conn, "QUIT")
	resp = readLine(conn)
	if !strings.HasPrefix(resp, "221") {
		t.Fatalf("expected 221 for QUIT, got: %s", resp)
	}

	// Verify message was stored
	time.Sleep(100 * time.Millisecond)
	msgs := st.GetMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Subject != "Test Subject" {
		t.Fatalf("expected subject 'Test Subject', got '%s'", msgs[0].Subject)
	}
	if msgs[0].From != "sender@test.com" {
		t.Fatalf("expected from 'sender@test.com', got '%s'", msgs[0].From)
	}
}

func TestSMTPMultipleRecipients(t *testing.T) {
	srv, st, addr := startTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", addr)
	defer conn.Close()

	readLine(conn) // greeting
	writeLine(conn, "EHLO test")
	readMultiLine(conn)
	writeLine(conn, "MAIL FROM:<sender@test.com>")
	readLine(conn)
	writeLine(conn, "RCPT TO:<a@test.com>")
	readLine(conn)
	writeLine(conn, "RCPT TO:<b@test.com>")
	readLine(conn)
	writeLine(conn, "DATA")
	readLine(conn)
	writeLine(conn, "Subject: Multi")
	writeLine(conn, "")
	writeLine(conn, "body")
	writeLine(conn, ".")
	readLine(conn)
	writeLine(conn, "QUIT")
	readLine(conn)

	time.Sleep(100 * time.Millisecond)
	msgs := st.GetMessages()
	if len(msgs) != 1 {
		t.Fatal("expected 1 message")
	}
	if len(msgs[0].To) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(msgs[0].To))
	}
}

func TestSMTPRSET(t *testing.T) {
	srv, _, addr := startTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", addr)
	defer conn.Close()

	readLine(conn)
	writeLine(conn, "EHLO test")
	readMultiLine(conn)
	writeLine(conn, "MAIL FROM:<sender@test.com>")
	readLine(conn)
	writeLine(conn, "RSET")
	resp := readLine(conn)
	if !strings.HasPrefix(resp, "250") {
		t.Fatalf("expected 250 for RSET, got: %s", resp)
	}
	writeLine(conn, "QUIT")
	readLine(conn)
}

func TestSMTPNOOP(t *testing.T) {
	srv, _, addr := startTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", addr)
	defer conn.Close()

	readLine(conn)
	writeLine(conn, "EHLO test")
	readMultiLine(conn)
	writeLine(conn, "NOOP")
	resp := readLine(conn)
	if !strings.HasPrefix(resp, "250") {
		t.Fatalf("expected 250 for NOOP, got: %s", resp)
	}
	writeLine(conn, "QUIT")
	readLine(conn)
}

func TestSMTPVRFY(t *testing.T) {
	srv, _, addr := startTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", addr)
	defer conn.Close()

	readLine(conn)
	writeLine(conn, "EHLO test")
	readMultiLine(conn)
	writeLine(conn, "VRFY user@test.com")
	resp := readLine(conn)
	if !strings.HasPrefix(resp, "252") {
		t.Fatalf("expected 252 for VRFY, got: %s", resp)
	}
	writeLine(conn, "QUIT")
	readLine(conn)
}

func TestSMTPAUTH(t *testing.T) {
	srv, _, addr := startTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", addr)
	defer conn.Close()

	readLine(conn)
	writeLine(conn, "EHLO test")
	readMultiLine(conn)
	writeLine(conn, "AUTH PLAIN dGVzdAB0ZXN0AHBhc3M=")
	resp := readLine(conn)
	if !strings.HasPrefix(resp, "235") {
		t.Fatalf("expected 235 for AUTH, got: %s", resp)
	}
	writeLine(conn, "QUIT")
	readLine(conn)
}

func TestSMTPUnknownCommand(t *testing.T) {
	srv, _, addr := startTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", addr)
	defer conn.Close()

	readLine(conn)
	writeLine(conn, "EHLO test")
	readMultiLine(conn)
	writeLine(conn, "FOOBAR")
	resp := readLine(conn)
	if !strings.HasPrefix(resp, "500") {
		t.Fatalf("expected 500 for unknown cmd, got: %s", resp)
	}
	writeLine(conn, "QUIT")
	readLine(conn)
}

func TestSMTPHELO(t *testing.T) {
	srv, _, addr := startTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", addr)
	defer conn.Close()

	readLine(conn)
	writeLine(conn, "HELO test")
	resp := readMultiLine(conn)
	if !strings.Contains(resp, "250") {
		t.Fatalf("expected 250 for HELO, got: %s", resp)
	}
	writeLine(conn, "QUIT")
	readLine(conn)
}

func TestSMTPHTMLMessage(t *testing.T) {
	srv, st, addr := startTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", addr)
	defer conn.Close()

	readLine(conn)
	writeLine(conn, "EHLO test")
	readMultiLine(conn)
	writeLine(conn, "MAIL FROM:<s@t.com>")
	readLine(conn)
	writeLine(conn, "RCPT TO:<r@t.com>")
	readLine(conn)
	writeLine(conn, "DATA")
	readLine(conn)
	writeLine(conn, "Subject: HTML Test")
	writeLine(conn, "Content-Type: text/html")
	writeLine(conn, "")
	writeLine(conn, "<h1>Hello</h1>")
	writeLine(conn, ".")
	readLine(conn)
	writeLine(conn, "QUIT")
	readLine(conn)

	time.Sleep(100 * time.Millisecond)
	msgs := st.GetMessages()
	if len(msgs) != 1 {
		t.Fatal("expected 1 message")
	}
	if !strings.Contains(msgs[0].HTMLBody, "<h1>Hello</h1>") {
		t.Fatalf("expected HTML body, got: %s", msgs[0].HTMLBody)
	}
}

func TestSMTPMultipartMessage(t *testing.T) {
	srv, st, addr := startTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", addr)
	defer conn.Close()

	readLine(conn)
	writeLine(conn, "EHLO test")
	readMultiLine(conn)
	writeLine(conn, "MAIL FROM:<s@t.com>")
	readLine(conn)
	writeLine(conn, "RCPT TO:<r@t.com>")
	readLine(conn)
	writeLine(conn, "DATA")
	readLine(conn)

	msg := "Subject: Multipart Test\r\n" +
		"Content-Type: multipart/alternative; boundary=\"boundary42\"\r\n\r\n" +
		"--boundary42\r\n" +
		"Content-Type: text/plain\r\n\r\n" +
		"Plain text body\r\n" +
		"--boundary42\r\n" +
		"Content-Type: text/html\r\n\r\n" +
		"<p>HTML body</p>\r\n" +
		"--boundary42--\r\n"

	conn.Write([]byte(msg))
	writeLine(conn, ".")
	readLine(conn)
	writeLine(conn, "QUIT")
	readLine(conn)

	time.Sleep(100 * time.Millisecond)
	msgs := st.GetMessages()
	if len(msgs) != 1 {
		t.Fatal("expected 1 message")
	}
	if !strings.Contains(msgs[0].TextBody, "Plain text") {
		t.Fatalf("expected text body, got: %s", msgs[0].TextBody)
	}
	if !strings.Contains(msgs[0].HTMLBody, "HTML body") {
		t.Fatalf("expected html body, got: %s", msgs[0].HTMLBody)
	}
}

func TestSMTPSessionLog(t *testing.T) {
	srv, st, addr := startTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", addr)
	defer conn.Close()

	readLine(conn)
	writeLine(conn, "EHLO test")
	readMultiLine(conn)
	writeLine(conn, "QUIT")
	readLine(conn)

	time.Sleep(100 * time.Millisecond)
	sessions := st.GetSessions()
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Log == "" {
		t.Fatal("expected session log to be non-empty")
	}
	if !strings.Contains(sessions[0].Log, "EHLO") {
		t.Fatal("expected session log to contain EHLO")
	}
}

func TestExtractAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"MAIL FROM:<user@example.com>", "user@example.com"},
		{"RCPT TO:<admin@test.org>", "admin@test.org"},
		{"MAIL FROM: user@example.com", "user@example.com"},
		{"RCPT TO:<>", ""},
	}

	for _, tt := range tests {
		result := extractAddress(tt.input)
		if result != tt.expected {
			t.Errorf("extractAddress(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestDecodeHeader(t *testing.T) {
	// Plain header
	if decodeHeader("Hello") != "Hello" {
		t.Fatal("plain header should pass through")
	}

	// Encoded header
	encoded := "=?UTF-8?Q?Hello_World?="
	result := decodeHeader(encoded)
	if result != "Hello World" {
		t.Fatalf("expected 'Hello World', got '%s'", result)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()
	if id1 == id2 {
		t.Fatal("generated IDs should be unique")
	}
	if len(id1) != 32 {
		t.Fatalf("expected 32 char hex ID, got %d chars", len(id1))
	}
}

func TestParseBodyPlainText(t *testing.T) {
	raw := "Content-Type: text/plain\r\n\r\nHello plain"
	msg, _ := mail.ReadMessage(strings.NewReader(raw))
	html, text, atts := parseBody(msg)
	if html != "" {
		t.Fatal("expected no html")
	}
	if text != "Hello plain" {
		t.Fatalf("expected 'Hello plain', got '%s'", text)
	}
	if len(atts) != 0 {
		t.Fatal("expected no attachments")
	}
}

func TestParseBodyHTML(t *testing.T) {
	raw := "Content-Type: text/html\r\n\r\n<b>Hello</b>"
	msg, _ := mail.ReadMessage(strings.NewReader(raw))
	html, text, _ := parseBody(msg)
	if html != "<b>Hello</b>" {
		t.Fatalf("expected html, got '%s'", html)
	}
	if text != "" {
		t.Fatal("expected no text")
	}
}

func TestParseBodyNoContentType(t *testing.T) {
	raw := "Subject: test\r\n\r\nPlain body"
	msg, _ := mail.ReadMessage(strings.NewReader(raw))
	_, text, _ := parseBody(msg)
	if text != "Plain body" {
		t.Fatalf("expected plain body, got '%s'", text)
	}
}

func TestParseBodyBadContentType(t *testing.T) {
	raw := "Content-Type: ;;;bad\r\n\r\nFallback"
	msg, _ := mail.ReadMessage(strings.NewReader(raw))
	_, text, _ := parseBody(msg)
	if text != "Fallback" {
		t.Fatalf("expected fallback text, got '%s'", text)
	}
}

func TestParseMultipartWithAttachment(t *testing.T) {
	boundary := "boundary123"
	body := fmt.Sprintf("--%s\r\nContent-Type: text/plain\r\n\r\nText part\r\n--%s\r\nContent-Type: application/pdf\r\nContent-Disposition: attachment; filename=\"test.pdf\"\r\n\r\nPDF content\r\n--%s--\r\n", boundary, boundary, boundary)

	html, text, atts := parseMultipart(strings.NewReader(body), boundary)
	if html != "" {
		t.Fatal("expected no html")
	}
	if text != "Text part" {
		t.Fatalf("expected text part, got '%s'", text)
	}
	if len(atts) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(atts))
	}
	if atts[0].Filename != "test.pdf" {
		t.Fatalf("expected test.pdf, got '%s'", atts[0].Filename)
	}
}

func TestSMTPUnparseableMessage(t *testing.T) {
	srv, st, addr := startTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", addr)
	defer conn.Close()

	readLine(conn)
	writeLine(conn, "EHLO test")
	readMultiLine(conn)
	writeLine(conn, "MAIL FROM:<s@t.com>")
	readLine(conn)
	writeLine(conn, "RCPT TO:<r@t.com>")
	readLine(conn)
	writeLine(conn, "DATA")
	readLine(conn)

	// Send data that is not valid RFC822 (no headers, just garbage)
	conn.Write([]byte("this is not a valid email at all\r\n.\r\n"))
	readLine(conn)
	writeLine(conn, "QUIT")
	readLine(conn)

	time.Sleep(100 * time.Millisecond)
	msgs := st.GetMessages()
	if len(msgs) != 1 {
		t.Fatal("expected 1 message")
	}
	if msgs[0].Subject != "(unparseable)" {
		t.Fatalf("expected unparseable subject, got: %s", msgs[0].Subject)
	}
}

func TestExtractAddressNoColon(t *testing.T) {
	result := extractAddress("justplaintext")
	if result != "" {
		t.Fatalf("expected empty, got %q", result)
	}
}

func TestDecodeHeaderInvalid(t *testing.T) {
	// Invalid encoded-word should return original string
	result := decodeHeader("=?INVALID?Q?broken")
	if result != "=?INVALID?Q?broken" {
		t.Fatalf("expected original string, got %q", result)
	}
}

func TestSMTPConnectionClose(t *testing.T) {
	srv, _, addr := startTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", addr)
	readLine(conn)
	// Close connection without QUIT
	conn.Close()

	time.Sleep(100 * time.Millisecond)
	// Should not crash and session should be recorded
}

func TestParseNestedMultipart(t *testing.T) {
	inner := "--inner\r\nContent-Type: text/plain\r\n\r\nInner text\r\n--inner\r\nContent-Type: text/html\r\n\r\n<b>Inner HTML</b>\r\n--inner--\r\n"
	outer := fmt.Sprintf("--outer\r\nContent-Type: multipart/alternative; boundary=\"inner\"\r\n\r\n%s\r\n--outer--\r\n", inner)

	html, text, _ := parseMultipart(strings.NewReader(outer), "outer")
	if text != "Inner text" {
		t.Fatalf("expected 'Inner text', got '%s'", text)
	}
	if html != "<b>Inner HTML</b>" {
		t.Fatalf("expected inner HTML, got '%s'", html)
	}
}
