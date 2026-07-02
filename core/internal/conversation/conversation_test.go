package conversation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeFilename(t *testing.T) {
	got := safeFilename("../relay:switch/session?")
	if strings.Contains(got, "/") || strings.Contains(got, "\\") || strings.Contains(got, "..") {
		t.Fatalf("unsafe filename %q", got)
	}
}

func TestLoadJSONLSessionKeepsWarnings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	content := strings.Join([]string{
		`{"role":"user","content":"hello","timestamp":"2026-06-30T10:00:00Z"}`,
		`not-json`,
		`{"role":"assistant","content":"world","timestamp":"2026-06-30T10:00:01Z"}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	session, err := loadJSONLSession(sessionRef{
		Source:      SourceCodexCLI,
		ID:          "session",
		ProjectID:   "project",
		ProjectName: "project",
		ProjectPath: dir,
		SourcePath:  path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(session.Events) != 2 {
		t.Fatalf("events = %d, want 2", len(session.Events))
	}
	if len(session.Warnings) != 1 {
		t.Fatalf("warnings = %d, want 1", len(session.Warnings))
	}
	if session.Summary.MessageCount != 2 {
		t.Fatalf("message count = %d, want 2", session.Summary.MessageCount)
	}
	if !strings.Contains(session.Markdown, "> hello") || !strings.Contains(session.Markdown, "world") {
		t.Fatalf("session markdown was not generated:\n%s", session.Markdown)
	}
}

func TestDiscoverClaudeSessionsSkipsSubagents(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "-Users-test-project")
	subagentDir := filepath.Join(projectDir, "session-id", "subagents")
	if err := os.MkdirAll(subagentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(projectDir, "session-id.jsonl")
	subPath := filepath.Join(subagentDir, "agent-a.jsonl")
	content := []byte(`{"type":"user","message":{"role":"user","content":"main session"},"timestamp":"2026-06-30T10:00:00Z"}`)
	if err := os.WriteFile(mainPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subPath, []byte(`{"type":"user","message":{"role":"user","content":"subagent session"},"timestamp":"2026-06-30T10:00:00Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	refs, warnings := discoverJSONLSessions(SourceClaudeCode, root, "Claude Code", root)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if len(refs) != 1 {
		t.Fatalf("refs = %d, want 1: %#v", len(refs), refs)
	}
	if refs[0].SourcePath != mainPath {
		t.Fatalf("source path = %q, want %q", refs[0].SourcePath, mainPath)
	}
}

func TestRenderMarkdownRedactsSecrets(t *testing.T) {
	markdown := RenderMarkdown(Session{
		Summary: SessionSummary{
			ID:          "session",
			Source:      SourceCodexCLI,
			ProjectName: "relay-switch",
			Title:       "Secret check",
		},
		Events: []Event{
			{Role: "user", Kind: "message", Text: "API_KEY=sk-testsecret123456"},
		},
	}, true)
	if strings.Contains(markdown, "sk-testsecret") {
		t.Fatalf("markdown was not redacted:\n%s", markdown)
	}
	if !strings.Contains(markdown, "[REDACTED]") {
		t.Fatalf("markdown missing redaction marker:\n%s", markdown)
	}
}

func TestRenderMarkdownUsesReadableTranscriptFormat(t *testing.T) {
	markdown := RenderMarkdown(Session{
		Summary: SessionSummary{
			ID:          "session",
			Source:      SourceCodexCLI,
			ProjectName: "relay-switch",
			Title:       "Readable export",
		},
		Events: []Event{
			{Role: "user", Kind: "message", Text: "请生成 Markdown\n并隐藏工具输出"},
			{Role: "assistant", Kind: "message", Text: "## 方案\n\n- 保留正文\n- 折叠工具"},
			{Role: "assistant", Kind: "tool_call", Text: "go test ./...", Language: "sh"},
			{Role: "assistant", Kind: "tool", Text: "ok", Language: "text"},
			{Role: "system", Kind: "meta", Text: "hidden"},
		},
	}, false)
	if strings.HasPrefix(markdown, "---") || strings.Contains(markdown, "source:") || strings.Contains(markdown, "session_id:") {
		t.Fatalf("markdown leaked front matter:\n%s", markdown)
	}
	if !strings.Contains(markdown, "> 请生成 Markdown\n> 并隐藏工具输出") {
		t.Fatalf("markdown missing user blockquote:\n%s", markdown)
	}
	if !strings.Contains(markdown, "## 方案\n\n- 保留正文") {
		t.Fatalf("markdown missing assistant markdown:\n%s", markdown)
	}
	if strings.Contains(markdown, "process events hidden") || strings.Contains(markdown, "go test ./...") || strings.Contains(markdown, "Tool call") || strings.Contains(markdown, "ok") {
		t.Fatalf("markdown leaked process details:\n%s", markdown)
	}
	if strings.Contains(markdown, "## User") || strings.Contains(markdown, "## Assistant") {
		t.Fatalf("markdown contains noisy transcript markers:\n%s", markdown)
	}
}

func TestClaudeJSONLSkipsInternalEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude.jsonl")
	content := strings.Join([]string{
		`{"type":"mode","mode":"normal"}`,
		`{"type":"user","message":{"role":"user","content":"请修复首页刷新问题"},"cwd":"` + dir + `","timestamp":"2026-06-30T10:00:00Z"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","thinking":"hidden"},{"type":"text","text":"## 修复方案\n\n- 刷新缓存\n- 验证首页"}]},"cwd":"` + dir + `","timestamp":"2026-06-30T10:00:01Z"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"rg cache"}}]},"cwd":"` + dir + `","timestamp":"2026-06-30T10:00:02Z"}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	session, err := loadJSONLSession(sessionRef{
		Source:      SourceClaudeCode,
		ID:          "claude",
		ProjectID:   "project",
		ProjectName: "project",
		ProjectPath: dir,
		SourcePath:  path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Summary.Title != "请修复首页刷新问题" {
		t.Fatalf("title = %q", session.Summary.Title)
	}
	if session.Summary.MessageCount != 2 {
		t.Fatalf("message count = %d, want 2", session.Summary.MessageCount)
	}
	if strings.Contains(RenderMarkdown(session, false), "hidden") {
		t.Fatalf("thinking text leaked into markdown")
	}
}

func TestCodexJSONLSkipsDeveloperContext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codex.jsonl")
	content := strings.Join([]string{
		`{"timestamp":"2026-06-30T10:00:00Z","type":"session_meta","payload":{"cwd":"` + dir + `"}}`,
		`{"timestamp":"2026-06-30T10:00:01Z","type":"response_item","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"<permissions instructions>hidden</permissions instructions>"}]}}`,
		`{"timestamp":"2026-06-30T10:00:02Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"保存文章为 Markdown"}]}}`,
		"{\"timestamp\":\"2026-06-30T10:00:03Z\",\"type\":\"response_item\",\"payload\":{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"已保存到 `posts/demo.md`。\"}]}}",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	summary, warnings := summarizeJSONL(path)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if summary.ProjectPath != dir {
		t.Fatalf("project path = %q, want %q", summary.ProjectPath, dir)
	}
	if summary.Title != "保存文章为 Markdown" {
		t.Fatalf("title = %q", summary.Title)
	}
	if summary.MessageCount != 2 {
		t.Fatalf("message count = %d, want 2", summary.MessageCount)
	}
}

func TestCodexJSONLExtractsActualRequestFromIDEContext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codex-context.jsonl")
	content := strings.Join([]string{
		`{"timestamp":"2026-06-30T10:00:00Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"<environment_context>\n  <cwd>` + dir + `</cwd>\n</environment_context>"}]}}`,
		`{"timestamp":"2026-06-30T10:00:01Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"# Context from my IDE setup:\n\n## Active file: README.md\n\n## My request for Codex:\n请增加 Provider 文档链接\n"}]}}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	summary, warnings := summarizeJSONL(path)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if summary.Title != "请增加 Provider 文档链接" {
		t.Fatalf("title = %q", summary.Title)
	}
	if summary.MessageCount != 1 {
		t.Fatalf("message count = %d, want 1", summary.MessageCount)
	}
}

func TestUserSkillInstructionsAreRemoved(t *testing.T) {
	got := normalizeMessageText(`# Skill Body

Long skill markdown that should not be shown.

ARGUMENTS: $ui-ux-pro-max 请基于文档设计 UI`, "user")
	if got != "$ui-ux-pro-max 请基于文档设计 UI" {
		t.Fatalf("skill arguments = %q", got)
	}

	got = normalizeMessageText(`<skills_instructions>
## Skills
Very long injected markdown.
</skills_instructions>
请继续实现功能`, "user")
	if got != "请继续实现功能" {
		t.Fatalf("skills instructions were not removed: %q", got)
	}

	got = normalizeMessageText(`<skill>
<name>ui-ux-pro-max</name>
<path>/tmp/SKILL.md</path>
# ui-ux-pro-max
Long skill markdown.
</skill>`, "user")
	if got != "" {
		t.Fatalf("skill block was not removed: %q", got)
	}
}
