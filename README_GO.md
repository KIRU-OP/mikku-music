# Mikku Music — Go migration (foundation)

This is a separate, additive Go migration branch. The original Python bot remains untouched.

## Status: foundation, not a complete feature-parity release

Included now:
- Go module and Telegram long-polling entry point
- `BOT_TOKEN` environment configuration
- `/start`, `/help`, `/ping`, `/play`, `/queue`, `/skip`, and `/stop`
- Thread-safe, per-chat in-memory request queue
- Provider-neutral track and source-adapter interfaces

Not ported/connected yet:
- Telegram voice-chat streaming / PyTgCalls replacement
- YouTube, Spotify, Apple Music, SoundCloud, Resso, and Telegram media adapters
- MongoDB persistence, multi-assistant/userbot sessions, inline mode
- Admin/sudo/security, language packs, settings, broadcasts, logs, thumbnails, and remaining commands
- Loop, shuffle, seek, speed, pause/resume against an active audio stream

The commands deliberately state that playback is not connected. This avoids pretending the migration has feature parity before the streaming layer is implemented.

## Run the foundation

Install Go 1.22+ and set a Telegram bot token from BotFather:

```sh
export BOT_TOKEN="your-token"
go mod tidy
go run ./cmd/mikku
```

## Migration plan

1. Port configuration, Telegram handlers, permission checks, and database interfaces.
2. Implement provider adapters one at a time with tests and documented API limitations.
3. Port the voice-chat streaming layer and assistant lifecycle.
4. Port admin, settings, inline, language, and maintenance features.
5. Add integration tests, deployment files, and a feature-by-feature parity checklist.

Do not remove the Python entry point or deploy this foundation as a replacement for the existing bot until the required features are ported and tested.
