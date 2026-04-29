package gatewayadapter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/settings"
)

func TestEmbeddedLocalRuntimeAdapterAdminOperations(t *testing.T) {
	selectedModels := []provider.SelectedModel{{ModelID: "claude-3-7-sonnet", Position: 0}}
	adapter := NewEmbeddedLocalRuntimeAdapter(settings.LocalGatewaySettings{
		ListenHost: "127.0.0.1",
		ListenPort: 8788,
	}, "/bin/echo", t.TempDir())
	adapter.client = &http.Client{
		Transport: externalRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.Method + " " + req.URL.Path {
			case http.MethodGet + " /health":
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			case http.MethodGet + " /admin/selected-models":
				payload, err := json.Marshal(selectedModels)
				if err != nil {
					t.Fatalf("marshal selected models: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(string(payload))),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			case http.MethodPut + " /admin/selected-models":
				var input []provider.SelectedModel
				if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
					t.Fatalf("decode selected models input: %v", err)
				}
				payload, err := json.Marshal(input)
				if err != nil {
					t.Fatalf("marshal selected models response: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(string(payload))),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			default:
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
				return nil, nil
			}
		}),
	}
	adapter.admin.client = adapter.client

	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper process: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	adapter.process = cmd

	gotSelected, err := adapter.ListSelectedModels(context.Background())
	if err != nil {
		t.Fatalf("ListSelectedModels returned error: %v", err)
	}
	if len(gotSelected) != 1 || gotSelected[0].ModelID != "claude-3-7-sonnet" {
		t.Fatalf("unexpected selected models: %+v", gotSelected)
	}

	replacedSelected, err := adapter.ReplaceSelectedModels(context.Background(), selectedModels)
	if err != nil {
		t.Fatalf("ReplaceSelectedModels returned error: %v", err)
	}
	if len(replacedSelected) != 1 || replacedSelected[0].ModelID != "claude-3-7-sonnet" {
		t.Fatalf("unexpected replaced selected models: %+v", replacedSelected)
	}
}
