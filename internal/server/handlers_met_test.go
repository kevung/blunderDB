package server

import (
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// latin1Table is the Rockwell table declared ISO-8859-1, its name holding an
// "é" stored as the single byte E9.
func latin1Table(t *testing.T) (utf8Text string, raw []byte) {
	t.Helper()
	b, err := os.ReadFile("../../pkg/blunderdb/engine/testdata/met/Rockwell-Kazaross.xml")
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Replace(string(b), `<?xml version = "1.0"?>`, `<?xml version="1.0" encoding="ISO-8859-1"?>`, 1)
	s = strings.Replace(s, "Rockwell/Kazaross 25 point MET", "Table é Latin1", 1)
	for _, r := range s {
		raw = append(raw, byte(r))
	}
	return s, raw
}

func TestMetImportDecodesLatin1Once(t *testing.T) {
	text, raw := latin1Table(t)
	for name, body := range map[string]map[string]any{
		"text":   {"source": text},
		"base64": {"source_base64": base64.StdEncoding.EncodeToString(raw)},
	} {
		t.Run(name, func(t *testing.T) {
			ts := newTestServer(t)
			var table domain.MatchEquityTable
			postDecode(t, ts, "/v1/met.import", body, &table)
			if table.Name != "Table é Latin1" {
				t.Errorf("name = %q; want %q", table.Name, "Table é Latin1")
			}
			var again domain.MatchEquityTable
			postDecode(t, ts, "/v1/met.import", body, &again)
			if !again.AlreadyHeld || again.ID != table.ID {
				t.Errorf("reimport = %+v; want the held table flagged already_held", again)
			}
		})
	}
}

func TestMetImportLatin1TextEdgeCases(t *testing.T) {
	ts := newTestServer(t)
	text, _ := latin1Table(t)

	resp := post(t, ts, "/v1/met.import", map[string]any{"source": strings.Replace(text, "Table é", "Table €", 1)})
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(b), "source_base64") {
		t.Errorf("char above U+00FF: status %d: %s; want 400 pointing at source_base64", resp.StatusCode, b)
	}

	var table domain.MatchEquityTable
	postDecode(t, ts, "/v1/met.import", map[string]any{"source": "\uFEFF" + text}, &table)
	if table.Name != "Table é Latin1" {
		t.Errorf("name with a BOM = %q; want %q", table.Name, "Table é Latin1")
	}
}

func TestMetImportRefusalsSayWhatToDo(t *testing.T) {
	ts := newTestServer(t)
	text, _ := latin1Table(t)
	for name, c := range map[string]struct {
		body map[string]any
		want string
	}{
		"windows-1252": {map[string]any{"source": strings.Replace(text, "ISO-8859-1", "windows-1252", 1)}, "windows-1252"},
		"undeclared":   {map[string]any{"source_base64": base64.StdEncoding.EncodeToString([]byte("<?xml version=\"1.0\"?><met><info><name>\xe9</name>"))}, "ISO-8859-1"},
		"utf16":        {map[string]any{"source_base64": base64.StdEncoding.EncodeToString([]byte{0xFF, 0xFE, '<', 0})}, "UTF-16"},
	} {
		t.Run(name, func(t *testing.T) {
			resp := post(t, ts, "/v1/met.import", c.body)
			defer resp.Body.Close()
			b, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(b), c.want) || strings.Contains(string(b), "not an explicit gnubg") {
				t.Errorf("status %d: %s; want 400 naming %s", resp.StatusCode, b, c.want)
			}
		})
	}
}
