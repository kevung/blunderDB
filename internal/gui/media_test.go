package gui

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func newMediaFile(t *testing.T, name string, n int) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// reply keeps what the tests read of a response once its body is closed.
type reply struct {
	StatusCode int
	Header     http.Header
}

func get(t *testing.T, url, rng string) (reply, []byte) {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rng != "" {
		req.Header.Set("Range", rng)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return reply{resp.StatusCode, resp.Header}, b
}

func TestMediaRangeAndHeaders(t *testing.T) {
	var h mediaHost
	t.Cleanup(h.stop)
	u, err := h.register(newMediaFile(t, "a.MOV", 1000))
	if err != nil {
		t.Fatal(err)
	}
	resp, b := get(t, u, "bytes=0-99")
	if resp.StatusCode != 206 || len(b) != 100 {
		t.Fatalf("status %d len %d", resp.StatusCode, len(b))
	}
	if ct := resp.Header.Get("Content-Type"); ct != "video/mp4" {
		t.Fatalf("content-type %q", ct)
	}
	if c := resp.Header.Get("X-Media-Container"); c != "mov" {
		t.Fatalf("container %q", c)
	}
	if u2, _ := h.register(newMediaFile(t, "a.MOV", 10)); u2 == u {
		t.Fatal("distinct files share a token")
	}
}

func TestMediaUnknownTokenAndUnregisteredFile(t *testing.T) {
	var h mediaHost
	t.Cleanup(h.stop)
	secret := newMediaFile(t, "secret.mp4", 10)
	u, err := h.register(newMediaFile(t, "ok.mp4", 10))
	if err != nil {
		t.Fatal(err)
	}
	base := u[:strings.Index(u, "/media/")]
	// A Windows path (C:\...) is no URL path: probe it as /C:/... like the others.
	secret = "/" + strings.TrimPrefix(filepath.ToSlash(secret), "/")
	for _, p := range []string{"/media/deadbeef", "/media/", "/media/.." + secret, secret, "/media" + secret, "/yt/abcdefghijk"} {
		if resp, _ := get(t, base+p, ""); resp.StatusCode != 404 {
			t.Errorf("%s: status %d", p, resp.StatusCode)
		}
	}
	if _, err := h.register(filepath.Join(t.TempDir(), "missing.mp4")); err == nil {
		t.Fatal("missing file registered")
	}
}

func TestYouTube(t *testing.T) {
	cases := map[string]string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=3": "dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ?t=9":                "dQw4w9WgXcQ",
		"https://youtube.com/live/dQw4w9WgXcQ":            "dQw4w9WgXcQ",
		"https://www.youtube.com/shorts/dQw4w9WgXcQ":      "dQw4w9WgXcQ",
		"https://example.com/watch?v=dQw4w9WgXcQ":         "",
		"youtu.be/dQw4w9WgXcQ":                            "dQw4w9WgXcQ",
		"www.youtube.com/watch?v=dQw4w9WgXcQ":             "dQw4w9WgXcQ",
		"m.youtube.com/watch?v=dQw4w9WgXcQ":               "dQw4w9WgXcQ",
		"youtube.com/shorts/dQw4w9WgXcQ":                  "dQw4w9WgXcQ",
		"/home/x/a.mp4":                                   "",
	}
	for in, want := range cases {
		if got := youTubeID(in); got != want {
			t.Errorf("%s: %q", in, got)
		}
	}
	a := &App{}
	t.Cleanup(a.stopMedia)
	if a.VideoSourceKind("https://youtu.be/dQw4w9WgXcQ") != "youtube" || a.VideoSourceKind("https://x.org/a.mp4") != "url" ||
		a.VideoSourceKind("/a.mp4") != "file" || a.VideoSourceKind("") != "" {
		t.Fatal("kinds")
	}
	u, err := a.YouTubeEmbedURL("https://youtu.be/dQw4w9WgXcQ")
	if err != nil {
		t.Fatal(err)
	}
	if resp, b := get(t, u, ""); resp.StatusCode != 200 || !strings.Contains(string(b), "dQw4w9WgXcQ") {
		t.Fatalf("yt page %d", resp.StatusCode)
	}
	// A webview that sends the frame no referrer leaves the page the origin the pane passes.
	if resp, b := get(t, u+"?origin=wails%3A%2F%2Fwails", ""); resp.StatusCode != 200 || !strings.Contains(string(b), "URLSearchParams(location.search).get('origin')") {
		t.Fatalf("yt page with origin %d", resp.StatusCode)
	}
	if resp, _ := get(t, strings.Replace(u, "dQw4w9WgXcQ", "AAAAAAAAAAA", 1), ""); resp.StatusCode != 404 {
		t.Fatal("unattached id served")
	}
}

func TestMediaServesTheOpenPanesAndThemOnly(t *testing.T) {
	var h mediaHost
	t.Cleanup(h.stop)
	pa := newMediaFile(t, "a.mp4", 10)
	u1, _ := h.register(pa)
	u2, err := h.register(newMediaFile(t, "b.mp4", 10))
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []string{u1, u2} {
		if resp, _ := get(t, u, ""); resp.StatusCode != 200 {
			t.Fatalf("%s: %d", u, resp.StatusCode)
		}
	}
	// A second pane on the same file shares the token and keeps it alive.
	if again, _ := h.register(pa); again != u1 {
		t.Fatalf("same path, new token: %s != %s", again, u1)
	}
	h.release(u1)
	if resp, _ := get(t, u1, ""); resp.StatusCode != 200 {
		t.Fatal("released while another pane holds it")
	}
	h.release(u1)
	if resp, _ := get(t, u1, ""); resp.StatusCode != 404 {
		t.Fatal("served after the last release")
	}
	if resp, _ := get(t, u2, ""); resp.StatusCode != 200 {
		t.Fatal("another pane's video released")
	}
	y1, _ := h.registerYouTube("dQw4w9WgXcQ")
	y2, _ := h.registerYouTube("AAAAAAAAAAA")
	h.release(y1)
	if resp, _ := get(t, y1, ""); resp.StatusCode != 404 {
		t.Fatal("released id served")
	}
	if resp, _ := get(t, y2, ""); resp.StatusCode != 200 {
		t.Fatal("other id released")
	}
	h.release("http://127.0.0.1:1/media/unknown")
	h.release(u2)
	if resp, _ := get(t, u2, ""); resp.StatusCode != 404 {
		t.Fatal("served after release")
	}
}

func TestVideoDialogPatternAcceptsUpperCase(t *testing.T) {
	for _, want := range []string{"*.mov", "*.MOV", "*.mp4", "*.MP4"} {
		if !slices.Contains(strings.Split(videoDialogPattern, ";"), want) {
			t.Errorf("pattern %q lacks %s", videoDialogPattern, want)
		}
	}
}

func TestYouTubeWatchURL(t *testing.T) {
	a := &App{}
	for source, want := range map[string]string{
		"https://youtu.be/dQw4w9WgXcQ":                 "https://youtu.be/dQw4w9WgXcQ",
		" https://www.youtube.com/watch?v=dQw4w9WgXcQ": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"youtu.be/dQw4w9WgXcQ":                         "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"youtube.com/live/dQw4w9WgXcQ":                 "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"/home/me/match.mp4":                           "",
		"https://example.com/match.mp4":                "",
		"":                                             "",
	} {
		if got := a.YouTubeWatchURL(source); got != want {
			t.Errorf("YouTubeWatchURL(%q) = %q, want %q", source, got, want)
		}
	}
}
