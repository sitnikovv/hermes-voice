"""Conversation platform for Hermes Voice."""

from __future__ import annotations

import logging
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Any, Literal
from urllib.parse import urljoin

from homeassistant.components import conversation
from homeassistant.config_entries import ConfigEntry
from homeassistant.const import CONF_NAME, MATCH_ALL
from homeassistant.core import HomeAssistant
from homeassistant.helpers import intent
from homeassistant.helpers.aiohttp_client import async_get_clientsession
from homeassistant.helpers.entity_platform import AddConfigEntryEntitiesCallback

from .const import (
    CONF_ALIAS,
    CONF_DEVICE_ID,
    CONF_ENDPOINT,
    CONF_STATIC_RESPONSE,
    DEFAULT_ALIAS,
    DEFAULT_DEVICE_ID,
    DEFAULT_NAME,
    DEFAULT_STATIC_RESPONSE,
)

_LOGGER = logging.getLogger(__name__)


_TASK_STATUS_PHRASES = (
    "проверь задачу",
    "что с задачей",
    "готова задача",
    "результат задачи",
)


@dataclass(slots=True)
class LastAcceptedTask:
    """In-memory state for the last accepted Hermes task."""

    task_id: str
    original_text: str
    created_at: datetime


async def async_setup_entry(
    hass: HomeAssistant,
    config_entry: ConfigEntry,
    async_add_entities: AddConfigEntryEntitiesCallback,
) -> None:
    """Set up Hermes Voice conversation entities."""
    async_add_entities([HermesVoiceConversationEntity(config_entry)])


class HermesVoiceConversationEntity(
    conversation.ConversationEntity,
    conversation.AbstractConversationAgent,
):
    """Hermes Voice conversation agent selectable by Assist pipelines."""

    _attr_should_poll = False
    _attr_supports_streaming = False

    def __init__(self, entry: ConfigEntry) -> None:
        """Initialize the conversation entity."""
        self.entry = entry
        self._attr_unique_id = entry.entry_id
        self._attr_name = entry.data.get(CONF_NAME, entry.title or DEFAULT_NAME)
        self._last_accepted_task: LastAcceptedTask | None = None

    @property
    def supported_languages(self) -> list[str] | Literal["*"]:
        """Return supported languages."""
        return MATCH_ALL

    async def async_added_to_hass(self) -> None:
        """Register this entity as a conversation agent."""
        await super().async_added_to_hass()
        conversation.async_set_agent(self.hass, self.entry, self)
        _LOGGER.info("Registered Hermes Voice conversation entity as %s", self.entity_id)

    async def async_will_remove_from_hass(self) -> None:
        """Unregister this entity as a conversation agent."""
        conversation.async_unset_agent(self.hass, self.entry)
        await super().async_will_remove_from_hass()

    async def _async_handle_message(
        self,
        user_input: conversation.ConversationInput,
        chat_log: conversation.ChatLog,
    ) -> conversation.ConversationResult:
        """Process a conversation turn."""
        speech = self.static_response
        if self.endpoint:
            if self._is_task_status_request(user_input.text):
                speech = await self._poll_last_task()
            else:
                speech = await self._forward_to_hermes(user_input)

        response = intent.IntentResponse(language=user_input.language)
        response.async_set_speech(speech)
        return conversation.ConversationResult(
            response=response,
            conversation_id=user_input.conversation_id,
            continue_conversation=False,
        )

    @property
    def endpoint(self) -> str | None:
        """Return the Hermes Voice endpoint."""
        return self.entry.data.get(CONF_ENDPOINT)

    @property
    def device_id(self) -> str:
        """Return the bridge device id."""
        return self.entry.data.get(CONF_DEVICE_ID, DEFAULT_DEVICE_ID)

    @property
    def alias(self) -> str:
        """Return the bridge alias."""
        return self.entry.data.get(CONF_ALIAS, DEFAULT_ALIAS)

    @property
    def static_response(self) -> str:
        """Return the static fallback response."""
        return self.entry.data.get(CONF_STATIC_RESPONSE, DEFAULT_STATIC_RESPONSE)

    async def _forward_to_hermes(
        self, user_input: conversation.ConversationInput
    ) -> str:
        """Forward text to the Hermes Voice dev HTTP endpoint and map response to speech."""
        session = async_get_clientsession(self.hass)
        payload: dict[str, Any] = {
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
        except Exception as err:  # noqa: BLE001 - HA should speak a safe fallback.
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
                self._last_accepted_task = LastAcceptedTask(
                    task_id=task_id,
                    original_text=user_input.text,
                    created_at=datetime.now(UTC),
                )
                return "Задача принята в работу. Позже скажите: проверь задачу."
            return "Задача принята в работу."
        if status == "failed":
            return output or "Hermes Voice сообщил о неуспешном выполнении."

        return output or f"Hermes Voice вернул неизвестный статус: {status}."

    def _is_task_status_request(self, text: str) -> bool:
        """Return true when the user asks for the last accepted task result."""
        normalized = " ".join(text.casefold().split())
        if any(phrase in normalized for phrase in _TASK_STATUS_PHRASES):
            return True
        if self._last_accepted_task is None:
            return False
        return "задач" in normalized and any(
            marker in normalized
            for marker in ("провер", "готов", "результат", "статус", "что")
        )

    def _task_url(self, task_id: str) -> str:
        """Build task status URL from the configured text endpoint."""
        endpoint = self.endpoint or ""
        base = endpoint.rsplit("/v1/dev/text", 1)[0]
        if base == endpoint:
            base = endpoint.rstrip("/")
        return urljoin(f"{base}/", f"v1/dev/tasks/{task_id}")

    async def _poll_last_task(self) -> str:
        """Poll the last accepted task and map its state to speech."""
        task = self._last_accepted_task
        if task is None:
            return "У меня нет ожидающей задачи."

        session = async_get_clientsession(self.hass)
        try:
            async with session.get(self._task_url(task.task_id), timeout=20) as resp:
                data = await resp.json(content_type=None)
        except Exception as err:  # noqa: BLE001 - HA should speak a safe fallback.
            _LOGGER.exception("Error polling Hermes Voice task %s", task.task_id)
            return f"Не удалось проверить задачу: {err}"

        err_data = data.get("error")
        if isinstance(err_data, dict):
            message = err_data.get("message") or "unknown error"
            self._last_accepted_task = None
            return f"Задача недоступна: {message}"

        status = data.get("status")
        if status == "accepted":
            return "Задача ещё выполняется."

        self._last_accepted_task = None
        if status == "completed":
            response = data.get("response") or {}
            output = response.get("output") or ""
            return output or "Задача завершилась, но результат пустой."
        if status == "failed":
            err = data.get("error") or {}
            message = err.get("message") or "Hermes Voice сообщил о неуспешном выполнении."
            return f"Задача завершилась с ошибкой: {message}"

        return f"Hermes Voice вернул неизвестный статус задачи: {status}."
