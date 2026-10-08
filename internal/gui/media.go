package gui

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// mediaHost serves the videos of the open panes from a loopback HTTP server
// rather than Wails' asset server, which does not honour Range requests. It
// serves only files registered through MediaURL, each under an unguessable
// token: the URL never carries a path, so nothing else on disk is reachable.
// A registration is counted: two panes on the same file share a token, and the
// token goes when the last of them releases its URL.
type mediaHost struct {
	mu      sync.Mutex
	srv     *http.Server
	base    string
	tokens  map[string]string // token -> absolute path
	byPath  map[string]string // absolute path -> token
	refs    map[string]int    // token or YouTube id -> panes holding it
	youtube map[string]bool   // ids with an attached YouTube source
}

var youTubeIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{6,20}$`)

// mediaContentType maps a container extension to the type the webview needs.
func mediaContentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".m4v", ".mov":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mkv":
		return "video/x-matroska"
	case ".ogv":
		return "video/ogg"
	}
	return "application/octet-stream"
}

func mediaContainer(path string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
}

func (h *mediaHost) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/media/", h.serveMedia)
	mux.HandleFunc("/yt/", h.serveYouTube)
	return mux
}

func (h *mediaHost) serveMedia(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/media/")
	h.mu.Lock()
	path, ok := h.tokens[token]
	h.mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mediaContentType(path))
	w.Header().Set("X-Media-Container", mediaContainer(path))
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, "", st.ModTime(), f)
}

// The page gives the IFrame Player a real HTTP referrer and relays
// currentTime, play/pause and seek with the parent through postMessage. The
// parent's origin comes from the referrer, or from ?origin= when a webview
// sends the frame none; replies still go only to the parent window.
const youTubePage = `<!doctype html><html><head><meta charset="utf-8"><style>html,body,#p{margin:0;width:100%%;height:100%%;background:#000}</style></head><body><div id="p"></div><script src="https://www.youtube.com/iframe_api"></script><script>
var player;
function onYouTubeIframeAPIReady(){player=new YT.Player('p',{width:'100%%',height:'100%%',videoId:'%s',playerVars:{playsinline:1,rel:0},events:{onReady:function(){post({type:'ready',duration:player.getDuration()})},onStateChange:function(e){post({type:'state',state:e.data,time:player.getCurrentTime()})}}})}
var origin=''; try{origin=new URL(document.referrer).origin}catch(e){}
if(!origin||origin==='null'){try{origin=new URLSearchParams(location.search).get('origin')||''}catch(e){}}
function post(m){if(!origin||origin==='null')return;m.source='blunderdb-yt';parent.postMessage(m,origin)}
setInterval(function(){if(player&&player.getCurrentTime)post({type:'time',time:player.getCurrentTime()})},250);
addEventListener('message',function(e){if(!origin||e.origin!==origin||e.source!==parent)return;var m=e.data||{};if(!player)return;
if(m.type==='play')player.playVideo();else if(m.type==='pause')player.pauseVideo();else if(m.type==='seek')player.seekTo(m.time,true)});
</script></body></html>`

func (h *mediaHost) serveYouTube(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/yt/")
	h.mu.Lock()
	ok := h.youtube[id]
	h.mu.Unlock()
	if !ok || !youTubeIDRe.MatchString(id) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, youTubePage, html.EscapeString(id))
}

// startLocked brings the server up on a free loopback port on first use.
func (h *mediaHost) startLocked() error {
	if h.srv != nil {
		return nil
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	h.srv = &http.Server{Handler: h.handler(), ReadHeaderTimeout: 10 * time.Second}
	h.base = "http://" + ln.Addr().String()
	srv := h.srv
	go func() { _ = srv.Serve(ln) }()
	return nil
}

// register makes path servable and returns its URL.
func (h *mediaHost) register(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if st.IsDir() {
		return "", errors.New("not a file: " + abs)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.startLocked(); err != nil {
		return "", err
	}
	tok, ok := h.byPath[abs]
	if !ok {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			return "", err
		}
		tok = hex.EncodeToString(b[:])
		if h.tokens == nil {
			h.tokens, h.byPath = map[string]string{}, map[string]string{}
		}
		h.tokens[tok], h.byPath[abs] = abs, tok
	}
	h.hold(tok)
	return h.base + "/media/" + tok, nil
}

func (h *mediaHost) registerYouTube(id string) (string, error) {
	if !youTubeIDRe.MatchString(id) {
		return "", errors.New("invalid YouTube id")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := h.startLocked(); err != nil {
		return "", err
	}
	if h.youtube == nil {
		h.youtube = map[string]bool{}
	}
	h.youtube[id] = true
	h.hold(id)
	return h.base + "/yt/" + id, nil
}

// hold counts one more pane on key. Called with h.mu held.
func (h *mediaHost) hold(key string) {
	if h.refs == nil {
		h.refs = map[string]int{}
	}
	h.refs[key]++
}

// release forgets one pane's registration, named by the URL it was given; the
// file or YouTube id stops being served with the last pane that held it, and
// the server stays up. An unknown URL is ignored.
func (h *mediaHost) release(u string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var key, kind string
	switch {
	case h.base != "" && strings.HasPrefix(u, h.base+"/media/"):
		key, kind = strings.TrimPrefix(u, h.base+"/media/"), "media"
	case h.base != "" && strings.HasPrefix(u, h.base+"/yt/"):
		key, kind = strings.TrimPrefix(u, h.base+"/yt/"), "yt"
	default:
		return
	}
	if h.refs[key] == 0 {
		return
	}
	if h.refs[key]--; h.refs[key] > 0 {
		return
	}
	delete(h.refs, key)
	if kind == "yt" {
		delete(h.youtube, key)
		return
	}
	delete(h.byPath, h.tokens[key])
	delete(h.tokens, key)
}

func (h *mediaHost) stop() {
	h.mu.Lock()
	srv := h.srv
	h.srv, h.base = nil, ""
	h.tokens, h.byPath, h.youtube, h.refs = nil, nil, nil, nil
	h.mu.Unlock()
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if srv.Shutdown(ctx) != nil {
			_ = srv.Close()
		}
	}
}

// ReleaseMedia stops serving the video behind url, a URL MediaURL or
// YouTubeEmbedURL returned (its pane closed or changed source); the other
// panes' videos keep playing.
func (a *App) ReleaseMedia(url string) { a.media.release(url) }

// stopMedia stops the loopback media server, at shutdown.
func (a *App) stopMedia() { a.media.stop() }

// youTubeID extracts the video id from the recognised YouTube URL shapes.
func youTubeID(source string) string {
	source = strings.TrimSpace(source)
	if !strings.Contains(source, "://") {
		source = "https://" + source
	}
	u, err := url.Parse(source)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	host = strings.TrimPrefix(host, "m.")
	var id string
	switch host {
	case "youtu.be":
		id = strings.Trim(u.Path, "/")
	case "youtube.com", "music.youtube.com":
		p := strings.Trim(u.Path, "/")
		switch {
		case p == "watch":
			id = u.Query().Get("v")
		case strings.HasPrefix(p, "live/"):
			id = strings.TrimPrefix(p, "live/")
		case strings.HasPrefix(p, "shorts/"):
			id = strings.TrimPrefix(p, "shorts/")
		case strings.HasPrefix(p, "embed/"):
			id = strings.TrimPrefix(p, "embed/")
		}
	}
	if i := strings.IndexByte(id, '/'); i >= 0 {
		id = id[:i]
	}
	if !youTubeIDRe.MatchString(id) {
		return ""
	}
	return id
}

// PickTranscriptionVideo asks for a video file; empty when cancelled.
func (a *App) PickTranscriptionVideo() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose the match video",
		Filters: []runtime.FileFilter{
			{DisplayName: "Videos (*.mp4, *.m4v, *.mov, *.webm, *.mkv, *.ogv)", Pattern: "*.mp4;*.m4v;*.mov;*.webm;*.mkv;*.ogv"},
		},
	})
}

// MediaURL registers a video file and returns its loopback URL; the error
// (missing file) is what lets the frontend offer to relocate it.
func (a *App) MediaURL(path string) (string, error) { return a.media.register(path) }

// VideoSourceKind classifies a stored source: "file", "youtube", "url" or "".
func (a *App) VideoSourceKind(source string) string {
	s := strings.TrimSpace(source)
	switch {
	case s == "":
		return ""
	case youTubeID(s) != "":
		return "youtube"
	case strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://"):
		return "url"
	}
	return "file"
}

// YouTubeEmbedURL registers the source's video and returns the loopback
// player page; the page is served only for a source attached this way.
func (a *App) YouTubeEmbedURL(source string) (string, error) {
	id := youTubeID(source)
	if id == "" {
		return "", errors.New("not a YouTube URL")
	}
	return a.media.registerYouTube(id)
}

// OpenVideoExternally opens a YouTube source at ms in the browser, any other
// URL as is.
func (a *App) OpenVideoExternally(source string, ms int64) error {
	if id := youTubeID(source); id != "" {
		runtime.BrowserOpenURL(a.ctx, fmt.Sprintf("https://youtu.be/%s?t=%d", id, ms/1000))
		return nil
	}
	if k := a.VideoSourceKind(source); k == "url" {
		runtime.BrowserOpenURL(a.ctx, strings.TrimSpace(source))
		return nil
	}
	return errors.New("not a web video source")
}
