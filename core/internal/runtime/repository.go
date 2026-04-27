package runtime

import "context"

type Repository interface {
	GetConfig(ctx context.Context) (Config, error)
	SaveConfig(ctx context.Context, cfg Config) (Config, error)
}
