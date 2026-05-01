package localgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestAIMiniGatewayAdapterGetCapabilities(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/capabilities" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		return jsonResponse(http.StatusOK, map[string]any{
			"supports_openai_compatible":    true,
			"supports_anthropic_compatible": true,
			"supports_models_api":           true,
			"supports_stream":               true,
			"supports_admin_api":            true,
			"supports_model_source_admin":   true,
			"supports_selected_model_admin": true,
		}), nil
	})}

	adapter := NewAIMiniGatewayAdapter(client)
	adapter.status = RuntimeStatus{
		RuntimeKind: RuntimeKindAIMiniGateway,
		State:       RuntimeStateRunning,
		Running:     true,
		Healthy:     true,
		APIBase:     "http://runtime.test",
	}

	caps, err := adapter.GetCapabilities(context.Background())
	if err != nil {
		t.Fatalf("get capabilities: %v", err)
	}

	if !caps.SupportsOpenAICompatible || !caps.SupportsSelectedModelAdmin {
		t.Fatalf("unexpected capabilities: %+v", caps)
	}
	if caps.SupportsAtomicSourceSync {
		t.Fatalf("unexpected atomic sync support: %+v", caps)
	}
}

func TestAIMiniGatewayAdapterSyncFromProductState(t *testing.T) {
	t.Parallel()

	state := newFakeAIMiniGatewayRuntime()
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return state.roundTrip(t, r)
	})}

	adapter := NewAIMiniGatewayAdapter(client)
	adapter.status = RuntimeStatus{
		RuntimeKind: RuntimeKindAIMiniGateway,
		State:       RuntimeStateRunning,
		Running:     true,
		Healthy:     true,
		APIBase:     "http://runtime.test",
	}

	result, err := adapter.SyncFromProductState(context.Background(), SyncInput{
		Sources: []SyncModelSource{
			{
				ID:              "source-b",
				Name:            "Anthropic",
				BaseURL:         "https://api.anthropic.com/v1",
				APIKey:          "sk-ant",
				ProviderType:    "anthropic-compatible",
				DefaultModelID:  "claude-sonnet-4-0",
				ExposedModelIDs: []string{"claude-haiku-4-0"},
				Enabled:         true,
				Position:        1,
			},
			{
				ID:             "source-a",
				Name:           "OpenAI",
				BaseURL:        "https://api.openai.com/v1",
				APIKey:         "sk-openai",
				ProviderType:   "openai-compatible",
				DefaultModelID: "gpt-4.1",
				Enabled:        true,
				Position:       0,
			},
		},
		SelectedModels: []SelectedModel{
			{ModelID: "claude-sonnet-4-0", Position: 4},
			{ModelID: "gpt-4.1", Position: 9},
		},
	})
	if err != nil {
		t.Fatalf("sync from product state: %v", err)
	}

	if result.AppliedSources != 2 {
		t.Fatalf("unexpected applied sources: %d", result.AppliedSources)
	}
	if result.AppliedSelectedModels != 2 {
		t.Fatalf("unexpected applied selected models: %d", result.AppliedSelectedModels)
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if len(state.sources) != 2 {
		t.Fatalf("unexpected runtime source count: %d", len(state.sources))
	}
	if state.sources[0].Name != "OpenAI" || state.sources[1].Name != "Anthropic" {
		t.Fatalf("unexpected runtime source order: %+v", state.sources)
	}
	if len(state.selectedModels) != 2 {
		t.Fatalf("unexpected runtime selected models: %+v", state.selectedModels)
	}
	if state.selectedModels[0].ModelID != "claude-sonnet-4-0" || state.selectedModels[0].Position != 0 {
		t.Fatalf("unexpected first runtime selected model: %+v", state.selectedModels[0])
	}
}

func TestNormalizeSelectedModels(t *testing.T) {
	t.Parallel()

	items := normalizeSelectedModels([]SelectedModel{
		{ModelID: "b", Position: 4},
		{ModelID: "a", Position: 7},
	})

	if len(items) != 2 {
		t.Fatalf("unexpected items length: %d", len(items))
	}
	if items[0].Position != 0 || items[1].Position != 1 {
		t.Fatalf("unexpected normalized positions: %+v", items)
	}

	if !slices.Equal([]string{items[0].ModelID, items[1].ModelID}, []string{"b", "a"}) {
		t.Fatalf("unexpected normalized order: %+v", items)
	}
}

type fakeAIMiniGatewayRuntime struct {
	mu             sync.Mutex
	nextID         int
	sources        []RuntimeModelSource
	selectedModels []SelectedModel
}

func newFakeAIMiniGatewayRuntime() *fakeAIMiniGatewayRuntime {
	return &fakeAIMiniGatewayRuntime{
		nextID: 1,
		sources: []RuntimeModelSource{
			{
				ID:             "src-existing",
				Name:           "Legacy",
				BaseURL:        "https://legacy.example/v1",
				ProviderType:   "openai-compatible",
				DefaultModelID: "legacy-model",
				Enabled:        true,
				Position:       0,
			},
		},
	}
}

func (f *fakeAIMiniGatewayRuntime) roundTrip(t *testing.T, r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/admin/model-sources":
		return jsonResponse(http.StatusOK, f.sources), nil
	case r.Method == http.MethodPost && r.URL.Path == "/admin/model-sources":
		var input RuntimeModelSourceInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("decode create source: %v", err)
		}
		item := RuntimeModelSource{
			ID:              "src-" + strconv.Itoa(f.nextID),
			Name:            input.Name,
			BaseURL:         input.BaseURL,
			ProviderType:    input.ProviderType,
			DefaultModelID:  input.DefaultModelID,
			ExposedModelIDs: append([]string(nil), input.ExposedModelIDs...),
			Enabled:         input.Enabled,
			Position:        len(f.sources),
			APIKeyMasked:    "sk-****",
		}
		f.nextID++
		f.sources = append(f.sources, item)
		return jsonResponse(http.StatusCreated, item), nil
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/admin/model-sources/"):
		id := strings.TrimPrefix(r.URL.Path, "/admin/model-sources/")
		next := make([]RuntimeModelSource, 0, len(f.sources))
		for _, item := range f.sources {
			if item.ID == id {
				continue
			}
			next = append(next, item)
		}
		f.sources = next
		for index := range f.sources {
			f.sources[index].Position = index
		}
		return jsonResponse(http.StatusNoContent, nil), nil
	case r.Method == http.MethodPut && r.URL.Path == "/admin/selected-models":
		var input []SelectedModel
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("decode selected models: %v", err)
		}
		f.selectedModels = append([]SelectedModel(nil), input...)
		return jsonResponse(http.StatusOK, f.selectedModels), nil
	case r.Method == http.MethodGet && r.URL.Path == "/admin/selected-models":
		return jsonResponse(http.StatusOK, f.selectedModels), nil
	default:
		t.Fatalf("unexpected runtime request: %s %s", r.Method, r.URL.Path)
		return nil, nil
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, payload any) *http.Response {
	if payload == nil {
		return &http.Response{
			StatusCode: status,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}
	}

	body, _ := json.Marshal(payload)
	return &http.Response{
		StatusCode: status,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewReader(body)),
	}
}
