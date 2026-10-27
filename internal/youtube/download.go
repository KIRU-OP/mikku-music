package youtube

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

// DownloadResult reports the local path and which resolver produced the file.
type DownloadResult struct {
	Path   string
	Source string
}

// AudioDownload downloads audio locally, using the Worker fallback if yt-dlp fails.
func (a *API) AudioDownload(ctx context.Context, videoID string) (DownloadResult, error) {
	return a.downloadWithFallback(ctx, videoID, "audio", ".mp3")
}

// VideoDownload downloads video locally, using the Worker fallback if yt-dlp fails.
func (a *API) VideoDownload(ctx context.Context, videoID string) (DownloadResult, error) {
	return a.downloadWithFallback(ctx, videoID, "video", ".mp4")
}

func (a *API) downloadWithFallback(ctx context.Context, videoID, mediaFormat, ext string) (DownloadResult, error) {
	if !videoIDPattern.MatchString(videoID) {
		return DownloadResult{}, errors.New("invalid YouTube video ID")
	}
	if err := os.MkdirAll(a.cfg.DownloadDir, 0750); err != nil {
		return DownloadResult{}, err
	}
	destination := filepath.Join(a.cfg.DownloadDir, videoID+ext)
	if info, err := os.Stat(destination); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
		return DownloadResult{Path: destination, Source: "LOCAL CACHE"}, nil
	}
	path, primaryErr := a.Download(ctx, videoID, mediaFormat)
	if primaryErr == nil && path != "" {
		return DownloadResult{Path: path, Source: "YTDLP DIRECT"}, nil
	}
	media, fallbackErr := a.WorkerFallback(ctx, videoID, mediaFormat)
	if fallbackErr != nil {
		if primaryErr != nil {
			return DownloadResult{}, errors.Join(primaryErr, fallbackErr)
		}
		return DownloadResult{}, fallbackErr
	}
	if err := a.DownloadWorkerMedia(ctx, media.URL, destination); err != nil {
		if primaryErr != nil {
			return DownloadResult{}, errors.Join(primaryErr, err)
		}
		return DownloadResult{}, err
	}
	return DownloadResult{Path: destination, Source: "WORKER FALLBACK"}, nil
}
