package localgateway

const (
	SourceSyncStatusPending = "pending"
	SourceSyncStatusSynced  = "synced"
	SourceSyncStatusError   = "error"
)

type ModelSource struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	BaseURL         string   `json:"base_url"`
	ModelsPath      string   `json:"models_path"`
	APIKeyRef       string   `json:"-"`
	APIKey          string   `json:"api_key"`
	ProviderType    string   `json:"provider_type"`
	DefaultModelID  string   `json:"default_model_id"`
	ExposedModelIDs []string `json:"exposed_model_ids"`
	Enabled         bool     `json:"enabled"`
	Position        int      `json:"position"`
	APIKeyMasked    string   `json:"api_key_masked"`
	LastSyncStatus  string   `json:"last_sync_status"`
	LastSyncError   string   `json:"last_sync_error,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type PublicModelSource struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	BaseURL         string   `json:"base_url"`
	ModelsPath      string   `json:"models_path"`
	APIKey          string   `json:"api_key"`
	ProviderType    string   `json:"provider_type"`
	DefaultModelID  string   `json:"default_model_id"`
	ExposedModelIDs []string `json:"exposed_model_ids"`
	Enabled         bool     `json:"enabled"`
	Position        int      `json:"position"`
	APIKeyMasked    string   `json:"api_key_masked"`
	LastSyncStatus  string   `json:"last_sync_status"`
	LastSyncError   string   `json:"last_sync_error,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type SelectedModel struct {
	ModelID  string `json:"model_id"`
	Position int    `json:"position"`
}

func ToPublicModelSource(item ModelSource) PublicModelSource {
	return PublicModelSource{
		ID:              item.ID,
		Name:            item.Name,
		BaseURL:         item.BaseURL,
		ModelsPath:      item.ModelsPath,
		APIKey:          item.APIKey,
		ProviderType:    item.ProviderType,
		DefaultModelID:  item.DefaultModelID,
		ExposedModelIDs: append([]string(nil), item.ExposedModelIDs...),
		Enabled:         item.Enabled,
		Position:        item.Position,
		APIKeyMasked:    item.APIKeyMasked,
		LastSyncStatus:  item.LastSyncStatus,
		LastSyncError:   item.LastSyncError,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}
