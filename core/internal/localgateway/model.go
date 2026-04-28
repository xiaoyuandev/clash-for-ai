package localgateway

type InboundProtocol string

const (
	InboundOpenAI    InboundProtocol = "openai"
	InboundAnthropic InboundProtocol = "anthropic"
	InboundGemini    InboundProtocol = "gemini"
)

type Operation string

const (
	OperationChatCompletions Operation = "chat.completions"
	OperationResponses       Operation = "responses"
	OperationMessages        Operation = "messages"
	OperationCountTokens     Operation = "messages.count_tokens"
	OperationModels          Operation = "models"
)

type Request struct {
	Protocol  InboundProtocol
	Operation Operation
	Method    string
	Path      string
	Model     string
	Stream    bool
	Headers   map[string]string
	Body      []byte
}

type ModelSource struct {
	ID             string
	Name           string
	BaseURL        string
	APIKey         string
	ProviderType   string
	DefaultModelID string
	Enabled        bool
}

type UpstreamTarget struct {
	Name           string
	BaseURL        string
	APIKey         string
	ProviderType   string
	DefaultModelID string
}

type Response struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}
