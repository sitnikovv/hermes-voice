"""Config flow for Hermes Voice."""

from __future__ import annotations

from typing import Any

import voluptuous as vol

from homeassistant import config_entries
from homeassistant.const import CONF_NAME
from homeassistant.helpers import config_validation as cv

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


def _schema(user_input: dict[str, Any] | None = None) -> vol.Schema:
    """Return the config-flow form schema."""
    user_input = user_input or {}
    return vol.Schema(
        {
            vol.Optional(
                CONF_NAME, default=user_input.get(CONF_NAME, DEFAULT_NAME)
            ): cv.string,
            vol.Optional(CONF_ENDPOINT, default=user_input.get(CONF_ENDPOINT, "")): str,
            vol.Optional(
                CONF_DEVICE_ID,
                default=user_input.get(CONF_DEVICE_ID, DEFAULT_DEVICE_ID),
            ): cv.string,
            vol.Optional(
                CONF_ALIAS, default=user_input.get(CONF_ALIAS, DEFAULT_ALIAS)
            ): cv.string,
            vol.Optional(
                CONF_STATIC_RESPONSE,
                default=user_input.get(CONF_STATIC_RESPONSE, DEFAULT_STATIC_RESPONSE),
            ): cv.string,
        }
    )


def _normalize(data: dict[str, Any]) -> dict[str, Any]:
    """Normalize config-flow/YAML data."""
    normalized = dict(data)
    endpoint = normalized.get(CONF_ENDPOINT)
    if endpoint == "":
        normalized.pop(CONF_ENDPOINT, None)
    return normalized


class HermesVoiceConfigFlow(config_entries.ConfigFlow, domain=DOMAIN):
    """Handle a config flow for Hermes Voice."""

    VERSION = 1

    async def async_step_user(
        self, user_input: dict[str, Any] | None = None
    ) -> config_entries.ConfigFlowResult:
        """Handle manual setup from UI."""
        errors: dict[str, str] = {}

        if user_input is not None:
            data = _normalize(user_input)
            if endpoint := data.get(CONF_ENDPOINT):
                try:
                    vol.Schema(cv.url)(endpoint)
                except vol.Invalid:
                    errors[CONF_ENDPOINT] = "url"

            if not errors:
                await self.async_set_unique_id(DOMAIN)
                self._abort_if_unique_id_configured(updates=data)
                return self.async_create_entry(
                    title=data.get(CONF_NAME, DEFAULT_NAME),
                    data=data,
                )

        return self.async_show_form(
            step_id="user",
            data_schema=_schema(user_input),
            errors=errors,
        )

    async def async_step_import(
        self, import_config: dict[str, Any]
    ) -> config_entries.ConfigFlowResult:
        """Import YAML config as a config entry."""
        data = _normalize(import_config)
        await self.async_set_unique_id(DOMAIN)
        self._abort_if_unique_id_configured(updates=data)
        return self.async_create_entry(
            title=data.get(CONF_NAME, DEFAULT_NAME),
            data=data,
        )
