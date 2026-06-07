package extension

import (
	"context"
	"encoding/json"
)

type Repository interface {
	List(ctx context.Context) ([]Plugin, error)
	GetByID(ctx context.Context, id string) (*Plugin, error)
	Upsert(ctx context.Context, item Plugin) (Plugin, error)
	SetEnabled(ctx context.Context, id string, enabled bool) (*Plugin, error)
	MarkMissing(ctx context.Context, scope PluginScope, seenIDs []string) error
	UpsertInstall(ctx context.Context, install PluginInstall) (PluginInstall, error)
	GetInstallByPluginID(ctx context.Context, pluginID string) (*PluginInstall, error)
	GetInstallBySource(ctx context.Context, sourceType string, sourceURL string) (*PluginInstall, error)
	DeletePlugin(ctx context.Context, pluginID string) error
	GetSettings(ctx context.Context, pluginID string) (map[string]json.RawMessage, string, error)
	ReplaceSettings(ctx context.Context, pluginID string, values map[string]json.RawMessage) (string, error)
	RecordAudit(ctx context.Context, entry AuditLogEntry) (AuditLogEntry, error)
	ListAudit(ctx context.Context, pluginID string, limit int) ([]AuditLogEntry, error)
}
