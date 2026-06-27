package userprefs

import "context"

type UserPrefs struct {
	Email         string
	DefaultPublic bool
}

type Repository interface {
	Get(ctx context.Context, email string) (UserPrefs, error)
	Upsert(ctx context.Context, prefs UserPrefs) error
	Delete(ctx context.Context, email string) error
}
