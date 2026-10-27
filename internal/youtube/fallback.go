package youtube

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ResolveStream resolves a stream using yt-dlp first, then the configured
// Worker API if the primary resolver fails. Signed URLs are short-lived.
func (a *API) ResolveStream(ctx context.Context, input, mediaFormat string) (string, error) {
	id := a.VideoID(input)
	if id == "" {
		return "", errors.New("a valid YouTube video URL or 11-character video ID is required")
	}
	streamURL, primaryErr := a.VideoStream(ctx, id)
	if primaryErr == nil && streamURL != "" {
		if a.CheckStream(ctx, streamURL) {
			return streamURL, nil
		}
		primaryErr = errors.New("primary stream URL failed the HTTP probe")
	}
	fallback, fallbackErr := a.WorkerFallback(ctx, id, mediaFormat)
	if fallbackErr == nil {
		return fallback.URL, nil
	}
	if primaryErr != nil {
		return "", fmt.Errorf("primary resolver: %v; worker fallback: %w", primaryErr, fallbackErr)
	}
	return "", fallbackErr
}

// DownloadWorkerMedia downloads an HTTPS media URL to a temporary file and
// atomically renames it to the requested path after a successful non-empty read.
func (a *API) DownloadWorkerMedia(ctx context.Context, mediaURL, destination string) error {
	if !strings.HasPrefix(mediaURL, "https://") {
		return errors.New("media URL must use HTTPS")
	}
	if destination == "" {
		return errors.New("destination path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0750); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return err
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("media server returned HTTP %d", resp.StatusCode)
	}
	tmp := destination + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	info, err := os.Stat(tmp)
	if err != nil || info.Size() == 0 {
		_ = os.Remove(tmp)
		if err != nil {
			return err
		}
		return errors.New("downloaded media is empty")
	}
	if err := os.Rename(tmp, destination); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
