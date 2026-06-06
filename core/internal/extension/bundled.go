package extension

import "encoding/json"

const MarkdownArchivePluginID = "relay-switch.markdown-archive"

type BundledPlugin struct {
	Manifest     Manifest
	ManifestPath string
}

func MarkdownArchiveBundledPlugin() BundledPlugin {
	return BundledPlugin{
		ManifestPath: "bundled://relay-switch.markdown-archive/relay-switch-plugin.json",
		Manifest: Manifest{
			ManifestVersion: 1,
			ID:              MarkdownArchivePluginID,
			Name:            "Markdown Archive",
			Version:         "0.1.0",
			Description:     "Exports local AI tool transcripts to Markdown.",
			Publisher:       "relay-switch",
			Engines: ManifestEngines{
				RelaySwitch: ">=1.0.0",
			},
			Entry: ManifestEntry{
				Type: "none",
			},
			Contributes: ManifestContributes{
				"commands":              mustRawJSON(markdownArchiveCommandsJSON),
				"settings":              mustRawJSON(markdownArchiveSettingsJSON),
				"pages":                 mustRawJSON(markdownArchivePagesJSON),
				"transcriptSources":     mustRawJSON(markdownArchiveTranscriptSourcesJSON),
				"conversationExporters": mustRawJSON(markdownArchiveExportersJSON),
				"backgroundTasks":       mustRawJSON(markdownArchiveBackgroundTasksJSON),
			},
			Permissions: []string{
				"tool.transcripts.read",
				"filesystem.userSelectedDirectory.write",
				"filesystem.pluginData",
				"background.task",
				"ui.page",
				"ui.toast",
			},
		},
	}
}

func mustRawJSON(value string) json.RawMessage {
	return json.RawMessage(value)
}

const markdownArchiveCommandsJSON = `[
  {
    "id": "markdownArchive.syncNow",
    "title": "Sync Conversations Now",
    "category": "Archive"
  },
  {
    "id": "markdownArchive.openOutputDirectory",
    "title": "Open Output Directory",
    "category": "Archive"
  }
]`

const markdownArchiveSettingsJSON = `{
  "type": "object",
  "properties": {
    "enabled": {
      "type": "boolean",
      "title": "Enabled",
      "default": false
    },
    "outputDirectory": {
      "type": "string",
      "title": "Output Directory",
      "default": ""
    },
    "includeClaudeCode": {
      "type": "boolean",
      "title": "Include Claude Code",
      "default": true
    },
    "includeCodexCLI": {
      "type": "boolean",
      "title": "Include Codex CLI",
      "default": true
    },
    "autoSync": {
      "type": "boolean",
      "title": "Auto Sync",
      "default": false
    },
    "syncIntervalSeconds": {
      "type": "integer",
      "title": "Sync Interval Seconds",
      "default": 300
    },
    "includeSystemEvents": {
      "type": "boolean",
      "title": "Include System Events",
      "default": false
    },
    "redactSecrets": {
      "type": "boolean",
      "title": "Redact Secrets",
      "default": true
    },
    "overwriteExisting": {
      "type": "boolean",
      "title": "Overwrite Existing Files",
      "default": true
    }
  }
}`

const markdownArchivePagesJSON = `[
  {
    "id": "markdown-archive",
    "title": "Markdown Archive",
    "route": "/plugins/markdown-archive",
    "kind": "settings-form"
  }
]`

const markdownArchiveTranscriptSourcesJSON = `[
  {
    "id": "claude-code",
    "title": "Claude Code",
    "kind": "local-jsonl"
  },
  {
    "id": "codex-cli",
    "title": "Codex CLI",
    "kind": "local-jsonl"
  }
]`

const markdownArchiveExportersJSON = `[
  {
    "id": "obsidian-markdown",
    "title": "Obsidian Markdown",
    "format": "markdown"
  }
]`

const markdownArchiveBackgroundTasksJSON = `[
  {
    "id": "markdownArchive.autoSync",
    "title": "Auto Sync Conversations",
    "minimumIntervalSeconds": 30
  }
]`
