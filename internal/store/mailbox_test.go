package store

import (
	"testing"
	"time"
)

func newTestMessage(to []string, subject string, read bool, when time.Time) *Message {
	return &Message{
		ID:         "m" + subject,
		From:       "sender@x",
		To:         to,
		Subject:    subject,
		ReceivedAt: when,
		IsRead:     read,
	}
}

func TestMailboxes_GroupsByLowercasedAddress(t *testing.T) {
	s := New(0)
	now := time.Now().UTC()
	s.AddMessage(newTestMessage([]string{"Alice@Example.com"}, "first", false, now))
	s.AddMessage(newTestMessage([]string{"alice@example.com"}, "second", false, now.Add(time.Second)))
	s.AddMessage(newTestMessage([]string{"bob@example.com"}, "third", true, now.Add(2*time.Second)))

	mbs := s.Mailboxes()
	if len(mbs) != 2 {
		t.Fatalf("expected 2 mailboxes, got %d: %+v", len(mbs), mbs)
	}

	byAddr := map[string]MailboxSummary{}
	for _, mb := range mbs {
		byAddr[mb.Address] = mb
	}

	alice := byAddr["alice@example.com"]
	if alice.MessageCount != 2 {
		t.Errorf("alice messageCount=%d, want 2", alice.MessageCount)
	}
	if alice.UnreadCount != 2 {
		t.Errorf("alice unreadCount=%d, want 2", alice.UnreadCount)
	}

	bob := byAddr["bob@example.com"]
	if bob.MessageCount != 1 {
		t.Errorf("bob messageCount=%d", bob.MessageCount)
	}
	if bob.UnreadCount != 0 {
		t.Errorf("bob unreadCount=%d", bob.UnreadCount)
	}
}

func TestMailboxes_OrderedByLastReceivedDescending(t *testing.T) {
	s := New(0)
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.AddMessage(newTestMessage([]string{"old@x"}, "old", false, t0))
	s.AddMessage(newTestMessage([]string{"new@x"}, "new", false, t0.Add(time.Hour)))
	s.AddMessage(newTestMessage([]string{"newest@x"}, "newest", false, t0.Add(2*time.Hour)))

	mbs := s.Mailboxes()
	if len(mbs) < 3 {
		t.Fatalf("expected 3 mailboxes")
	}
	if mbs[0].Address != "newest@x" {
		t.Errorf("first should be newest; got %s", mbs[0].Address)
	}
	if mbs[2].Address != "old@x" {
		t.Errorf("last should be old; got %s", mbs[2].Address)
	}
}

func TestMailboxes_MultipleRecipientsContributeToEach(t *testing.T) {
	s := New(0)
	s.AddMessage(newTestMessage([]string{"a@x", "b@x"}, "broadcast", false, time.Now()))

	mbs := s.Mailboxes()
	if len(mbs) != 2 {
		t.Fatalf("expected 2 mailboxes from multi-recipient message; got %d", len(mbs))
	}
	for _, mb := range mbs {
		if mb.MessageCount != 1 {
			t.Errorf("%s messageCount=%d", mb.Address, mb.MessageCount)
		}
	}
}

func TestMessagesForMailbox_CaseInsensitive(t *testing.T) {
	s := New(0)
	s.AddMessage(newTestMessage([]string{"Alice@X.Com"}, "hi", false, time.Now()))

	if got := s.MessagesForMailbox("alice@x.com"); len(got) != 1 {
		t.Errorf("expected case-insensitive match; got %d", len(got))
	}
	if got := s.MessagesForMailbox("ALICE@X.COM"); len(got) != 1 {
		t.Errorf("uppercase query should match; got %d", len(got))
	}
}

func TestMessagesForMailbox_EmptyAddressReturnsNothing(t *testing.T) {
	s := New(0)
	s.AddMessage(newTestMessage([]string{"a@x"}, "hi", false, time.Now()))

	if got := s.MessagesForMailbox(""); len(got) != 0 {
		t.Errorf("empty address should return nothing; got %d", len(got))
	}
}

func TestMessagesForMailbox_OnlyReturnsMatchingMessages(t *testing.T) {
	s := New(0)
	s.AddMessage(newTestMessage([]string{"a@x"}, "1", false, time.Now()))
	s.AddMessage(newTestMessage([]string{"b@x"}, "2", false, time.Now()))
	s.AddMessage(newTestMessage([]string{"a@x", "c@x"}, "3", false, time.Now()))

	got := s.MessagesForMailbox("a@x")
	if len(got) != 2 {
		t.Errorf("expected 2 messages for a@x; got %d", len(got))
	}
}

func TestMailboxes_SkipsEmptyRecipients(t *testing.T) {
	s := New(0)
	s.AddMessage(newTestMessage([]string{"", "  ", "a@x"}, "subj", false, time.Now()))

	mbs := s.Mailboxes()
	if len(mbs) != 1 || mbs[0].Address != "a@x" {
		t.Errorf("expected single a@x mailbox; got %+v", mbs)
	}
}
