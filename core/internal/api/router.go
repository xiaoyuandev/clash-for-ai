package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/gatewayadapter"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/health"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgatewaycontrol"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/logging"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type Router struct {
	providers         *provider.Service
	health            *health.Service
	logs              *logging.Service
	gateway           http.Handler
	localGatewayAdmin *localgatewaycontrol.Service
}

func NewRouter(
	providers *provider.Service,
	healthService *health.Service,
	loggingService *logging.Service,
	gatewayHandler http.Handler,
	localGatewayAdmin *localgatewaycontrol.Service,
) http.Handler {
	router := &Router{
		providers:         providers,
		health:            healthService,
		logs:              loggingService,
		gateway:           gatewayHandler,
		localGatewayAdmin: localGatewayAdmin,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", router.handleHealth)
	mux.HandleFunc("/api/logs", router.handleLogs)
	mux.HandleFunc("/api/local-gateway/runtime", router.handleLocalGatewayRuntime)
	mux.HandleFunc("/api/local-gateway/model-sources", router.handleLocalGatewayModelSources)
	mux.HandleFunc("/api/local-gateway/model-sources/", router.handleLocalGatewayModelSourceActions)
	mux.HandleFunc("/api/local-gateway/selected-models", router.handleLocalGatewaySelectedModels)
	mux.HandleFunc("/api/providers", router.handleProviders)
	mux.HandleFunc("/api/providers/", router.handleProviderActions)
	mux.Handle("/v1/", router.gateway)

	return withCORS(mux)
}

func (r *Router) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "0.1.0",
	})
}

