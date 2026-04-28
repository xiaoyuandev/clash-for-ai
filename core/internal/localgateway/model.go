package localgateway

type InboundProtocol string

const (
	InboundOpenAI    InboundProtocol = "openai"
	InboundAnthropic InboundProtocol = "anthropic"
	InboundGemini    InboundProtocol = "gemini"
)

type Request struct {
	Protocol InboundProtocol
	Path     string
	Model    string
	Stream   bool
	Headers  map[string]string
	Body     []byte
}

type UpstreamTarget struct {
	Name         string
	BaseURL      string
	APIKey       string
	ProviderType string
	ModelID      string
}

type Response struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}
