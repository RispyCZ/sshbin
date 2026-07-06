// Package pruner periodically deletes orphaned blobs and their share records.
package pruner

import (
	"context"
	"time"

	"github.com/charmbracelet/log"

	"github.com/rispycz/sshbin/internal/sharing"
	"github.com/rispycz/sshbin/internal/storage"
)

// Config controls pruning behaviour.
type Config struct {
	// Interval is how often a prune pass runs. Zero disables the pruner.
	Interval time.Duration
	// UnconfiguredAfter is how long an uploaded-but-never-configured share is
	// kept before it (and its blob) is pruned. It also serves as the minimum
	// age for deleting row-less blobs during the sweep.
	UnconfiguredAfter time.Duration
}

// Pruner removes expired and orphaned blobs on a schedule.
type Pruner struct {
	repo  sharing.Repository
	store storage.Storage
	cfg   Config
	now   func() time.Time
}

func New(repo sharing.Repository, store storage.Storage, cfg Config) *Pruner {
	return &Pruner{repo: repo, store: store, cfg: cfg, now: time.Now}
}

// Run executes a prune pass immediately and then every cfg.Interval until the
// context is cancelled. It returns nil on cancellation.
func (p *Pruner) Run(ctx context.Context) error {
	if err := p.prune(ctx); err != nil {
		log.Error("prune pass failed", "err", err)
	}
	ticker := time.NewTicker(p.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.prune(ctx); err != nil {
				log.Error("prune pass failed", "err", err)
			}
		}
	}
}

// prune runs a single pass: it deletes expired and stale-unconfigured shares
// with their blobs, then sweeps blobs that no longer have a share record.
func (p *Pruner) prune(ctx context.Context) error {
	now := p.now()

	shares, err := p.repo.Prunable(ctx, now, now.Add(-p.cfg.UnconfiguredAfter))
	if err != nil {
		return err
	}
	pruned := make(map[string]struct{}, len(shares))
	for _, s := range shares {
		if err := p.store.Delete(ctx, s.FileID, s.FileName); err != nil {
			log.Error("prune: delete blob", "id", s.FileID, "err", err)
			continue
		}
		if err := p.repo.Delete(ctx, s.ID); err != nil {
			log.Error("prune: delete share", "id", s.ID, "err", err)
			continue
		}
		pruned[s.FileID] = struct{}{}
		log.Info("pruned share", "id", s.ID, "file", s.FileName)
	}

	return p.sweep(ctx, now, pruned)
}

// sweep deletes blobs that have no matching share record and are older than the
// orphan grace period. The grace avoids racing an in-flight upload whose share
// record is written just after its bytes.
func (p *Pruner) sweep(ctx context.Context, now time.Time, alreadyPruned map[string]struct{}) error {
	blobs, err := p.store.List(ctx)
	if err != nil {
		return err
	}
	ids, err := p.repo.FileIDs(ctx)
	if err != nil {
		return err
	}
	referenced := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		referenced[id] = struct{}{}
	}

	grace := p.cfg.UnconfiguredAfter
	if grace <= 0 {
		grace = time.Hour
	}
	cutoff := now.Add(-grace)

	for _, b := range blobs {
		if _, ok := referenced[b.ID]; ok {
			continue
		}
		if _, ok := alreadyPruned[b.ID]; ok {
			continue
		}
		if b.ModTime.After(cutoff) {
			continue
		}
		if err := p.store.Delete(ctx, b.ID, b.Name); err != nil {
			log.Error("prune: delete orphan blob", "id", b.ID, "err", err)
			continue
		}
		log.Info("pruned orphan blob", "id", b.ID, "file", b.Name)
	}
	return nil
}
