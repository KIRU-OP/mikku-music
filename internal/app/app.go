package app

import (
	"context"
	"encoding/json"
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
	started time.Time
	requests uint64
	admins map[int64]map[int64]bool
}

func New() *Service {
	s := &Service{
		queues: music.NewQueues(),
		youtube: youtube.New(youtube.Config{YTDLPPath: envOr("YTDLP_PATH", "yt-dlp"), DownloadDir: envOr("DOWNLOAD_DIR", "downloads")}),
		active: make(map[int64]bool),
		cancel: make(map[int64]context.CancelFunc),
		started: time.Now(),
		admins: make(map[int64]map[int64]bool),
	}
	s.loadAdmins()
	return s
}

func (s *Service) authStorePath() string {
	return envOr("AUTH_STORE_PATH", "data/auth.json")
}

func (s *Service) loadAdmins() {
	path := s.authStorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) { log.Printf("read auth store: %v", err) }
		return
	}
	var stored map[int64]map[int64]bool
	if err := json.Unmarshal(data, &stored); err != nil {
		log.Printf("decode auth store %s: %v", path, err)
		return
	}
	if stored != nil { s.admins = stored }
}

// persistAdminsLocked must be called while s.mu is held.
func (s *Service) persistAdminsLocked() error {
	path := s.authStorePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil { return err }
	data, err := json.MarshalIndent(s.admins, "", "  ")
	if err != nil { return err }
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil { return err }
	return os.Rename(tmp, path)
}

func isOwner(userID int64) bool {
	raw := strings.TrimSpace(os.Getenv("OWNER_ID"))
	if raw == "" { return false }
	var owner int64
	if _, err := fmt.Sscan(raw, &owner); err != nil { return false }
	return owner > 0 && owner == userID
}

