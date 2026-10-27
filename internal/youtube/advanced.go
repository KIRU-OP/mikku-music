package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Format describes a downloadable format returned by yt-dlp.
type Format struct {
	Format   string `json:"format"`
	Filesize int64  `json:"filesize"`
	FormatID string `json:"format_id"`
	Ext      string `json:"ext"`
	Note     string `json:"format_note"`
	URL      string `json:"url"`
}

// Formats returns available non-DASH formats for a YouTube video.
func (a *API) Formats(ctx context.Context, input string) ([]Format, string, error) {
	u, err := cleanURL(input)
	if err != nil {
		if id := a.VideoID(input); id != "" {
			u = defaultBase + id
		} else {
			return nil, "", err
		}
	}
	r, err := a.runJSON(ctx, u)
	if err != nil {
		return nil, "", err
	}
	out, err := a.command(ctx, "--no-warnings", "--no-playlist", "-J", u)
	if err != nil {
		return nil, "", err
	}
	var raw struct {
		Formats []Format `json:"formats"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, "", fmt.Errorf("decode formats: %w", err)
	}
	_ = r
	formats := make([]Format, 0, len(raw.Formats))
	for _, f := range raw.Formats {
		if f.FormatID == "" || f.Ext == "" || f.Filesize <= 0 {
			continue
		}
		if strings.Contains(strings.ToLower(f.Format), "dash") {
			continue
		}
		f.URL = u
		formats = append(formats, f)
	}
	return formats, u, nil
}

// Slider returns the indexed search result, excluding live and over-one-hour videos.
func (a *API) Slider(ctx context.Context, query string, index int) (Track, error) {
	if index < 0 {
		return Track{}, errors.New("result index must not be negative")
	}
	results, err := a.Search(ctx, query, 10)
	if err != nil {
		return Track{}, err
	}
	filtered := make([]Track, 0, len(results))
	for _, track := range results {
		if track.DurationSec > 0 && track.DurationSec <= 3600 {
			filtered = append(filtered, track)
		}
	}
	if index >= len(filtered) {
		return Track{}, ErrNotFound
	}
	return filtered[index], nil
}

// FileSize sums known sizes from yt-dlp's format metadata.
func (a *API) FileSize(ctx context.Context, input string) (int64, error) {
	u, err := cleanURL(input)
	if err != nil {
		if id := a.VideoID(input); id != "" {
			u = defaultBase + id
		} else {
			return 0, err
		}
	}
	out, err := a.command(ctx, "--no-warnings", "--no-playlist", "-J", u)
	if err != nil {
		return 0, err
	}
	var raw struct {
		Formats []struct {
			Filesize int64 `json:"filesize"`
			FilesizeApprox int64 `json:"filesize_approx"`
		} `json:"formats"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return 0, fmt.Errorf("decode file sizes: %w", err)
	}
	var total int64
	for _, f := range raw.Formats {
		if f.Filesize > 0 {
			total += f.Filesize
		} else if f.FilesizeApprox > 0 {
			total += f.FilesizeApprox
		}
	}
	return total, nil
}

// WorkerMedia contains a resolved media URL returned by the optional Worker API.
// Signed URLs are temporary and should be consumed immediately.
type WorkerMedia struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

// WorkerFallback asks the configured Worker API for a media URL.
// It returns an error rather than silently inventing a URL when the API is disabled
// or responds without a valid HTTPS media link.
func (a *API) WorkerFallback(ctx context.Context, videoID, mediaFormat string) (WorkerMedia, error) {
	if a.cfg.WorkerFallback == "" || a.cfg.WorkerAPIKey == "" {
		return WorkerMedia{}, errors.New("worker fallback is not configured")
	}
	if !videoIDPattern.MatchString(videoID) {
		return WorkerMedia{}, errors.New("invalid YouTube video ID")
	}
	if mediaFormat != "audio" && mediaFormat != "video" && mediaFormat != "mp3" && mediaFormat != "mp4" {
		return WorkerMedia{}, errors.New("format must be audio/video or mp3/mp4")
	}
	endpoint, err := url.Parse(strings.TrimRight(a.cfg.WorkerFallback, "/") + "/api")
	if err != nil {
		return WorkerMedia{}, err
	}
	q := endpoint.Query()
	q.Set("key", a.cfg.WorkerAPIKey)
	q.Set("url", defaultBase+videoID)
	q.Set("format", mediaFormat)
	endpoint.RawQuery = q.Encode()

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return WorkerMedia{}, err
		}
		resp, err := a.http.Do(req)
		if err != nil {
			lastErr = err
		} else {
			var body map[string]any
			decodeErr := json.NewDecoder(resp.Body).Decode(&body)
			resp.Body.Close()
			if decodeErr != nil {
				lastErr = decodeErr
			} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				if success, _ := body["success"].(bool); success {
					mediaURL := findMediaURL(body)
					if mediaURL != "" {
						parsed, parseErr := url.Parse(mediaURL)
						if parseErr == nil && parsed.Scheme == "https" && parsed.Host != "" {
							title, _ := body["title"].(string)
							return WorkerMedia{URL: mediaURL, Title: title}, nil
						}
						return WorkerMedia{}, errors.New("worker returned an invalid media URL")
					}
					lastErr = errors.New("worker response contains no media URL")
				} else {
					reason, _ := body["message"].(string)
					if reason == "" {
						reason, _ = body["error"].(string)
					}
					if reason == "" {
						reason = "worker reported failure"
					}
					lastErr = errors.New(reason)
				}
			} else {
				lastErr = fmt.Errorf("worker returned HTTP %d", resp.StatusCode)
				if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
					return WorkerMedia{}, lastErr
				}
			}
		}
		if attempt < 2 {
			select {
			case <-ctx.Done():
				return WorkerMedia{}, ctx.Err()
			case <-time.After(time.Duration(attempt+1) * 250 * time.Millisecond):
			}
		}
	}
	if lastErr == nil {
		lastErr = errors.New("worker fallback failed")
	}
	return WorkerMedia{}, lastErr
}

func findMediaURL(value any) string {
	switch v := value.(type) {
	case map[string]any:
		for _, key := range []string{"play_url", "stream_url", "download_url", "url", "audio_url", "video_url"} {
			if raw, ok := v[key].(string); ok && strings.HasPrefix(raw, "https://") {
				return raw
			}
		}
		for _, child := range v {
			if found := findMediaURL(child); found != "" {
				return found
			}
		}
	case []any:
		for _, child := range v {
			if found := findMediaURL(child); found != "" {
				return found
			}
		}
	case string:
		if strings.HasPrefix(v, "https://") {
			parsed, err := url.Parse(v)
			if err == nil && parsed.Host != "" && (strings.Contains(parsed.Host, "googlevideo.com") || strings.Contains(parsed.Host, "youtube.com") || strings.Contains(parsed.Host, "workers.dev") || strings.Contains(parsed.Host, "cloudflare")) {
				return v
			}
		}
	}
	return ""
}

// ParseSizeString converts yt-dlp's numeric duration output without locale assumptions.
func ParseSizeString(value string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return n
}
