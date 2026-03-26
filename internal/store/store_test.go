package store

import (
	"sync"
	"testing"
	"time"
)

func makeMsg(id, subject string) *Message {
	return &Message{
		ID:         id,
		From:       "test@example.com",
		To:         []string{"rcpt@example.com"},
		Subject:    subject,
		ReceivedAt: time.Now(),
		Size:       100,
		RawData:    []byte("raw"),
	}
}

func TestNew(t *testing.T) {
	s := New(10)
	if s.maxItems != 10 {
		t.Fatalf("expected maxItems=10, got %d", s.maxItems)
	}
	if len(s.messages) != 0 {
		t.Fatal("expected empty messages")
	}
}

func TestNewDefaultMaxItems(t *testing.T) {
	s := New(0)
	if s.maxItems != 500 {
		t.Fatalf("expected default maxItems=500, got %d", s.maxItems)
	}

	s2 := New(-1)
	if s2.maxItems != 500 {
		t.Fatalf("expected default maxItems=500, got %d", s2.maxItems)
	}
}

func TestAddMessage(t *testing.T) {
	s := New(10)
	msg := makeMsg("1", "Hello")
	s.AddMessage(msg)

	if s.MessageCount() != 1 {
		t.Fatalf("expected 1 message, got %d", s.MessageCount())
	}

	msgs := s.GetMessages()
	if len(msgs) != 1 {
		t.Fatal("expected 1 message in list")
	}
	if msgs[0].ID != "1" {
		t.Fatalf("expected id=1, got %s", msgs[0].ID)
	}
}

func TestAddMessagePrependsNewest(t *testing.T) {
	s := New(10)
	s.AddMessage(makeMsg("1", "First"))
	s.AddMessage(makeMsg("2", "Second"))

	msgs := s.GetMessages()
	if msgs[0].ID != "2" {
		t.Fatal("newest message should be first")
	}
}

func TestAddMessageMaxItems(t *testing.T) {
	s := New(3)
	s.AddMessage(makeMsg("1", "A"))
	s.AddMessage(makeMsg("2", "B"))
	s.AddMessage(makeMsg("3", "C"))
	s.AddMessage(makeMsg("4", "D"))

	if s.MessageCount() != 3 {
		t.Fatalf("expected 3 messages, got %d", s.MessageCount())
	}
	// Oldest (id=1) should have been evicted
	if s.GetMessage("1") != nil {
		t.Fatal("message 1 should have been evicted")
	}
}

func TestGetMessage(t *testing.T) {
	s := New(10)
	s.AddMessage(makeMsg("abc", "Test"))

	msg := s.GetMessage("abc")
	if msg == nil {
		t.Fatal("expected to find message")
	}
	if msg.Subject != "Test" {
		t.Fatalf("expected subject=Test, got %s", msg.Subject)
	}
}

func TestGetMessageNotFound(t *testing.T) {
	s := New(10)
	if s.GetMessage("nonexistent") != nil {
		t.Fatal("expected nil for nonexistent message")
	}
}

func TestMarkRead(t *testing.T) {
	s := New(10)
	s.AddMessage(makeMsg("1", "Unread"))

	if s.UnreadCount() != 1 {
		t.Fatal("expected 1 unread")
	}

	s.MarkRead("1")

	msg := s.GetMessage("1")
	if !msg.IsRead {
		t.Fatal("expected message to be marked read")
	}
	if s.UnreadCount() != 0 {
		t.Fatal("expected 0 unread")
	}
}

func TestMarkReadNonexistent(t *testing.T) {
	s := New(10)
	// Should not panic
	s.MarkRead("nonexistent")
}

func TestDeleteMessage(t *testing.T) {
	s := New(10)
	s.AddMessage(makeMsg("1", "Delete me"))

	ok := s.DeleteMessage("1")
	if !ok {
		t.Fatal("expected delete to return true")
	}
	if s.MessageCount() != 0 {
		t.Fatal("expected 0 messages after delete")
	}
}

func TestDeleteMessageNotFound(t *testing.T) {
	s := New(10)
	ok := s.DeleteMessage("nonexistent")
	if ok {
		t.Fatal("expected delete to return false for nonexistent")
	}
}

func TestDeleteAllMessages(t *testing.T) {
	s := New(10)
	s.AddMessage(makeMsg("1", "A"))
	s.AddMessage(makeMsg("2", "B"))
	s.AddMessage(makeMsg("3", "C"))

	count := s.DeleteAllMessages()
	if count != 3 {
		t.Fatalf("expected 3 deleted, got %d", count)
	}
	if s.MessageCount() != 0 {
		t.Fatal("expected 0 messages")
	}
}

