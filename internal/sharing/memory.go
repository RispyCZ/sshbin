package sharing

import (
	"context"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu      sync.RWMutex
	records map[string]Sharing
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{records: make(map[string]Sharing)}
}

func (r *MemoryRepository) Create(ctx context.Context, s Sharing) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[s.ID] = s
	return nil
}

func (r *MemoryRepository) Get(ctx context.Context, id string) (Sharing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.records[id]
	if !ok {
		return Sharing{}, ErrNotFound
	}
	return s, nil
}

func (r *MemoryRepository) Update(ctx context.Context, s Sharing) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.records[s.ID]; !ok {
		return ErrNotFound
	}
	r.records[s.ID] = s
	return nil
}

func (r *MemoryRepository) ListByOwner(ctx context.Context, email string) ([]Sharing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Sharing
	for _, s := range r.records {
		if s.OwnerEmail == email {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *MemoryRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.records[id]; !ok {
		return ErrNotFound
	}
	delete(r.records, id)
	return nil
}

func (r *MemoryRepository) DeleteByOwner(ctx context.Context, email string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, s := range r.records {
		if s.OwnerEmail == email {
			delete(r.records, id)
		}
	}
	return nil
}

func (r *MemoryRepository) Prunable(ctx context.Context, now, unconfiguredBefore time.Time) ([]Sharing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Sharing
	for _, s := range r.records {
		if s.Expired(now) || (!s.Configured && s.CreatedAt.Before(unconfiguredBefore)) {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *MemoryRepository) FileIDs(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.records))
	for _, s := range r.records {
		ids = append(ids, s.FileID)
	}
	return ids, nil
}
