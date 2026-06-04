package api

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/xiaoyuandev/relay-switch/core/internal/gateway"
	"github.com/xiaoyuandev/relay-switch/core/internal/health"
	"github.com/xiaoyuandev/relay-switch/core/internal/localgateway"
	"github.com/xiaoyuandev/relay-switch/core/internal/logging"
	"github.com/xiaoyuandev/relay-switch/core/internal/provider"
	"github.com/xiaoyuandev/relay-switch/core/internal/tooling"
)

type Router struct {
	providers *provider.Service
	health    *health.Service
	logs      *logging.Service
	local     *localgateway.Manager
	tools     *tooling.Service
	httpPort  int
	webDir    string
	gateway   http.Handler
}

func NewRouter(
	providers *provider.Service,
	healthService *health.Service,
	loggingService *logging.Service,
	localGatewayManager *localgateway.Manager,
	toolingService *tooling.Service,
	httpPort int,
	webAssetsDir string,
	gatewayHandler *gateway.Handler,
) http.Handler {
	router := &Router{
		providers: providers,
		health:    healthService,
		logs:      loggingService,
		local:     localGatewayManager,
		tools:     toolingService,
		httpPort:  httpPort,
		webDir:    webAssetsDir,
		gateway:   gatewayHandler,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", router.handleHealth)
	mux.HandleFunc("/api/logs", router.handleLogs)
	mux.HandleFunc("/api/release", router.handleRelease)
	mux.HandleFunc("/api/runtime", router.handleRuntime)
	mux.HandleFunc("/api/tools", router.handleTools)
	mux.HandleFunc("/api/tools/codex-model-catalog", router.handleCodexModelCatalog)
	mux.HandleFunc("/api/tools/", router.handleToolActions)
	mux.HandleFunc("/api/local-gateway/runtime", router.handleLocalGatewayRuntime)
	mux.HandleFunc("/api/local-gateway/capabilities", router.handleLocalGatewayCapabilities)
	mux.HandleFunc("/api/local-gateway/source-capabilities", router.handleLocalGatewaySourceCapabilities)
	mux.HandleFunc("/api/local-gateway/source-models/preview", router.handleLocalGatewaySourceModelsPreview)
	mux.HandleFunc("/api/local-gateway/sync", router.handleLocalGatewaySync)
	mux.HandleFunc("/api/local-gateway/sources", router.handleLocalGatewaySources)
	mux.HandleFunc("/api/local-gateway/sources/", router.handleLocalGatewaySourceActions)
	mux.HandleFunc("/api/local-gateway/selected-models", router.handleLocalGatewaySelectedModels)
	mux.HandleFunc("/api/providers", router.handleProviders)
	mux.HandleFunc("/api/providers/", router.handleProviderActions)
	mux.Handle("/v1/", router.gateway)
	if webAssetsDir != "" {
		mux.Handle("/", router.webHandler())
	}

	return withCORS(mux)
}

func (r *Router) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "1.1.0",
	})
}

func (r *Router) handleRelease(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type releaseMetadata struct {
		ReleaseVersion string `json:"release_version"`
		Platform       string `json:"platform"`
		Arch           string `json:"arch"`
		RuntimeKind    string `json:"runtime_kind"`
		RuntimeVersion string `json:"runtime_version"`
		RuntimeCommit  string `json:"runtime_commit"`
		PackagedAt     string `json:"packaged_at"`
	}

	metadataPath := filepath.Join(r.webDir, "..", "release.json")
	content, err := os.ReadFile(metadataPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			writeJSON(w, http.StatusOK, map[string]any{
				"available": false,
			})
			return
		}

		http.Error(w, "failed to read release metadata", http.StatusInternalServerError)
		return
	}

	var payload releaseMetadata
	if err := json.Unmarshal(content, &payload); err != nil {
		http.Error(w, "failed to decode release metadata", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"available": true,
		"release":   payload,
	})
}

