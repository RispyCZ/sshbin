package auth

import (
	"sync"
	"time"
)

// SessionStore persists verified sessions keyed by their token.
type SessionStore interface {
	Put(token string, s Session) error
	// Get returns the session for a token. The bool is false when the token is
	// unknown; expired sessions are treated as unknown and removed.
	Get(token string) (Session, bool, error)
	Delete(token string) error
	DeleteByEmail(email string) error
}

// MemorySessionStore is the default in-process SessionStore.
type MemorySessionStore struct {
	mu       sync.Mutex
	now      func() time.Time
	sessions map[string]Session
}

func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{now: time.Now, sessions: make(map[string]Session)}
}

func (m *MemorySessionStore) Put(token string, s Session) error {
	m.mu.Lock()
	m.sessions[token] = s
	m.mu.Unlock()
	return nil
}

func (m *MemorySessionStore) Get(token string) (Session, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[token]
	if !ok {
		return Session{}, false, nil
	}
	if m.now().After(s.ExpiresAt) {
		delete(m.sessions, token)
		return Session{}, false, nil
	}
	return s, true, nil
}

func (m *MemorySessionStore) Delete(token string) error {
	m.mu.Lock()
	delete(m.sessions, token)
	m.mu.Unlock()
	return nil
}

func (m *MemorySessionStore) DeleteByEmail(email string) error {
	m.mu.Lock()
	for token, s := range m.sessions {
		if s.Email == email {
			delete(m.sessions, token)
		}
	}
	m.mu.Unlock()
	return nil
}
