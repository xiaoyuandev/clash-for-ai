# Relay Switch Plugin Development

Relay Switch owns the extension platform only. Concrete plugins live outside this repository, typically in their own public GitHub repositories.

## Platform Boundary

Relay Switch provides:

- Plugin discovery from `relay-switch-plugin.json`.
- Public GitHub install by `git clone`.
- Manual update from the installed Git repository.
- Enable, disable, and uninstall lifecycle state.
- Manifest-declared commands, settings, tool integrations, background tasks, permissions, and audit records.
- Controlled runtime entry planning for local processes and npm packages.

Relay Switch does not provide:

- Built-in concrete plugins in the main repository.
- Repo install/build script execution during plugin install.
- Arbitrary shell command execution from plugin manifests.
- Provider API key injection into plugin runtimes by default.

## Install Model

Install uses:

```http
POST /api/extensions/install
Content-Type: application/json

{
  "source": "github",
  "url": "https://github.com/owner/repo"
}
```

Rules:

- Only public `https://github.com/{owner}/{repo}` URLs are accepted.
- Relay Switch clones the repository into its managed data directory.
- The repository must contain exactly one `relay-switch-plugin.json`.
- The plugin is enabled automatically after a successful install.
- Update is manual through `POST /api/extensions/{id}/update`.
- Uninstall is `POST /api/extensions/{id}/uninstall` and removes plugin code, settings, grants, data, and audit records.

## Manifest

Minimum manifest:

```json
{
  "manifestVersion": 1,
  "id": "publisher.plugin-name",
  "name": "Plugin Name",
  "version": "1.0.0",
  "publisher": "publisher",
  "engines": {
    "relaySwitch": ">=1.0.0"
  },
  "entry": {
    "type": "none"
  },
  "contributes": {},
  "permissions": []
}
```

`id` must be a slug or reverse-domain identifier. `version` must be semver.

## Runtime Entries

`entry.type: "none"` declares a manifest-only plugin.

`entry.type: "process"` runs a repository-local executable. `entry.command` must be a relative path inside the plugin repository.

```json
{
  "entry": {
    "type": "process",
    "command": "./bin/plugin",
    "args": ["serve"]
  }
}
```

`entry.type: "nodePackage"` is the supported npx-style runtime. Relay Switch constructs the command itself.

```json
{
  "entry": {
    "type": "nodePackage",
    "package": "@publisher/relay-switch-plugin-example",
    "version": "1.2.3",
    "bin": "relay-switch-plugin-example",
    "args": ["serve"]
  }
}
```

Rules for `nodePackage`:

- `version` must be an exact semver version such as `1.2.3`.
- `latest`, ranges, dist-tags, Git URLs, and custom registries are not accepted in v1.
- Relay Switch plans `npx --yes --package <package>@<version> <bin> ...args`.
- Plugin authors cannot provide free-form shell command strings.

## JSON-RPC Runtime Protocol

Runtime processes are expected to use JSON-RPC 2.0 over stdio. The platform methods are:

- `initialize`
- `shutdown`
- `executeCommand`
- `settingsChanged`
- `getStatus`

Host responsibilities:

- Start the runtime when an enabled plugin has a process or node package entry.
- Stop the runtime on disable, update, and uninstall.
- Treat crashes and startup failures as degraded runtime state.
- Cap stderr/stdout capture and apply request timeouts.

Plugin runtimes should not assume access to Relay Switch provider credentials unless a future API explicitly grants them.

## Contributions

Supported contribution points:

- `commands`
- `settings`
- `toolIntegrations`
- `declaredProcesses`
- `backgroundTasks`
- `pages`
- `transcriptSources`
- `conversationExporters`
- `gatewayHooks`
- `gatewayProtocolBridges`
- `managedToolBinaries`
- `providerRequestTemplates`
- `runtimeAdapters`

Unknown contribution points load with warnings so newer plugins can remain inspectable on older hosts.
