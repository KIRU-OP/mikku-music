import time
import re
import asyncio

from pyrogram import filters
from pyrogram.enums import ChatType
from pyrogram.types import InlineKeyboardButton, InlineKeyboardMarkup, Message
from pyrogram.errors import (
    ChannelPrivate,
    SlowmodeWait,
    PeerIdInvalid,
    UserNotParticipant,
    ChatAdminRequired,
    FloodWait,
)
from youtubesearchpython.__future__ import VideosSearch

import config
from mikku import app
from mikku.misc import _boot_
from mikku.plugins.sudo.sudoers import sudoers_list
from mikku.utils.database import (
    add_served_chat,
    add_served_user,
    blacklisted_chats,
    get_lang,
    is_banned_user,
    is_on_off,
    blacklist_chat,
)
from mikku.utils.decorators.language import LanguageStart
from mikku.utils.formatters import get_readable_time
from mikku.utils.inline import help_pannel, private_panel, start_panel
from config import BANNED_USERS, LOGGER_ID
from strings import get_string


# ──────────────────────────────────────────────
# Media helper: link dekhkar photo / video / gif khud choose karta hai
# ──────────────────────────────────────────────
VIDEO_EXT = (".mp4", ".mov", ".mkv", ".webm")
GIF_EXT = (".gif",)


async def send_start_media(message: Message, media: str, caption: str, reply_markup=None):
    """URL ke hisaab se photo, video ya gif bhejta hai.
    Agar media fail ho jaye to sirf text bhej deta hai."""
    path = media.lower().split("?")[0]
    try:
        if path.endswith(VIDEO_EXT):
            return await message.reply_video(
                video=media,
                caption=caption,
                has_spoiler=True,
                supports_streaming=True,
                reply_markup=reply_markup,
            )
        if path.endswith(GIF_EXT):
            return await message.reply_animation(
                animation=media,
                caption=caption,
                has_spoiler=True,
                reply_markup=reply_markup,
            )
        return await message.reply_photo(
            photo=media,
            caption=caption,
            has_spoiler=True,
            reply_markup=reply_markup,
        )
    except (ChannelPrivate, SlowmodeWait, ChatAdminRequired, UserNotParticipant, FloodWait):
        raise  # inko niche ke handlers sambhalte hain
    except Exception as e:
        print(f"Media send failed ({media}): {e}")
        return await message.reply_text(caption, reply_markup=reply_markup)


