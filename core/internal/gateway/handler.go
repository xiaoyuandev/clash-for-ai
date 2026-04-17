package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/credential"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type ActiveProviderResolver interface {
	GetActive(ctx context.Context) (*provider.Provider, error)
}

type Handler struct {
	providers   ActiveProviderResolver
	credentials credential.Store
}

func NewHandler(providers ActiveProviderResolver, credentials credential.Store) *Handler {
	return &Handler{
		providers:   providers,
		credentials: credentials,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1" && !strings.HasPrefix(r.URL.Path, "/v1/") {
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

	baseURL, err := url.Parse(activeProvider.BaseURL)
	if err != nil {
		http.Error(w, "invalid provider base_url", http.StatusBadGateway)
		return
	}

	apiKey, err := h.credentials.Get(r.Context(), activeProvider.APIKeyRef)
	if err != nil {
		http.Error(w, "failed to load provider credential", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(baseURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		req.URL.Path = joinURLPath(baseURL.Path, strings.TrimPrefix(r.URL.Path, "/v1"))
		req.URL.RawPath = req.URL.Path
		req.Host = baseURL.Host
		req.Header.Del("Authorization")
		req.Header.Del("X-API-Key")
		req.Header.Del("x-api-key")

		switch activeProvider.AuthMode {
		case provider.AuthModeBearer:
			req.Header.Set("Authorization", "Bearer "+apiKey)
		case provider.AuthModeAPIKey:
			req.Header.Set("x-api-key", apiKey)
		case provider.AuthModeBoth:
			req.Header.Set("Authorization", "Bearer "+apiKey)
			req.Header.Set("x-api-key", apiKey)
		}

		for key, value := range activeProvider.ExtraHeaders {
			req.Header.Set(key, value)
		}
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, proxyErr error) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(rw).Encode(map[string]string{
			"error":   "upstream_request_failed",
			"message": proxyErr.Error(),
		})
	}

	proxy.ServeHTTP(w, r)
}

func joinURLPath(basePath string, requestPath string) string {
	switch {
	case basePath == "":
		if requestPath == "" {
			return "/"
		}
		return requestPath
	case requestPath == "":
		return basePath
	case strings.HasSuffix(basePath, "/") && strings.HasPrefix(requestPath, "/"):
		return basePath + strings.TrimPrefix(requestPath, "/")
	case !strings.HasSuffix(basePath, "/") && !strings.HasPrefix(requestPath, "/"):
		return basePath + "/" + requestPath
	default:
		return basePath + requestPath
	}
}