func TestAddSession(t *testing.T) {
	s := New(10)
	sess := &Session{
		ID:          "s1",
		ClientAddr:  "127.0.0.1:1234",
		StartedAt:   time.Now(),
		EndedAt:     time.Now(),
		MessageCount: 1,
		Log:         "test log",
	}
	s.AddSession(sess)

	sessions := s.GetSessions()
	if len(sessions) != 1 {
		t.Fatal("expected 1 session")
	}
}

func TestAddSessionMaxItems(t *testing.T) {
	s := New(2)
	s.AddSession(&Session{ID: "s1"})
	s.AddSession(&Session{ID: "s2"})
	s.AddSession(&Session{ID: "s3"})

	sessions := s.GetSessions()
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if s.GetSession("s1") != nil {
		t.Fatal("oldest session should have been evicted")
	}
}

func TestGetSession(t *testing.T) {
	s := New(10)
	s.AddSession(&Session{ID: "s1", ClientAddr: "1.2.3.4:5678"})

	sess := s.GetSession("s1")
	if sess == nil {
		t.Fatal("expected to find session")
	}
	if sess.ClientAddr != "1.2.3.4:5678" {
		t.Fatalf("wrong client addr: %s", sess.ClientAddr)
	}
}

func TestGetSessionNotFound(t *testing.T) {
	s := New(10)
	if s.GetSession("nonexistent") != nil {
		t.Fatal("expected nil")
	}
}

func TestUnreadCount(t *testing.T) {
	s := New(10)
	s.AddMessage(makeMsg("1", "A"))
	s.AddMessage(makeMsg("2", "B"))
	s.AddMessage(makeMsg("3", "C"))

	if s.UnreadCount() != 3 {
		t.Fatalf("expected 3 unread, got %d", s.UnreadCount())
	}

	s.MarkRead("2")
	if s.UnreadCount() != 2 {
		t.Fatalf("expected 2 unread, got %d", s.UnreadCount())
	}
}

func TestOnChange(t *testing.T) {
	s := New(10)
	called := make(chan bool, 10)
	s.OnChange(func() {
		called <- true
	})

	s.AddMessage(makeMsg("1", "A"))

	select {
	case <-called:
		// ok
	case <-time.After(time.Second):
		t.Fatal("onChange not called on AddMessage")
	}

	s.DeleteMessage("1")
	select {
	case <-called:
		// ok
	case <-time.After(time.Second):
		t.Fatal("onChange not called on DeleteMessage")
	}
}

func TestOnChangeDeleteAll(t *testing.T) {
	s := New(10)
	called := make(chan bool, 10)
	s.OnChange(func() {
		called <- true
	})

	s.AddMessage(makeMsg("1", "A"))
	<-called // consume AddMessage notification

	s.DeleteAllMessages()
	select {
	case <-called:
		// ok
	case <-time.After(time.Second):
		t.Fatal("onChange not called on DeleteAllMessages")
	}
}

func TestGetMessagesCopy(t *testing.T) {
	s := New(10)
	s.AddMessage(makeMsg("1", "A"))

	msgs := s.GetMessages()
	msgs[0] = nil // modify the copy

	// Original should be unaffected
	if s.GetMessage("1") == nil {
		t.Fatal("original message should not be affected by modifying copy")
	}
}

func TestGetSessionsCopy(t *testing.T) {
	s := New(10)
	s.AddSession(&Session{ID: "s1"})

	sessions := s.GetSessions()
	sessions[0] = nil

	if s.GetSession("s1") == nil {
		t.Fatal("original session should not be affected by modifying copy")
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := New(100)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			s.AddMessage(makeMsg(id, "subject"))
		}(string(rune('a' + i)))
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.GetMessages()
			s.MessageCount()
			s.UnreadCount()
		}()
	}

	wg.Wait()
}

func TestOnChangeNil(t *testing.T) {
	s := New(10)
	// No onChange set - should not panic
	s.AddMessage(makeMsg("1", "A"))
	s.DeleteMessage("1")
	s.DeleteAllMessages()
}

func TestSessionPrependsNewest(t *testing.T) {
	s := New(10)
	s.AddSession(&Session{ID: "s1"})
	s.AddSession(&Session{ID: "s2"})

	sessions := s.GetSessions()
	if sessions[0].ID != "s2" {
		t.Fatal("newest session should be first")
	}
}
