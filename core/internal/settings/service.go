package settings

import (
	"context"
	"strings"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func DefaultSettings() AppSettings {
	return AppSettings{
		LocalGateway: LocalGatewaySettings{
			Enabled:    true,
			ListenHost: "127.0.0.1",
			ListenPort: 8788,
		},
	}
}

func NormalizeSettings(input AppSettings) AppSettings {
	settings := input
	defaults := DefaultSettings()

	if strings.TrimSpace(settings.LocalGateway.ListenHost) == "" {
		settings.LocalGateway.ListenHost = defaults.LocalGateway.ListenHost
	}
	if settings.LocalGateway.ListenPort <= 0 {
		settings.LocalGateway.ListenPort = defaults.LocalGateway.ListenPort
	}

	return settings
}

func (s *Service) Get(ctx context.Context) (AppSettings, error) {
	settings, err := s.repository.Get(ctx)
	if err != nil {
		return AppSettings{}, err
	}
	return NormalizeSettings(settings), nil
}

func (s *Service) Save(ctx context.Context, settings AppSettings) (AppSettings, error) {
	return s.repository.Save(ctx, NormalizeSettings(settings))
}
