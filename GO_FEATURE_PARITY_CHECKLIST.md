# Go migration status

This checklist is intentionally honest: a feature is complete only after implementation and tests.

## Implemented in this branch
- [x] Go Telegram Bot API polling
- [x] Basic command dispatch and queue commands\n- [x] Runtime stats, active-worker view, and settings summary\n- [x] Owner-ID-gated temporary auth/unauth/admin-list commands
- [x] YouTube search/metadata via yt-dlp
- [x] Per-chat bounded queue
- [x] Sequential download and delivery as Telegram audio messages
- [x] Cancellation of current download and queue clear/skip
- [x] Download cleanup and upload-size guard
- [x] Unit tests and CI workflow

## Not yet equivalent to the original Python bot
- [ ] Telegram group voice-chat playback and assistant sessions
- [ ] Spotify/Apple Music/SoundCloud/Resso/Telegram media source adapters
- [ ] MongoDB persistence and settings migrations
- [ ] Original admin/sudo permissions, bans, maintenance and all plugin commands
- [ ] Inline UI, callback handlers, language packs and broadcasts
- [ ] Pause/resume/seek/loop/shuffle/speed for an active voice-chat stream
- [ ] Integration and end-to-end tests against a real Telegram chat and media source

The original default branch is preserved. Do not merge this migration branch as a production replacement until the remaining requirements are implemented and verified.