func (s *Service) canControlChat(chatID, userID int64) bool {
	if isOwner(userID) { return true }
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.admins[chatID][userID]
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

	if s.handleExtendedCommand(ctx, b, msg, command, args, respond) { return }
	if s.isFeatureBanned(msg.From) { respond("⛔ Your account is blocked from using this bot."); return }
	if s.isMaintenanceEnabled() && (command == "/play" || command == "/queue") && (msg.From == nil || (!isOwner(msg.From.ID) && !s.isSudo(msg.From.ID))) { respond("🛠️ Bot is temporarily in maintenance mode."); return }
	switch command {
	case "/start":
		respond("🎧 Mikku Music Go is online.\n\nUse /play <song or YouTube URL> to search and send an audio file in this chat. Use /queue, /skip, and /stop to manage requests.\n\nNote: this Go edition sends audio files as Telegram messages; Telegram Bot API does not provide group voice-chat streaming controls.")
	case "/help":
		respond("🎵 Mikku Music — Go edition\n/start — welcome\n/help — commands\n/ping — health check\n/stats — uptime and request counters\n/active — active download chats\n/settings — current runtime settings\n/auth <user ID> — owner-only authorization\n/unauth <user ID> — owner-only revoke authorization\n/admins — list temporary authorized IDs\n/play <song or YouTube URL> — search, download and send audio\n/queue — show upcoming requests\n/skip — cancel current download or skip the next request\n/stop — cancel current download and clear the queue\n\nRequires yt-dlp and ffmpeg installed on the host. YouTube availability is controlled by YouTube and may vary.")
	case "/ping":
		respond("🏓 Pong! Mikku Go bot is online.")
	case "/stats":
		s.mu.Lock()
		activeCount := 0
		for _, running := range s.active { if running { activeCount++ } }
		requests := s.requests
		uptime := time.Since(s.started).Round(time.Second)
		s.mu.Unlock()
		respond(fmt.Sprintf("📊 Mikku Go Stats\nUptime: %s\nActive chat workers: %d\nRequests accepted: %d", uptime, activeCount, requests))
	case "/active":
		s.mu.Lock()
		var chats []string
		for id, running := range s.active { if running { chats = append(chats, fmt.Sprint(id)) } }
		s.mu.Unlock()
		if len(chats) == 0 { respond("No active downloads right now."); return }
		respond("🎧 Active download chats:\n" + strings.Join(chats, "\n"))
	case "/settings":
		respond("⚙️ Current settings\nQueue limit: 100 requests per chat\nPlayback mode: audio file sent to this chat\nDownload timeout: 8 minutes\nSource: YouTube via yt-dlp\n\nGroup voice-chat streaming controls are not available in this Go build yet.")
	case "/auth":
		if msg.From == nil || !isOwner(msg.From.ID) { respond("⛔ Only the configured OWNER_ID can manage bot authorizations."); return }
		var target int64
		if _, err := fmt.Sscan(strings.TrimSpace(args), &target); err != nil || target <= 0 { respond("Usage: /auth <Telegram user ID>"); return }
		s.mu.Lock()
		if s.admins[msg.Chat.ID] == nil { s.admins[msg.Chat.ID] = make(map[int64]bool) }
		s.admins[msg.Chat.ID][target] = true
		err := s.persistAdminsLocked()
		s.mu.Unlock()
		if err != nil { respond("❌ Could not save authorization: " + short(err.Error(), 160)); return }
		respond(fmt.Sprintf("✅ User %d authorized for this chat.", target))
	case "/unauth":
		if msg.From == nil || !isOwner(msg.From.ID) { respond("⛔ Only the configured OWNER_ID can manage bot authorizations."); return }
		var target int64
		if _, err := fmt.Sscan(strings.TrimSpace(args), &target); err != nil || target <= 0 { respond("Usage: /unauth <Telegram user ID>"); return }
		s.mu.Lock()
		delete(s.admins[msg.Chat.ID], target)
		err := s.persistAdminsLocked()
		s.mu.Unlock()
		if err != nil { respond("❌ Could not save authorization: " + short(err.Error(), 160)); return }
		respond(fmt.Sprintf("✅ User %d removed from this chat's authorization list.", target))
	case "/admins":
		s.mu.Lock()
		ids := make([]string, 0, len(s.admins[msg.Chat.ID]))
		for id := range s.admins[msg.Chat.ID] { ids = append(ids, fmt.Sprint(id)) }
		s.mu.Unlock()
		if len(ids) == 0 { respond("No extra authorized users in this chat."); return }
		respond("🔐 Authorized user IDs :\n" + strings.Join(ids, "\n"))
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
		s.mu.Lock(); s.requests++; s.mu.Unlock()
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
		if msg.From == nil || !s.canControlChat(msg.Chat.ID, msg.From.ID) { respond("⛔ Only the configured owner or a user authorized with /auth can skip requests."); return }
		s.mu.Lock()
		cancelCurrent := s.cancel[msg.Chat.ID]
		if cancelCurrent != nil { cancelCurrent() }
		s.mu.Unlock()
		if cancelCurrent != nil { respond("⏭️ Skipping the current download/request."); return }
		if item, ok := s.queues.Skip(msg.Chat.ID); ok { respond("⏭️ Skipped: " + item.Title) } else { respond("Nothing is queued.") }
	case "/stop":
		if msg.From == nil || !s.canControlChat(msg.Chat.ID, msg.From.ID) { respond("⛔ Only the configured owner or a user authorized with /auth can stop playback."); return }
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
		path, err = s.applyConfiguredSpeed(downloadCtx, chatID, path)
		if err != nil { cancel(); s.clearCurrentCancel(chatID); s.sendText(parent, b, chatID, "❌ Could not apply speed: "+short(err.Error(), 160)); continue }
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
		if err != nil && playCtx.Err() == nil { s.sendText(parent, b, chatID, "❌ Could not send audio: "+short(err.Error(), 180)) } else if err == nil { s.afterSuccessfulTrack(chatID, item) }
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
