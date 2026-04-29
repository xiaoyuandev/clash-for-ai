package modelsource

import "strings"

type ExposedModel struct {
	ID      string
	OwnedBy string
}

func BuildExposedModels(items []Source) []ExposedModel {
	models := make([]ExposedModel, 0, len(items))
	seen := make(map[string]struct{}, len(items))

	for _, item := range items {
		if !item.Enabled {
			continue
		}

		modelID := strings.TrimSpace(item.DefaultModelID)
		if modelID == "" {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}

		seen[modelID] = struct{}{}
		ownedBy := strings.TrimSpace(item.ProviderType)
		if ownedBy == "" {
			ownedBy = strings.TrimSpace(item.Name)
		}

		models = append(models, ExposedModel{
			ID:      modelID,
			OwnedBy: ownedBy,
		})
	}

	return models
}
