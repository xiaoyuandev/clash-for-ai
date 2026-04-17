package logging

type RequestLog struct {
	ID           string  `json:"id"`
	Timestamp    string  `json:"timestamp"`
	ProviderID   string  `json:"provider_id"`
	ProviderName string  `json:"provider_name"`
	Method       string  `json:"method"`
	Path         string  `json:"path"`
	Model        *string `json:"model,omitempty"`
	StatusCode   *int    `json:"status_code,omitempty"`
	IsStream     bool    `json:"is_stream"`
	UpstreamHost string  `json:"upstream_host"`
	LatencyMs    int64   `json:"latency_ms"`
	FirstByteMs  *int64  `json:"first_byte_ms,omitempty"`
	FirstTokenMs *int64  `json:"first_token_ms,omitempty"`
	ErrorType    *string `json:"error_type,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
	ErrorSnippet *string `json:"error_snippet,omitempty"`
}
