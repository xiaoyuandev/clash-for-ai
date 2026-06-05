package extension

import "context"

type Repository interface {
	List(ctx context.Context) ([]Plugin, error)
	GetByID(ctx context.Context, id string) (*Plugin, error)
	Upsert(ctx context.Context, item Plugin) (Plugin, error)
	SetEnabled(ctx context.Context, id string, enabled bool) (*Plugin, error)
	MarkMissing(ctx context.Context, scope PluginScope, seenIDs []string) error
}
