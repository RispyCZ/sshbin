package sharing

import (
	"context"
	"time"
)

type Sharing struct {
	ID        string
	FileID    string
	FileName  string
	CreatedAt time.Time
}

type Repository interface {
	Create(ctx context.Context, s Sharing) error
	Get(ctx context.Context, id string) (Sharing, error)
}
