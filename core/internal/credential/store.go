package credential

import "context"

type Store interface {
	Save(ctx context.Context, key string, value string) (string, error)
	Get(ctx context.Context, ref string) (string, error)
	Delete(ctx context.Context, ref string) error
}
