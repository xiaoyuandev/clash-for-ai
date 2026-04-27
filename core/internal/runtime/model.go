package runtime

import "time"

type Mode string

const (
	ModeLegacy          Mode = "legacy"
	ModeExternalPortkey Mode = "external-portkey"
)

type Config struct {
	Mode    Mode   `json:"mode"`
	BaseURL string `json:"base_url"`
}

type Health struct {
	Mode      Mode      `json:"mode"`
	BaseURL   string    `json:"base_url"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	CheckedAt time.Time `json:"checked_at"`
}
