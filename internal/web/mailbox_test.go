package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/wrxck/smtp-dev-server/internal/store"
)

func newTestWebServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	st := store.New(0)
	ws := NewServer(":0", st)
	srv := httptest.NewServer(ws.mux)
	t.Cleanup(srv.Close)
	return srv, st
}

func TestHandleGetMailboxes(t *testing.T) {
	srv, st := newTestWebServer(t)

	st.AddMessage(&store.Message{ID: "1", From: "x@y", To: []string{"alice@x"}, Subject: "a", ReceivedAt: time.Now()})
	st.AddMessage(&store.Message{ID: "2", From: "x@y", To: []string{"alice@x"}, Subject: "b", ReceivedAt: time.Now()})
	st.AddMessage(&store.Message{ID: "3", From: "x@y", To: []string{"bob@x"}, Subject: "c", ReceivedAt: time.Now()})

	resp, err := http.Get(srv.URL + "/api/mailboxes")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var mbs []store.MailboxSummary
	if err := json.Unmarshal(body, &mbs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(mbs) != 2 {
		t.Errorf("expected 2 mailboxes; got %d", len(mbs))
	}
}

func TestHandleGetMailboxMessages(t *testing.T) {
	srv, st := newTestWebServer(t)
	st.AddMessage(&store.Message{ID: "1", To: []string{"alice@x"}, Subject: "for alice", ReceivedAt: time.Now()})
	st.AddMessage(&store.Message{ID: "2", To: []string{"bob@x"}, Subject: "for bob", ReceivedAt: time.Now()})

	resp, _ := http.Get(srv.URL + "/api/mailboxes/" + url.PathEscape("alice@x") + "/messages")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var msgs []*store.Message
	json.Unmarshal(body, &msgs)
	if len(msgs) != 1 || msgs[0].Subject != "for alice" {
		t.Errorf("expected only alice's message; got %+v", msgs)
	}
}

func TestHandleGetMessages_WithToFilter(t *testing.T) {
	srv, st := newTestWebServer(t)
	st.AddMessage(&store.Message{ID: "1", To: []string{"alice@x"}, Subject: "for alice", ReceivedAt: time.Now()})
	st.AddMessage(&store.Message{ID: "2", To: []string{"bob@x"}, Subject: "for bob", ReceivedAt: time.Now()})

	resp, _ := http.Get(srv.URL + "/api/messages?to=" + url.QueryEscape("bob@x"))
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var msgs []*store.Message
	json.Unmarshal(body, &msgs)
	if len(msgs) != 1 || msgs[0].Subject != "for bob" {
		t.Errorf("?to=bob@x should return only bob's; got %+v", msgs)
	}
}
