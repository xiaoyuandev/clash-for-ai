package conversation

import (
	"regexp"
	"strings"
)

func RenderMarkdown(session Session, redact bool) string {
	var builder strings.Builder
	summary := session.Summary
	writeLine(&builder, "# "+escapeMarkdownHeading(summary.Title))
	writeLine(&builder, "")

	processEvents := []Event{}
	for _, event := range session.Events {
		if shouldSkipMarkdownEvent(event) {
			continue
		}
		text := event.Text
		if redact {
			text = redactSecrets(text)
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		switch {
		case event.Role == "user" && event.Kind == "message":
			flushProcessGroup(&builder, processEvents, redact)
			processEvents = []Event{}
			writeBlockquote(&builder, text)
			writeLine(&builder, "")
		case event.Role == "assistant" && event.Kind == "message":
			flushProcessGroup(&builder, processEvents, redact)
			processEvents = []Event{}
			writeLine(&builder, text)
			writeLine(&builder, "")
		default:
			processEvents = append(processEvents, event)
		}
	}
	flushProcessGroup(&builder, processEvents, redact)

	return builder.String()
}

func shouldSkipMarkdownEvent(event Event) bool {
	if event.Kind == "meta" {
		return true
	}
	if event.Kind == "raw" && strings.TrimSpace(event.Text) == "" {
		return true
	}
	return false
}

func markdownEventTitle(event Event) string {
	switch event.Kind {
	case "tool_call":
		return "Tool call"
	case "tool":
		return "Tool output"
	case "raw":
		return "Raw event"
	default:
		if event.Role == "" {
			return "Additional event"
		}
		return strings.Title(strings.ReplaceAll(event.Role, "_", " "))
	}
}

func writeBlockquote(builder *strings.Builder, value string) {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			writeLine(builder, ">")
			continue
		}
		writeLine(builder, "> "+line)
	}
}

func flushProcessGroup(_ *strings.Builder, _ []Event, _ bool) {
	// The readable transcript intentionally omits process/tool/raw details.
}

func writeLine(builder *strings.Builder, value string) {
	builder.WriteString(value)
	builder.WriteByte('\n')
}

func escapeMarkdownHeading(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Untitled session"
	}
	return strings.TrimLeft(value, "# ")
}

func fenceSafe(value string) string {
	return strings.ReplaceAll(value, "```", "`\u200b``")
}

func redactSecrets(value string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password)=([^\s]+)`),
		regexp.MustCompile(`(?i)(bearer\s+)[a-z0-9._\-]+`),
		regexp.MustCompile(`sk-[A-Za-z0-9_\-]{12,}`),
	}
	redacted := value
	for _, pattern := range patterns {
		redacted = pattern.ReplaceAllString(redacted, "$1[REDACTED]")
	}
	return redacted
}
