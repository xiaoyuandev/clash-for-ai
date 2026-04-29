package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
	dispatcher "github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway/inbound/dispatcher"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgatewaystate"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type LocalRuntimeHandler struct {
	modelSources ModelSourceResolver
	selected     LocalRuntimeSelectedModelStore
	executor     localgateway.Service
}

type ModelSourceResolver interface {
	List(ctx context.Context) ([]modelsource.Source, error)
}

type LocalRuntimeSelectedModelStore interface {
	ListSelectedModels(ctx context.Context) ([]localgatewaystate.SelectedModel, error)
	ReplaceSelectedModels(ctx context.Context, items []localgatewaystate.SelectedModel) ([]localgatewaystate.SelectedModel, error)
}

func NewLocalRuntimeHandler(
	modelSources ModelSourceResolver,
	selected LocalRuntimeSelectedModelStore,
	executor localgateway.Service,
) http.Handler {
	handler := &LocalRuntimeHandler{
		modelSources: modelSources,
		selected:     selected,
		executor:     executor,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.handleHealth)
	mux.HandleFunc("/admin/model-sources", handler.handleModelSources)
	mux.HandleFunc("/admin/model-sources/", handler.handleModelSourceActions)
	mux.HandleFunc("/admin/selected-models", handler.handleSelectedModels)
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

	selectedModels, err := h.listSelectedModels(r.Context())
	if err != nil {
		http.Error(w, "failed to load local gateway selected models", http.StatusInternalServerError)
		return
	}

	attempts, _ := buildSelectedModelAttempts(r, body, selectedModels)
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

func (h *LocalRuntimeHandler) handleModelSources(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		items, err := h.modelSources.List(req.Context())
		if err != nil {
			http.Error(w, "failed to list model sources", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		service, ok := h.modelSources.(interface {
			Create(ctx context.Context, input modelsource.CreateInput) (modelsource.Source, error)
		})
		if !ok {
			http.Error(w, "model source create unavailable", http.StatusNotImplemented)
			return
		}

		var input modelsource.CreateInput
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		item, err := service.Create(req.Context(), input)
		if err != nil {
			http.Error(w, "failed to create model source", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *LocalRuntimeHandler) handleModelSourceActions(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/admin/model-sources/")
	parts := strings.Split(path, "/")
	switch {
	case len(parts) == 1 && parts[0] == "order" && req.Method == http.MethodPut:
		service, ok := h.modelSources.(interface {
			ReplaceOrder(ctx context.Context, items []modelsource.Source) ([]modelsource.Source, error)
		})
		if !ok {
			http.Error(w, "model source reorder unavailable", http.StatusNotImplemented)
			return
		}

		var input []modelsource.Source
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		items, err := service.ReplaceOrder(req.Context(), input)
		if err != nil {
			http.Error(w, "failed to update model source order", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case len(parts) == 1 && req.Method == http.MethodPut:
		service, ok := h.modelSources.(interface {
			Update(ctx context.Context, id string, input modelsource.UpdateInput) (modelsource.Source, error)
		})
		if !ok {
			http.Error(w, "model source update unavailable", http.StatusNotImplemented)
			return
		}

		var input modelsource.UpdateInput
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		item, err := service.Update(req.Context(), parts[0], input)
		if err != nil {
			if errors.Is(err, modelsource.ErrSourceNotFound) {
				http.Error(w, "model source not found", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to update model source", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case len(parts) == 1 && req.Method == http.MethodDelete:
		service, ok := h.modelSources.(interface {
			Delete(ctx context.Context, id string) error
		})
		if !ok {
			http.Error(w, "model source delete unavailable", http.StatusNotImplemented)
			return
		}

		if err := service.Delete(req.Context(), parts[0]); err != nil {
			if errors.Is(err, modelsource.ErrSourceNotFound) {
				http.Error(w, "model source not found", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to delete model source", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *LocalRuntimeHandler) handleSelectedModels(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		items, err := h.listSelectedModels(req.Context())
		if err != nil {
			http.Error(w, "failed to list selected models", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPut:
		var input []provider.SelectedModel
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		items, err := h.replaceSelectedModels(req.Context(), input)
		if err != nil {
			http.Error(w, "failed to update selected models", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, items)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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
	selected []localgatewaystate.SelectedModel,
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

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *LocalRuntimeHandler) listSelectedModels(ctx context.Context) ([]localgatewaystate.SelectedModel, error) {
	items, err := h.selected.ListSelectedModels(ctx)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func buildSelectedModelAttempts(
	r *http.Request,
	body []byte,
	selected []localgatewaystate.SelectedModel,
) ([]attemptSpec, *string) {
	currentModel, payload := extractModelFromBody(body)
	if len(selected) == 0 || payload == nil || r.Method != http.MethodPost {
		return []attemptSpec{{model: currentModel, body: body}}, currentModel
	}

	orderedModels := make([]string, 0, len(selected))
	for _, item := range selected {
		orderedModels = append(orderedModels, item.ModelID)
	}

	startIndex := 0
	if currentModel != nil {
		found := -1
		for index, modelID := range orderedModels {
			if modelID == *currentModel {
				found = index
				break
			}
		}
		if found < 0 {
			return []attemptSpec{{model: currentModel, body: body}}, currentModel
		}
		startIndex = found
	}

	attempts := make([]attemptSpec, 0, len(orderedModels)-startIndex)
	for _, modelID := range orderedModels[startIndex:] {
		attemptModel := modelID
		attempts = append(attempts, attemptSpec{
			model: &attemptModel,
			body:  bodyWithModel(payload, modelID, body),
		})
	}

	return attempts, currentModel
}

func (h *LocalRuntimeHandler) replaceSelectedModels(ctx context.Context, items []provider.SelectedModel) ([]provider.SelectedModel, error) {
	runtimeItems := make([]localgatewaystate.SelectedModel, 0, len(items))
	for _, item := range items {
		runtimeItems = append(runtimeItems, localgatewaystate.SelectedModel{
			ModelID:  item.ModelID,
			Position: item.Position,
		})
	}

	saved, err := h.selected.ReplaceSelectedModels(ctx, runtimeItems)
	if err != nil {
		return nil, err
	}

	result := make([]provider.SelectedModel, 0, len(saved))
	for _, item := range saved {
		result = append(result, provider.SelectedModel{
			ModelID:  item.ModelID,
			Position: item.Position,
		})
	}
	return result, nil
}
