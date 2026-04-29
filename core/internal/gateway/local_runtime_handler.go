package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
	dispatcher "github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway/inbound/dispatcher"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type LocalRuntimeProviderResolver interface {
	GetByID(ctx context.Context, id string) (*provider.Provider, error)
	ListSelectedModels(ctx context.Context, id string) ([]provider.SelectedModel, error)
}

type LocalRuntimeHandler struct {
	providers    LocalRuntimeProviderResolver
	modelSources ModelSourceResolver
	executor     localgateway.Service
}

func NewLocalRuntimeHandler(
	providers LocalRuntimeProviderResolver,
	modelSources ModelSourceResolver,
	executor localgateway.Service,
) http.Handler {
	handler := &LocalRuntimeHandler{
		providers:    providers,
		modelSources: modelSources,
		executor:     executor,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.handleHealth)
	mux.HandleFunc("/v1", handler.handleV1)
	mux.HandleFunc("/v1/", handler.handleV1)
	return mux
}

func (h *LocalRuntimeHandler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func (h *LocalRuntimeHandler) handleV1(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1" && !strings.HasPrefix(r.URL.Path, "/v1/") {
		http.NotFound(w, r)
		return
	}

	body, err := readRequestBody(r)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	activeProvider, err := h.providers.GetByID(r.Context(), provider.LocalGatewayProviderID)
	if err != nil {
		http.Error(w, "failed to resolve local gateway provider", http.StatusInternalServerError)
		return
	}
	if activeProvider == nil {
		http.Error(w, "local gateway provider unavailable", http.StatusServiceUnavailable)
		return
	}

	sources, err := h.modelSources.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list local model sources", http.StatusInternalServerError)
		return
	}

	request, err := dispatcher.ParseRequest(r, body)
	if err != nil {
		http.Error(w, "unsupported local gateway route", http.StatusNotFound)
		return
	}

	if request.Operation == localgateway.OperationModels {
		writeResponse(w, buildLocalModelsResponse(sources))
		return
	}

	if len(sources) == 0 {
		writeLocalRuntimeError(w, http.StatusBadGateway, fmt.Errorf("no enabled model source available"))
		return
	}

	selectedModels, err := h.providers.ListSelectedModels(r.Context(), provider.LocalGatewayProviderID)
	if err != nil {
		http.Error(w, "failed to load local gateway selected models", http.StatusInternalServerError)
		return
	}

	attempts, _ := buildModelAttempts(r, body, selectedModels, activeProvider.ClaudeCodeModelMap)
	for index, attempt := range attempts {
		currentRequest := request
		currentRequest.Body = attempt.body
		if attempt.model != nil {
			currentRequest.Model = *attempt.model
		}
		currentRequest.Stream = isStreamingRequest(r, attempt.body)

		source := chooseModelSource(sources, currentRequest.Model, selectedModels)
		if source == nil {
			writeLocalRuntimeError(w, http.StatusBadGateway, fmt.Errorf("no enabled model source available"))
			return
		}

		response, execErr := h.executor.Handle(r.Context(), currentRequest, *source)
		if execErr != nil {
			if index < len(attempts)-1 {
				continue
			}
			writeLocalRuntimeError(w, http.StatusBadGateway, execErr)
			return
		}

		result := forwardResult{
			statusCode:  response.StatusCode,
			header:      http.Header(response.Headers),
			body:        response.Body,
			firstByteAt: time.Now(),
			finalModel:  attempt.model,
		}

		if response.StatusCode < 400 {
			writeResponse(w, result)
			return
		}

		message := strings.TrimSpace(string(response.Body))
		if message != "" {
			result.errorMessage = &message
			result.errorSnippet = &message
		}

		if !isRetryableStatus(response.StatusCode) || index == len(attempts)-1 {
			writeResponse(w, result)
			return
		}
	}

	writeLocalRuntimeError(w, http.StatusBadGateway, fmt.Errorf("no enabled model source available"))
}

func buildLocalModelsResponse(sources []modelsource.Source) forwardResult {
	models := provider.BuildSystemLocalGatewayModelInfo(sources)
	payload := map[string]any{
		"data": models,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return forwardResult{networkError: err}
	}

	return forwardResult{
		statusCode:  http.StatusOK,
		header:      http.Header{"Content-Type": []string{"application/json"}},
		body:        body,
		firstByteAt: time.Now(),
	}
}

func chooseModelSource(
	sources []modelsource.Source,
	requestModel string,
	selected []provider.SelectedModel,
) *localgateway.ModelSource {
	byModelID := make(map[string]modelsource.Source, len(sources))
	var firstEnabled *modelsource.Source
	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		byModelID[source.DefaultModelID] = source
		if firstEnabled == nil {
			next := source
			firstEnabled = &next
		}
	}

	if source, ok := byModelID[requestModel]; ok {
		next := source
		return &localgateway.ModelSource{
			ID:             next.ID,
			Name:           next.Name,
			BaseURL:        next.BaseURL,
			APIKey:         next.APIKey,
			ProviderType:   next.ProviderType,
			DefaultModelID: next.DefaultModelID,
			Enabled:        next.Enabled,
		}
	}

	for _, item := range selected {
		source, ok := byModelID[item.ModelID]
		if !ok {
			continue
		}
		next := source
		return &localgateway.ModelSource{
			ID:             next.ID,
			Name:           next.Name,
			BaseURL:        next.BaseURL,
			APIKey:         next.APIKey,
			ProviderType:   next.ProviderType,
			DefaultModelID: next.DefaultModelID,
			Enabled:        next.Enabled,
		}
	}

	if firstEnabled == nil {
		return nil
	}

	return &localgateway.ModelSource{
		ID:             firstEnabled.ID,
		Name:           firstEnabled.Name,
		BaseURL:        firstEnabled.BaseURL,
		APIKey:         firstEnabled.APIKey,
		ProviderType:   firstEnabled.ProviderType,
		DefaultModelID: firstEnabled.DefaultModelID,
		Enabled:        firstEnabled.Enabled,
	}
}

func writeLocalRuntimeError(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   "upstream_request_failed",
		"message": err.Error(),
	})
}
