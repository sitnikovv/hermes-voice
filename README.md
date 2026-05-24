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
- deterministic resolution from `device_id + optional alias` to person/profile/model/backend;
- typed lookup errors;
- inline secrets are rejected; use references such as `env:HERMES_API_KEY`.

Run tests:

```bash
go test ./...
```
