"""Hermes Voice Home Assistant integration."""

from __future__ import annotations

import logging
from typing import Any

import voluptuous as vol

from homeassistant.config_entries import SOURCE_IMPORT, ConfigEntry
from homeassistant.const import CONF_NAME, Platform
from homeassistant.core import HomeAssistant
from homeassistant.helpers import config_validation as cv
from homeassistant.helpers.typing import ConfigType

from .const import (
    CONF_ALIAS,
    CONF_DEVICE_ID,
    CONF_ENDPOINT,
    CONF_STATIC_RESPONSE,
    DEFAULT_ALIAS,
    DEFAULT_DEVICE_ID,
    DEFAULT_NAME,
    DEFAULT_STATIC_RESPONSE,
    DOMAIN,
)

_LOGGER = logging.getLogger(__name__)

PLATFORMS: list[Platform] = [Platform.CONVERSATION]

CONFIG_SCHEMA = vol.Schema(
    {
        vol.Optional(DOMAIN): vol.Schema(
            {
                vol.Optional(CONF_NAME, default=DEFAULT_NAME): cv.string,
                vol.Optional(CONF_ENDPOINT): cv.url,
                vol.Optional(CONF_DEVICE_ID, default=DEFAULT_DEVICE_ID): cv.string,
                vol.Optional(CONF_ALIAS, default=DEFAULT_ALIAS): cv.string,
                vol.Optional(
                    CONF_STATIC_RESPONSE, default=DEFAULT_STATIC_RESPONSE
                ): cv.string,
            }
        ),
    },
    extra=vol.ALLOW_EXTRA,
)


async def async_setup(hass: HomeAssistant, config: ConfigType) -> bool:
    """Set up Hermes Voice from YAML by importing it into a config entry."""
    if DOMAIN not in config:
        return True

    hass.async_create_task(
        hass.config_entries.flow.async_init(
            DOMAIN,
            context={"source": SOURCE_IMPORT},
            data=dict(config[DOMAIN]),
        )
    )
    return True


async def async_setup_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Set up Hermes Voice from a config entry."""
    await hass.config_entries.async_forward_entry_setups(entry, PLATFORMS)
    return True


async def async_unload_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Unload a Hermes Voice config entry."""
    return await hass.config_entries.async_unload_platforms(entry, PLATFORMS)


async def async_migrate_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Migrate old Hermes Voice config entries."""
    if entry.version > 1:
        return False

    if entry.version < 1:
        data: dict[str, Any] = dict(entry.data)
        hass.config_entries.async_update_entry(entry, data=data, version=1)

    return True
