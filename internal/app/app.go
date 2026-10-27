package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/KIRU-OP/mikku-music/internal/music"
	"github.com/KIRU-OP/mikku-music/internal/youtube"
)

const (
	maxQueueLines = 20
	maxTelegramAudioBytes = 49 * 1024 * 1024
)

type Service struct {
	queues *music.Queues
	youtube *youtube.API
	mu sync.Mutex
	active map[int64]bool
	cancel map[int64]context.CancelFunc
}

func New() *Service {
	return &Service{
		queues: music.NewQueues(),
		youtube: youtube.New(youtube.Config{YTDLPPath: envOr("YTDLP_PATH", "yt-dlp"), DownloadDir: envOr("DOWNLOAD_DIR", "downloads")}),
		active: make(map[int64]bool),
		cancel: make(map[int64]context.CancelFunc),
	}
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" { return value }
	return fallback
}

func (s *Service) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil { return }
	msg := update.Message
	text := strings.TrimSpace(msg.Text)
	if text == "" { return }
	fields := strings.Fields(text)
	command := strings.ToLower(strings.SplitN(fields[0], "@", 2)[0])
	args := strings.TrimSpace(strings.TrimPrefix(text, fields[0]))
	respond := func(value string) {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: msg.Chat.ID, Text: value}); err != nil {
			log.Printf("send Telegram reply: %v", err)
		}
	}

	switch command {
	case "/start":
		respond("🎧 Mikku Music Go is online.\n\nUse /play <song or YouTube URL> to search and send an audio file in this chat. Use /queue, /skip, and /stop to manage requests.\n\nNote: this Go edition sends audio files as Telegram messages; Telegram Bot API does not provide group voice-chat streaming controls.")
	case "/help":
		respond("🎵 Mikku Music — Go edition\n/start — welcome\n/help — commands\n/ping — health check\n/play <song or YouTube URL> — search, download and send audio\n/queue — show upcoming requests\n/skip — cancel current download or skip the next request\n/stop — cancel current download and clear the queue\n\nRequires yt-dlp and ffmpeg installed on the host. YouTube availability is controlled by YouTube and may vary.")
	case "/ping":
		respond("🏓 Pong! Mikku Go bot is online.")
	case "/play":
		if args == "" { respond("Usage: /play <song name or YouTube URL>"); return }
		searchCtx, cancel := context.WithTimeout(ctx, 50*time.Second)
		tracks, err := s.youtube.Search(searchCtx, args, 1)
		cancel()
		if err != nil { respond("❌ Could not find that track: " + short(err.Error(), 220)); return }
		track := tracks[0]
		requestedBy := ""
		if msg.From != nil {
			requestedBy = msg.From.Username
			if requestedBy == "" { requestedBy = msg.From.FirstName }
		}
		item := music.Item{Title: track.Title, RequestedBy: requestedBy, URL: track.URL, ID: track.ID, DurationSec: track.DurationSec, Thumbnail: track.Thumbnail}
		if err := s.queues.Enqueue(msg.Chat.ID, item); err != nil { respond("❌ " + err.Error()); return }
		respond(fmt.Sprintf("📥 Added to queue: %s (%s)", track.Title, track.DurationMin))
		s.startWorker(ctx, b, msg.Chat.ID)
	case "/queue":
		items := s.queues.List(msg.Chat.ID)
		if len(items) == 0 { respond("The upcoming queue is empty."); return }
		var out strings.Builder
		out.WriteString("🎶 Upcoming queue:\n")
		limit := len(items); if limit > maxQueueLines { limit = maxQueueLines }
		for i := 0; i < limit; i++ {
			fmt.Fprintf(&out, "%d. %s", i+1, items[i].Title)
			if items[i].RequestedBy != "" { fmt.Fprintf(&out, " — requested by %s", items[i].RequestedBy) }
			out.WriteByte('\n')
		}
		if len(items) > limit { fmt.Fprintf(&out, "…and %d more request(s).", len(items)-limit) }
		respond(strings.TrimSpace(out.String()))
	case "/skip":
		s.mu.Lock()
		cancelCurrent := s.cancel[msg.Chat.ID]
		if cancelCurrent != nil { cancelCurrent() }
		s.mu.Unlock()
		if cancelCurrent != nil { respond("⏭️ Skipping the current download/request."); return }
		if item, ok := s.queues.Skip(msg.Chat.ID); ok { respond("⏭️ Skipped: " + item.Title) } else { respond("Nothing is queued.") }
	case "/stop":
		s.mu.Lock()
		if cancelCurrent := s.cancel[msg.Chat.ID]; cancelCurrent != nil { cancelCurrent() }
		s.queues.Clear(msg.Chat.ID)
		s.mu.Unlock()
		respond("🛑 Current download cancelled and queue cleared.")
	default:
		return
	}
}

