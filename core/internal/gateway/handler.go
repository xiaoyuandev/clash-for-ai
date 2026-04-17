package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type ActiveProviderResolver interface {
	GetActive(ctx context.Context) (*provider.Provider, error)
}

type Handler struct {
	providers ActiveProviderResolver
}

func NewHandler(providers ActiveProviderResolver) *Handler {
	return &Handler{providers: providers}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/v1/") {
		http.NotFound(w, r)
		return
	}

	activeProvider, err := h.providers.GetActive(r.Context())
	if err != nil {
		http.Error(w, "failed to resolve active provider", http.StatusInternalServerError)
		return
	}

	if activeProvider == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "no_active_provider",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":  "gateway skeleton online",
		"provider": activeProvider.Name,
		"path":     r.URL.Path,
	})
}
