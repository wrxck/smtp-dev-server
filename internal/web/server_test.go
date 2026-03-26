package web

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wrxck/smtp-dev-server/internal/store"
)

func newTestServer() (*Server, *store.Store) {
	s := store.New(100)
	ws := NewServer("127.0.0.1:0", s)
	return ws, s
}

func addTestMessage(s *store.Store, id, subject, from, htmlBody, textBody string) {
	s.AddMessage(&store.Message{
		ID:         id,
		From:       from,
		To:         []string{"to@test.com"},
		Subject:    subject,
		ReceivedAt: time.Now(),
		Size:       100,
		RawData:    []byte("From: " + from + "\r\nSubject: " + subject + "\r\n\r\n" + textBody),
		HTMLBody:   htmlBody,
		TextBody:   textBody,
		Headers:    [][2]string{{"From", from}, {"Subject", subject}},
		Attachments: []store.Attachment{
			{Filename: "test.txt", ContentType: "text/plain", Size: 5, Content: []byte("hello")},
		},
	})
}

func TestHandleGetMessages(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "Hello", "a@b.com", "<h1>Hi</h1>", "Hi")

	req := httptest.NewRequest("GET", "/api/messages", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var msgs []store.Message
	json.Unmarshal(w.Body.Bytes(), &msgs)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Subject != "Hello" {
		t.Fatalf("expected subject Hello, got %s", msgs[0].Subject)
	}
}

