package localgatewayruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgatewaystate"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/settings"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/storage"
)

const stateMigrationVersion = 1

type stateMigrationMarker struct {
	Version int `json:"version"`
}

type SettingsStore interface {
	Save(ctx context.Context, input settings.AppSettings) (settings.AppSettings, error)
}

type ProviderSelectedModelsReader interface {
	ListSelectedModels(ctx context.Context, id string) ([]provider.SelectedModel, error)
}

func PrepareStateIfNeeded(
	ctx context.Context,
	runtimeDataDir string,
	coreDB *sql.DB,
	coreCredentials credential.Store,
	coreSettings SettingsStore,
	providers ProviderSelectedModelsReader,
	currentSettings settings.AppSettings,
) error {
	migrated, err := HasPreparedState(runtimeDataDir)
	if err != nil {
		return err
	}
	if migrated {
		return nil
	}

	coreModelSources := modelsource.NewService(modelsource.NewSQLiteRepository(coreDB), coreCredentials)
	coreLocalGatewayState := localgatewaystate.NewService(localgatewaystate.NewSQLiteRepository(coreDB))

	return prepareState(
		ctx,
		runtimeDataDir,
		coreModelSources,
		coreLocalGatewayState,
		coreSettings,
		providers,
		currentSettings,
	)
}

func StateMarkerPath(runtimeDataDir string) string {
	return filepath.Join(runtimeDataDir, "state-migration.json")
}

func HasPreparedState(runtimeDataDir string) (bool, error) {
	body, err := os.ReadFile(StateMarkerPath(runtimeDataDir))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	var marker stateMigrationMarker
	if err := json.Unmarshal(body, &marker); err != nil {
		return false, err
	}

	return marker.Version >= stateMigrationVersion, nil
}

func prepareState(
	ctx context.Context,
	runtimeDataDir string,
	coreModelSources *modelsource.Service,
	coreLocalGatewayState *localgatewaystate.Service,
	coreSettings SettingsStore,
	providers ProviderSelectedModelsReader,
	currentSettings settings.AppSettings,
) error {
	migrated, err := HasPreparedState(runtimeDataDir)
	if err != nil {
		return err
	}
	if migrated {
		return nil
	}

	runtimeStore, err := storage.NewSQLite(filepath.Join(runtimeDataDir, "clash-for-ai.db"))
	if err != nil {
		return err
	}
	defer runtimeStore.Close()

	runtimeCredentialStore, err := credential.NewFileStore(filepath.Join(runtimeDataDir, "credentials.json"))
	if err != nil {
		return err
	}

	runtimeModelSourceRepository := modelsource.NewSQLiteRepository(runtimeStore.DB)
	runtimeLocalGatewayStateRepository := localgatewaystate.NewSQLiteRepository(runtimeStore.DB)
	runtimeModelSources := modelsource.NewService(runtimeModelSourceRepository, runtimeCredentialStore)
	runtimeLocalGatewayState := localgatewaystate.NewService(runtimeLocalGatewayStateRepository)

	runtimeSources, err := runtimeModelSources.List(ctx)
	if err != nil {
		return err
	}
	migratedSources := false
	if len(runtimeSources) == 0 {
		coreSources, listErr := coreModelSources.List(ctx)
		if listErr != nil {
			return listErr
		}
		for _, item := range coreSources {
			if _, createErr := runtimeModelSources.Create(ctx, modelsource.CreateInput{
				Name:           item.Name,
				BaseURL:        item.BaseURL,
				ProviderType:   item.ProviderType,
				DefaultModelID: item.DefaultModelID,
				Enabled:        item.Enabled,
				APIKey:         item.APIKey,
			}); createErr != nil {
				return createErr
			}
		}
		migratedSources = len(coreSources) > 0
	}

	runtimeSelected, err := runtimeLocalGatewayState.ListSelectedModels(ctx)
	if err != nil {
		return err
	}
	migratedSelected := false
	if len(runtimeSelected) == 0 {
		legacySelected := make([]localgatewaystate.SelectedModel, 0, len(currentSettings.LocalGatewaySelected))
		for _, item := range currentSettings.LocalGatewaySelected {
			legacySelected = append(legacySelected, localgatewaystate.SelectedModel{
				ModelID:  item.ModelID,
				Position: item.Position,
			})
		}
		if len(legacySelected) == 0 {
			providerSelected, listErr := providers.ListSelectedModels(ctx, provider.LocalGatewayProviderID)
			if listErr != nil {
				return listErr
			}
			legacySelected = make([]localgatewaystate.SelectedModel, 0, len(providerSelected))
			for _, item := range providerSelected {
				legacySelected = append(legacySelected, localgatewaystate.SelectedModel{
					ModelID:  item.ModelID,
					Position: item.Position,
				})
			}
		}

		if len(legacySelected) > 0 {
			if _, updateErr := runtimeLocalGatewayState.ReplaceSelectedModels(ctx, legacySelected); updateErr != nil {
				return updateErr
			}
			migratedSelected = true
		}
	}

	if migratedSources {
		coreSources, listErr := coreModelSources.List(ctx)
		if listErr != nil {
			return listErr
		}
		for _, item := range coreSources {
			if deleteErr := coreModelSources.Delete(ctx, item.ID); deleteErr != nil {
				return deleteErr
			}
		}
	}

	if migratedSelected && len(currentSettings.LocalGatewaySelected) > 0 {
		currentSettings.LocalGatewaySelected = []settings.SelectedModel{}
		if _, saveErr := coreSettings.Save(ctx, currentSettings); saveErr != nil {
			return saveErr
		}
	}

	if migratedSelected {
		if _, replaceErr := coreLocalGatewayState.ReplaceSelectedModels(ctx, nil); replaceErr != nil {
			return replaceErr
		}
	}

	return markStatePrepared(runtimeDataDir)
}

func markStatePrepared(runtimeDataDir string) error {
	if err := os.MkdirAll(runtimeDataDir, 0o755); err != nil {
		return err
	}

	body, err := json.Marshal(stateMigrationMarker{Version: stateMigrationVersion})
	if err != nil {
		return err
	}

	return os.WriteFile(StateMarkerPath(runtimeDataDir), body, 0o644)
}
