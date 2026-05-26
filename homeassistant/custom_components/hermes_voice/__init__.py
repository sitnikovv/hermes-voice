"""Minimal Hermes Voice conversation agent for Home Assistant Assist spikes."""

from __future__ import annotations

import logging
from typing import Any, Literal

import voluptuous as vol

from homeassistant.components.conversation.agent_manager import get_agent_manager
from homeassistant.components.conversation.models import (
    AbstractConversationAgent,
    ConversationInput,
    ConversationResult,
)
from homeassistant.const import CONF_NAME, MATCH_ALL
from homeassistant.core import HomeAssistant
from homeassistant.helpers import config_validation as cv, intent
from homeassistant.helpers.aiohttp_client import async_get_clientsession
from homeassistant.helpers.typing import ConfigType

_LOGGER = logging.getLogger(__name__)

DOMAIN = "hermes_voice"
CONF_ENDPOINT = "endpoint"
CONF_DEVICE_ID = "device_id"
CONF_ALIAS = "alias"
CONF_STATIC_RESPONSE = "static_response"

DEFAULT_NAME = "Hermes Voice"
DEFAULT_DEVICE_ID = "phone_ha"
DEFAULT_ALIAS = ""
DEFAULT_STATIC_RESPONSE = "Hermes Voice custom conversation agent is loaded."

CONFIG_SCHEMA = vol.Schema(
    {
        DOMAIN: vol.Schema(
            {
                vol.Optional(CONF_NAME, default=DEFAULT_NAME): cv.string,
                vol.Optional(CONF_ENDPOINT): cv.url,
                vol.Optional(CONF_DEVICE_ID, default=DEFAULT_DEVICE_ID): cv.string,
                vol.Optional(CONF_ALIAS, default=DEFAULT_ALIAS): cv.string,
                vol.Optional(CONF_STATIC_RESPONSE, default=DEFAULT_STATIC_RESPONSE): cv.string,
            }
        )
    },
    extra=vol.ALLOW_EXTRA,
)


async def async_setup(hass: HomeAssistant, config: ConfigType) -> bool:
    """Set up the YAML-loaded Hermes Voice conversation agent."""
    conf = config.get(DOMAIN)
    if conf is None:
        return True

    agent = HermesVoiceAgent(hass, conf)
    get_agent_manager(hass).async_set_agent(DOMAIN, agent)
    _LOGGER.info("Registered Hermes Voice conversation agent as %s", DOMAIN)
    return True


class HermesVoiceAgent(AbstractConversationAgent):
    """Tiny conversation agent that optionally forwards text to Hermes Voice dev HTTP."""

    def __init__(self, hass: HomeAssistant, config: dict[str, Any]) -> None:
        """Initialize the agent."""
        self.hass = hass
        self.name = config[CONF_NAME]
        self.endpoint: str | None = config.get(CONF_ENDPOINT)
        self.device_id: str = config[CONF_DEVICE_ID]
        self.alias: str = config[CONF_ALIAS]
        self.static_response: str = config[CONF_STATIC_RESPONSE]

    @property
    def supported_languages(self) -> list[str] | Literal["*"]:
        """Return supported languages."""
        return MATCH_ALL

    async def async_process(self, user_input: ConversationInput) -> ConversationResult:
        """Process a conversation turn."""
        speech = self.static_response
        if self.endpoint:
            speech = await self._forward_to_hermes(user_input)

        response = intent.IntentResponse(language=user_input.language)
        response.async_set_speech(speech)
        return ConversationResult(
            response=response,
            conversation_id=user_input.conversation_id,
            continue_conversation=False,
        )

    async def _forward_to_hermes(self, user_input: ConversationInput) -> str:
        """Forward text to the Hermes Voice dev HTTP endpoint and map the response to speech."""
        session = async_get_clientsession(self.hass)
        payload = {
            "request_id": user_input.conversation_id or "ha-assist",
            "device_id": self.device_id,
            "alias": self.alias,
            "input": user_input.text,
            "metadata": {
                "source": "home-assistant-assist",
                "language": user_input.language,
                "agent_id": user_input.agent_id,
            },
        }

        try:
            async with session.post(self.endpoint, json=payload, timeout=20) as resp:
                data = await resp.json(content_type=None)
        except Exception as err:  # noqa: BLE001 - Home Assistant should speak a safe fallback.
            _LOGGER.exception("Error calling Hermes Voice endpoint")
            return f"Hermes Voice недоступен: {err}"

        if "error" in data:
            message = data.get("error", {}).get("message") or "unknown error"
            return f"Hermes Voice вернул ошибку: {message}"

        status = data.get("status")
        output = data.get("output") or ""
        task_id = data.get("task_id") or ""

        if status == "completed":
            return output or "Hermes Voice вернул пустой ответ."
        if status == "accepted":
            if task_id:
                return f"Задача принята в работу. Идентификатор: {task_id}."
            return "Задача принята в работу."
        if status == "failed":
            return output or "Hermes Voice сообщил о неуспешном выполнении."

        return output or f"Hermes Voice вернул неизвестный статус: {status}."
