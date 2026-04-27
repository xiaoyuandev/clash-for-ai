package runtime

import "context"

type Repository interface {
	GetConfig(ctx context.Context) (Config, error)
	SaveConfig(ctx context.Context, cfg Config) (Config, error)
	GetLocalGatewayClaudeMap(ctx context.Context) (ClaudeCodeModelMap, error)
	SaveLocalGatewayClaudeMap(ctx context.Context, cfg ClaudeCodeModelMap) (ClaudeCodeModelMap, error)
}
