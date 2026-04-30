package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway"
	anthropicprovider "github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway/providers/anthropic"
	openaiprovider "github.com/xiaoyuandev/clash-for-ai/core/internal/localgateway/providers/openai"
	"github.com/xiaoyuandev/clash-for-ai/core/internal/provideradapter"
)

type Service struct {
	client *http.Client
}

func New(client *http.Client) *Service {
	if client == nil {
		client = &http.Client{}
	}
	return &Service{client: client}
}

func (e *Service) Handle(ctx context.Context, request localgateway.Request, source localgateway.ModelSource) (localgateway.Response, error) {
	target := provideradapter.ResolveTarget(
		source.ProviderType,
		source.BaseURL,
		source.APIKey,
		source.DefaultModelID,
	)

	upstreamRequest, err := buildUpstreamRequest(request, target)
	if err != nil {
		return localgateway.Response{}, err
	}

	req, err := http.NewRequestWithContext(ctx, upstreamRequest.Method, upstreamRequest.URL, bytes.NewReader(upstreamRequest.Body))
	if err != nil {
		return localgateway.Response{}, fmt.Errorf("build upstream http request: %w", err)
	}
	req.Header = upstreamRequest.Headers.Clone()

	resp, err := e.client.Do(req)
	if err != nil {
		return localgateway.Response{}, fmt.Errorf("execute upstream request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return localgateway.Response{}, fmt.Errorf("read upstream response: %w", err)
	}

	return localgateway.Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Body:       body,
	}, nil
}

func buildUpstreamRequest(request localgateway.Request, target provideradapter.Target) (localgateway.UpstreamRequest, error) {
	upstreamTarget := localgateway.UpstreamTarget{
		Name:           target.Route.UpstreamKind,
		BaseURL:        target.BaseURL,
		APIKey:         target.APIKey,
		ProviderType:   string(target.ProviderType),
		DefaultModelID: target.DefaultModelID,
	}

	switch target.Route.UpstreamKind {
	case "anthropic":
		return anthropicprovider.BuildRequest(request, upstreamTarget)
	case "openai":
		return openaiprovider.BuildRequest(request, upstreamTarget)
	default:
		return localgateway.UpstreamRequest{}, fmt.Errorf("unsupported upstream kind: %s", target.Route.UpstreamKind)
	}
}
