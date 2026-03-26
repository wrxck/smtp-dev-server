package store

import (
	"sync"
	"time"
)

type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int    `json:"size"`
	Content     []byte `json:"-"`
}

type Message struct {
	ID          string       `json:"id"`
	From        string       `json:"from"`
	To          []string     `json:"to"`
	Subject     string       `json:"subject"`
	ReceivedAt  time.Time    `json:"receivedAt"`
	Size        int          `json:"size"`
	RawData     []byte       `json:"-"`
	HTMLBody    string       `json:"-"`
	TextBody    string       `json:"-"`
	Headers     [][2]string  `json:"-"`
	Attachments []Attachment `json:"-"`
	IsRead      bool         `json:"isRead"`
}

type Session struct {
	ID          string    `json:"id"`
	ClientAddr  string    `json:"clientAddr"`
	StartedAt   time.Time `json:"startedAt"`
	EndedAt     time.Time `json:"endedAt"`
	MessageCount int      `json:"messageCount"`
	Log         string    `json:"-"`
}

type Store struct {
	mu       sync.RWMutex
	messages []*Message
	sessions []*Session
	maxItems int
	onChange func()
}

func New(maxItems int) *Store {
	if maxItems <= 0 {
		maxItems = 500
	}
	return &Store{
		messages: make([]*Message, 0),
		sessions: make([]*Session, 0),
		maxItems: maxItems,
	}
}

func (s *Store) OnChange(fn func()) {
	s.mu.Lock()
	s.onChange = fn
	s.mu.Unlock()
}

func (s *Store) notify() {
	if s.onChange != nil {
		go s.onChange()
	}
}

func (s *Store) AddMessage(msg *Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append([]*Message{msg}, s.messages...)
	if len(s.messages) > s.maxItems {
		s.messages = s.messages[:s.maxItems]
	}
	s.notify()
}

func (s *Store) AddSession(sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = append([]*Session{sess}, s.sessions...)
	if len(s.sessions) > s.maxItems {
		s.sessions = s.sessions[:s.maxItems]
	}
}

func (s *Store) GetMessages() []*Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Message, len(s.messages))
	copy(result, s.messages)
	return result
}

func (s *Store) GetMessage(id string) *Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.messages {
		if m.ID == id {
			return m
		}
	}
	return nil
}

func (s *Store) MarkRead(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.messages {
		if m.ID == id {
			m.IsRead = true
			break
		}
	}
}

func (s *Store) DeleteMessage(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, m := range s.messages {
		if m.ID == id {
			s.messages = append(s.messages[:i], s.messages[i+1:]...)
			s.notify()
			return true
		}
	}
	return false
}

func (s *Store) DeleteAllMessages() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := len(s.messages)
	s.messages = s.messages[:0]
	s.notify()
	return count
}

func (s *Store) GetSessions() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Session, len(s.sessions))
	copy(result, s.sessions)
	return result
}

func (s *Store) GetSession(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sess := range s.sessions {
		if sess.ID == id {
			return sess
		}
	}
	return nil
}

func (s *Store) MessageCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.messages)
}

func (s *Store) UnreadCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, m := range s.messages {
		if !m.IsRead {
			count++
		}
	}
	return count
}
