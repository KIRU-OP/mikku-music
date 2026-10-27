# Mikku Music — Go edition

This branch is the Go-only migration track for KIRU-OP/mikku-music.

## What works in this branch
- Telegram Bot API polling and command dispatch.
- /start, /help, /ping, /stats, /active, /settings, /auth, /unauth, /admins, /play, /queue, /skip, and /stop.\n- Owner-ID-gated temporary authorization management; authorization currently resets when the process restarts.
- YouTube search and metadata through yt-dlp.
- Per-chat bounded in-memory queues with sequential audio download and delivery as Telegram audio messages.
- Cancellation of an active download, cleanup of downloaded files, upload-size checks, and unit tests.
- Optional configured YouTube Worker fallback in the low-level resolver/download API.

## Important scope
This branch sends audio as Telegram chat messages. It does not implement group voice-chat streaming or the original PyTgCalls assistant/session system. It is not a feature-for-feature replacement for the original bot's Spotify/Apple Music/SoundCloud/Resso/Telegram providers, MongoDB settings, admin/sudo plugins, inline UI, or language packs. Those features must not be described as ported until they have real Go implementations and tests. The original default branch is not changed by this branch; this migration branch intentionally removes the Python source tree so the Go-only work can be reviewed separately.

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
