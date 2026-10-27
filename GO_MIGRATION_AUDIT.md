# Mikku Music Go migration audit

## Verified repository-level scope

The original Python implementation remains the production implementation. The Go branch is additive and must not be treated as a drop-in replacement yet.

### Implemented in the current Go foundation
- Telegram bot long polling using `BOT_TOKEN`
- `/start`, `/help`, `/ping`, `/play`, `/queue`, `/skip`, `/stop`
- In-memory, per-chat queue with basic unit tests
- Provider-neutral track-provider interfaces
- Docker build recipe
- CI workflow for formatting, tests, race detection, and compilation

### Not implemented / not verified
- Real audio playback in Telegram voice chats
- YouTube, Spotify, Apple Music, SoundCloud, Resso, and Telegram media providers
- MongoDB persistence
- Pyrogram userbot/assistant sessions and multi-assistant lifecycle
- Admin/sudo commands, ban/block lists, maintenance mode, permissions
- Inline mode, language packs, settings, broadcast, logging, thumbnails, seek/loop/shuffle/speed/pause/resume
- Live Telegram integration tests or end-to-end audio tests

## Release gate

Do not replace the existing Python deployment until each required feature has a Go implementation, automated tests, and a documented result. Provider integrations must use supported APIs and respect source service terms; a search result or metadata response is not proof that audio playback works.

## Local verification

Install Go 1.22 or newer, then run:

```sh
go mod tidy
gofmt -w .
go test -race ./...
go build ./cmd/mikku
```

The GitHub Actions workflow runs formatting checks, tests, race detection, and build checks. A green workflow validates only the code present in this branch; it does not imply feature parity with the Python bot.