func (r *Router) handleProviders(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		items, err := r.providers.List(req.Context())
		if err != nil {
			http.Error(w, "failed to list providers", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var input provider.CreateInput
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.BaseURL) == "" {
			http.Error(w, "name and base_url are required", http.StatusBadRequest)
			return
		}

		if input.AuthMode == "" {
			input.AuthMode = provider.InferAuthMode(input.Name, input.BaseURL)
		}

		item, err := r.providers.Create(req.Context(), input)
		if err != nil {
			http.Error(w, "failed to create provider", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, item)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (r *Router) handleLogs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 100
	if rawLimit := req.URL.Query().Get("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	items, err := r.logs.List(req.Context(), limit)
	if err != nil {
		http.Error(w, "failed to list request logs", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (r *Router) handleLocalGatewayRuntime(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status, err := r.localGatewayAdmin.GetRuntimeStatus(req.Context())
	if err != nil {
		if errors.Is(err, provider.ErrProviderNotFound) {
			http.Error(w, "local gateway provider not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to load local gateway runtime", http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func (r *Router) handleLocalGatewayModelSources(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		items, err := r.localGatewayAdmin.ListModelSources(req.Context())
		if err != nil {
			writeLocalGatewayAdminError(w, err, "failed to list local gateway model sources")
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var input modelsource.CreateInput
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		item, err := r.localGatewayAdmin.CreateModelSource(req.Context(), input)
		if err != nil {
			writeLocalGatewayAdminError(w, err, "failed to create local gateway model source")
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (r *Router) handleLocalGatewayModelSourceActions(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/api/local-gateway/model-sources/")
	parts := strings.Split(path, "/")
	switch {
	case len(parts) == 1 && parts[0] == "order" && req.Method == http.MethodPut:
		var input []modelsource.Source
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		items, err := r.localGatewayAdmin.ReplaceModelSourceOrder(req.Context(), input)
		if err != nil {
			writeLocalGatewayAdminError(w, err, "failed to reorder local gateway model sources")
			return
		}
		writeJSON(w, http.StatusOK, items)
	case len(parts) == 1 && req.Method == http.MethodPut:
		var input modelsource.UpdateInput
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		item, err := r.localGatewayAdmin.UpdateModelSource(req.Context(), parts[0], input)
		if err != nil {
			writeLocalGatewayAdminError(w, err, "failed to update local gateway model source")
			return
		}
		writeJSON(w, http.StatusOK, item)
	case len(parts) == 1 && req.Method == http.MethodDelete:
		if err := r.localGatewayAdmin.DeleteModelSource(req.Context(), parts[0]); err != nil {
			writeLocalGatewayAdminError(w, err, "failed to delete local gateway model source")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (r *Router) handleLocalGatewaySelectedModels(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		items, err := r.localGatewayAdmin.ListSelectedModels(req.Context())
		if err != nil {
			writeLocalGatewayAdminError(w, err, "failed to list local gateway selected models")
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPut:
		var input []provider.SelectedModel
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		items, err := r.localGatewayAdmin.ReplaceSelectedModels(req.Context(), input)
		if err != nil {
			writeLocalGatewayAdminError(w, err, "failed to update local gateway selected models")
			return
		}
		writeJSON(w, http.StatusOK, items)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (r *Router) handleProviderActions(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/api/providers/")
	parts := strings.Split(path, "/")
	switch {
	case len(parts) == 2 && parts[1] == "activate" && req.Method == http.MethodPost:
		item, err := r.providers.Activate(req.Context(), parts[0])
		if err != nil {
			if errors.Is(err, provider.ErrProviderNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}

			http.Error(w, "failed to activate provider", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, item)
	case len(parts) == 2 && parts[1] == "claude-code-model-map" && req.Method == http.MethodGet:
		item, err := r.providers.GetClaudeCodeModelMap(req.Context(), parts[0])
		if err != nil {
			if errors.Is(err, provider.ErrProviderNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}

			http.Error(w, "failed to load claude code model map", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, item)
	case len(parts) == 2 && parts[1] == "claude-code-model-map" && req.Method == http.MethodPut:
		var input provider.ClaudeCodeModelMap
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		item, err := r.providers.UpdateClaudeCodeModelMap(req.Context(), parts[0], input)
		if err != nil {
			if errors.Is(err, provider.ErrProviderNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}

			http.Error(w, "failed to update claude code model map", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, item)
	case len(parts) == 2 && parts[1] == "models" && req.Method == http.MethodGet:
		items, err := r.providers.FetchModels(req.Context(), parts[0])
		if err != nil {
			if errors.Is(err, provider.ErrProviderNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}

			http.Error(w, "failed to fetch provider models", http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, items)
	case len(parts) == 2 && parts[1] == "selected-models" && req.Method == http.MethodGet:
		if parts[0] == provider.LocalGatewayProviderID {
			http.Error(w, "local gateway selected models are runtime-internal", http.StatusNotFound)
			return
		}

		items, err := r.providers.ListSelectedModels(req.Context(), parts[0])
		if err != nil {
			if errors.Is(err, provider.ErrProviderNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}

			http.Error(w, "failed to list selected models", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, items)
	case len(parts) == 2 && parts[1] == "selected-models" && req.Method == http.MethodPut:
		if parts[0] == provider.LocalGatewayProviderID {
			http.Error(w, "local gateway selected models are runtime-internal", http.StatusNotFound)
			return
		}

		var input []provider.SelectedModel
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		items, err := r.providers.ReplaceSelectedModels(req.Context(), parts[0], input)
		if err != nil {
			if errors.Is(err, provider.ErrProviderNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}

			http.Error(w, "failed to update selected models", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, items)
	case len(parts) == 2 && parts[1] == "healthcheck" && req.Method == http.MethodPost:
		result, err := r.health.CheckProvider(req.Context(), parts[0])
		if err != nil {
			if errors.Is(err, provider.ErrProviderNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}

			http.Error(w, "failed to run provider healthcheck", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, result)
	case len(parts) == 1 && req.Method == http.MethodDelete:
		if err := r.providers.Delete(req.Context(), parts[0]); err != nil {
			if errors.Is(err, provider.ErrProviderNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}
			if errors.Is(err, provider.ErrProviderImmutable) {
				http.Error(w, "system provider cannot be deleted", http.StatusForbidden)
				return
			}

			http.Error(w, "failed to delete provider", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	case len(parts) == 1 && req.Method == http.MethodPut:
		var input provider.UpdateInput
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if input.AuthMode == "" {
			input.AuthMode = provider.InferAuthMode(input.Name, input.BaseURL)
		}

		item, err := r.providers.Update(req.Context(), parts[0], input)
		if err != nil {
			if errors.Is(err, provider.ErrProviderNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}
			if errors.Is(err, provider.ErrProviderImmutable) {
				http.Error(w, "system provider cannot be updated", http.StatusForbidden)
				return
			}

			http.Error(w, "failed to update provider", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, item)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeLocalGatewayAdminError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, gatewayadapter.ErrRuntimeAdminUnsupported):
		http.Error(w, "local gateway runtime admin unsupported", http.StatusNotImplemented)
	default:
		http.Error(w, fallback, http.StatusBadGateway)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, x-api-key, api-key")

		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, req)
	})
}