# ──────────────────────────────────────────────
# /start in private chat
# ──────────────────────────────────────────────
@app.on_message(filters.command(["start"]) & filters.private & ~BANNED_USERS)
@LanguageStart
async def start_pm(client, message: Message, _):
    await add_served_user(message.from_user.id)

    if len(message.text.split()) > 1:
        name = message.text.split(None, 1)[1]

        # /start help
        if name[0:4] == "help":
            keyboard = help_pannel(_)
            return await send_start_media(
                message,
                config.START_IMG_URL,
                caption=_["help_1"].format(config.SUPPORT_CHAT),
                reply_markup=keyboard,
            )

        # /start sudolist
        if name[0:3] == "sud":
            await sudoers_list(client=client, message=message, _=_)
            if await is_on_off(2):
                try:
                    await app.send_message(
                        chat_id=config.LOGGER_ID,
                        text=(
                            f"{message.from_user.mention} ᴊᴜsᴛ sᴛᴀʀᴛᴇᴅ ᴛʜᴇ ʙᴏᴛ ᴛᴏ ᴄʜᴇᴄᴋ <b>sᴜᴅᴏʟɪsᴛ</b>.\n\n"
                            f"<b>ᴜsᴇʀ ɪᴅ :</b> <code>{message.from_user.id}</code>\n"
                            f"<b>ᴜsᴇʀɴᴀᴍᴇ :</b> @{message.from_user.username}"
                        ),
                    )
                except Exception as e:
                    print(f"Error sending log to LOGGER_ID: {e}")
            return

        # /start info_<video_id>
        if name[0:3] == "inf":
            m = await message.reply_text("🔎")
            query = (str(name)).replace("info_", "", 1)
            query = f"https://www.youtube.com/watch?v={query}"

            try:
                results = VideosSearch(query, limit=1)
                search_result = (await results.next())["result"]
                if not search_result:
                    await m.edit_text(_.get("start_7", "No results found."))
                    return

                result = search_result[0]
                title = result.get("title", "N/A")
                duration = result.get("duration", "N/A")
                views = result.get("viewCount", {}).get("short", "N/A")
                thumbnail = result.get("thumbnails", [{}])[0].get("url", "").split("?")[0]
                channellink = result.get("channel", {}).get("link", "N/A")
                channel = result.get("channel", {}).get("name", "N/A")
                link = result.get("link", "N/A")
                published = result.get("publishedTime", "N/A")

                searched_text = _["start_6"].format(
                    title, duration, views, published, channellink, channel, app.mention
                )
                key = InlineKeyboardMarkup(
                    [
                        [
                            InlineKeyboardButton(text=_["S_B_8"], url=link),
                            InlineKeyboardButton(text=_["S_B_9"], url=config.SUPPORT_CHAT),
                        ],
                    ]
                )
                await m.delete()
                # Ye YouTube thumbnail hai (hamesha photo), isliye send_photo hi rakha hai
                await app.send_photo(
                    chat_id=message.chat.id,
                    photo=thumbnail,
                    has_spoiler=True,
                    caption=searched_text,
                    reply_markup=key,
                )
                if await is_on_off(2):
                    try:
                        await app.send_message(
                            chat_id=config.LOGGER_ID,
                            text=(
                                f"{message.from_user.mention} ᴊᴜsᴛ sᴛᴀʀᴛᴇᴅ ᴛʜᴇ ʙᴏᴛ ᴛᴏ ᴄʜᴇᴄᴋ <b>ᴛʀᴀᴄᴋ ɪɴғᴏʀᴍᴀᴛɪᴏɴ</b>.\n\n"
                                f"<b>ᴜsᴇʀ ɪᴅ :</b> <code>{message.from_user.id}</code>\n"
                                f"<b>ᴜsᴇʀɴᴀᴍᴇ :</b> @{message.from_user.username}"
                            ),
                        )
                    except Exception as e:
                        print(f"Error sending log to LOGGER_ID: {e}")
            except Exception as e:
                await m.edit_text(_.get("error_message", "Error: {0}").format(e))
                print(f"Error in info section: {e}")
    else:
        # Normal /start
        out = private_panel(_)
        await send_start_media(
            message,
            config.START_IMG_URL,
            caption=_["start_2"].format(message.from_user.mention, app.mention),
            reply_markup=InlineKeyboardMarkup(out),
        )
        if await is_on_off(2):
            try:
                await app.send_message(
                    chat_id=config.LOGGER_ID,
                    text=(
                        f"{message.from_user.mention} ᴊᴜsᴛ sᴛᴀʀᴛᴇᴅ ᴛʜᴇ ʙᴏᴛ.\n\n"
                        f"<b>ᴜsᴇʀ ɪᴅ :</b> <code>{message.from_user.id}</code>\n"
                        f"<b>ᴜsᴇʀɴᴀᴍᴇ :</b> @{message.from_user.username}"
                    ),
                )
            except Exception as e:
                print(f"Error sending log to LOGGER_ID: {e}")


# ──────────────────────────────────────────────
# /start in group
# ──────────────────────────────────────────────
@app.on_message(filters.command(["start"]) & filters.group & ~BANNED_USERS)
@LanguageStart
async def start_gp(client, message: Message, _):
    out = start_panel(_)
    uptime = int(time.time() - _boot_)
    caption = _["start_1"].format(app.mention, get_readable_time(uptime))

    try:
        await send_start_media(
            message,
            config.START_IMG_URL,
            caption=caption,
            reply_markup=InlineKeyboardMarkup(out),
        )
        await add_served_chat(message.chat.id)
    except ChannelPrivate:
        print(f"Cannot send message in private channel: {message.chat.id}")
        return
    except SlowmodeWait as e:
        print(f"Slowmode active in chat {message.chat.id}. Waiting for {e.value} seconds.")
        await asyncio.sleep(e.value)
        try:
            await send_start_media(
                message,
                config.START_IMG_URL,
                caption=caption,
                reply_markup=InlineKeyboardMarkup(out),
            )
            await add_served_chat(message.chat.id)
        except Exception as retry_e:
            print(f"Error after slowmode wait in chat {message.chat.id}: {retry_e}")
    except (ChatAdminRequired, UserNotParticipant) as e:
        print(f"Bot lacks permissions or is not a participant in chat {message.chat.id}: {e}")
    except Exception as e:
        print(f"An unexpected error occurred in start_gp for chat {message.chat.id}: {e}")


