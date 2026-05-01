package localgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type AIMiniGatewayAdapter struct {
	mu     sync.RWMutex
	client *http.Client
	cmd    *exec.Cmd
	status RuntimeStatus
}

func NewAIMiniGatewayAdapter(client *http.Client) *AIMiniGatewayAdapter {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	return &AIMiniGatewayAdapter{
		client: client,
		status: RuntimeStatus{
			RuntimeKind: RuntimeKindAIMiniGateway,
			State:       RuntimeStateStopped,
		},
	}
}

func (a *AIMiniGatewayAdapter) RuntimeKind() string {
	return RuntimeKindAIMiniGateway
}

func (a *AIMiniGatewayAdapter) StartRuntime(ctx context.Context, input StartRuntimeInput) (RuntimeStatus, error) {
	if input.Executable == "" || input.Host == "" || input.Port <= 0 || input.DataDir == "" {
		return RuntimeStatus{}, &AdapterError{
			Code:        AdapterErrorInvalidConfig,
			Operation:   "start_runtime",
			RuntimeKind: a.RuntimeKind(),
			Message:     "executable, host, port, and data_dir are required",
		}
	}

	if err := os.MkdirAll(filepath.Clean(input.DataDir), 0o755); err != nil {
		return RuntimeStatus{}, &AdapterError{
			Code:        AdapterErrorInvalidConfig,
			Operation:   "start_runtime",
			RuntimeKind: a.RuntimeKind(),
			Message:     "create runtime data directory failed",
			Err:         err,
		}
	}

	apiBase := buildAPIBase(input.Host, input.Port)
	args := append([]string{}, input.Arguments...)
	args = append(args,
		"--host", input.Host,
		"--port", strconv.Itoa(input.Port),
		"--data-dir", input.DataDir,
	)

	env := mergeEnvironment(os.Environ(), input.Environment, map[string]string{
		"LOCAL_GATEWAY_RUNTIME_HOST": input.Host,
		"LOCAL_GATEWAY_RUNTIME_PORT": strconv.Itoa(input.Port),
		"CORE_DATA_DIR":              input.DataDir,
	})

	cmd := exec.Command(input.Executable, args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return RuntimeStatus{}, &AdapterError{
			Code:        AdapterErrorUnavailable,
			Operation:   "start_runtime",
			RuntimeKind: a.RuntimeKind(),
			Message:     "start ai-mini-gateway runtime failed",
			Err:         err,
		}
	}

	status := RuntimeStatus{
		RuntimeKind: a.RuntimeKind(),
		State:       RuntimeStateStarting,
		Managed:     true,
		Running:     true,
		Healthy:     false,
		APIBase:     apiBase,
		Host:        input.Host,
		Port:        input.Port,
		PID:         cmd.Process.Pid,
	}

	a.mu.Lock()
	a.cmd = cmd
	a.status = status
	a.mu.Unlock()

	go a.waitForExit(cmd)

	if err := a.waitForHealth(ctx, apiBase); err != nil {
		_ = a.StopRuntime(context.Background())
		return a.currentStatus(), err
	}

	a.mu.Lock()
	a.status.State = RuntimeStateRunning
	a.status.Healthy = true
	a.mu.Unlock()

	return a.currentStatus(), nil
}

func (a *AIMiniGatewayAdapter) StopRuntime(ctx context.Context) error {
	a.mu.RLock()
	cmd := a.cmd
	a.mu.RUnlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return &AdapterError{
			Code:        AdapterErrorUnavailable,
			Operation:   "stop_runtime",
			RuntimeKind: a.RuntimeKind(),
			Message:     "signal ai-mini-gateway runtime failed",
			Err:         err,
		}
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		status := a.currentStatus()
		if !status.Running {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}

	if err := cmd.Process.Kill(); err != nil {
		return &AdapterError{
			Code:        AdapterErrorUnavailable,
			Operation:   "stop_runtime",
			RuntimeKind: a.RuntimeKind(),
			Message:     "force kill ai-mini-gateway runtime failed",
			Err:         err,
		}
	}

	return nil
}

