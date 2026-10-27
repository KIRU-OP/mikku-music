# Go migration status

This checklist is intentionally honest: a feature is complete only after implementation and tests.

## Implemented in this branch
- [x] Go Telegram Bot API polling and command dispatch
- [x] /start, /help, /ping, /stats, /active and /settings
- [x] /play audio and /vplay video message delivery aliases
- [x] YouTube search/metadata via yt-dlp and best-effort catalogue-link metadata resolution
- [x] YouTube playlist queueing (up to 25 tracks)
- [x] Bounded per-chat queue, queue display, shuffle, skip, stop and repeat
- [x] Audio speed processing for upcoming downloads
- [x] Owner-ID-gated /auth, /unauth and /admins
- [x] Persistent /lang, /mode, /maintenance, sudo list management, bot-level block list and /broadcast
- [x] Local JSON persistence for auth/settings/known chats with restrictive permissions
- [x] Download timeout, temporary file cleanup, upload-size guard, unit tests and CI workflow

## Not yet equivalent to the original Python bot
- [ ] Telegram MTProto userbot/assistant session login and group voice/video-call playback (PyTgCalls equivalent)
- [ ] Full source-specific adapters and native stream resolution for Spotify/Apple Music/SoundCloud/Resso/Telegram media
- [ ] MongoDB-backed persistence and migration of existing database records
- [ ] Telegram group admin permission checks and full original global-ban semantics across every served chat
- [ ] Original inline UI, callback handlers, full language packs, channel-play routing, autoplay/autoleave/watchers
- [ ] Live-call pause/resume/seek controls
- [ ] Integration and end-to-end tests against a real Telegram chat and media source

The original default branch is preserved. This branch is not a drop-in production replacement until the remaining requirements are implemented and verified.