# ──────────────────────────────────────────────
# Jab bot ya koi naya member group mein aaye
# ──────────────────────────────────────────────
@app.on_message(filters.new_chat_members, group=-1)
async def welcome(client, message: Message):
    for member in message.new_chat_members:
        try:
            language = await get_lang(message.chat.id)
            _ = get_string(language)

            # Globally banned user ko turant ban karo
            if await is_banned_user(member.id):
                try:
                    await message.chat.ban_member(member.id)
                    print(f"Banned user {member.id} from chat {message.chat.id} as they are globally banned.")
                except ChatAdminRequired:
                    print(f"Bot is not admin to ban user {member.id} in chat {message.chat.id}.")
                except Exception as e:
                    print(f"Error banning user {member.id} in chat {message.chat.id}: {e}")

            # Bot ko add kiya gaya
            if member.id == app.id:
                # Sirf supergroup allowed hai
                if message.chat.type != ChatType.SUPERGROUP:
                    await message.reply_text(_["start_4"])
                    print(f"Leaving chat {message.chat.id} because it's not a supergroup.")
                    return await app.leave_chat(message.chat.id)

                # Blacklisted chat
                if message.chat.id in await blacklisted_chats():
                    await message.reply_text(
                        _["start_5"].format(
                            app.mention,
                            f"https://t.me/{app.username}?start=sudolist",
                            config.SUPPORT_CHAT,
                        ),
                        disable_web_page_preview=True,
                    )
                    print(f"Leaving chat {message.chat.id} as it is blacklisted.")
                    return await app.leave_chat(message.chat.id)

                # Myanmar characters wale group ko auto blacklist karo
                try:
                    ch = await app.get_chat(message.chat.id)
                    if (ch.title and re.search(r"[\u1000-\u109F]", ch.title)) or (
                        ch.description and re.search(r"[\u1000-\u109F]", ch.description)
                    ):
                        await blacklist_chat(message.chat.id)
                        await message.reply_text(
                            "This group is not allowed to play songs due to detected unsupported characters."
                        )
                        await app.send_message(
                            LOGGER_ID,
                            f"This group has been blacklisted automatically due to Myanmar characters "
                            f"in the chat title or description.\nTitle: {ch.title}\nID: {message.chat.id}",
                        )
                        print(f"Blacklisted and leaving chat {message.chat.id} due to Myanmar characters.")
                        return await app.leave_chat(message.chat.id)

                except PeerIdInvalid:
                    print(f"Could not get chat info for {message.chat.id}: PeerIdInvalid.")
                except Exception as e:
                    print(f"Error checking chat title/description in chat {message.chat.id}: {e}")

                # Welcome message (photo / video / gif)
                out = start_panel(_)
                await send_start_media(
                    message,
                    config.START_IMG_URL,
                    caption=_["start_3"].format(
                        message.from_user.first_name,
                        app.mention,
                        message.chat.title,
                        app.mention,
                    ),
                    reply_markup=InlineKeyboardMarkup(out),
                )
                await add_served_chat(message.chat.id)
                await message.stop_propagation()

        except FloodWait as e:
            print(f"FloodWait in welcome for chat {message.chat.id}. Waiting for {e.value} seconds.")
            await asyncio.sleep(e.value)
        except Exception as ex:
            print(f"Unexpected error in welcome for chat {message.chat.id} and user {member.id}: {ex}")
            try:
                await app.send_message(LOGGER_ID, f"Error in welcome handler for chat {message.chat.id}: {ex}")
            except Exception as log_e:
                print(f"Could not send error log to LOGGER_ID: {log_e}")
