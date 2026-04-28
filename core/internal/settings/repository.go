package settings

import "context"

type Repository interface {
	Get(ctx context.Context) (AppSettings, error)
	Save(ctx context.Context, settings AppSettings) (AppSettings, error)
}
