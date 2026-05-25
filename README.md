# Hermes Voice Bridge

Local voice interface for Hermes Agent.

Baseline:
- Home Assistant Assist as the voice platform.
- Orange Pi 3B 8GB as the backend host.
- Armbian/Debian + Docker Compose + Home Assistant Container.
- Home Assistant Voice Preview Edition ESP32-S3 as the target voice satellite.
- Android smartphone with Home Assistant app / Assist as the temporary development client.

This repository intentionally starts platform-agnostic around Home Assistant/local voice assumptions, not Yandex Alice assumptions.

## Current MVP core

The first implemented core pieces are:

- a local YAML registry package:
  - schema v1 for backends, models, persons, profiles, devices, and device-local aliases;
  - strict loading and semantic validation for hand-edited YAML;
  - deterministic resolution from `device_id + optional alias` to person/profile/model/backend;
  - typed lookup and validation errors;
  - inline secrets are rejected; use references such as `env:HERMES_API_KEY`;
- an isolated rules-based speech cleanup package for utterance text;
- a transport-neutral backend adapter contract for one resolved invocation attempt.

### Registry routing contract

`Registry.Resolve(deviceID, alias)` is the MVP routing contract.

Input:
- `deviceID`: registry device key;
- `alias`: optional device-local alias key.

Default route:
- empty `alias` uses `devices.<device>.default_person` and `devices.<device>.default_profile`.

Alias route:
- non-empty `alias` is matched exactly inside the selected device only;
- no trimming, lowercasing, Unicode normalization, fuzzy matching, or global alias lookup;
- an alias may override `person`, `profile`, or both;
- a missing alias `person` inherits the device default person;
- a missing alias `profile` inherits the device default profile;
- unknown alias returns `ErrAliasNotFound` and does not fallback to the default route.

Output:
- `ResolvedContext` contains device, alias, person, profile, model, and backend IDs plus their resolved structs.

Precondition:
- use registries returned by `Load` / `LoadFile`, or call `Validate` before routing manually constructed registries.

### Speech cleanup boundary

`internal/cleanup` provides deterministic rules-based cleanup for the recognized utterance body before future backend calls.

Contract:
- cleanup operates only on utterance text, not on registry device IDs or alias keys;
- it is not registry or alias normalization and must not change `Registry.Resolve` matching semantics;
- rules run in declared order and support conservative whitespace trim/collapse, prefix removal, suffix removal, and literal phrase replacement;
- `CleanWithTrace` returns the exact original input, final safe cleaned text, and before/after snapshots for each rule that changed text;
- default rules are conservative: trim/collapse whitespace, remove leading Russian Hermes wake/filler phrases, and remove trailing `пожалуйста`;
- if non-whitespace input would be cleaned to an empty string, cleanup falls back to the trimmed/collapsed original text.

### Backend adapter boundary

`internal/backend` defines the execution boundary used after registry routing and speech cleanup have already produced a resolved request.

Contract:
- `Adapter.Invoke(ctx, Request)` represents exactly one backend call attempt;
- `Request` carries transport-neutral fields such as input text, device/alias context, person/profile/model/backend IDs, model name, system prompt, and metadata;
- `Request.Validate()` requires `Input`, `PersonID`, `ProfileID`, `ModelID`, `BackendID`, and `ModelName`;
- `Response` supports `completed`, `accepted`, and `failed` status values, optional token usage, metadata, and a `TaskID` shape for future long-running work;
- typed backend errors preserve `errors.Is` matching for invalid requests, unsupported backends, invocation failures, temporary failures, unauthorized failures, and context cancellation.

Non-goals for this package:
- it does not perform real Hermes HTTP or CLI invocation yet;
- it does not load or materialize API keys/secrets;
- cleanup, routing, and registry remain separate packages and are not imported by `internal/backend`;
- `StatusAccepted` plus `TaskID` is only response shape compatibility, not an async task store, polling loop, or cancellation lifecycle;
- no streaming, retries, conversation/session lifecycle, or user-facing error presentation is implemented here.

A deterministic static adapter is provided only for tests and contract proof; it validates the request, respects an already-canceled context, and returns the configured response or configured error without network calls.

### Temporary dev HTTP/text client

A dev-only localhost HTTP endpoint is available to exercise the current MVP flow:

```text
HTTP JSON input → registry resolve → speech cleanup → static backend response
```

This is not a production API and does not invoke real Hermes transport yet. It does not add auth, streaming, async task storage, retries, Home Assistant integration, or API key resolution.

Start it with localhost defaults:

```bash
go run ./cmd/hermes-voice \
  --registry testdata/registry.yaml \
  --listen 127.0.0.1:8081 \
  --static-output "static dev response"
```

Health check:

```bash
curl http://127.0.0.1:8081/healthz
```

Text request example:

```bash
curl -sS http://127.0.0.1:8081/v1/dev/text \
  -H 'content-type: application/json' \
  -d '{"request_id":"dev-1","device_id":"phone_ha","alias":"coding","input":"гермес помоги написать тест","metadata":{"source":"curl"}}'
```

The response includes the selected route IDs, cleanup trace, static backend output, usage shape, and response metadata. It intentionally does not expose backend endpoints or `api_key_ref` values.

Run tests:

```bash
go test ./...
```
