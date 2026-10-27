package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/KIRU-OP/mikku-music/internal/music"
)

type Service struct { queues *music.Queues }

func New() *Service { return &Service{queues: music.NewQueues()} }

func (s *Service) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Chat == nil { return }
	msg := update.Message
	text := strings.TrimSpace(msg.Text)
	if text == "" { return }

	fields := strings.Fields(text)
	command := strings.ToLower(strings.SplitN(fields[0], "@", 2)[0])
	args := ""
	if len(fields) > 1 { args = strings.TrimSpace(strings.TrimPrefix(text, fields[0])) }
	var reply string

	switch command {
	case "/start":
		reply = "🎧 Welcome to Mikku Music (Go edition)!\n\nThis is the migration foundation. Telegram commands and queue state are implemented; voice-chat playback and provider adapters are still being ported.\n\nUse /help to see available commands."
	case "/help":
		reply = "🎵 Mikku Music — Go migration foundation\n/start — welcome\n/help — commands\n/ping — health check\n/play <song> — add a request to the local queue (does not stream yet)\n/queue — show queued requests\n/skip — remove the first queued request\n/stop — clear the queue\n\nYouTube/Spotify/Apple Music/SoundCloud integrations and voice-chat playback are not connected in this foundation build."
	case "/ping":
		reply = "🏓 Pong! Mikku Go bot is online."
	case "/play":
		if args == "" { reply = "Usage: /play <song name or link>"; break }
		requestedBy := ""
		if msg.From != nil { requestedBy = msg.From.Username }
		if err := s.queues.Enqueue(msg.Chat.ID, music.Item{Title: args, RequestedBy: requestedBy}); err != nil {
			reply = err.Error()
			break
		}
		reply = fmt.Sprintf("📥 Added to the local queue: %s\n⚠️ Playback is not connected in this Go foundation yet.", args)
	case "/queue":
		items := s.queues.List(msg.Chat.ID)
		if len(items) == 0 { reply = "The queue is empty."; break }
		var out strings.Builder
		out.WriteString("🎶 Queue (local requests):\n")
		for i, item := range items { fmt.Fprintf(&out, "%d. %s\n", i+1, item.Title) }
		reply = out.String()
	case "/skip":
		item, ok := s.queues.Skip(msg.Chat.ID)
		if !ok { reply = "Nothing is queued." } else { reply = "Removed from queue: " + item.Title + "\nPlayback is not connected yet." }
	case "/stop":
		s.queues.Clear(msg.Chat.ID)
		reply = "🛑 Queue cleared. (No active audio stream is connected in this foundation.)"
	default:
		return
	}

	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: reply}); err != nil {
		fmt.Printf("send message failed: %v\n", err)
	}
}