func (a *AIMiniGatewayAdapter) GetRuntimeStatus(ctx context.Context) (RuntimeStatus, error) {
	status := a.currentStatus()
	if status.APIBase == "" || !status.Running {
		return status, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, status.APIBase+"/health", nil)
	if err != nil {
		return status, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		status.Healthy = false
		status.State = RuntimeStateDegraded
		status.LastError = err.Error()
		return status, nil
	}
	defer resp.Body.Close()

	status.Healthy = resp.StatusCode == http.StatusOK
	if status.Healthy {
		status.State = RuntimeStateRunning
		status.LastError = ""
	} else {
		status.State = RuntimeStateDegraded
		status.LastError = "runtime healthcheck returned non-200"
	}

	a.mu.Lock()
	a.status = status
	a.mu.Unlock()

	return status, nil
}

func (a *AIMiniGatewayAdapter) GetCapabilities(ctx context.Context) (RuntimeCapabilities, error) {
	var payload struct {
		SupportsOpenAICompatible    bool `json:"supports_openai_compatible"`
		SupportsAnthropicCompatible bool `json:"supports_anthropic_compatible"`
		SupportsModelsAPI           bool `json:"supports_models_api"`
		SupportsStream              bool `json:"supports_stream"`
		SupportsAdminAPI            bool `json:"supports_admin_api"`
		SupportsModelSourceAdmin    bool `json:"supports_model_source_admin"`
		SupportsSelectedModelAdmin  bool `json:"supports_selected_model_admin"`
	}

	if err := a.doJSON(ctx, http.MethodGet, "/capabilities", nil, &payload); err != nil {
		return RuntimeCapabilities{}, err
	}

	return RuntimeCapabilities{
		SupportsOpenAICompatible:    payload.SupportsOpenAICompatible,
		SupportsAnthropicCompatible: payload.SupportsAnthropicCompatible,
		SupportsModelsAPI:           payload.SupportsModelsAPI,
		SupportsStream:              payload.SupportsStream,
		SupportsAdminAPI:            payload.SupportsAdminAPI,
		SupportsModelSourceAdmin:    payload.SupportsModelSourceAdmin,
		SupportsSelectedModelAdmin:  payload.SupportsSelectedModelAdmin,
		SupportsSourceCapabilities:  true,
		SupportsAtomicSourceSync:    false,
		SupportsRuntimeVersion:      false,
	}, nil
}

func (a *AIMiniGatewayAdapter) ListModelSources(ctx context.Context) ([]RuntimeModelSource, error) {
	var payload []struct {
		ID              string   `json:"id"`
		Name            string   `json:"name"`
		BaseURL         string   `json:"base_url"`
		ProviderType    string   `json:"provider_type"`
		DefaultModelID  string   `json:"default_model_id"`
		ExposedModelIDs []string `json:"exposed_model_ids"`
		Enabled         bool     `json:"enabled"`
		Position        int      `json:"position"`
		APIKeyMasked    string   `json:"api_key_masked"`
	}

	if err := a.doJSON(ctx, http.MethodGet, "/admin/model-sources", nil, &payload); err != nil {
		return nil, err
	}

	items := make([]RuntimeModelSource, 0, len(payload))
	for _, item := range payload {
		items = append(items, RuntimeModelSource{
			ID:              item.ID,
			Name:            item.Name,
			BaseURL:         item.BaseURL,
			ProviderType:    item.ProviderType,
			DefaultModelID:  item.DefaultModelID,
			ExposedModelIDs: append([]string(nil), item.ExposedModelIDs...),
			Enabled:         item.Enabled,
			Position:        item.Position,
			APIKeyMasked:    item.APIKeyMasked,
		})
	}

	return items, nil
}

