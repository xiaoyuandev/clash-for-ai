package tooling

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strconv"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

var ErrNoBackupAvailable = errors.New("no backup file is available to restore")

type Service struct {
	providers *provider.Service
}

func NewService(providers *provider.Service) *Service {
	return &Service{providers: providers}
}

func (s *Service) List(ctx context.Context, apiPort int) ([]ToolIntegrationState, error) {
	codex, err := inspectCodexIntegration(apiPort)
	if err != nil {
		return nil, err
	}
	claude, err := inspectClaudeIntegration(apiPort)
	if err != nil {
		return nil, err
	}

	return []ToolIntegrationState{
		codex,
		claude,
		staticState(ToolCursor),
		staticState(ToolCherry),
		staticState(ToolOpenCode),
		staticState(ToolOpenAISDK),
	}, nil
}

func (s *Service) Configure(ctx context.Context, id ToolIntegrationID, apiPort int) (ToolIntegrationState, error) {
	switch id {
	case ToolCodexCLI:
		return applyCodexIntegration(apiPort)
	case ToolClaudeCode:
		modelMap, err := s.activeClaudeCodeModelMap(ctx)
		if err != nil {
			return ToolIntegrationState{}, err
		}
		return applyClaudeIntegration(apiPort, modelMap)
	default:
		return staticState(id), nil
	}
}

func (s *Service) Restore(_ context.Context, id ToolIntegrationID, apiPort int) (ToolIntegrationState, error) {
	switch id {
	case ToolCodexCLI:
		return restoreCodexIntegration(apiPort)
	case ToolClaudeCode:
		return restoreClaudeIntegration(apiPort)
	default:
		return staticState(id), nil
	}
}

func (s *Service) Runtime() RuntimeInfo {
	info := currentRuntime()
	if info.OS == "" {
		info.OS = runtime.GOOS
	}
	if info.Arch == "" {
		info.Arch = runtime.GOARCH
	}
	return info
}

func (s *Service) activeClaudeCodeModelMap(ctx context.Context) (provider.ClaudeCodeModelMap, error) {
	active, err := s.providers.GetActive(ctx)
	if err != nil || active == nil {
		return provider.ClaudeCodeModelMap{}, err
	}
	return active.ClaudeCodeModelMap, nil
}

func staticState(id ToolIntegrationID) ToolIntegrationState {
	return ToolIntegrationState{
		ID:              id,
		Detected:        false,
		Configured:      false,
		SupportsAdapter: false,
	}
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func intToString(value int) string {
	return strconv.Itoa(value)
}
