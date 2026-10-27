// Package youtube ports the core of mikku/Youtube.py to Go.
//
// It uses yt-dlp as an external resolver for YouTube metadata and media URLs.
// Install yt-dlp and ffmpeg on the host. YouTube may refuse requests or expire
// signed media URLs; callers should resolve URLs immediately before playback.
package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultBase = "https://www.youtube.com/watch?v="

var (
	videoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	ansiPattern    = regexp.MustCompile(`\x1b(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])`)
	ErrNotFound    = errors.New("youtube: no matching result")
)

type Config struct {
	YTDLPPath       string
	CookiesFile     string
	DownloadDir     string
	DurationLimit   time.Duration
	HTTPTimeout     time.Duration
	WorkerFallback  string
	WorkerAPIKey    string
	CacheTTL        time.Duration
}

type API struct {
	cfg Config
	http *http.Client
	mu sync.RWMutex
	cache map[string]cacheEntry
}

type cacheEntry struct { track Track; expires time.Time }

type Track struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"link"`
	DurationMin string `json:"duration_min"`
	DurationSec int    `json:"duration_sec"`
	Thumbnail   string `json:"thumb"`
}

type ytResult struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Webpage  string `json:"webpage_url"`
	Duration float64 `json:"duration"`
	Thumb    string `json:"thumbnail"`
	Thumbs   []struct { URL string `json:"url"` } `json:"thumbnails"`
	Entries  []*ytResult `json:"entries"`
}

func New(cfg Config) *API {
	if cfg.YTDLPPath == "" { cfg.YTDLPPath = "yt-dlp" }
	if cfg.CookiesFile == "" { cfg.CookiesFile = strings.TrimSpace(os.Getenv("YTDLP_COOKIES_FILE")) }
	if cfg.DownloadDir == "" { cfg.DownloadDir = "downloads" }
	if cfg.HTTPTimeout <= 0 { cfg.HTTPTimeout = 15 * time.Second }
	if cfg.CacheTTL <= 0 { cfg.CacheTTL = time.Hour }
	if cfg.WorkerFallback == "" { cfg.WorkerFallback = strings.TrimRight(os.Getenv("WORKER_FALLBACK_API_URL"), "/") }
	if cfg.WorkerAPIKey == "" { cfg.WorkerAPIKey = os.Getenv("WORKER_FALLBACK_API_KEY") }
	return &API{cfg: cfg, http: &http.Client{Timeout: cfg.HTTPTimeout}, cache: make(map[string]cacheEntry)}
}

func (a *API) Exists(input string) bool {
	u, err := safeYouTubeURL(input)
	return err == nil && (u.Host == "youtube.com" || strings.HasSuffix(u.Host, ".youtube.com") || u.Host == "youtu.be" || strings.HasSuffix(u.Host, ".youtu.be"))
}

func (a *API) VideoID(input string) string {
	if videoIDPattern.MatchString(strings.TrimSpace(input)) { return strings.TrimSpace(input) }
	u, err := safeYouTubeURL(input); if err != nil { return "" }
	if u.Host == "youtu.be" || strings.HasSuffix(u.Host, ".youtu.be") {
		id := strings.Trim(u.Path, "/"); if videoIDPattern.MatchString(id) { return id }; return ""
	}
	if id := u.Query().Get("v"); videoIDPattern.MatchString(id) { return id }
	for _, prefix := range []string{"/embed/", "/shorts/", "/live/"} {
		if strings.HasPrefix(u.Path, prefix) {
			id := strings.Split(strings.TrimPrefix(u.Path, prefix), "/")[0]
			if videoIDPattern.MatchString(id) { return id }
		}
	}
	return ""
}

func safeYouTubeURL(input string) (*url.URL, error) {
	input = strings.TrimSpace(input)
	if input == "" || strings.ContainsAny(input, "\r\n\x00") { return nil, errors.New("empty or invalid URL") }
	if !strings.Contains(input, "://") { input = "https://" + input }
	u, err := url.Parse(input); if err != nil { return nil, err }
	if u.Scheme != "https" && u.Scheme != "http" { return nil, errors.New("unsupported URL scheme") }
	host := strings.ToLower(u.Hostname())
	if host == "" || net.ParseIP(host) != nil { return nil, errors.New("invalid YouTube host") }
	if host != "youtube.com" && !strings.HasSuffix(host, ".youtube.com") && host != "youtu.be" && !strings.HasSuffix(host, ".youtu.be") {
		return nil, errors.New("URL is not a YouTube URL")
	}
	u.Host = host
	return u, nil
}

func cleanURL(input string) (string, error) {
	u, err := safeYouTubeURL(input); if err != nil { return "", err }
	if id := u.Query().Get("v"); id != "" {
		return defaultBase + url.QueryEscape(id), nil
	}
	u.Fragment = ""
	q := u.Query(); q.Del("si"); q.Del("feature"); u.RawQuery = q.Encode()
	return u.String(), nil
}

func (a *API) command(ctx context.Context, args ...string) ([]byte, error) {
	base := []string{}
	if a.cfg.CookiesFile != "" {
		if st, err := os.Stat(a.cfg.CookiesFile); err == nil && !st.IsDir() { base = append(base, "--cookies", a.cfg.CookiesFile) }
	}
	cmd := exec.CommandContext(ctx, a.cfg.YTDLPPath, append(base, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	cmd.Env = os.Environ()
	err := cmd.Run()
	if err != nil { msg := strings.TrimSpace(ansiPattern.ReplaceAllString(stderr.String(), "")); if msg == "" { msg = err.Error() }; return nil, fmt.Errorf("yt-dlp: %s", msg) }
	return stdout.Bytes(), nil
}

func parseDuration(seconds float64) int { if seconds < 0 { return 0 }; return int(seconds) }
func durationLabel(seconds int) string {
	if seconds <= 0 { return "" }
	h := seconds / 3600; m := (seconds % 3600) / 60; s := seconds % 60
	if h > 0 { return fmt.Sprintf("%d:%02d:%02d", h, m, s) }
	return fmt.Sprintf("%d:%02d", m, s)
}

func toTrack(r *ytResult) (Track, error) {
	if r == nil || r.ID == "" || r.Title == "" { return Track{}, ErrNotFound }
	u := r.Webpage
	if u == "" { u = defaultBase + r.ID }
	thumb := r.Thumb
	if thumb == "" && len(r.Thumbs) > 0 { thumb = r.Thumbs[0].URL }
	if parsed, err := url.Parse(thumb); err == nil && parsed != nil { parsed.RawQuery = ""; thumb = parsed.String() }
	sec := parseDuration(r.Duration)
	return Track{ID:r.ID, Title:r.Title, URL:u, DurationMin:durationLabel(sec), DurationSec:sec, Thumbnail:thumb}, nil
}

func (a *API) runJSON(ctx context.Context, args ...string) (*ytResult, error) {
	out, err := a.command(ctx, append([]string{"--no-warnings", "--no-playlist", "--dump-single-json"}, args...)...)
	if err != nil { return nil, err }
	var r ytResult
	if err := json.Unmarshal(out, &r); err != nil { return nil, fmt.Errorf("decode yt-dlp metadata: %w", err) }
	return &r, nil
}

func (a *API) Search(ctx context.Context, query string, limit int) ([]Track, error) {
	query = strings.TrimSpace(query)
	if query == "" { return nil, errors.New("search query is empty") }
	if limit < 1 { limit = 1 }; if limit > 50 { limit = 50 }
	target := query
	if videoIDPattern.MatchString(query) {
		target = defaultBase + query
	} else if normalized, normalizeErr := cleanURL(query); normalizeErr == nil {
		target = normalized
	} else {
		target = fmt.Sprintf("ytsearch%d:%s", limit, query)
	}
	r, err := a.runJSON(ctx, "--flat-playlist", target)
	if err != nil { return nil, err }
	items := r.Entries; if len(items) == 0 { items = []*ytResult{r} }
	tracks := make([]Track, 0, len(items))
	for _, item := range items {
		if item == nil || item.ID == "" || item.Title == "" { continue }
		if item.Webpage == "" { item.Webpage = defaultBase + item.ID }
		t, e := toTrack(item); if e == nil { tracks = append(tracks, t) }
	}
	if len(tracks) == 0 { return nil, ErrNotFound }
	return tracks, nil
}

func (a *API) Details(ctx context.Context, input string) (Track, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Track{}, errors.New("YouTube URL, video ID, or search query is required")
	}
	if id := a.VideoID(input); id != "" {
		input = defaultBase + id
	} else if u, err := cleanURL(input); err == nil {
		input = u
	} else {
		results, searchErr := a.Search(ctx, input, 1)
		if searchErr != nil {
			return Track{}, searchErr
		}
		return results[0], nil
	}

	a.mu.RLock()
	cached, ok := a.cache[input]
	a.mu.RUnlock()
	if ok && time.Now().Before(cached.expires) {
		return cached.track, nil
	}
	r, err := a.runJSON(ctx, input)
	if err != nil {
		return Track{}, err
	}
	t, err := toTrack(r)
	if err != nil {
		return Track{}, err
	}
	a.mu.Lock()
	a.cache[input] = cacheEntry{track: t, expires: time.Now().Add(a.cfg.CacheTTL)}
	a.mu.Unlock()
	return t, nil
}

func (a *API) Title(ctx context.Context, input string) (string, error) { t,e := a.Details(ctx,input); return t.Title,e }
func (a *API) Duration(ctx context.Context, input string) (string, error) { t,e := a.Details(ctx,input); return t.DurationMin,e }
func (a *API) Thumbnail(ctx context.Context, input string) (string, error) { t,e := a.Details(ctx,input); return t.Thumbnail,e }
func (a *API) TrackDetails(ctx context.Context, input string) (Track, error) { return a.Details(ctx,input) }

// VideoStream resolves a video-only-or-combined stream up to 720p. The returned
// signed URL can expire, so resolve it immediately before handing it to a player.
func (a *API) VideoStream(ctx context.Context, input string) (string, error) {
	u, err := cleanURL(input)
	if err != nil {
		if id := a.VideoID(input); id != "" {
			u = defaultBase + id
		} else {
			return "", errors.New("a valid YouTube URL or video ID is required")
		}
	}
	out, err := a.command(ctx, "-g", "-f", "best[height<=720][width<=1280]/best", u)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line), nil
		}
	}
	return "", errors.New("yt-dlp returned no stream URL")
}

func (a *API) Playlist(ctx context.Context, input string, limit int) ([]string, error) {
	if limit < 1 { limit = 1 }; if limit > 500 { limit = 500 }
	u, err := safeYouTubeURL(input); if err != nil { return nil, err }
	if !strings.Contains(u.RawQuery, "list=") && u.Query().Get("list")=="" { return nil, errors.New("URL has no playlist ID") }
	out, err := a.command(ctx, "--flat-playlist", "--playlist-end", strconv.Itoa(limit), "--get-id", "--skip-download", u.String())
	if err != nil { return nil, err }
	var ids []string
	for _, line := range strings.Split(string(out), "\n") { id:=strings.TrimSpace(line); if videoIDPattern.MatchString(id) { ids=append(ids,id) } }
	return ids,nil
}

func (a *API) Download(ctx context.Context, input, format string) (string, error) {
	u, err := cleanURL(input)
	if err != nil {
		if id := a.VideoID(input); id != "" {
			u = defaultBase + id
		} else {
			return "", errors.New("a valid YouTube URL or video ID is required")
		}
	}
	if format != "audio" && format != "video" {
		return "", errors.New("format must be audio or video")
	}
	if err := os.MkdirAll(a.cfg.DownloadDir, 0750); err != nil {
		return "", err
	}
	outTemplate := filepath.Join(a.cfg.DownloadDir, "%(id)s.%(ext)s")
	args := []string{"--no-playlist", "--no-part", "-o", outTemplate}
	if format == "audio" {
		args = append(args, "-x", "--audio-format", "mp3")
	} else {
		args = append(args, "-f", "best[height<=720]/best")
	}
	args = append(args, u)
	if _, err := a.command(ctx, args...); err != nil {
		return "", err
	}
	r, err := a.runJSON(ctx, u)
	if err != nil {
		return "", err
	}
	pattern := filepath.Join(a.cfg.DownloadDir, r.ID+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	for _, name := range matches {
		if st, e := os.Stat(name); e == nil && st.Size() > 0 {
			return name, nil
		}
	}
	return "", errors.New("download completed but output file was not found")
}

// Autoplay chooses a different, playable candidate from search results.
// This is a conservative search fallback, not YouTube's private recommendation API.
func (a *API) Autoplay(ctx context.Context, currentID, title string, maxDuration time.Duration) (*Track, error) {
	query := strings.TrimSpace(title)
	if query == "" { return nil, ErrNotFound }
	query = regexp.MustCompile(`\[[^\]]*\]|\([^\)]*\)`).ReplaceAllString(query," ")
	query = regexp.MustCompile(`(?i)\b(official|video|audio|lyrics?|lyrical|4k|hd|remix|song|music|feat\.?|ft\.?)\b`).ReplaceAllString(query," ")
	results, err := a.Search(ctx, query, 12); if err != nil { return nil,err }
	for _, t := range results {
		if t.ID == currentID || t.DurationSec <= 0 { continue }
		if maxDuration > 0 && time.Duration(t.DurationSec)*time.Second > maxDuration { continue }
		return &t,nil
	}
	return nil,ErrNotFound
}

// CheckStream probes a resolved URL with a byte range request. It does not
// guarantee that a later player request will succeed because signed URLs expire.
func (a *API) CheckStream(ctx context.Context, streamURL string) bool {
	req, err := http.NewRequestWithContext(ctx,http.MethodGet,streamURL,nil); if err != nil { return false }
	req.Header.Set("User-Agent","Mozilla/5.0")
	req.Header.Set("Range","bytes=0-0")
	resp, err := a.http.Do(req); if err != nil { return false }; defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent
}
