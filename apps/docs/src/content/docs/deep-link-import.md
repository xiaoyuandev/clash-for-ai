---
title: Deep Link Import for Relay Providers
description: How third-party relay websites can open Relay Switch and import provider or model source data with user confirmation.
slug: deep-link-import
---

## What this is

Relay Switch supports a desktop deep link import flow.

Third-party websites can open the desktop app with a URL like:

```text
relay-switch://v1/import?resource=provider&payload=BASE64URL_JSON
```

The desktop app will:

1. open or bring the existing window to the front,
2. parse the import request,
3. show an import confirmation dialog,
4. import the data only after the user confirms.

The current minimum scope supports:

1. `provider`
2. `model`

If you want a ready-to-use generator page, open:

<a href="../deeplink.html" target="_blank" rel="noreferrer">/deeplink.html</a>

## Quick integration function

If your relay website only wants to add a Provider import button, use this function.

Required inputs:

1. `name`
2. `baseUrl`
3. `apiKey`

```js
function createRelaySwitchProviderImportUrl({ name, baseUrl, apiKey }) {
  const payload = { name, baseUrl, apiKey };
  const json = JSON.stringify(payload);
  const bytes = new TextEncoder().encode(json);
  let binary = "";

  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }

  const encodedPayload = btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/g, "");

  return `relay-switch://v1/import?resource=provider&payload=${encodedPayload}`;
}

// Example usage in a button click handler:
const url = createRelaySwitchProviderImportUrl({
  name: "OpenRouter",
  baseUrl: "https://openrouter.ai/api/v1",
  apiKey: "sk-or-example"
});

window.location.href = url;
```

For a third-party relay website, a practical integration usually looks like this:

```html
<button id="import-to-relay-switch">Import to Relay Switch</button>

<script>
  document.getElementById("import-to-relay-switch").addEventListener("click", () => {
    window.location.href = createRelaySwitchProviderImportUrl({
      name: "Your Relay Name",
      baseUrl: "https://relay.example.com/v1",
      apiKey: "USER_VISIBLE_OR_USER_GENERATED_API_KEY"
    });
  });
</script>
```

## URL format

Use this structure:

```text
relay-switch://v1/import?resource=<provider|model>&payload=<base64url-json>
```

Rules:

1. `resource` must be `provider` or `model`
2. `payload` must be a base64url-encoded JSON object
3. the app will reject unsupported routes, invalid payloads, or missing required fields

## Local verification

If you want to verify this flow on your own machine, use a packaged desktop build instead of relying only on `pnpm dev`.

Why:

1. browsers ask the operating system to open `relay-switch://...`
2. the operating system needs a registered handler for that scheme
3. the packaged app declares that handler reliably
4. a dev process alone is often not enough to make the browser recognize the scheme

Recommended verification steps:

1. build a local packaged desktop app
2. launch the packaged app at least once
3. open <a href="../deeplink.html" target="_blank" rel="noreferrer">/deeplink.html</a>
4. click `Open Deep Link`
5. confirm that Relay Switch opens and shows the import confirmation dialog

If the browser reports that the scheme has no registered handler, the most common reason is that the packaged app has not been installed or launched yet.

## Provider payload

Supported fields:

```json
{
  "name": "OpenRouter",
  "baseUrl": "https://openrouter.ai/api/v1",
  "apiKey": "sk-or-example"
}
```

Notes:

1. `name`, `baseUrl`, and `apiKey` are required
2. this public payload is intentionally aligned with the current desktop add form
3. third-party integrators do not need to provide an auth mode field for the minimum import flow
4. Relay Switch currently treats imported Provider links with the default bearer-style behavior used by the existing add form

Example deep link:

```text
relay-switch://v1/import?resource=provider&payload=eyJuYW1lIjoiT3BlblJvdXRlciIsImJhc2VVcmwiOiJodHRwczovL29wZW5yb3V0ZXIuYWkvYXBpL3YxIiwiYXBpS2V5Ijoic2stb3ItZXhhbXBsZSJ9
```

## Model payload

Supported fields:

```json
{
  "name": "Relay Models",
  "baseUrl": "https://relay.example.com/v1",
  "apiKey": "sk-model-example",
  "providerType": "openai-compatible",
  "modelIds": ["gpt-4o-mini", "claude-3-7-sonnet"]
}
```

Notes:

1. `name`, `baseUrl`, `apiKey`, and at least one model id are required
2. `providerType` is optional
3. supported `providerType` values are:
   `openai-compatible`
   `anthropic-compatible`
4. if `providerType` is omitted, Relay Switch defaults to `openai-compatible`
5. the first model in `modelIds` becomes the default model id

Example deep link:

```text
relay-switch://v1/import?resource=model&payload=eyJuYW1lIjoiUmVsYXkgTW9kZWxzIiwiYmFzZVVybCI6Imh0dHBzOi8vcmVsYXkuZXhhbXBsZS5jb20vdjEiLCJhcGlLZXkiOiJzay1tb2RlbC1leGFtcGxlIiwicHJvdmlkZXJUeXBlIjoib3BlbmFpLWNvbXBhdGlibGUiLCJtb2RlbElkcyI6WyJncHQtNG8tbWluaSIsImNsYXVkZS0zLTctc29ubmV0Il19
```

## How to build the payload

The payload is:

1. a JSON object
2. UTF-8 encoded
3. base64url encoded

Base64url means:

1. replace `+` with `-`
2. replace `/` with `_`
3. remove trailing `=`

Example in JavaScript:

```js
function toBase64Url(value) {
  return btoa(JSON.stringify(value))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/g, "");
}

const payload = toBase64Url({
  name: "OpenRouter",
  baseUrl: "https://openrouter.ai/api/v1",
  apiKey: "sk-or-example"
});

const url = `relay-switch://v1/import?resource=provider&payload=${payload}`;
```

## User experience

When a user clicks the deep link:

1. the system asks whether to open Relay Switch,
2. Relay Switch opens,
3. the app shows an import confirmation dialog,
4. the user confirms or cancels,
5. the app imports the configuration only after confirmation.

## Security notes

Do not treat this flow as silent import.

Recommended practices:

1. always expect a user confirmation step
2. do not publicly expose real API keys in shared links
3. prefer short-lived or user-generated links when possible
4. validate `baseUrl` and payload fields on the sender side too

## Current limitations

This minimum implementation does not yet support:

1. `provider + models` combined import
2. signed payloads
3. remote config fetch by token
4. automatic overwrite or merge strategies
