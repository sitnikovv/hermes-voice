# Home Assistant integration artifacts

This directory contains reproducible Home Assistant-side artifacts for Hermes Voice Bridge.

## `custom_components/hermes_voice`

Minimal YAML-loaded custom conversation agent for Веха 3 feasibility spike.

It registers a conversation agent with:

```text
agent_id: hermes_voice
```

Static smoke-test configuration in Home Assistant `configuration.yaml`:

```yaml
hermes_voice:
  static_response: "Hermes Voice custom conversation agent is loaded."
```

Bridge configuration for the current dev HTTP endpoint:

```yaml
hermes_voice:
  endpoint: "http://127.0.0.1:8081/v1/dev/text"
  device_id: "phone_ha"
  alias: ""
```

Test through Home Assistant Conversation API:

```bash
curl -sS -X POST http://127.0.0.1:8123/api/conversation/process \
  -H "Authorization: Bearer <HA_LONG_LIVED_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"text":"тест hermes voice","language":"ru","agent_id":"hermes_voice"}'
```

This is a spike artifact, not a production HA integration package.
