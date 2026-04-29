package gatewayadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provider"
)

type runtimeAdminClient struct {
	baseURL string
	client  *http.Client
}

func (c runtimeAdminClient) ListModelSources(ctx context.Context) ([]modelsource.Source, error) {
	var items []modelsource.Source
	if err := c.doJSON(ctx, http.MethodGet, "/admin/model-sources", nil, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c runtimeAdminClient) CreateModelSource(ctx context.Context, input modelsource.CreateInput) (modelsource.Source, error) {
	var item modelsource.Source
	if err := c.doJSON(ctx, http.MethodPost, "/admin/model-sources", input, &item); err != nil {
		return modelsource.Source{}, err
	}
	return item, nil
}

func (c runtimeAdminClient) UpdateModelSource(ctx context.Context, id string, input modelsource.UpdateInput) (modelsource.Source, error) {
	var item modelsource.Source
	path := "/admin/model-sources/" + strings.TrimSpace(id)
	if err := c.doJSON(ctx, http.MethodPut, path, input, &item); err != nil {
		return modelsource.Source{}, err
	}
	return item, nil
}

func (c runtimeAdminClient) DeleteModelSource(ctx context.Context, id string) error {
	path := "/admin/model-sources/" + strings.TrimSpace(id)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

func (c runtimeAdminClient) ReplaceModelSourceOrder(ctx context.Context, items []modelsource.Source) ([]modelsource.Source, error) {
	var updated []modelsource.Source
	if err := c.doJSON(ctx, http.MethodPut, "/admin/model-sources/order", items, &updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func (c runtimeAdminClient) ListSelectedModels(ctx context.Context) ([]provider.SelectedModel, error) {
	var items []provider.SelectedModel
	if err := c.doJSON(ctx, http.MethodGet, "/admin/selected-models", nil, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c runtimeAdminClient) ReplaceSelectedModels(ctx context.Context, items []provider.SelectedModel) ([]provider.SelectedModel, error) {
	var updated []provider.SelectedModel
	if err := c.doJSON(ctx, http.MethodPut, "/admin/selected-models", items, &updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func (c runtimeAdminClient) doJSON(ctx context.Context, method string, path string, input any, output any) error {
	var body io.Reader
	if input != nil {
		payload, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if isAdminUnsupportedStatus(resp.StatusCode) {
		return ErrRuntimeAdminUnsupported
	}
	if resp.StatusCode >= 400 {
		snippet, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return fmt.Errorf("runtime admin request failed: HTTP %d", resp.StatusCode)
		}
		message := strings.TrimSpace(string(snippet))
		if message == "" {
			return fmt.Errorf("runtime admin request failed: HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("runtime admin request failed: HTTP %d: %s", resp.StatusCode, message)
	}

	if output == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func isAdminUnsupportedStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusNotImplemented:
		return true
	default:
		return false
	}
}
