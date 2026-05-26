# Home Assistant integration artifacts

This directory contains reproducible Home Assistant-side artifacts for Hermes Voice Bridge.

## `custom_components/hermes_voice`

Config-entry-backed custom conversation agent for Home Assistant Assist.

It provides a selectable Assist pipeline conversation agent entity, typically:

```text
conversation.hermes_voice
```

The integration keeps YAML import support for spike deployments. Existing YAML like this is imported into a config entry on HA startup:

```yaml
hermes_voice:
  endpoint: "http://127.0.0.1:8081/v1/dev/text"
  device_id: "phone_ha"
  alias: ""
```

Manual setup is also available through:

```text
Settings -> Devices & services -> Add integration -> Hermes Voice
```

After HA restart and import/setup, select it in the Assist pipeline:

```text
Settings -> Voice assistants -> <pipeline> -> Conversation agent / Диалоговая система -> Hermes Voice
```

Test through Home Assistant Conversation API after the entity exists:

```bash
curl -sS -X POST http://127.0.0.1:8123/api/conversation/process \
  -H "Authorization: Bearer <HA_LONG_LIVED_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"text":"тест hermes voice","language":"ru","agent_id":"conversation.hermes_voice"}'
```

This is a spike artifact, not a production HA integration package.
