package tooling

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestApplyCodexIntegrationWritesBearerTokenAndNullAPIKey(t *testing.T) {
	setupCodexIntegrationTestHome(t)
	writeTestText(t, codexConfigPath(), strings.Join([]string{
		`model_provider = "Other"`,
		`experimental_bearer_token = "old"`,
		"",
		"[model_providers.OpenAI]",
		`name = "OpenAI"`,
		`base_url = "https://api.openai.com/v1"`,
	}, "\n")+"\n")
	writeTestText(t, codexAuthPath(), `{"OPENAI_API_KEY":"sk-old","OTHER_FIELD":"kept"}`+"\n")

	state, err := applyCodexIntegration(3456)
	if err != nil {
		t.Fatalf("apply codex integration: %v", err)
	}
	if !state.Configured {
		t.Fatalf("expected configured state: %+v", state)
	}

	config := readTestText(t, codexConfigPath())
	for _, expected := range []string{
		`model_provider = "OpenAI"`,
		`experimental_bearer_token = "dummy"`,
		`base_url = "http://127.0.0.1:3456/v1"`,
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("config missing %q:\n%s", expected, config)
		}
	}

	var auth map[string]any
	if err := json.Unmarshal([]byte(readTestText(t, codexAuthPath())), &auth); err != nil {
		t.Fatalf("decode codex auth: %v", err)
	}
	if value, ok := auth["OPENAI_API_KEY"]; !ok || value != nil {
		t.Fatalf("OPENAI_API_KEY should be JSON null: %+v", auth)
	}
	if auth["OTHER_FIELD"] != "kept" {
		t.Fatalf("auth should preserve existing fields: %+v", auth)
	}
}

func TestIsCodexConfiguredRequiresNullAPIKeyAndBearerToken(t *testing.T) {
	setupCodexIntegrationTestHome(t)

	writeValidCodexIntegrationConfig(t, 3456)
	writeTestText(t, codexAuthPath(), `{"OPENAI_API_KEY":null}`+"\n")
	configured, err := isCodexConfigured(3456)
	if err != nil {
		t.Fatalf("inspect valid codex config: %v", err)
	}
	if !configured {
		t.Fatal("expected valid codex config to be configured")
	}

	cases := []struct {
		name    string
		config  string
		auth    string
		apiPort int
	}{
		{
			name:    "missing api key",
			config:  validCodexIntegrationConfig(3456),
			auth:    `{}`,
			apiPort: 3456,
		},
		{
			name:    "string api key",
			config:  validCodexIntegrationConfig(3456),
			auth:    `{"OPENAI_API_KEY":"dummy"}`,
			apiPort: 3456,
		},
		{
			name: "missing bearer token",
			config: strings.Join([]string{
				`model_provider = "OpenAI"`,
				"",
				"[model_providers.OpenAI]",
				`base_url = "http://127.0.0.1:3456/v1"`,
			}, "\n") + "\n",
			auth:    `{"OPENAI_API_KEY":null}`,
			apiPort: 3456,
		},
		{
			name:    "wrong bearer token",
			config:  strings.Replace(validCodexIntegrationConfig(3456), `experimental_bearer_token = "dummy"`, `experimental_bearer_token = "wrong"`, 1),
			auth:    `{"OPENAI_API_KEY":null}`,
			apiPort: 3456,
		},
		{
			name:    "wrong base url port",
			config:  validCodexIntegrationConfig(3457),
			auth:    `{"OPENAI_API_KEY":null}`,
			apiPort: 3456,
		},
		{
			name:    "wrong model provider",
			config:  strings.Replace(validCodexIntegrationConfig(3456), `model_provider = "OpenAI"`, `model_provider = "Other"`, 1),
			auth:    `{"OPENAI_API_KEY":null}`,
			apiPort: 3456,
		},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			writeTestText(t, codexConfigPath(), item.config)
			writeTestText(t, codexAuthPath(), item.auth+"\n")
			configured, err := isCodexConfigured(item.apiPort)
			if err != nil {
				t.Fatalf("inspect codex config: %v", err)
			}
			if configured {
				t.Fatal("expected codex config to be unconfigured")
			}
		})
	}
}

func setupCodexIntegrationTestHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func writeValidCodexIntegrationConfig(t *testing.T, apiPort int) {
	t.Helper()
	writeTestText(t, codexConfigPath(), validCodexIntegrationConfig(apiPort))
}

func validCodexIntegrationConfig(apiPort int) string {
	return strings.Join([]string{
		`model_provider = "OpenAI"`,
		`experimental_bearer_token = "dummy"`,
		"",
		"[model_providers.OpenAI]",
		`base_url = "http://127.0.0.1:` + intToString(apiPort) + `/v1"`,
	}, "\n") + "\n"
}
