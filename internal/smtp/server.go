package smtp

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"strings"
	"time"

	"github.com/wrxck/smtp-dev-server/internal/store"
)

type Server struct {
	addr     string
	store    *store.Store
	listener net.Listener
	quit     chan struct{}
}

func NewServer(addr string, s *store.Store) *Server {
	return &Server{
		addr:  addr,
		store: s,
		quit:  make(chan struct{}),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("smtp: listen %s: %w", s.addr, err)
	}
	s.listener = ln
	log.Printf("SMTP server listening on %s", s.addr)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-s.quit:
					return
				default:
					log.Printf("smtp: accept error: %v", err)
					continue
				}
			}
			go s.handleConnection(conn)
		}
	}()
	return nil
}

func (s *Server) Stop() {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type session struct {
	conn     net.Conn
	server   *Server
	from     string
	to       []string
	data     []byte
	log      strings.Builder
	msgCount int
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Minute))

	sess := &session{conn: conn, server: s}
	sessRecord := &store.Session{
		ID:         generateID(),
		ClientAddr: conn.RemoteAddr().String(),
		StartedAt:  time.Now(),
	}

	sess.writeLine("220 smtp-dev-server ESMTP ready")
	sess.logLine("S: 220 smtp-dev-server ESMTP ready")

	for {
		line, err := sess.readLine()
		if err != nil {
			break
		}
		sess.logLine("C: %s", line)

		cmd := strings.ToUpper(strings.TrimSpace(line))
		if len(cmd) > 4 {
			cmd = cmd[:4]
		}

		switch cmd {
		case "EHLO", "HELO":
			sess.writeLine("250-smtp-dev-server Hello")
			sess.writeLine("250-SIZE 52428800")
			sess.writeLine("250-8BITMIME")
			sess.writeLine("250-SMTPUTF8")
			sess.writeLine("250 OK")
			sess.logLine("S: 250 OK (EHLO)")

		case "MAIL":
			sess.from = extractAddress(line)
			sess.to = nil
			sess.data = nil
			sess.writeLine("250 OK")
			sess.logLine("S: 250 OK (MAIL FROM: %s)", sess.from)

		case "RCPT":
			addr := extractAddress(line)
			sess.to = append(sess.to, addr)
			sess.writeLine("250 OK")
			sess.logLine("S: 250 OK (RCPT TO: %s)", addr)

		case "DATA":
			sess.writeLine("354 Start mail input; end with <CRLF>.<CRLF>")
			sess.logLine("S: 354 Start mail input")
			data, err := sess.readData()
			if err != nil {
				sess.logLine("Error reading data: %v", err)
				break
			}
			sess.data = data
			sess.processMessage()
			sess.msgCount++
			sess.writeLine("250 OK: message queued")
			sess.logLine("S: 250 OK: message queued")

		case "RSET":
			sess.from = ""
			sess.to = nil
			sess.data = nil
			sess.writeLine("250 OK")

		case "NOOP":
			sess.writeLine("250 OK")

		case "QUIT":
			sess.writeLine("221 Bye")
			sess.logLine("S: 221 Bye")
			goto done

		case "VRFY":
			sess.writeLine("252 Cannot VRFY user")

		case "AUTH":
			sess.writeLine("235 2.7.0 Authentication successful")
			sess.logLine("S: 235 Authentication accepted (dev mode)")

		default:
			sess.writeLine("500 Command not recognized")
			sess.logLine("S: 500 Command not recognized")
		}
	}

done:
	sessRecord.EndedAt = time.Now()
	sessRecord.MessageCount = sess.msgCount
	sessRecord.Log = sess.log.String()
	s.store.AddSession(sessRecord)
}

func (sess *session) writeLine(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	sess.conn.Write([]byte(line + "\r\n"))
}

func (sess *session) readLine() (string, error) {
	var buf bytes.Buffer
	one := make([]byte, 1)
	for {
		_, err := sess.conn.Read(one)
		if err != nil {
			return "", err
		}
		if one[0] == '\n' {
			return strings.TrimRight(buf.String(), "\r"), nil
		}
		buf.WriteByte(one[0])
	}
}

