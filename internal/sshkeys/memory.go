package sshkeys

import (
	"context"
	"sort"
	"sync"
	"time"
)

// MemoryRepository is an in-memory Repository for use in tests.
type MemoryRepository struct {
	mu      sync.Mutex
	records map[string]PublicKey // keyed by ID
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{records: make(map[string]PublicKey)}
}

func (r *MemoryRepository) ListByEmail(_ context.Context, email string) ([]PublicKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []PublicKey
	for _, k := range r.records {
		if k.Email == email {
			out = append(out, k)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (r *MemoryRepository) Add(_ context.Context, k PublicKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.records {
		if existing.Fingerprint == k.Fingerprint {
			return ErrDuplicate
		}
	}
	r.records[k.ID] = k
	return nil
}

func (r *MemoryRepository) Delete(_ context.Context, id, email string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.records[id]
	if !ok || k.Email != email {
		return ErrNotFound
	}
	delete(r.records, id)
	return nil
}

func (r *MemoryRepository) DeleteByEmail(_ context.Context, email string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, k := range r.records {
		if k.Email == email {
			delete(r.records, id)
		}
	}
	return nil
}

func (r *MemoryRepository) FindOwnerByMarshaledKey(_ context.Context, marshaled []byte) (string, bool, error) {
	fp, err := FingerprintOfMarshaled(marshaled)
	if err != nil {
		return "", false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, k := range r.records {
		if k.Fingerprint == fp {
			return k.Email, true, nil
		}
	}
	return "", false, nil
}

func (r *MemoryRepository) TouchLastUsed(_ context.Context, fingerprint string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, k := range r.records {
		if k.Fingerprint == fingerprint {
			now := time.Now()
			k.LastUsedAt = &now
			r.records[id] = k
		}
	}
	return nil
}
