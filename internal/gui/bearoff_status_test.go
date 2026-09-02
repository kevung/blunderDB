package gui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// isolateBearoffDataDir points the download directory at a throwaway
// directory and restores the real one on cleanup, so a test run never sees
// (or races against) whatever the developer's machine has already downloaded.
func isolateBearoffDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	race.SetDataDir(dir)
	t.Cleanup(func() { race.SetDataDir("") })
	return dir
}

func TestBearoffStatusNothingDownloaded(t *testing.T) {
	isolateBearoffDataDir(t)
	a := NewApp(nil)

	st := a.BearoffStatus()

	if st.Downloaded {
		t.Errorf("Downloaded = true, want false: %+v", st)
	}
	if st.Downloading {
		t.Errorf("Downloading = true, want false (no download in flight): %+v", st)
	}
	if st.SizeBytes != 0 {
		t.Errorf("SizeBytes = %d, want 0: %+v", st.SizeBytes, st)
	}
	if st.PartialBytes != 0 {
		t.Errorf("PartialBytes = %d, want 0 (no .part file): %+v", st.PartialBytes, st)
	}
	if st.ExpectedBytes != bearoffExpectedBytes {
		t.Errorf("ExpectedBytes = %d, want %d", st.ExpectedBytes, bearoffExpectedBytes)
	}
	if st.Path != race.DownloadedPath() {
		t.Errorf("Path = %q, want %q", st.Path, race.DownloadedPath())
	}
	// With nothing downloaded and no external path, Resolve() falls back to
	// the embedded TS-06-06 source, which is always present.
	if st.ActiveOrigin == "" {
		t.Errorf("ActiveOrigin should always name a source, got empty: %+v", st)
	}
}

func TestBearoffStatusDownloadedFilePresent(t *testing.T) {
	dir := isolateBearoffDataDir(t)
	a := NewApp(nil)

	target := filepath.Join(dir, race.DownloadedFileName)
	content := []byte("not a real bearoff database, just sized content")
	if err := os.WriteFile(target, content, 0644); err != nil {
		t.Fatalf("write fake downloaded db: %v", err)
	}

	st := a.BearoffStatus()

	if !st.Downloaded {
		t.Errorf("Downloaded = false, want true: %+v", st)
	}
	if st.SizeBytes != int64(len(content)) {
		t.Errorf("SizeBytes = %d, want %d", st.SizeBytes, len(content))
	}
	if st.PartialBytes != 0 {
		t.Errorf("PartialBytes = %d, want 0 (no .part file present): %+v", st.PartialBytes, st)
	}
}

func TestBearoffStatusPartialDownload(t *testing.T) {
	dir := isolateBearoffDataDir(t)
	a := NewApp(nil)

	partial := filepath.Join(dir, race.DownloadedFileName+".part")
	content := []byte("partial content, download was interrupted")
	if err := os.WriteFile(partial, content, 0644); err != nil {
		t.Fatalf("write fake partial db: %v", err)
	}

	st := a.BearoffStatus()

	if st.Downloaded {
		t.Errorf("Downloaded = true, want false: a .part file is not a completed download: %+v", st)
	}
	if st.PartialBytes != int64(len(content)) {
		t.Errorf("PartialBytes = %d, want %d", st.PartialBytes, len(content))
	}
}

func TestBearoffStatusExternalPath(t *testing.T) {
	isolateBearoffDataDir(t)
	a := NewApp(nil)

	race.SetExternalPath("/some/external/gnubg_ts0.bd")
	t.Cleanup(func() { race.SetExternalPath("") })

	// BearoffStatus itself doesn't read config.BearoffTSPath (the GUI's
	// external-path setting is applied to the race engine separately, from
	// config.SaveBearoffTSPath / main.go's startup); it reports the engine's
	// currently resolved source, so this only exercises that the struct
	// reflects race.Resolve() consistently while an external path is set.
	st := a.BearoffStatus()
	if st.ActiveOrigin == "" {
		t.Errorf("ActiveOrigin should name a source even with an (invalid) external path set: %+v", st)
	}
}
