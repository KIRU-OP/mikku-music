# Go rewrite feature-parity checklist

This file is the release checklist for replacing the Python implementation. A checkbox is only marked complete after a Go implementation and a relevant test exist.

## Telegram and bot lifecycle
- [ ] Bot startup, graceful shutdown, reconnect/retry policy, logging, startup/shutdown notices
- [x] Basic long polling and basic command dispatch
- [ ] HTML formatting, inline keyboards, callback query routing, deep links
- [ ] Group/member permission checks, chat migration handling, flood-wait/backoff
- [ ] Userbot sessions, assistant pool (up to five), assistant health and lifecycle
- [ ] Session persistence and secure credential handling

## Playback and media
- [ ] Telegram voice-chat join/leave and audio/video stream transport
- [ ] Play, pause, resume, stop, skip, seek, loop, shuffle, speed, volume
- [ ] Stream-ended handling, auto-end, active chat/video chat tracking
- [ ] YouTube feature parity with `mikku/Youtube.py` (partial port added at `internal/youtube/youtube.go`: search, metadata, URL/ID validation, playlist IDs, stream URL resolution, downloads, autoplay fallback and stream probe; original download-cache/worker-fallback/autoplay/source-status integrations and complete parity remain outstanding)
- [ ] Spotify track/album/playlist resolution (API credentials optional)
- [ ] Apple Music metadata resolution
- [ ] SoundCloud resolution
- [ ] Resso resolution where accessible and permitted
- [ ] Telegram audio/video file and message-link resolution
- [ ] Autoplay/recommendations and source fallback/status tracking
- [ ] Download/cache cleanup, disk-space limits, safe subprocess execution
- [ ] Provider rate limits, retries, cancellation, source terms and access restrictions

## Data, settings, and security
- [ ] MongoDB schema/repositories and migrations
- [ ] Persisted language, loop, auto-end, on/off toggles, active chats and settings
- [ ] Owner, sudo, admin permissions
- [ ] Global ban, block lists, maintenance mode
- [ ] Input validation, SSRF-safe URL handling, command injection protection
- [ ] Secret redaction, safe logging, audit trail
- [ ] Per-chat queue persistence or explicit documented in-memory-only behavior

## User-facing modules
- [ ] All original play/admin/bot/misc/other/tools plugins and their command aliases
- [ ] Inline search/results and callback handlers
- [ ] Language strings/translations
- [ ] Thumbnails, formatting helpers, duration/size/time helpers
- [ ] Broadcast and logging tools
- [ ] Carbon/code-image feature and speed-test/system tools
- [ ] Repository update/install utilities and health/status commands

## Delivery and verification
- [x] Separate Go branch and draft PR; original Python branch remains untouched
- [x] Go unit tests for local queue behavior
- [x] CI workflow for formatting, tests/race detector, and compilation
- [ ] Latest CI run green
- [ ] Integration tests with Telegram test bot and database
- [ ] End-to-end voice-chat playback tests
- [ ] Deployment tested with documented environment variables
- [ ] Cutover/rollback guide completed
- [ ] Only after all required boxes above are complete: remove Python runtime/dependencies

## Current status
This is a tracking document, not a claim that the unchecked features are implemented. The Go foundation currently does not stream audio. Do not deploy it as a replacement for the Python bot yet.