func (s *Service) startWorker(ctx context.Context, b *bot.Bot, chatID int64) {
	s.mu.Lock()
	if s.active[chatID] { s.mu.Unlock(); return }
	s.active[chatID] = true
	s.mu.Unlock()
	go s.processQueue(ctx, b, chatID)
}

func (s *Service) processQueue(parent context.Context, b *bot.Bot, chatID int64) {
	defer func() {
		s.mu.Lock()
		s.active[chatID] = false
		s.cancel[chatID] = nil
		hasMore := len(s.queues.List(chatID)) > 0
		s.mu.Unlock()
		if hasMore { s.startWorker(parent, b, chatID) }
	}()

	for {
		s.mu.Lock()
		items := s.queues.List(chatID)
		if len(items) == 0 { s.mu.Unlock(); return }
		item, ok := s.queues.Skip(chatID)
		if !ok { s.mu.Unlock(); return }
		playCtx, cancel := context.WithCancel(parent)
		s.cancel[chatID] = cancel
		s.mu.Unlock()

		downloadCtx, timeoutCancel := context.WithTimeout(playCtx, 8*time.Minute)
		path, err := s.youtube.Download(downloadCtx, item.URL, "audio")
		timeoutCancel()
		if err != nil {
			cancel()
			s.clearCurrentCancel(chatID)
			if playCtx.Err() == nil { s.sendText(parent, b, chatID, "❌ Download failed for "+item.Title+": "+short(err.Error(), 180)) }
			continue
		}
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			cancel(); _ = os.Remove(path); s.clearCurrentCancel(chatID)
			s.sendText(parent, b, chatID, "❌ Download output was empty or unavailable.")
			continue
		}
		if info.Size() > maxTelegramAudioBytes {
			cancel(); _ = os.Remove(path); s.clearCurrentCancel(chatID)
			s.sendText(parent, b, chatID, "❌ Audio is larger than Telegram's bot upload limit; try a shorter track.")
			continue
		}
		file, err := os.Open(path)
		if err == nil {
			_, err = b.SendAudio(playCtx, &bot.SendAudioParams{
				ChatID: chatID,
				Audio: &models.InputFileUpload{Filename: filepath.Base(path), Data: file},
				Title: item.Title, Duration: item.DurationSec, Caption: "🎧 " + item.Title,
			})
			_ = file.Close()
		}
		cancel()
		_ = os.Remove(path)
		s.clearCurrentCancel(chatID)
		if err != nil && playCtx.Err() == nil { s.sendText(parent, b, chatID, "❌ Could not send audio: "+short(err.Error(), 180)) }
	}
}

func (s *Service) clearCurrentCancel(chatID int64) {
	s.mu.Lock()
	s.cancel[chatID] = nil
	s.mu.Unlock()
}

func (s *Service) sendText(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text}); err != nil {
		log.Printf("send Telegram message: %v", err)
	}
}

func short(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= max { return value }
	return value[:max] + "…"
}
