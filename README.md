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

The first implemented core piece is a local YAML registry package:

- schema v1 for backends, models, persons, profiles, devices, and device-local aliases;
- strict loading and semantic validation for hand-edited YAML;
- deterministic resolution from `device_id + optional alias` to person/profile/model/backend;
- typed lookup and validation errors;
- inline secrets are rejected; use references such as `env:HERMES_API_KEY`.

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

Run tests:

```bash
go test ./...
```
