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
- a dispatcher wrapper that enforces a quick-response budget and can return a minimal `accepted + task_id` fallback for slow backend work.

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
- it does not load or materialize API keys/secrets;
- cleanup, routing, and registry remain separate packages and are not imported by `internal/backend`;
- `StatusAccepted` plus `TaskID` is only response shape compatibility, not an async task store, polling loop, or cancellation lifecycle;
- no streaming, retries, conversation/session lifecycle, or user-facing error presentation is implemented here.

A deterministic static adapter is provided only for tests and contract proof; it validates the request, respects an already-canceled context, and returns the configured response or configured error without network calls.

A dev/MVP Hermes CLI adapter is available for real invocation through `hermes chat` in quiet non-interactive mode. It executes an argv vector directly (no shell), strips the `session_id:` line from stdout, returns the remaining final answer, bounds output size, and maps timeout/non-zero/missing-command failures to backend errors. It expects the target host/user to already have a working Hermes CLI configuration.

### Fast response, accepted fallback, and local task storage boundary

`internal/dispatch` wraps a `backend.Adapter` with a small quick-response timeout.

Behavior:
- if the backend completes or fails before the timeout, the original response/error is returned unchanged;
- if the caller context is canceled before the timeout, the context error is returned and no background task is started;
- if the timeout fires first, the dispatcher returns `StatusAccepted` with a non-empty `task_id` and metadata `accepted_by=dispatcher`, `reason=quick_timeout`;
- the original backend invoke may continue on a dispatcher-owned context detached from later caller cancellation;
- the dispatcher stores accepted/completed/failed lifecycle updates in the local task store when configured;
- the minimal task store is process-local memory: no restart persistence, retention policy, cancellation, retries, or real Hermes transport yet.

`GET /v1/dev/tasks/{task_id}` exposes local dev task status/result lookup for accepted fallback tasks.

Goal 008 intentionally does not add disk persistence, task listing, cancellation, retries, streaming, auth, or real Hermes transport. Those remain later boundaries.

### Temporary dev HTTP/text client

A dev-only localhost HTTP endpoint is available to exercise the current MVP flow:

```text
HTTP JSON input → registry resolve → speech cleanup → backend adapter response
```

This is not a production API. It does not add auth, streaming, durable async task storage, retries, Home Assistant packaging guarantees, or API key resolution.

Start it with static localhost defaults:

```bash
go run ./cmd/hermes-voice \
  --registry testdata/registry.yaml \
  --listen 127.0.0.1:8081 \
  --static-output "static dev response"
```

To run a real Hermes CLI backend on a host where `hermes chat -q ... -Q` already works:

```bash
go run ./cmd/hermes-voice \
  --registry testdata/registry.yaml \
  --listen 127.0.0.1:8081 \
  --backend hermes-cli \
  --hermes-command /home/sve/.local/bin/hermes \
  --hermes-source hermes-voice-local-smoke \
  --hermes-timeout 180s \
  --hermes-max-turns 3
```

The Hermes CLI backend is dev/MVP only. It assumes the process user already has valid Hermes config/auth. It uses direct argv execution, not shell evaluation. Do not expose this dev HTTP endpoint broadly on an untrusted network.

### Dev voice lifecycle scripts

For the current WSL + Orange Pi dev topology, use:

```bash
./scripts/dev-voice-up
./scripts/dev-voice-status
./scripts/dev-voice-down
```

`dev-voice-up` builds/deploys the linux/arm64 edge binary, starts the WSL central backend, starts the Orange Pi forwarder, and runs a smoke check. State and logs live under `.hermes/dev-voice/`.

### Edge forwarder mode

For development with one central backend and one or more Home Assistant / voice nodes, run a lightweight forwarder near HA:

```bash
go run ./cmd/hermes-voice \
  --mode forwarder \
  --listen 127.0.0.1:8081 \
  --forwarder-upstream http://192.168.7.10:18081 \
  --forwarder-edge-id orange-pi-ha \
  --forwarder-edge-room cabinet \
  --forwarder-device-id phone_ha
```

The forwarder exposes the same local endpoints (`/healthz`, `/v1/dev/text`, `/v1/dev/tasks/{task_id}`), adds edge metadata, and forwards to the central backend. It intentionally does not perform registry routing, cleanup, Hermes invocation, or task storage.

To demonstrate the minimal accepted fallback without real Hermes transport, delay the static backend longer than the dispatcher quick timeout:

```bash
go run ./cmd/hermes-voice \
  --registry testdata/registry.yaml \
  --listen 127.0.0.1:8081 \
  --quick-timeout 10ms \
  --static-delay 200ms
```

`--accepted-task-id` is only for deterministic tests and one-shot contract demos. Do not use it for a live/dev bridge connected to Home Assistant Assist: repeated accepted requests would reuse the same task id and conflict. Live smoke checks should send two accepted requests in a row and verify they return different `task_id` values.

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

The response includes the selected route IDs, cleanup trace, backend output, usage shape, and response metadata. It intentionally does not expose backend endpoints or `api_key_ref` values.

With the accepted fallback flags above, the same HTTP endpoint still returns HTTP `200`, but the JSON backend shape contains `"status":"accepted"` and a generated `"task_id"`.

For deterministic tests only, `--accepted-task-id dev-task-1` forces a fixed `task_id`.

Fetch local task status/result:

```bash
curl -sS http://127.0.0.1:8081/v1/dev/tasks/<task_id-from-response>
```

The task endpoint returns process-local state only. Tasks are lost on process restart, there is no list endpoint, no cancellation endpoint, no retry policy, and no durable result storage yet.

Backup registry before manual/future automated changes:

```bash
go run ./cmd/hermes-voice \
  --registry testdata/registry.yaml \
  --backup-registry
```

By default backups are written to `<registry-dir>/.registry-backups`. Override with `--registry-backup-dir <path>`. The backup is an exact byte-for-byte copy; this command does not validate, rewrite, restore, or mutate the registry.

Run tests:

```bash
go test ./...
```
