package sharing

import (
	"context"
	"fmt"
	"sync"
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
		return Sharing{}, fmt.Errorf("sharing %q not found", id)
	}
	return s, nil
}
