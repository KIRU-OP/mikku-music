package youtube

import (
	"net/url"
	"testing"
	"time"
)

func TestVideoID(t *testing.T) {
	a := New(Config{})
	cases := map[string]string{
		"dQw4w9WgXcQ": "dQw4w9WgXcQ",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=2": "dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ?si=abc": "dQw4w9WgXcQ",
		"https://youtube.com/shorts/dQw4w9WgXcQ": "dQw4w9WgXcQ",
		"https://youtube.com/embed/dQw4w9WgXcQ": "dQw4w9WgXcQ",
		"https://example.com/watch?v=dQw4w9WgXcQ": "",
		"invalid": "",
	}
	for input, want := range cases {
		if got := a.VideoID(input); got != want { t.Errorf("VideoID(%q) = %q, want %q", input, got, want) }
	}
}

func TestSafeYouTubeURL(t *testing.T) {
	for _, input := range []string{"https://youtube.com/watch?v=dQw4w9WgXcQ", "youtu.be/dQw4w9WgXcQ", "https://music.youtube.com/watch?v=dQw4w9WgXcQ"} {
		if _, err := safeYouTubeURL(input); err != nil { t.Errorf("expected valid URL %q: %v", input, err) }
	}
	for _, input := range []string{"", "https://example.com/watch?v=dQw4w9WgXcQ", "javascript:alert(1)", "https://127.0.0.1/x"} {
		if _, err := safeYouTubeURL(input); err == nil { t.Errorf("expected rejection for %q", input) }
	}
}

func TestToTrack(t *testing.T) {
	r := &ytResult{ID:"dQw4w9WgXcQ",Title:"Example",Duration:213,Thumb:"https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg?x=1"}
	got, err := toTrack(r)
	if err != nil { t.Fatal(err) }
	if got.URL != defaultBase+r.ID || got.DurationSec != 213 || got.DurationMin != "3:33" { t.Fatalf("unexpected track: %#v",got) }
	u, _ := url.Parse(got.Thumbnail)
	if u.RawQuery != "" { t.Errorf("thumbnail query was not removed: %q",got.Thumbnail) }
}

func TestDurationLabel(t *testing.T) {
	if got:=durationLabel(0); got!="" { t.Errorf("zero duration = %q",got) }
	if got:=durationLabel(65); got!="1:05" { t.Errorf("durationLabel(65) = %q",got) }
	if got:=durationLabel(3661); got!="1:01:01" { t.Errorf("durationLabel(3661) = %q",got) }
}

func TestNewDefaults(t *testing.T) {
	a := New(Config{CacheTTL: time.Minute})
	if a.cfg.YTDLPPath != "yt-dlp" || a.cfg.CacheTTL != time.Minute { t.Fatalf("unexpected config defaults: %#v",a.cfg) }
}
