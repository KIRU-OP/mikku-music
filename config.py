import re
from os import getenv

from dotenv import load_dotenv
from pyrogram import filters

# ──────────────────────────────────────────────
# 1. .env file load karo (sabse pehle, taaki neeche ki saari values sahi padhein)
# ──────────────────────────────────────────────
load_dotenv()


def str_to_bool(value, default=False):
    """'true', '1', 'yes' ko True maanta hai, baaki sab False."""
    if value is None:
        return default
    return str(value).strip().lower() in ("true", "1", "yes", "on")


# ──────────────────────────────────────────────
# 2. Telegram credentials (https://my.telegram.org/apps se milte hain)
# ──────────────────────────────────────────────
API_ID = int(getenv("API_ID"))
API_HASH = getenv("API_HASH")
BOT_TOKEN = getenv("BOT_TOKEN")  # @BotFather se milta hai

# ──────────────────────────────────────────────
# 3. Database (user aur session data yahan save hota hai)
# ──────────────────────────────────────────────
MONGO_DB_URI = getenv("MONGO_DB_URI", None)

# ──────────────────────────────────────────────
# 4. Owner aur Logger
# ──────────────────────────────────────────────
OWNER_ID = int(getenv("OWNER_ID"))      # Aapki Telegram user ID
LOGGER_ID = int(getenv("LOGGER_ID"))    # Jis chat/group mein logs aayenge uski ID

# ──────────────────────────────────────────────
# 5. Support aur Community links
# ──────────────────────────────────────────────
SUPPORT_CHANNEL = getenv("SUPPORT_CHANNEL", "https://t.me/about_deadly_venom")
SUPPORT_CHAT = getenv("SUPPORT_CHAT", "https://t.me/group_voice_chat")
UPDATE_CHANNEL = getenv("UPDATE_CHANNEL", "https://t.me/NEXO_BOT_SUPPORT")   # "Channel Update" button
POLICY_LINK = getenv("POLICY_LINK", "https://telegra.ph/your-policy-page")  # "Policy" button

# ──────────────────────────────────────────────
# 6. YouTube / Streaming settings
# ──────────────────────────────────────────────
YTPROXY_URL = getenv("YTPROXY_URL", "http://80.211.135.205:1470/youtube/")
YT_API_KEY = getenv("YT_API_KEY", "")            # Key sirf .env mein rakho
YTDL_COOKIES = getenv("YTDL_COOKIES", "cookies.txt")
COOKIES_URL = getenv("COOKIES_URL", None)
DOWNLOADS_DIR = getenv("DOWNLOADS_DIR", "downloads")

ESSE_API_URL = getenv("ESSE_API_URL", "https://esse.skybotsdeveloper.workers.dev")
ESSE_API_KEY = getenv("ESSE_API_KEY", "")
ESSE_TIMEOUT = getenv("ESSE_TIMEOUT", "20")

# ──────────────────────────────────────────────
# 7. Spotify (https://developer.spotify.com/dashboard se milte hain)
# ──────────────────────────────────────────────
SPOTIFY_CLIENT_ID = getenv("SPOTIFY_CLIENT_ID", "")
SPOTIFY_CLIENT_SECRET = getenv("SPOTIFY_CLIENT_SECRET", "")
PLAYLIST_FETCH_LIMIT = int(getenv("PLAYLIST_FETCH_LIMIT", 25))  # Ek baar mein kitne tracks

# ──────────────────────────────────────────────
# 8. Limits aur Timing
# ──────────────────────────────────────────────
DURATION_LIMIT_MIN = int(getenv("DURATION_LIMIT", 300))             # Max song length (minutes)
ASSISTANT_LEAVE_TIME = int(getenv("ASSISTANT_LEAVE_TIME", 5400))    # Assistant kitne seconds baad leave kare
CACHE_DURATION = int(getenv("CACHE_DURATION", 86400))               # File cache kitni der rakhni hai (seconds)
CACHE_SLEEP = int(getenv("CACHE_SLEEP", 3600))                      # Cache safai ka gap (seconds)

TG_AUDIO_FILESIZE_LIMIT = int(getenv("TG_AUDIO_FILESIZE_LIMIT", 204857600))    # ~195 MB
TG_VIDEO_FILESIZE_LIMIT = int(getenv("TG_VIDEO_FILESIZE_LIMIT", 2073741824))   # ~1.93 GB

PRIVATE_BOT_MODE_MEM = int(getenv("PRIVATE_BOT_MODE_MEM", 1))

