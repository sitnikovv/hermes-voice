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
- an isolated rules-based speech cleanup package for utterance text.

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

Run tests:

```bash
go test ./...
```
