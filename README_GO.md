# Mikku Music — Go edition

This branch is the Go-only migration track for KIRU-OP/mikku-music.

## What works in this branch
- Telegram Bot API polling and command dispatch.
- /start, /help, /ping, /stats, /active, /settings, /play, /vplay, /playlist, /queue, /skip, and /stop.
- Persistent /lang, /mode, /loop, /shuffle, /speed, /maintenance, /auth, /unauth, /admins, /addsudo, /delsudo, /block, /unblock, and /broadcast.
- YouTube search and metadata through yt-dlp.
- Per-chat bounded queues with sequential audio/video download and delivery as Telegram media messages.
- Cancellation of an active download, cleanup of downloaded files, upload-size checks, and unit tests.
- Optional configured YouTube Worker fallback in the low-level resolver/download API.

## Important scope
This branch sends audio as Telegram chat messages. It does not implement group voice-chat streaming or the original PyTgCalls assistant/session system. Music catalogue links are resolved by metadata and a YouTube search match, not native provider streams. MongoDB migration, real Telegram group-admin checks, full inline UI/callbacks, channel play and full language translations remain incomplete. The original default branch is not changed by this branch; this migration branch intentionally removes the Python source tree so the Go-only work can be reviewed separately.

YouTube access is controlled by YouTube and may fail for restricted videos, bot checks, or unavailable content. Signed media URLs are temporary; this bot downloads the audio immediately instead of exposing a stale URL.

## Requirements
- Go 1.22+
- yt-dlp
- ffmpeg (required by yt-dlp for MP3 conversion)
- Telegram bot token from @BotFather

## Run
Set BOT_TOKEN, then run go mod tidy and go test -race ./.... To compile without conflicting with the legacy assets directory named mikku, use go build -o /tmp/mikku ./cmd/mikku. Set YTDLP_PATH to the yt-dlp executable if it is not on PATH. Set DOWNLOAD_DIR to change the temporary download directory. The app reads environment variables directly and does not auto-load a .env file.

Optional resolver fallback environment variables: WORKER_FALLBACK_API_URL, WORKER_FALLBACK_API_KEY, and YTDLP_COOKIES_FILE. Use only cookies and credentials you own or are authorized to use.

## Docker
Use a runtime image that contains CA certificates, yt-dlp, and ffmpeg. The Go source itself does not require Python; some yt-dlp installation methods use Python as a runtime dependency. For a strictly Python-free runtime, install yt-dlp as a standalone binary or replace it with a supported Go-native resolver.

## Tests
Run gofmt -w ., go test -race ./..., and go build -o /tmp/mikku ./cmd/mikku.


## Additional Go command ports

The Go branch now includes persistent chat language/play-mode settings, queue repeat and shuffle, audio speed processing for future downloads, maintenance mode, sudo user management, and persistent bot-level user blocks and owner/sudo broadcast to known chats. These settings are stored in `FEATURE_STORE_PATH` (default `data/features.json`) with restrictive file permissions.

Compatibility boundary: commands that require Pyrogram MTProto sessions, PyTgCalls group voice/video chat, inline callback query handlers, or MongoDB-backed group state are not equivalent yet. In particular, `/pause`, `/resume`, and `/seek` explain that the Bot API edition sends complete audio messages and cannot control an active group call. Global block commands prevent use of this bot; they do not ban users across every Telegram group.

Music-source compatibility: `/play` accepts YouTube searches/URLs and attempts metadata resolution for supported Spotify, Apple Music, SoundCloud, Bandcamp, Deezer, Tidal, Amazon Music, and Resso links through the installed `yt-dlp`. For those catalogue links it searches YouTube for a playable match; it does not extract DRM-protected source audio. `/playlist` queues up to 25 tracks from a YouTube playlist.
