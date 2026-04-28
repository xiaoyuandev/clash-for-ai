package provideradapter

type Target struct {
	ProviderType   Type          `json:"provider_type"`
	Route          ResolvedRoute `json:"route"`
	BaseURL        string        `json:"base_url"`
	APIKey         string        `json:"api_key"`
	DefaultModelID string        `json:"default_model_id"`
}

func ResolveTarget(providerType string, baseURL string, apiKey string, defaultModelID string) Target {
	normalizedType := NormalizeType(providerType)
	return Target{
		ProviderType:   normalizedType,
		Route:          ResolveRoute(string(normalizedType)),
		BaseURL:        baseURL,
		APIKey:         apiKey,
		DefaultModelID: defaultModelID,
	}
}
