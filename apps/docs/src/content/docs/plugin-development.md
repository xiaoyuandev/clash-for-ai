---
title: Plugin Development
description: Relay Switch external plugin manifest, install, update, uninstall, and runtime API reference.
slug: plugin-development
---

Relay Switch owns the extension platform only. Concrete plugins live outside this repository, typically in their own public GitHub repositories.

## Platform boundary

Relay Switch provides manifest discovery, public GitHub install by `git clone`, manual update, enable/disable/uninstall lifecycle state, settings, contributions, permissions, audit records, and controlled runtime entry planning.

Relay Switch does not run plugin repository install/build scripts during install, does not keep concrete plugins in the main repository, does not accept arbitrary shell command strings, and does not inject provider API keys into plugin runtimes by default.

## Install API

```http
POST /api/extensions/install
Content-Type: application/json

{
  "source": "github",
  "url": "https://github.com/owner/repo"
}
```

Rules:

1. Only public `https://github.com/{owner}/{repo}` URLs are accepted.
2. Relay Switch clones the repository into its managed data directory.
3. The repository must contain exactly one `relay-switch-plugin.json`.
4. A successful install enables the plugin automatically.
5. Update is manual through `POST /api/extensions/{id}/update`.
6. Uninstall uses `POST /api/extensions/{id}/uninstall` and removes plugin code, settings, grants, data, and audit records.

## Manifest

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

## Runtime entries

`entry.type: "none"` declares a manifest-only plugin.

`entry.type: "process"` runs a repository-local executable:

```json
{
  "entry": {
    "type": "process",
    "command": "./bin/plugin",
    "args": ["serve"]
  }
}
```

`entry.type: "nodePackage"` is the supported npx-style runtime:

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

Relay Switch plans `npx --yes --package <package>@<version> <bin> ...args`. Versions must be exact semver. `latest`, ranges, dist-tags, Git URLs, and custom registries are not accepted in v1.

## Runtime protocol

Runtime processes are expected to use JSON-RPC 2.0 over stdio with these methods:

- `initialize`
- `shutdown`
- `executeCommand`
- `settingsChanged`
- `getStatus`

The host starts enabled runtime entries, stops them on disable/update/uninstall, treats crashes as degraded runtime state, and applies output caps and request timeouts.
