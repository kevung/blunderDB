package gui

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// The V0.1 webview bench (tasks/transcription-vision/bancs/01-webview-linux.md) only runs
// when BLUNDERDB_BENCH_V01 names a video: it serves that video with CORS, opens the test page
// and writes the page's JSON report to BLUNDERDB_BENCH_V01_OUT, then quits.
const (
	benchV01VideoEnv = "BLUNDERDB_BENCH_V01"
	benchV01OutEnv   = "BLUNDERDB_BENCH_V01_OUT"
)

func benchV01Enabled() bool { return os.Getenv(benchV01VideoEnv) != "" }

// benchV01CORS wraps the media handler: pixels read back from a loopback video need
// Access-Control-Allow-Origin on every answer, OPTIONS and 206 included.
func benchV01CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Accept-Ranges")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) benchV01Result(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	if out := os.Getenv(benchV01OutEnv); out != "" {
		_ = os.WriteFile(out, body, 0o644)
	}
	w.WriteHeader(http.StatusNoContent)
	if r.URL.Query().Get("final") == "1" {
		go func() { time.Sleep(500 * time.Millisecond); runtime.Quit(a.ctx) }()
	}
}

// benchV01Start opens the test page once the main page has had time to load.
func (a *App) benchV01Start() {
	if !benchV01Enabled() {
		return
	}
	a.media.mu.Lock()
	a.media.benchResult = a.benchV01Result
	a.media.mu.Unlock()
	u, err := a.media.register(os.Getenv(benchV01VideoEnv))
	if err != nil {
		println("bench v01:", err.Error())
		return
	}
	a.media.mu.Lock()
	base := a.media.base
	a.media.mu.Unlock()
	go func() {
		time.Sleep(3 * time.Second)
		target := "/bench/v01.html?v=" + url.QueryEscape(u) + "&r=" + url.QueryEscape(base+"/bench/result")
		runtime.WindowExecJS(a.ctx, "location.replace('"+target+"')")
	}()
}
