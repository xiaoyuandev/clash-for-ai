package localgatewayruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgatewaystate"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/modelsource"
)

type stubModelSources struct {
	replaceOrderCalled bool
	replaceOrderInput  []modelsource.Source
}

func (s *stubModelSources) List(context.Context) ([]modelsource.Source, error) {
	return nil, nil
}

func (s *stubModelSources) ReplaceOrder(_ context.Context, items []modelsource.Source) ([]modelsource.Source, error) {
	s.replaceOrderCalled = true
	s.replaceOrderInput = items
	return items, nil
}

type stubSelectedStore struct{}

func (s *stubSelectedStore) ListSelectedModels(context.Context) ([]localgatewaystate.SelectedModel, error) {
	return nil, nil
}

func (s *stubSelectedStore) ReplaceSelectedModels(context.Context, []localgatewaystate.SelectedModel) ([]localgatewaystate.SelectedModel, error) {
	return nil, nil
}

type stubExecutor struct{}

func (s *stubExecutor) Handle(context.Context, localgateway.Request, localgateway.ModelSource) (localgateway.Response, error) {
	return localgateway.Response{}, nil
}

func TestHandlerSupportsModelSourceReorderContractPath(t *testing.T) {
	modelSources := &stubModelSources{}
	handler := NewHandler(modelSources, &stubSelectedStore{}, &stubExecutor{})

	req := httptest.NewRequest(http.MethodPut, "/admin/model-sources/order", strings.NewReader(`[{"id":"source-1"}]`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !modelSources.replaceOrderCalled {
		t.Fatalf("expected ReplaceOrder to be called")
	}
	if len(modelSources.replaceOrderInput) != 1 || modelSources.replaceOrderInput[0].ID != "source-1" {
		t.Fatalf("unexpected reorder input: %+v", modelSources.replaceOrderInput)
	}

	var response []modelsource.Source
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response) != 1 || response[0].ID != "source-1" {
		t.Fatalf("unexpected response: %+v", response)
	}
}
