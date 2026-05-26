"""Constants for Hermes Voice integration."""

from __future__ import annotations

from typing import Final

from homeassistant.const import CONF_NAME

DOMAIN: Final = "hermes_voice"

CONF_ENDPOINT: Final = "endpoint"
CONF_DEVICE_ID: Final = "device_id"
CONF_ALIAS: Final = "alias"
CONF_STATIC_RESPONSE: Final = "static_response"

DEFAULT_NAME: Final = "Hermes Voice"
DEFAULT_DEVICE_ID: Final = "phone_ha"
DEFAULT_ALIAS: Final = ""
DEFAULT_STATIC_RESPONSE: Final = "Hermes Voice custom conversation agent is loaded."

CONFIG_KEYS: Final = (
    CONF_NAME,
    CONF_ENDPOINT,
    CONF_DEVICE_ID,
    CONF_ALIAS,
    CONF_STATIC_RESPONSE,
)
