package userprefs

import (
	"context"
	"sync"
)

// MemoryRepository is an in-memory Repository for use in tests.
type MemoryRepository struct {
	mu      sync.RWMutex
	records map[string]UserPrefs
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{records: make(map[string]UserPrefs)}
}

func (r *MemoryRepository) Get(_ context.Context, email string) (UserPrefs, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.records[email]
	if !ok {
		return UserPrefs{Email: email}, nil
	}
	return p, nil
}

func (r *MemoryRepository) Upsert(_ context.Context, prefs UserPrefs) error {
	r.mu.Lock()
	r.records[prefs.Email] = prefs
	r.mu.Unlock()
	return nil
}

func (r *MemoryRepository) Delete(_ context.Context, email string) error {
	r.mu.Lock()
	delete(r.records, email)
	r.mu.Unlock()
	return nil
}
