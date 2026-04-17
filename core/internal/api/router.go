package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/gateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type Router struct {
	providers *provider.Service
	gateway   http.Handler
}

func NewRouter(providers *provider.Service, gatewayHandler *gateway.Handler) http.Handler {
	router := &Router{
		providers: providers,
		gateway:   gatewayHandler,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", router.handleHealth)
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
			input.AuthMode = provider.AuthModeBearer
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

func (r *Router) handleProviderActions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/api/providers/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "activate" {
		http.NotFound(w, req)
		return
	}

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
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
