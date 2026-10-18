import sys
import asyncio

from pyrogram import Client
from pyrogram.errors import (
    FloodWait,
    UserAlreadyParticipant,
    InviteHashExpired,
    InviteHashInvalid,
    UsernameNotOccupied,
    PeerIdInvalid,
    ChatIdInvalid,
)

import config

from ..logging import LOGGER

assistants = []
assistantids = []

ASSISTANT_START_TIMEOUT = 60
ASSISTANT_STEP_TIMEOUT = 15
JOIN_DELAY = 2  # har join ke beech gap (seconds)

# Username (bina @), public link ya private invite link, sab chalega
GROUPS_TO_JOIN = [
    "about_me_venom",
    "group_voice_chat",
    "about_deadly_venom",
    "NEXO_BOT_SUPPORT",
]


async def _run_with_timeout(awaitable, timeout: int, label: str):
    try:
        return await asyncio.wait_for(awaitable, timeout=timeout)
    except asyncio.TimeoutError:
        raise TimeoutError(f"{label} timed out after {timeout}s") from None


def _has_session(value) -> bool:
    session = str(value or "").strip()
    return bool(session and session.lower() not in {"none", "null"})


def _clean_target(group: str) -> str:
    target = str(group).strip()
    if "t.me/+" in target or "joinchat" in target:
        return target  # private invite link as it is
    return target.rstrip("/").split("/")[-1].lstrip("@")


async def _join_group(client: Client, index: int, group: str):
    target = _clean_target(group)
    for _ in range(2):
        try:
            await _run_with_timeout(
                client.join_chat(target),
                ASSISTANT_STEP_TIMEOUT,
                f"Assistant {index} join_chat({target})",
            )
            LOGGER(__name__).info(f"Assistant {index} joined {target}")
            return
        except UserAlreadyParticipant:
            LOGGER(__name__).info(f"Assistant {index} already in {target}")
            return
        except FloodWait as e:
            wait = getattr(e, "value", None) or getattr(e, "x", 5)
            LOGGER(__name__).warning(
                f"Assistant {index} FloodWait {wait}s while joining {target}"
            )
            if wait > 60:
                return
            await asyncio.sleep(wait)
        except (InviteHashExpired, InviteHashInvalid):
            LOGGER(__name__).warning(
                f"Assistant {index}: invite link invalid/expired for {target}"
            )
            return
        except (UsernameNotOccupied, PeerIdInvalid, ChatIdInvalid):
            LOGGER(__name__).warning(f"Assistant {index}: {target} not found")
            return
        except Exception as e:
            LOGGER(__name__).warning(
                f"Assistant {index} could not join {target}: {e}"
            )
            return


class Userbot:
    def __init__(self):
        self.one = self._create_client(config.STRING1, "AnonXAss1")
        self.two = self._create_client(config.STRING2, "AnonXAss2")
        self.three = self._create_client(config.STRING3, "AnonXAss3")
        self.four = self._create_client(config.STRING4, "AnonXAss4")
        self.five = self._create_client(config.STRING5, "AnonXAss5")

    def _create_client(self, string, name):
        if not _has_session(string):
            return None
        return Client(
            name=name,
            api_id=config.API_ID,
            api_hash=config.API_HASH,
            session_string=str(string),
            no_updates=True,
        )

    async def start_assistant(self, client: Client, index: int):
        if not client:
            return False

        try:
            LOGGER(__name__).info(f"Starting Assistant {index}...")
            await _run_with_timeout(
                client.start(),
                ASSISTANT_START_TIMEOUT,
                f"Assistant {index} client.start()",
            )

            me = await _run_with_timeout(
                client.get_me(),
                ASSISTANT_STEP_TIMEOUT,
                f"Assistant {index} get_me()",
            )
            client.id, client.name, client.username = me.id, me.mention, me.username

            if not client.username:
                LOGGER(__name__).warning(
                    f"Assistant {index} has no username set, please set one."
                )

            if index not in assistants:
                assistants.append(index)
            if me.id not in assistantids:
                assistantids.append(me.id)

            LOGGER(__name__).info(f"Assistant {index} authenticated as {client.name}")

            for group in GROUPS_TO_JOIN:
                await _join_group(client, index, group)
                await asyncio.sleep(JOIN_DELAY)

            try:
                await _run_with_timeout(
                    client.send_message(
                        config.LOGGER_ID, f"Assistant {index} Started"
                    ),
                    ASSISTANT_STEP_TIMEOUT,
                    f"Assistant {index} log message",
                )
            except Exception as e:
                LOGGER(__name__).warning(
                    f"Assistant {index} can't send log group startup message: {e}"
                )

            LOGGER(__name__).info(f"Assistant {index} Started as {client.name}")
            return True

        except Exception as e:
            LOGGER(__name__).error(f"Failed to start Assistant {index}: {e}")
            try:
                await _run_with_timeout(
                    client.stop(),
                    ASSISTANT_STEP_TIMEOUT,
                    f"Assistant {index} stop after failed start",
                )
            except Exception:
                pass
            return False

    async def start(self):
        LOGGER(__name__).info("Starting Assistants...")
        await self.start_assistant(self.one, 1)
        await self.start_assistant(self.two, 2)
        await self.start_assistant(self.three, 3)
        await self.start_assistant(self.four, 4)
        await self.start_assistant(self.five, 5)

        if not assistants:
            LOGGER(__name__).error(
                "No assistants started. Check STRING_SESSION values and assistant logs."
            )
            sys.exit(1)
        LOGGER(__name__).info(f"Assistants ready: {', '.join(map(str, assistants))}")

    async def stop(self):
        LOGGER(__name__).info("Stopping Assistants...")
        clients = [self.one, self.two, self.three, self.four, self.five]
        for i, c in enumerate(clients, start=1):
            if c and i in assistants:
                try:
                    await c.stop()
                except Exception as e:
                    LOGGER(__name__).error(f"Error while stopping Assistant {i}: {e}")
        LOGGER(__name__).info("Assistants stopped.")
