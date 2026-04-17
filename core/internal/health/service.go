package health

import "time"

type CheckResult struct {
	Status      string    `json:"status"`
	StatusCode  int       `json:"status_code"`
	LatencyMs   int64     `json:"latency_ms"`
	Summary     string    `json:"summary"`
	CheckedAt   time.Time `json:"checked_at"`
	ProviderID  string    `json:"provider_id"`
	ProviderURL string    `json:"provider_url"`
}
