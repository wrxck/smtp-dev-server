package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/wrxck/smtp-dev-server/internal/store"
)

type Server struct {
	addr     string
	store    *store.Store
	mux      *http.ServeMux
	server   *http.Server
	sseClients map[chan struct{}]struct{}
	sseMu      sync.Mutex
}

func NewServer(addr string, s *store.Store) *Server {
	ws := &Server{
		addr:       addr,
		store:      s,
		mux:        http.NewServeMux(),
		sseClients: make(map[chan struct{}]struct{}),
	}
	ws.routes()
	ws.store.OnChange(ws.notifyClients)
	return ws
}

func (s *Server) notifyClients() {
	s.sseMu.Lock()
	defer s.sseMu.Unlock()
	for ch := range s.sseClients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.handleUI)
	s.mux.HandleFunc("GET /api/messages", s.handleGetMessages)
	s.mux.HandleFunc("GET /api/messages/{id}", s.handleGetMessage)
	s.mux.HandleFunc("GET /api/messages/{id}/html", s.handleGetMessageHTML)
	s.mux.HandleFunc("GET /api/messages/{id}/text", s.handleGetMessageText)
	s.mux.HandleFunc("GET /api/messages/{id}/raw", s.handleGetMessageRaw)
	s.mux.HandleFunc("GET /api/messages/{id}/headers", s.handleGetMessageHeaders)
	s.mux.HandleFunc("GET /api/messages/{id}/attachments", s.handleGetMessageAttachments)
	s.mux.HandleFunc("GET /api/messages/{id}/attachments/{index}", s.handleGetAttachment)
	s.mux.HandleFunc("DELETE /api/messages/{id}", s.handleDeleteMessage)
	s.mux.HandleFunc("DELETE /api/messages", s.handleDeleteAllMessages)
	s.mux.HandleFunc("POST /api/messages/{id}/read", s.handleMarkRead)
	s.mux.HandleFunc("GET /api/sessions", s.handleGetSessions)
	s.mux.HandleFunc("GET /api/sessions/{id}", s.handleGetSession)
	s.mux.HandleFunc("GET /api/mailboxes", s.handleGetMailboxes)
	s.mux.HandleFunc("GET /api/mailboxes/{address}/messages", s.handleGetMailboxMessages)
	s.mux.HandleFunc("GET /api/events", s.handleSSE)
}

func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:    s.addr,
		Handler: s.mux,
	}

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("web: listen %s: %w", s.addr, err)
	}

	log.Printf("Web UI available at http://%s", ln.Addr().String())

	go s.server.Serve(ln)
	return nil
}

func (s *Server) Stop() {
	if s.server != nil {
		s.server.Close()
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	// optional ?to= filter shows messages addressed to a single recipient.
	// case-insensitive; matches any address in the to list.
	if to := strings.TrimSpace(r.URL.Query().Get("to")); to != "" {
		writeJSON(w, s.store.MessagesForMailbox(to))
		return
	}
	writeJSON(w, s.store.GetMessages())
}

// handleGetMailboxes returns the list of distinct recipient addresses
// (lowercased) with per-mailbox message and unread counts.
func (s *Server) handleGetMailboxes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.store.Mailboxes())
}

// handleGetMailboxMessages returns the messages addressed to a single
// recipient. address must be in the path and is treated as opaque.
func (s *Server) handleGetMailboxMessages(w http.ResponseWriter, r *http.Request) {
	addr := r.PathValue("address")
	if addr == "" {
		http.Error(w, "address required", http.StatusBadRequest)
		return
	}
	writeJSON(w, s.store.MessagesForMailbox(addr))
}

func (s *Server) handleGetMessage(w http.ResponseWriter, r *http.Request) {
	msg := s.store.GetMessage(r.PathValue("id"))
	if msg == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, msg)
}

func (s *Server) handleGetMessageHTML(w http.ResponseWriter, r *http.Request) {
	msg := s.store.GetMessage(r.PathValue("id"))
	if msg == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(msg.HTMLBody))
}

func (s *Server) handleGetMessageText(w http.ResponseWriter, r *http.Request) {
	msg := s.store.GetMessage(r.PathValue("id"))
	if msg == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(msg.TextBody))
}

func (s *Server) handleGetMessageRaw(w http.ResponseWriter, r *http.Request) {
	msg := s.store.GetMessage(r.PathValue("id"))
	if msg == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "message/rfc822")
	w.Write(msg.RawData)
}

func (s *Server) handleGetMessageHeaders(w http.ResponseWriter, r *http.Request) {
	msg := s.store.GetMessage(r.PathValue("id"))
	if msg == nil {
		http.NotFound(w, r)
		return
	}
	type header struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	var headers []header
	for _, h := range msg.Headers {
		headers = append(headers, header{Name: h[0], Value: h[1]})
	}
	writeJSON(w, headers)
}

func (s *Server) handleGetMessageAttachments(w http.ResponseWriter, r *http.Request) {
	msg := s.store.GetMessage(r.PathValue("id"))
	if msg == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, msg.Attachments)
}

func (s *Server) handleGetAttachment(w http.ResponseWriter, r *http.Request) {
	msg := s.store.GetMessage(r.PathValue("id"))
	if msg == nil {
		http.NotFound(w, r)
		return
	}
	var idx int
	fmt.Sscanf(r.PathValue("index"), "%d", &idx)
	if idx < 0 || idx >= len(msg.Attachments) {
		http.NotFound(w, r)
		return
	}
	att := msg.Attachments[idx]
	w.Header().Set("Content-Type", att.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", att.Filename))
	w.Write(att.Content)
}

func (s *Server) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	if s.store.DeleteMessage(r.PathValue("id")) {
		w.WriteHeader(http.StatusNoContent)
	} else {
		http.NotFound(w, r)
	}
}

func (s *Server) handleDeleteAllMessages(w http.ResponseWriter, r *http.Request) {
	count := s.store.DeleteAllMessages()
	writeJSON(w, map[string]int{"deleted": count})
}

func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	s.store.MarkRead(r.PathValue("id"))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetSessions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.store.GetSessions())
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sess := s.store.GetSession(r.PathValue("id"))
	if sess == nil {
		http.NotFound(w, r)
		return
	}
	type sessionDetail struct {
		*store.Session
		Log string `json:"log"`
	}
	writeJSON(w, sessionDetail{Session: sess, Log: sess.Log})
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan struct{}, 1)
	s.sseMu.Lock()
	s.sseClients[ch] = struct{}{}
	s.sseMu.Unlock()

	defer func() {
		s.sseMu.Lock()
		delete(s.sseClients, ch)
		s.sseMu.Unlock()
	}()

	// Send initial count
	fmt.Fprintf(w, "data: {\"type\":\"update\",\"messageCount\":%d,\"unreadCount\":%d}\n\n",
		s.store.MessageCount(), s.store.UnreadCount())
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ch:
			fmt.Fprintf(w, "data: {\"type\":\"update\",\"messageCount\":%d,\"unreadCount\":%d}\n\n",
				s.store.MessageCount(), s.store.UnreadCount())
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