func TestHandleGetMessagesEmpty(t *testing.T) {
	ws, _ := newTestServer()

	req := httptest.NewRequest("GET", "/api/messages", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleGetMessage(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "Hello", "a@b.com", "", "text")

	req := httptest.NewRequest("GET", "/api/messages/msg1", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleGetMessageNotFound(t *testing.T) {
	ws, _ := newTestServer()

	req := httptest.NewRequest("GET", "/api/messages/nonexistent", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleGetMessageHTML(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "Test", "a@b.com", "<h1>HTML</h1>", "")

	req := httptest.NewRequest("GET", "/api/messages/msg1/html", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<h1>HTML</h1>") {
		t.Fatal("expected HTML content")
	}
	if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("wrong content-type: %s", w.Header().Get("Content-Type"))
	}
}

func TestHandleGetMessageHTMLNotFound(t *testing.T) {
	ws, _ := newTestServer()

	req := httptest.NewRequest("GET", "/api/messages/nonexistent/html", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleGetMessageText(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "Test", "a@b.com", "", "Plain text here")

	req := httptest.NewRequest("GET", "/api/messages/msg1/text", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Plain text here") {
		t.Fatal("expected text content")
	}
}

func TestHandleGetMessageTextNotFound(t *testing.T) {
	ws, _ := newTestServer()

	req := httptest.NewRequest("GET", "/api/messages/nonexistent/text", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleGetMessageRaw(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "Test", "a@b.com", "", "body")

	req := httptest.NewRequest("GET", "/api/messages/msg1/raw", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "message/rfc822" {
		t.Fatalf("wrong content type: %s", w.Header().Get("Content-Type"))
	}
}

func TestHandleGetMessageRawNotFound(t *testing.T) {
	ws, _ := newTestServer()

	req := httptest.NewRequest("GET", "/api/messages/nonexistent/raw", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleGetMessageHeaders(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "Test", "a@b.com", "", "")

	req := httptest.NewRequest("GET", "/api/messages/msg1/headers", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	type header struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	var headers []header
	json.Unmarshal(w.Body.Bytes(), &headers)
	if len(headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(headers))
	}
}

func TestHandleGetMessageHeadersNotFound(t *testing.T) {
	ws, _ := newTestServer()

	req := httptest.NewRequest("GET", "/api/messages/nonexistent/headers", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleGetMessageAttachments(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "Test", "a@b.com", "", "")

	req := httptest.NewRequest("GET", "/api/messages/msg1/attachments", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleGetMessageAttachmentsNotFound(t *testing.T) {
	ws, _ := newTestServer()

	req := httptest.NewRequest("GET", "/api/messages/nonexistent/attachments", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleGetAttachment(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "Test", "a@b.com", "", "")

	req := httptest.NewRequest("GET", "/api/messages/msg1/attachments/0", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Fatalf("expected 'hello', got '%s'", w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Content-Disposition"), "test.txt") {
		t.Fatal("expected Content-Disposition with filename")
	}
}

func TestHandleGetAttachmentNotFoundMessage(t *testing.T) {
	ws, _ := newTestServer()

	req := httptest.NewRequest("GET", "/api/messages/nonexistent/attachments/0", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleGetAttachmentOutOfRange(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "Test", "a@b.com", "", "")

	req := httptest.NewRequest("GET", "/api/messages/msg1/attachments/99", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleDeleteMessage(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "Test", "a@b.com", "", "")

	req := httptest.NewRequest("DELETE", "/api/messages/msg1", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if s.MessageCount() != 0 {
		t.Fatal("expected 0 messages")
	}
}

func TestHandleDeleteMessageNotFound(t *testing.T) {
	ws, _ := newTestServer()

	req := httptest.NewRequest("DELETE", "/api/messages/nonexistent", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleDeleteAllMessages(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "A", "a@b.com", "", "")
	addTestMessage(s, "msg2", "B", "a@b.com", "", "")

	req := httptest.NewRequest("DELETE", "/api/messages", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result map[string]int
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["deleted"] != 2 {
		t.Fatalf("expected 2 deleted, got %d", result["deleted"])
	}
}

func TestHandleMarkRead(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "Test", "a@b.com", "", "")

	req := httptest.NewRequest("POST", "/api/messages/msg1/read", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	msg := s.GetMessage("msg1")
	if !msg.IsRead {
		t.Fatal("expected message to be read")
	}
}

func TestHandleGetSessions(t *testing.T) {
	ws, s := newTestServer()
	s.AddSession(&store.Session{ID: "s1", ClientAddr: "127.0.0.1:1234"})

	req := httptest.NewRequest("GET", "/api/sessions", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleGetSession(t *testing.T) {
	ws, s := newTestServer()
	s.AddSession(&store.Session{ID: "s1", ClientAddr: "127.0.0.1:1234", Log: "EHLO test"})

	req := httptest.NewRequest("GET", "/api/sessions/s1", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "EHLO test") {
		t.Fatal("expected session log in response")
	}
}

func TestHandleGetSessionNotFound(t *testing.T) {
	ws, _ := newTestServer()

	req := httptest.NewRequest("GET", "/api/sessions/nonexistent", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleUI(t *testing.T) {
	ws, _ := newTestServer()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("wrong content type: %s", w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Body.String(), "smtp-dev-server") {
		t.Fatal("expected UI HTML to contain app name")
	}
}

func TestHandleSSE(t *testing.T) {
	ws, s := newTestServer()

	// Start a real HTTP server for SSE testing
	ts := httptest.NewServer(ws.mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %s", resp.Header.Get("Content-Type"))
	}

	// Read initial event
	scanner := bufio.NewScanner(resp.Body)
	if scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			t.Fatalf("expected data: line, got: %s", line)
		}
	}

	// Add a message to trigger an update
	s.AddMessage(&store.Message{
		ID:      "sse-test",
		Subject: "SSE Test",
	})

	// Read the update event
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			if strings.Contains(line, "\"messageCount\":1") {
				break // success
			}
		}
	}
}

func TestServerStartStop(t *testing.T) {
	s := store.New(10)
	ws := NewServer("127.0.0.1:0", s)
	if err := ws.Start(); err != nil {
		t.Fatal(err)
	}
	ws.Stop()
}

func TestServerStartError(t *testing.T) {
	s := store.New(10)
	// Bind a port first
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	defer ln.Close()

	// Try to bind the same address
	ws := NewServer(addr, s)
	if err := ws.Start(); err == nil {
		ws.Stop()
		t.Fatal("expected error binding to occupied port")
	}
}

func TestNotifyClients(t *testing.T) {
	ws, _ := newTestServer()

	ch := make(chan struct{}, 1)
	ws.sseMu.Lock()
	ws.sseClients[ch] = struct{}{}
	ws.sseMu.Unlock()

	ws.notifyClients()

	select {
	case <-ch:
		// ok
	case <-time.After(time.Second):
		t.Fatal("client not notified")
	}
}

func TestNotifyClientsFullChannel(t *testing.T) {
	ws, _ := newTestServer()

	// Channel with no buffer that already has a pending message
	ch := make(chan struct{}, 1)
	ch <- struct{}{} // fill the buffer

	ws.sseMu.Lock()
	ws.sseClients[ch] = struct{}{}
	ws.sseMu.Unlock()

	// Should not block
	ws.notifyClients()
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, map[string]string{"key": "value"})

	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("wrong content type: %s", w.Header().Get("Content-Type"))
	}

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["key"] != "value" {
		t.Fatal("wrong JSON content")
	}
}

func TestHandleGetAttachmentNegativeIndex(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "Test", "a@b.com", "", "")

	// Negative index - Sscanf will parse it as 0 or fail, but we handle bounds checking
	req := httptest.NewRequest("GET", "/api/messages/msg1/attachments/-1", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	// -1 is out of range
	if w.Code != 404 {
		t.Fatalf("expected 404 for negative index, got %d", w.Code)
	}
}

func TestWebServerStopNilServer(t *testing.T) {
	s := store.New(10)
	ws := NewServer("127.0.0.1:0", s)
	// Stop without starting - should not panic
	ws.Stop()
}

func TestSSEStreamWriteOnClose(t *testing.T) {
	ws, _ := newTestServer()
	ts := httptest.NewServer(ws.mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}

	// Read initial event
	buf := make([]byte, 512)
	resp.Body.Read(buf)

	// Close response, triggering context cancellation
	resp.Body.Close()

	// Notify after close - should not panic
	ws.notifyClients()
}

func TestMultipleMessagesAPI(t *testing.T) {
	ws, s := newTestServer()
	for i := 0; i < 5; i++ {
		addTestMessage(s, fmt.Sprintf("msg%d", i), fmt.Sprintf("Subject %d", i), "a@b.com", "", "")
	}

	req := httptest.NewRequest("GET", "/api/messages", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	var msgs []store.Message
	json.Unmarshal(w.Body.Bytes(), &msgs)
	if len(msgs) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(msgs))
	}
}

func TestHandleGetMessageRawContent(t *testing.T) {
	ws, s := newTestServer()
	addTestMessage(s, "msg1", "Test", "a@b.com", "", "body text")

	req := httptest.NewRequest("GET", "/api/messages/msg1/raw", nil)
	w := httptest.NewRecorder()
	ws.mux.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	if !strings.Contains(string(body), "Subject: Test") {
		t.Fatal("raw body should contain headers")
	}
}
