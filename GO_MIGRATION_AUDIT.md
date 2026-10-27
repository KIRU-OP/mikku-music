# Go migration audit

## This branch
This is a Go-only migration branch. Python source files and Python deployment entry points are removed from this branch; the default branch remains unchanged as a rollback/reference.

Implemented:
- Telegram Bot API polling and basic commands.
- YouTube search/metadata using yt-dlp.
- Bounded per-chat in-memory request queue.
- Sequential audio/video download and delivery as Telegram messages; YouTube playlist queueing; repeat, shuffle and audio speed processing.
- Cancel current download, skip queued item, clear queue.
- Download cleanup, timeout, upload-size guard, unit tests and CI build/test checks.

## Remaining feature gaps
This is not yet feature parity with the original music bot. In particular, it does not provide Telegram group voice-chat playback, PyTgCalls replacement, userbot/assistant sessions, Spotify/Apple Music/SoundCloud/Resso/Telegram provider adapters, MongoDB persistence, full group-admin checks, inline callbacks/translations, channel play, autoplay/autoleave/watchers, or live-call controls such as pause/resume/seek. Persistent local settings, sudo/block commands, maintenance and broadcast have partial Go implementations. Do not deploy it as a drop-in replacement for the original production bot until these features are implemented and integration-tested.

## Verification
The GitHub Actions workflow checks gofmt, unit tests with the race detector, and compilation. These checks validate the code in this branch only. Real Telegram and YouTube end-to-end checks require deployment credentials and must be performed separately.