func (r *Router) webHandler() http.Handler {
	indexPath := filepath.Join(r.webDir, "index.html")
	fileServer := http.FileServer(http.Dir(r.webDir))

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet && req.Method != http.MethodHead {
			http.NotFound(w, req)
			return
		}

		cleanPath := filepath.Clean(strings.TrimPrefix(req.URL.Path, "/"))
		if cleanPath == "." {
			cleanPath = "index.html"
		}

		targetPath := filepath.Join(r.webDir, cleanPath)
		info, err := os.Stat(targetPath)
		switch {
		case err == nil && !info.IsDir():
			fileServer.ServeHTTP(w, req)
			return
		case err == nil && info.IsDir():
			indexInDir := filepath.Join(targetPath, "index.html")
			if dirInfo, dirErr := os.Stat(indexInDir); dirErr == nil && !dirInfo.IsDir() {
				req.URL.Path = strings.TrimSuffix(req.URL.Path, "/") + "/index.html"
				fileServer.ServeHTTP(w, req)
				return
			}
		case err != nil && !errors.Is(err, fs.ErrNotExist):
			http.Error(w, "failed to read web assets", http.StatusInternalServerError)
			return
		}

		if _, err := os.Stat(indexPath); err != nil {
			http.NotFound(w, req)
			return
		}
		http.ServeFile(w, req, indexPath)
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
	switch req.Method {
	case http.MethodGet:
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
	case http.MethodDelete:
		if err := r.logs.Clear(req.Context()); err != nil {
			http.Error(w, "failed to clear request logs", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (r *Router) handleRuntime(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.tools == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"os":       runtime.GOOS,
			"arch":     runtime.GOARCH,
			"is_wsl":   false,
			"home_dir": "",
		})
		return
	}
	writeJSON(w, http.StatusOK, r.tools.Runtime())
}

func (r *Router) handleTools(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.tools == nil {
		writeJSON(w, http.StatusOK, []tooling.ToolIntegrationState{})
		return
	}

	items, err := r.tools.List(req.Context(), r.httpPort)
	if err != nil {
		http.Error(w, "failed to list tools", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (r *Router) handleCodexModelCatalog(w http.ResponseWriter, req *http.Request) {
	if r.tools == nil {
		http.Error(w, "tooling service unavailable", http.StatusServiceUnavailable)
		return
	}

	switch req.Method {
	case http.MethodGet:
		state, err := r.tools.GetCodexModelCatalogState()
		if err != nil {
			http.Error(w, "failed to inspect codex model catalog", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, state)
	case http.MethodPut:
		var input struct {
			Enabled            *bool `json:"enabled"`
			HideOfficialModels *bool `json:"hide_official_models"`
		}
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		state, err := r.tools.UpdateCodexModelCatalogState(req.Context(), input.Enabled, input.HideOfficialModels)
		if err != nil {
			http.Error(w, "failed to update codex model catalog", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, state)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (r *Router) handleToolActions(w http.ResponseWriter, req *http.Request) {
	if r.tools == nil {
		http.Error(w, "tooling service unavailable", http.StatusServiceUnavailable)
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/api/tools/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	toolID := tooling.ToolIntegrationID(parts[0])
	switch {
	case parts[1] == "configure" && req.Method == http.MethodPost:
		state, err := r.tools.Configure(req.Context(), toolID, r.httpPort)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, state)
	case parts[1] == "restore" && req.Method == http.MethodPost:
		state, err := r.tools.Restore(req.Context(), toolID, r.httpPort)
		if err != nil {
			if errors.Is(err, tooling.ErrNoBackupAvailable) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, state)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (r *Router) handleProviderActions(w http.ResponseWriter, req *http.Request) {
	path := strings.TrimPrefix(req.URL.Path, "/api/providers/")
	parts := strings.Split(path, "/")
	switch {
	case len(parts) == 2 && parts[1] == "activate" && req.Method == http.MethodPost:
		if err := r.guardManagedLocalGatewayActivation(req.Context(), parts[0]); err != nil {
			if errors.Is(err, provider.ErrProviderNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}
			writeLocalGatewayManagerError(w, err)
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

		r.syncCodexModelCatalog(req.Context())
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
	case len(parts) == 2 && parts[1] == "model-tests" && req.Method == http.MethodPost:
		var input provider.TestModelInput
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		result, err := r.providers.TestModelAvailability(req.Context(), parts[0], input.ModelID)
		if err != nil {
			if errors.Is(err, provider.ErrProviderNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}

			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, result)
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
	case len(parts) == 2 && parts[1] == "codex-models" && req.Method == http.MethodGet:
		items, err := r.providers.ListCodexModels(req.Context(), parts[0])
		if err != nil {
			if errors.Is(err, provider.ErrProviderNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}

			http.Error(w, "failed to list codex models", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, items)
	case len(parts) == 2 && parts[1] == "codex-models" && req.Method == http.MethodPut:
		var input []provider.CodexModel
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		items, err := r.providers.ReplaceCodexModels(req.Context(), parts[0], input)
		if err != nil {
			if errors.Is(err, provider.ErrProviderNotFound) {
				http.Error(w, "provider not found", http.StatusNotFound)
				return
			}

			http.Error(w, "failed to update codex models", http.StatusInternalServerError)
			return
		}

		r.syncCodexModelCatalogForProvider(req.Context(), parts[0])
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

func (r *Router) syncCodexModelCatalog(ctx context.Context) {
	if r.tools == nil {
		return
	}
	if err := r.tools.SyncCodexModelCatalog(ctx); err != nil {
		log.Printf("[codex] model catalog sync failed: %v", err)
	}
}

func (r *Router) syncCodexModelCatalogForProvider(ctx context.Context, providerID string) {
	active, err := r.providers.GetActive(ctx)
	if err != nil {
		log.Printf("[codex] check active provider for catalog sync failed: %v", err)
		return
	}
	if active == nil || active.ID != providerID {
		return
	}
	r.syncCodexModelCatalog(ctx)
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

func (r *Router) handleLocalGatewaySourceCapabilities(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.local == nil {
		writeLocalGatewayError(w, http.StatusServiceUnavailable, "local gateway manager unavailable")
		return
	}

	items, err := r.local.ListSourceCapabilities(req.Context())
	if err != nil {
		writeLocalGatewayManagerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (r *Router) handleLocalGatewaySourceModelsPreview(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.local == nil {
		writeLocalGatewayError(w, http.StatusServiceUnavailable, "local gateway manager unavailable")
		return
	}

	var input localgateway.PreviewModelSourceInput
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		writeLocalGatewayError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	items, err := r.local.PreviewSourceModels(req.Context(), input)
	if err != nil {
		writeLocalGatewayError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, items)
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
		writeJSON(w, http.StatusOK, toPublicLocalGatewaySources(items))
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
		writeJSON(w, http.StatusCreated, localgateway.ToPublicModelSource(item))
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
	if req.Method == http.MethodPost && strings.HasSuffix(id, "/healthcheck") {
		sourceID := strings.TrimSuffix(id, "/healthcheck")
		sourceID = strings.TrimSuffix(sourceID, "/")
		if strings.TrimSpace(sourceID) == "" {
			writeLocalGatewayError(w, http.StatusNotFound, "local gateway source id is required")
			return
		}

		result, err := r.local.CheckSourceHealth(req.Context(), sourceID)
		if err != nil {
			writeLocalGatewayManagerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
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

		writeJSON(w, http.StatusOK, localgateway.ToPublicModelSource(item))
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

func (r *Router) guardManagedLocalGatewayActivation(ctx context.Context, providerID string) error {
	if r.providers == nil {
		return nil
	}

	item, err := r.providers.GetByID(ctx, providerID)
	if err != nil || item == nil {
		return err
	}
	if !item.IsSystemManaged || item.RuntimeKind != provider.RuntimeKindManagedLocalGate {
		return nil
	}
	if r.local == nil {
		return &localgateway.AdapterError{
			Code:        localgateway.AdapterErrorUnavailable,
			Operation:   "activate_local_gateway_provider",
			RuntimeKind: localgateway.RuntimeKindAIMiniGateway,
			Message:     "local gateway runtime manager is unavailable",
		}
	}

	status, err := r.local.GetRuntimeStatus(ctx)
	if err != nil {
		return err
	}
	if status.Running && status.Healthy {
		return nil
	}

	message := "local gateway runtime is not ready"
	if status.LastError != "" {
		message = "local gateway runtime is not ready: " + status.LastError
	}

	return &localgateway.AdapterError{
		Code:        localgateway.AdapterErrorConflict,
		Operation:   "activate_local_gateway_provider",
		RuntimeKind: status.RuntimeKind,
		Message:     message,
	}
}

func toPublicLocalGatewaySources(items []localgateway.ModelSource) []localgateway.PublicModelSource {
	result := make([]localgateway.PublicModelSource, 0, len(items))
	for _, item := range items {
		result = append(result, localgateway.ToPublicModelSource(item))
	}
	return result
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
