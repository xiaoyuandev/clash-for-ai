package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/gateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/health"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/logging"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type Router struct {
	providers *provider.Service
	health    *health.Service
	logs      *logging.Service
	local     *localgateway.Manager
	gateway   http.Handler
}

func NewRouter(
	providers *provider.Service,
	healthService *health.Service,
	loggingService *logging.Service,
	localGatewayManager *localgateway.Manager,
	gatewayHandler *gateway.Handler,
) http.Handler {
	router := &Router{
		providers: providers,
		health:    healthService,
		logs:      loggingService,
		local:     localGatewayManager,
		gateway:   gatewayHandler,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", router.handleHealth)
	mux.HandleFunc("/api/logs", router.handleLogs)
	mux.HandleFunc("/api/local-gateway/runtime", router.handleLocalGatewayRuntime)
	mux.HandleFunc("/api/local-gateway/capabilities", router.handleLocalGatewayCapabilities)
	mux.HandleFunc("/api/local-gateway/sync", router.handleLocalGatewaySync)
	mux.HandleFunc("/api/local-gateway/sources", router.handleLocalGatewaySources)
	mux.HandleFunc("/api/local-gateway/sources/", router.handleLocalGatewaySourceActions)
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
			if errors.Is(err, provider.ErrProviderNotDeletable) {
				http.Error(w, "provider is not deletable", http.StatusForbidden)
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
			if errors.Is(err, provider.ErrProviderNotEditable) {
				http.Error(w, "provider is not editable", http.StatusForbidden)
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

func (r *Router) handleLocalGatewayRuntime(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.local == nil {
		writeLocalGatewayError(w, http.StatusServiceUnavailable, "local gateway manager unavailable")
		return
	}

	status, err := r.local.GetRuntimeStatus(req.Context())
	if err != nil {
		writeLocalGatewayManagerError(w, err)
		return
	}

	lastSync, lastSyncError := r.local.GetLastSyncResult()
	writeJSON(w, http.StatusOK, map[string]any{
		"runtime":         status,
		"last_sync":       lastSync,
		"last_sync_error": lastSyncError,
	})
}

func (r *Router) handleLocalGatewayCapabilities(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.local == nil {
		writeLocalGatewayError(w, http.StatusServiceUnavailable, "local gateway manager unavailable")
		return
	}

	caps, err := r.local.GetCapabilities(req.Context())
	if err != nil {
		writeLocalGatewayManagerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, caps)
}

func (r *Router) handleLocalGatewaySync(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.local == nil {
		writeLocalGatewayError(w, http.StatusServiceUnavailable, "local gateway manager unavailable")
		return
	}

	result, err := r.local.Sync(req.Context())
	if err != nil {
		writeLocalGatewayManagerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (r *Router) handleLocalGatewaySources(w http.ResponseWriter, req *http.Request) {
	if r.local == nil {
		writeLocalGatewayError(w, http.StatusServiceUnavailable, "local gateway manager unavailable")
		return
	}

	switch req.Method {
	case http.MethodGet:
		items, err := r.local.ListSources(req.Context())
		if err != nil {
			writeLocalGatewayManagerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var input localgateway.CreateModelSourceInput
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			writeLocalGatewayError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		item, err := r.local.CreateSource(req.Context(), input)
		if err != nil {
			writeLocalGatewayManagerError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (r *Router) handleLocalGatewaySourceActions(w http.ResponseWriter, req *http.Request) {
	if r.local == nil {
		writeLocalGatewayError(w, http.StatusServiceUnavailable, "local gateway manager unavailable")
		return
	}

	id := strings.TrimPrefix(req.URL.Path, "/api/local-gateway/sources/")
	if strings.TrimSpace(id) == "" {
		writeLocalGatewayError(w, http.StatusNotFound, "local gateway source id is required")
		return
	}

	switch req.Method {
	case http.MethodPut:
		var input localgateway.UpdateModelSourceInput
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			writeLocalGatewayError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		item, err := r.local.UpdateSource(req.Context(), id, input)
		if err != nil {
			writeLocalGatewayManagerError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.local.DeleteSource(req.Context(), id); err != nil {
			writeLocalGatewayManagerError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (r *Router) handleLocalGatewaySelectedModels(w http.ResponseWriter, req *http.Request) {
	if r.local == nil {
		writeLocalGatewayError(w, http.StatusServiceUnavailable, "local gateway manager unavailable")
		return
	}

	switch req.Method {
	case http.MethodGet:
		items, err := r.local.ListSelectedModels(req.Context())
		if err != nil {
			writeLocalGatewayManagerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPut:
		var input []localgateway.SelectedModel
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			writeLocalGatewayError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		items, err := r.local.ReplaceSelectedModels(req.Context(), input)
		if err != nil {
			writeLocalGatewayManagerError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, items)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeLocalGatewayError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{
		"error":   "local_gateway_error",
		"message": message,
	})
}

func writeLocalGatewayManagerError(w http.ResponseWriter, err error) {
	if errors.Is(err, localgateway.ErrModelSourceNotFound) {
		writeLocalGatewayError(w, http.StatusNotFound, err.Error())
		return
	}

	var adapterErr *localgateway.AdapterError
	if errors.As(err, &adapterErr) {
		statusCode := http.StatusBadGateway
		switch adapterErr.Code {
		case localgateway.AdapterErrorInvalidConfig:
			statusCode = http.StatusBadRequest
		case localgateway.AdapterErrorConflict:
			statusCode = http.StatusConflict
		case localgateway.AdapterErrorUnsupported:
			statusCode = http.StatusNotImplemented
		case localgateway.AdapterErrorUnavailable:
			statusCode = http.StatusServiceUnavailable
		}
		writeJSON(w, statusCode, map[string]any{
			"error":        adapterErr.Code,
			"message":      adapterErr.Message,
			"operation":    adapterErr.Operation,
			"runtime_kind": adapterErr.RuntimeKind,
			"retryable":    adapterErr.Retryable,
		})
		return
	}

	writeLocalGatewayError(w, http.StatusInternalServerError, err.Error())
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
