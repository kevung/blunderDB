package gui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// resumableDownload streams url into dest atomically (via dest+".part"),
// verifying the whole file against expectedSHA256 (hex) before the final
// rename. It is pure Go with no Wails dependency so it can be unit-tested
// against an httptest server.
//
// Interrupted downloads RESUME: the .part file is kept on cancellation and
// failure, its prefix is re-hashed on the next attempt, and the remainder is
// requested with an HTTP Range header (GitHub release assets support ranges).
// Only a checksum mismatch discards the partial file — it is then corrupt and
// worthless.
//
// progress is called at most every 200ms with (received, total) in bytes,
// where received includes any resumed prefix.
func resumableDownload(ctx context.Context, url, dest, expectedSHA256 string, expectedBytes int64, progress func(received, total int64)) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp := dest + ".part"

	// Re-hash whatever a previous attempt left behind; the hash must cover
	// the whole file, so the prefix is read through the hasher first.
	h := sha256.New()
	var existing int64
	if fi, err := os.Stat(tmp); err == nil && fi.Size() > 0 {
		pf, err := os.Open(tmp)
		if err == nil {
			n, cerr := io.Copy(h, pf)
			pf.Close()
			if cerr == nil && n == fi.Size() {
				existing = n
			}
		}
		if existing == 0 {
			// Unreadable partial: start over.
			h = sha256.New()
			os.Remove(tmp)
		}
	}

	verifyAndInstall := func() error {
		if got := hex.EncodeToString(h.Sum(nil)); got != expectedSHA256 {
			os.Remove(tmp) // corrupt — resuming from it would never converge
			return fmt.Errorf("checksum mismatch: got %s", got)
		}
		return os.Rename(tmp, dest)
	}

	// A finished .part that missed its rename (crash between verify and
	// rename, or a 416 below) still installs without any network traffic.
	if expectedBytes > 0 && existing == expectedBytes {
		return verifyAndInstall()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if existing > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existing))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var f *os.File
	switch resp.StatusCode {
	case http.StatusOK:
		// Server ignored the range: restart from scratch.
		existing = 0
		h = sha256.New()
		f, err = os.Create(tmp)
	case http.StatusPartialContent:
		f, err = os.OpenFile(tmp, os.O_WRONLY|os.O_APPEND, 0o644)
	case http.StatusRequestedRangeNotSatisfiable:
		// The partial file already spans the asset: verify what we have.
		return verifyAndInstall()
	default:
		return fmt.Errorf("download failed: HTTP %s", resp.Status)
	}
	if err != nil {
		return err
	}

	total := expectedBytes
	if resp.ContentLength > 0 {
		total = existing + resp.ContentLength
	}

	received := existing
	lastEmit := time.Time{}
	buf := make([]byte, 1<<20)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				return werr // .part kept: next attempt resumes
			}
			h.Write(buf[:n])
			received += int64(n)
			if progress != nil && time.Since(lastEmit) > 200*time.Millisecond {
				lastEmit = time.Now()
				progress(received, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			return rerr // cancellation or network error: .part kept for resume
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	return verifyAndInstall()
}