func (a *AIMiniGatewayAdapter) ListModelSourceCapabilities(ctx context.Context) ([]ModelSourceCapability, error) {
	var payload []struct {
		ID                            string `json:"id"`
		Name                          string `json:"name"`
		ProviderType                  string `json:"provider_type"`
		SupportsModelsAPI             bool   `json:"supports_models_api"`
		ModelsAPIStatus               string `json:"models_api_status"`
		SupportsOpenAIChatCompletions bool   `json:"supports_openai_chat_completions"`
		OpenAIChatCompletionsStatus   string `json:"openai_chat_completions_status"`
		SupportsOpenAIResponses       bool   `json:"supports_openai_responses"`
		OpenAIResponsesStatus         string `json:"openai_responses_status"`
		SupportsAnthropicMessages     bool   `json:"supports_anthropic_messages"`
		AnthropicMessagesStatus       string `json:"anthropic_messages_status"`
		SupportsAnthropicCountTokens  bool   `json:"supports_anthropic_count_tokens"`
		AnthropicCountTokensStatus    string `json:"anthropic_count_tokens_status"`
		SupportsStream                bool   `json:"supports_stream"`
		StreamStatus                  string `json:"stream_status"`
	}

	if err := a.doJSON(ctx, http.MethodGet, "/admin/model-sources/capabilities", nil, &payload); err != nil {
		return nil, err
	}

	items := make([]ModelSourceCapability, 0, len(payload))
	for _, item := range payload {
		items = append(items, ModelSourceCapability{
			SourceID:                      item.ID,
			Name:                          item.Name,
			ProviderType:                  item.ProviderType,
			SupportsModelsAPI:             item.SupportsModelsAPI,
			ModelsAPIStatus:               item.ModelsAPIStatus,
			SupportsOpenAIChatCompletions: item.SupportsOpenAIChatCompletions,
			OpenAIChatCompletionsStatus:   item.OpenAIChatCompletionsStatus,
			SupportsOpenAIResponses:       item.SupportsOpenAIResponses,
			OpenAIResponsesStatus:         item.OpenAIResponsesStatus,
			SupportsAnthropicMessages:     item.SupportsAnthropicMessages,
			AnthropicMessagesStatus:       item.AnthropicMessagesStatus,
			SupportsAnthropicCountTokens:  item.SupportsAnthropicCountTokens,
			AnthropicCountTokensStatus:    item.AnthropicCountTokensStatus,
			SupportsStream:                item.SupportsStream,
			StreamStatus:                  item.StreamStatus,
		})
	}

	return items, nil
}

func (a *AIMiniGatewayAdapter) CreateModelSource(ctx context.Context, input RuntimeModelSourceInput) (RuntimeModelSource, error) {
	var payload RuntimeModelSource
	if err := a.doJSON(ctx, http.MethodPost, "/admin/model-sources", input, &payload); err != nil {
		return RuntimeModelSource{}, err
	}

	if payload.Position != input.Position {
		if err := a.reorderModelSource(ctx, payload.ID, input.Position); err != nil {
			return RuntimeModelSource{}, err
		}

		source, err := a.getModelSourceByID(ctx, payload.ID)
		if err != nil {
			return RuntimeModelSource{}, err
		}
		return source, nil
	}

	return payload, nil
}

func (a *AIMiniGatewayAdapter) UpdateModelSource(ctx context.Context, id string, input RuntimeModelSourceInput) (RuntimeModelSource, error) {
	var payload RuntimeModelSource
	if err := a.doJSON(ctx, http.MethodPut, "/admin/model-sources/"+id, input, &payload); err != nil {
		return RuntimeModelSource{}, err
	}

	if payload.Position != input.Position {
		if err := a.reorderModelSource(ctx, id, input.Position); err != nil {
			return RuntimeModelSource{}, err
		}

		source, err := a.getModelSourceByID(ctx, id)
		if err != nil {
			return RuntimeModelSource{}, err
		}
		return source, nil
	}

	return payload, nil
}

func (a *AIMiniGatewayAdapter) DeleteModelSource(ctx context.Context, id string) error {
	return a.doJSON(ctx, http.MethodDelete, "/admin/model-sources/"+id, nil, nil)
}

