//go:build simulation

// Command shim serves a real *database.Database over HTTP so that the real
// Svelte front, run under Vite and driven by Playwright, talks to the real Go
// backend instead of the e2e mock. It is measurement tooling for the
// simulated tournaments of #380, excluded from CI by its build tag.
//
//	go run -tags simulation ./tasks/nicomaque/simulation-2026-10/outils/shim -db X.db -port 8931 -out DIR
//
// POST /call/<Method> takes the JSON array of arguments in signature order and
// answers {"result": …} or {"error": "…"}, the Wails binding convention: a
// trailing error rejects, a lone error resolves to null.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
)

var (
	errType = reflect.TypeOf((*error)(nil)).Elem()
	ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()
)

type shim struct {
	db       *database.Database
	dbv      reflect.Value
	out      string
	mu       sync.Mutex
	warnings []database.PageWarning
	calls    map[string]int
}

func main() {
	path := flag.String("db", "", "SQLite database path (created when missing)")
	port := flag.Int("port", 8931, "listen port on 127.0.0.1")
	out := flag.String("out", "", "folder standing in for the native save/folder dialogs")
	flag.Parse()
	if *path == "" {
		log.Fatal("-db is required")
	}
	db := database.NewDatabase()
	// The front opens the file itself (StartupFilePath), as the GUI does on a
	// file-association launch; a missing file is created here and released so
	// that open finds the single-writer lock free.
	if _, err := os.Stat(*path); os.IsNotExist(err) {
		if err := db.SetupDatabase(*path); err != nil {
			log.Fatalf("setup: %v", err)
		}
		if err := db.Close(); err != nil {
			log.Fatalf("close after setup: %v", err)
		}
	}
	s := &shim{db: db, dbv: reflect.ValueOf(db), out: *out, calls: map[string]int{}, warnings: []database.PageWarning{}}
	// What the GUI turns into a status-bar event, kept for the spec to read.
	database.OnDirectionPageWarning(db, func(w database.PageWarning) {
		s.mu.Lock()
		s.warnings = append(s.warnings, w)
		s.mu.Unlock()
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/call/", s.call)
	mux.HandleFunc("/methods", s.methods)
	mux.HandleFunc("/warnings", s.pageWarnings)
	mux.HandleFunc("/stats", s.stats)
	mux.HandleFunc("/savecsv", s.saveCSV)
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	log.Printf("shim on %s, db %s", addr, *path)
	log.Fatal(http.ListenAndServe(addr, cors(mux)))
}

func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode: %v", err)
	}
}

func (s *shim) methods(w http.ResponseWriter, _ *http.Request) {
	t := s.dbv.Type()
	names := make([]string, 0, t.NumMethod())
	for i := 0; i < t.NumMethod(); i++ {
		names = append(names, t.Method(i).Name)
	}
	sort.Strings(names)
	writeJSON(w, names)
}

func (s *shim) pageWarnings(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, s.warnings)
}

func (s *shim) stats(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, s.calls)
}

// saveCSV stands in for gui.App.SaveCSV: the file lands in -out under the
// proposed name, so the spec can read what the director would have saved.
func (s *shim) saveCSV(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Body string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	dir := s.out
	if dir == "" {
		dir = os.TempDir()
	}
	name := filepath.Base(req.Name)
	if !strings.HasSuffix(strings.ToLower(name), ".csv") {
		name += ".csv"
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(req.Body), 0o644); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]string{"result": p})
}

func (s *shim) call(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/call/")
	m := s.dbv.MethodByName(name)
	if !m.IsValid() {
		writeJSON(w, map[string]string{"error": "shim: unknown method " + name})
		return
	}
	s.mu.Lock()
	s.calls[name]++
	s.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	var raw []json.RawMessage
	if len(body) > 0 {
		if err := json.Unmarshal(body, &raw); err != nil {
			writeJSON(w, map[string]string{"error": "shim: args: " + err.Error()})
			return
		}
	}
	mt := m.Type()
	args := make([]reflect.Value, 0, mt.NumIn())
	j := 0
	for i := 0; i < mt.NumIn(); i++ {
		pt := mt.In(i)
		if pt == ctxType {
			args = append(args, reflect.ValueOf(context.Background()))
			continue
		}
		v := reflect.New(pt)
		if j < len(raw) && string(raw[j]) != "null" {
			if err := json.Unmarshal(raw[j], v.Interface()); err != nil {
				writeJSON(w, map[string]string{"error": fmt.Sprintf("shim: arg %d of %s: %v", j, name, err)})
				return
			}
		}
		j++
		args = append(args, v.Elem())
	}
	var outs []reflect.Value
	func() {
		defer func() {
			if p := recover(); p != nil {
				outs = nil
				writeJSON(w, map[string]string{"error": fmt.Sprintf("shim: panic in %s: %v", name, p)})
			}
		}()
		outs = m.Call(args)
	}()
	if outs == nil && mt.NumOut() > 0 {
		return // the panic already answered
	}
	resp := map[string]any{"result": nil}
	for _, o := range outs {
		if o.Type() == errType {
			if !o.IsNil() {
				resp = map[string]any{"error": o.Interface().(error).Error()}
				writeJSON(w, resp)
				return
			}
			continue
		}
		resp["result"] = o.Interface()
	}
	writeJSON(w, resp)
}
