package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/gateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/health"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/logging"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type Router struct {
	providers *provider.Service
	health    *health.Service
	logs      *logging.Service
	gateway   http.Handler
}

func NewRouter(providers *provider.Service, healthService *health.Service, loggingService *logging.Service, gatewayHandler *gateway.Handler) http.Handler {
	router := &Router{
		providers: providers,
		health:    healthService,
		logs:      loggingService,
		gateway:   gatewayHandler,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", router.handleHealth)
	mux.HandleFunc("/api/logs", router.handleLogs)
	mux.HandleFunc("/api/providers", router.handleProviders)
	mux.HandleFunc("/api/providers/", router.handleProviderActions)
	mux.Handle("/v1/", router.gateway)

	return mux
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