func (a *AIMiniGatewayAdapter) ListSelectedModels(ctx context.Context) ([]SelectedModel, error) {
	var payload []SelectedModel
	if err := a.doJSON(ctx, http.MethodGet, "/admin/selected-models", nil, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (a *AIMiniGatewayAdapter) ReplaceSelectedModels(ctx context.Context, items []SelectedModel) ([]SelectedModel, error) {
	normalized := normalizeSelectedModels(items)
	var payload []SelectedModel
	if err := a.doJSON(ctx, http.MethodPut, "/admin/selected-models", normalized, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (a *AIMiniGatewayAdapter) SyncFromProductState(ctx context.Context, input SyncInput) (SyncResult, error) {
	currentSources, err := a.ListModelSources(ctx)
	if err != nil {
		return SyncResult{}, err
	}

	for _, source := range currentSources {
		if err := a.DeleteModelSource(ctx, source.ID); err != nil {
			return SyncResult{}, err
		}
	}

	sources := append([]SyncModelSource(nil), input.Sources...)
	slices.SortFunc(sources, func(a, b SyncModelSource) int {
		return a.Position - b.Position
	})

	for _, source := range sources {
		_, err := a.CreateModelSource(ctx, RuntimeModelSourceInput{
			Name:            source.Name,
			BaseURL:         source.BaseURL,
			APIKey:          source.APIKey,
			ProviderType:    source.ProviderType,
			DefaultModelID:  source.DefaultModelID,
			ExposedModelIDs: append([]string(nil), source.ExposedModelIDs...),
			Enabled:         source.Enabled,
			Position:        source.Position,
		})
		if err != nil {
			return SyncResult{}, &AdapterError{
				Code:        AdapterErrorSyncFailed,
				Operation:   "sync_runtime_sources",
				RuntimeKind: a.RuntimeKind(),
				Message:     "sync runtime model sources failed",
				Err:         err,
			}
		}
	}

	selectedModels, err := a.ReplaceSelectedModels(ctx, input.SelectedModels)
	if err != nil {
		return SyncResult{}, &AdapterError{
			Code:        AdapterErrorSyncFailed,
			Operation:   "sync_selected_models",
			RuntimeKind: a.RuntimeKind(),
			Message:     "sync runtime selected models failed",
			Err:         err,
		}
	}

	return SyncResult{
		AppliedSources:        len(sources),
		AppliedSelectedModels: len(selectedModels),
		LastSyncedAt:          time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (a *AIMiniGatewayAdapter) waitForExit(cmd *exec.Cmd) {
	err := cmd.Wait()

	a.mu.Lock()
	defer a.mu.Unlock()

	a.cmd = nil
	a.status.Running = false
	a.status.Healthy = false
	a.status.PID = 0
	a.status.State = RuntimeStateStopped
	if err != nil {
		a.status.State = RuntimeStateError
		a.status.LastError = err.Error()
		return
	}
	a.status.LastError = ""
}

func (a *AIMiniGatewayAdapter) waitForHealth(ctx context.Context, apiBase string) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/health", nil)
		if err != nil {
			return err
		}

		resp, err := a.client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}

	return &AdapterError{
		Code:        AdapterErrorUnavailable,
		Operation:   "wait_for_health",
		RuntimeKind: a.RuntimeKind(),
		Message:     "ai-mini-gateway runtime did not become healthy in time",
	}
}

func (a *AIMiniGatewayAdapter) currentStatus() RuntimeStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

func (a *AIMiniGatewayAdapter) getModelSourceByID(ctx context.Context, id string) (RuntimeModelSource, error) {
	items, err := a.ListModelSources(ctx)
	if err != nil {
		return RuntimeModelSource{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}

	return RuntimeModelSource{}, &AdapterError{
		Code:        AdapterErrorUnavailable,
		Operation:   "get_model_source_by_id",
		RuntimeKind: a.RuntimeKind(),
		Message:     "runtime model source not found after write",
	}
}

func (a *AIMiniGatewayAdapter) reorderModelSource(ctx context.Context, movedID string, targetPosition int) error {
	items, err := a.ListModelSources(ctx)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return nil
	}

	currentIndex := -1
	for index, item := range items {
		if item.ID == movedID {
			currentIndex = index
			break
		}
	}
	if currentIndex < 0 {
		return &AdapterError{
			Code:        AdapterErrorUnavailable,
			Operation:   "reorder_model_source",
			RuntimeKind: a.RuntimeKind(),
			Message:     "runtime model source not found for reorder",
		}
	}

	if targetPosition < 0 {
		targetPosition = 0
	}
	if targetPosition >= len(items) {
		targetPosition = len(items) - 1
	}
	if currentIndex == targetPosition {
		return nil
	}

	moved := items[currentIndex]
	items = append(items[:currentIndex], items[currentIndex+1:]...)
	items = slices.Insert(items, targetPosition, moved)

	type orderItem struct {
		ID       string `json:"id"`
		Position int    `json:"position"`
	}

	payload := make([]orderItem, 0, len(items))
	for index, item := range items {
		payload = append(payload, orderItem{
			ID:       item.ID,
			Position: index,
		})
	}

	return a.doJSON(ctx, http.MethodPut, "/admin/model-sources/order", payload, nil)
}

func (a *AIMiniGatewayAdapter) doJSON(ctx context.Context, method string, path string, input any, out any) error {
	apiBase := a.currentStatus().APIBase
	if apiBase == "" {
		return &AdapterError{
			Code:        AdapterErrorUnavailable,
			Operation:   method + " " + path,
			RuntimeKind: a.RuntimeKind(),
			Message:     "runtime api base is not available",
		}
	}

	var bodyReader *bytes.Reader
	if input == nil {
		bodyReader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(input)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiBase+path, bodyReader)
	if err != nil {
		return err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return &AdapterError{
			Code:        AdapterErrorUnavailable,
			Operation:   method + " " + path,
			RuntimeKind: a.RuntimeKind(),
			Message:     "runtime request failed",
			Err:         err,
			Retryable:   true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var payload struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&payload)

		return &AdapterError{
			Code:        mapHTTPStatusToAdapterError(resp.StatusCode),
			Operation:   method + " " + path,
			RuntimeKind: a.RuntimeKind(),
			Message:     firstNonEmpty(payload.Message, payload.Error, "runtime request failed"),
			Retryable:   resp.StatusCode >= 500,
		}
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func normalizeSelectedModels(items []SelectedModel) []SelectedModel {
	normalized := make([]SelectedModel, 0, len(items))
	for index, item := range items {
		normalized = append(normalized, SelectedModel{
			ModelID:  item.ModelID,
			Position: index,
		})
	}
	return normalized
}

func buildAPIBase(host string, port int) string {
	clientHost := host
	switch host {
	case "0.0.0.0", "::", "[::]":
		clientHost = "127.0.0.1"
	}

	return fmt.Sprintf("http://%s:%d", clientHost, port)
}

func mergeEnvironment(base []string, overrides ...map[string]string) []string {
	values := map[string]string{}
	for _, item := range base {
		key, value, ok := splitEnvironmentEntry(item)
		if !ok {
			continue
		}
		values[key] = value
	}

	for _, override := range overrides {
		for key, value := range override {
			values[key] = value
		}
	}

	result := make([]string, 0, len(values))
	for key, value := range values {
		result = append(result, key+"="+value)
	}
	slices.Sort(result)
	return result
}

func splitEnvironmentEntry(value string) (string, string, bool) {
	for index := 0; index < len(value); index++ {
		if value[index] == '=' {
			return value[:index], value[index+1:], true
		}
	}
	return "", "", false
}

func mapHTTPStatusToAdapterError(statusCode int) AdapterErrorCode {
	switch statusCode {
	case http.StatusBadRequest:
		return AdapterErrorInvalidConfig
	case http.StatusConflict:
		return AdapterErrorConflict
	case http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusNotImplemented:
		return AdapterErrorUnsupported
	default:
		if statusCode >= 500 {
			return AdapterErrorUnavailable
		}
		return AdapterErrorUpstream
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
