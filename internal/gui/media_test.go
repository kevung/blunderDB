package gui

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
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

func get(t *testing.T, url, rng string) (*http.Response, []byte) {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	if rng != "" {
		req.Header.Set("Range", rng)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, b
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
	for _, p := range []string{"/media/deadbeef", "/media/", "/media/../" + secret, secret, "/media/" + secret, "/yt/abcdefghijk"} {
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
		"/home/x/a.mp4": "",
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
	if resp, _ := get(t, strings.Replace(u, "dQw4w9WgXcQ", "AAAAAAAAAAA", 1), ""); resp.StatusCode != 404 {
		t.Fatal("unattached id served")
	}
}

func TestMediaRealMOV(t *testing.T) {
	home, _ := os.UserHomeDir()
	m, _ := filepath.Glob(filepath.Join(home, "Desktop/20260523-hsbtMarseille-8x7p-1x9p/*.MOV"))
	if len(m) == 0 {
		t.Skip("no sample video")
	}
	var h mediaHost
	t.Cleanup(h.stop)
	u, err := h.register(m[0])
	if err != nil {
		t.Fatal(err)
	}
	if resp, b := get(t, u, "bytes=0-99"); resp.StatusCode != 206 || len(b) != 100 {
		t.Fatalf("status %d len %d", resp.StatusCode, len(b))
	}
}