func (sess *session) readData() ([]byte, error) {
	var buf bytes.Buffer
	one := make([]byte, 1)
	for {
		_, err := sess.conn.Read(one)
		if err != nil {
			return nil, err
		}
		buf.WriteByte(one[0])
		if bytes.HasSuffix(buf.Bytes(), []byte("\r\n.\r\n")) {
			data := buf.Bytes()
			return data[:len(data)-5], nil
		}
	}
}

func (sess *session) logLine(format string, args ...any) {
	sess.log.WriteString(fmt.Sprintf(format, args...))
	sess.log.WriteString("\n")
}

func (sess *session) processMessage() {
	msg, err := mail.ReadMessage(bytes.NewReader(sess.data))
	if err != nil {
		log.Printf("smtp: failed to parse message: %v", err)
		sess.storeRawMessage()
		return
	}

	subject := decodeHeader(msg.Header.Get("Subject"))

	var headers [][2]string
	for key, vals := range msg.Header {
		for _, v := range vals {
			headers = append(headers, [2]string{key, v})
		}
	}

	htmlBody, textBody, attachments := parseBody(msg)

	record := &store.Message{
		ID:          generateID(),
		From:        sess.from,
		To:          sess.to,
		Subject:     subject,
		ReceivedAt:  time.Now(),
		Size:        len(sess.data),
		RawData:     sess.data,
		HTMLBody:    htmlBody,
		TextBody:    textBody,
		Headers:     headers,
		Attachments: attachments,
	}
	sess.server.store.AddMessage(record)
	log.Printf("Received message: %q from %s to %v", subject, sess.from, sess.to)
}

func (sess *session) storeRawMessage() {
	record := &store.Message{
		ID:         generateID(),
		From:       sess.from,
		To:         sess.to,
		Subject:    "(unparseable)",
		ReceivedAt: time.Now(),
		Size:       len(sess.data),
		RawData:    sess.data,
		TextBody:   string(sess.data),
	}
	sess.server.store.AddMessage(record)
}

func extractAddress(line string) string {
	start := strings.Index(line, "<")
	end := strings.Index(line, ">")
	if start >= 0 && end > start {
		return line[start+1 : end]
	}
	parts := strings.SplitN(line, ":", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func decodeHeader(s string) string {
	dec := new(mime.WordDecoder)
	result, err := dec.DecodeHeader(s)
	if err != nil {
		return s
	}
	return result
}

func parseBody(msg *mail.Message) (html, text string, attachments []store.Attachment) {
	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		body, _ := io.ReadAll(msg.Body)
		return "", string(body), nil
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary != "" {
			return parseMultipart(msg.Body, boundary)
		}
	}

	body, _ := io.ReadAll(msg.Body)
	if strings.HasPrefix(mediaType, "text/html") {
		return string(body), "", nil
	}
	return "", string(body), nil
}

func parseMultipart(r io.Reader, boundary string) (html, text string, attachments []store.Attachment) {
	mr := multipart.NewReader(r, boundary)
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		partData, err := io.ReadAll(part)
		if err != nil {
			continue
		}

		ct := part.Header.Get("Content-Type")
		cd := part.Header.Get("Content-Disposition")
		mediaType, params, _ := mime.ParseMediaType(ct)

		if strings.Contains(cd, "attachment") || (cd != "" && !strings.Contains(cd, "inline") && part.FileName() != "") {
			attachments = append(attachments, store.Attachment{
				Filename:    part.FileName(),
				ContentType: ct,
				Size:        len(partData),
				Content:     partData,
			})
			continue
		}

		if strings.HasPrefix(mediaType, "multipart/") {
			h, t, a := parseMultipart(bytes.NewReader(partData), params["boundary"])
			if h != "" {
				html = h
			}
			if t != "" {
				text = t
			}
			attachments = append(attachments, a...)
			continue
		}

		if strings.HasPrefix(mediaType, "text/html") {
			html = string(partData)
		} else if strings.HasPrefix(mediaType, "text/plain") {
			text = string(partData)
		}
	}
	return
}
