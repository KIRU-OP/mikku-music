package youtube

import (
	"testing"
)

func TestFindMediaURL(t *testing.T) {
	got := findMediaURL(map[string]any{
		"success": true,
		"data": map[string]any{
			"play_url": "https://rr.googlevideo.com/media?sig=abc",
		},
	})
	if got != "https://rr.googlevideo.com/media?sig=abc" {
		t.Fatalf("unexpected media URL: %q", got)
	}
}

func TestFindMediaURLRejectsNonHTTPS(t *testing.T) {
	got := findMediaURL(map[string]any{
		"download_url": "http://example.com/file.mp3",
	})
	if got != "" {
		t.Fatalf("expected unsafe URL to be rejected, got %q", got)
	}
}

func TestWorkerFallbackRequiresConfig(t *testing.T) {
	a := New(Config{})
	if _, err := a.WorkerFallback(t.Context(), "dQw4w9WgXcQ", "audio"); err == nil {
		t.Fatal("expected an error when Worker fallback is not configured")
	}
}

func TestWorkerFallbackValidatesInput(t *testing.T) {
	a := New(Config{WorkerFallback: "https://example.workers.dev", WorkerAPIKey: "test"})
	if _, err := a.WorkerFallback(t.Context(), "invalid", "audio"); err == nil {
		t.Fatal("expected invalid video ID to be rejected")
	}
	if _, err := a.WorkerFallback(t.Context(), "dQw4w9WgXcQ", "exe"); err == nil {
		t.Fatal("expected unsupported format to be rejected")
	}
}
