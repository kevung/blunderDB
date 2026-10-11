package ingest

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A draft leaves with its video named, not located: a local file keeps only its name, a URL
// travels as it is, and the rest of the document is untouched.
func TestExportedTranscriptionKeepsOnlyTheVideoFileName(t *testing.T) {
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	docs := map[string]string{
		"local":   `{"format_version":3,"header":{"match_length":7,"video_source":"/home/kevin/Vidéos/finale.mp4"},"actions":[],"cursor":0}`,
		"windows": `{"format_version":3,"header":{"match_length":5,"video_source":"C:\\Users\\kevin\\demi.mkv"},"actions":[],"cursor":0}`,
		"url":     `{"format_version":3,"header":{"match_length":3,"video_source":"https://www.youtube.com/watch?v=abc"},"actions":[],"cursor":0}`,
	}
	for label, doc := range docs {
		if _, err := src.Transcriptions().Save(ctx, "", &storage.Transcription{FormatVersion: "3", Label: label, Document: doc}); err != nil {
			t.Fatal(err)
		}
	}

	var buf bytes.Buffer
	if err := (SQLiteExporter{S: src}).Export(ctx, "", &buf, WholeTenant(FormatSQLite)); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(buf.Bytes(), []byte("/home/kevin")) || bytes.Contains(buf.Bytes(), []byte(`C:\\Users`)) {
		t.Error("a local video path is in the exported file")
	}
	path := filepath.Join(t.TempDir(), "export.sqlite")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	dst, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()

	want := map[string]string{
		"local":   `"video_source":"finale.mp4"`,
		"windows": `"video_source":"demi.mkv"`,
		"url":     `"video_source":"https://www.youtube.com/watch?v=abc"`,
	}
	seen := 0
	for tr, err := range dst.Transcriptions().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		seen++
		if !strings.Contains(tr.Document, want[tr.Label]) || !strings.Contains(tr.Document, `"match_length"`) {
			t.Errorf("%s: exported document %s, want %s and the rest of the header", tr.Label, tr.Document, want[tr.Label])
		}
	}
	if seen != len(docs) {
		t.Fatalf("exported %d drafts, want %d", seen, len(docs))
	}
}