# ──────────────────────────────────────────────
# 9. Assistant ke auto settings (true / false likho)
# ──────────────────────────────────────────────
AUTO_LEAVING_ASSISTANT = str_to_bool(getenv("AUTO_LEAVING_ASSISTANT"), False)
AUTO_END_STREAM = str_to_bool(getenv("AUTO_END_STREAM"), False)

# ──────────────────────────────────────────────
# 10. Assistant account ke Pyrogram session strings
# ──────────────────────────────────────────────
STRING1 = getenv("STRING_SESSION")
STRING2 = getenv("STRING_SESSION2")
STRING3 = getenv("STRING_SESSION3")
STRING4 = getenv("STRING_SESSION4")
STRING5 = getenv("STRING_SESSION5")

# ──────────────────────────────────────────────
# 11. Heroku aur Git (deploy aur update ke liye, optional)
# ──────────────────────────────────────────────
HEROKU_APP_NAME = getenv("HEROKU_APP_NAME")
HEROKU_API_KEY = getenv("HEROKU_API_KEY")

UPSTREAM_REPO = getenv("UPSTREAM_REPO", "https://github.com/VNI0X/ANONMUSIC")
UPSTREAM_BRANCH = getenv("UPSTREAM_BRANCH", "Master")
GIT_TOKEN = getenv("GIT_TOKEN")  # Sirf private repo ke liye

# ──────────────────────────────────────────────
# 12. Bot ke andar use hone wala temporary data (memory mein)
# ──────────────────────────────────────────────
BANNED_USERS = filters.user()
adminlist = {}
lyrical = {}
votemode = {}
autoclean = []
confirmer = {}
file_cache: dict[str, float] = {}

# ──────────────────────────────────────────────
# 13. Images / Videos jo bot ke messages mein dikhte hain
# ──────────────────────────────────────────────
START_IMG_URL = getenv("START_IMG_URL", "https://litter.catbox.moe/ja82tm.mp4")
PING_IMG_URL = getenv("PING_IMG_URL", "https://files.catbox.moe/x3a3nc.jpg")
PLAYLIST_IMG_URL = getenv("PLAYLIST_IMG_URL", "https://files.catbox.moe/v7u8ji.jpg")
STATS_IMG_URL = getenv("STATS_IMG_URL", "https://files.catbox.moe/7g7j9f.jpg")
TELEGRAM_AUDIO_URL = getenv("TELEGRAM_AUDIO_URL", "https://files.catbox.moe/fahcob.jpg")
TELEGRAM_VIDEO_URL = getenv("TELEGRAM_VIDEO_URL", "https://files.catbox.moe/gy14qk.jpg")
STREAM_IMG_URL = getenv("STREAM_IMG_URL", "https://files.catbox.moe/h0m0wz.jpg")
SOUNCLOUD_IMG_URL = getenv("SOUNCLOUD_IMG_URL", "https://files.catbox.moe/ieduw9.jpg")
YOUTUBE_IMG_URL = getenv("YOUTUBE_IMG_URL", "https://files.catbox.moe/rnwmfw.jpg")
SPOTIFY_ARTIST_IMG_URL = getenv("SPOTIFY_ARTIST_IMG_URL", "https://files.catbox.moe/66ye0m.jpg")
SPOTIFY_ALBUM_IMG_URL = getenv("SPOTIFY_ALBUM_IMG_URL", "https://files.catbox.moe/66ye0m.jpg")
SPOTIFY_PLAYLIST_IMG_URL = getenv("SPOTIFY_PLAYLIST_IMG_URL", "https://files.catbox.moe/66ye0m.jpg")


# ──────────────────────────────────────────────
# 14. Helper: "hh:mm:ss" ko seconds mein badalta hai
# ──────────────────────────────────────────────
def time_to_seconds(time: str) -> int:
    return sum(int(x) * 60 ** i for i, x in enumerate(reversed(time.split(":"))))


DURATION_LIMIT = time_to_seconds(f"{DURATION_LIMIT_MIN}:00")  # Limit seconds mein


# ──────────────────────────────────────────────
# 15. Links check: sab http:// ya https:// se shuru hone chahiye
# ──────────────────────────────────────────────
def validate_url(name: str, value: str):
    if value and not re.match(r"^(?:http|https)://", value):
        raise SystemExit(f"[ERROR] - {name} URL must start with http/https.")


validate_url("SUPPORT_CHANNEL", SUPPORT_CHANNEL)
validate_url("SUPPORT_CHAT", SUPPORT_CHAT)
validate_url("UPDATE_CHANNEL", UPDATE_CHANNEL)
validate_url("POLICY_LINK", POLICY_LINK)
