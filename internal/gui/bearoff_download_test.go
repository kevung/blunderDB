package gui

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// rangeServer serves blob with HTTP Range support, like a GitHub release
// asset. failAfter > 0 drops the connection after that many bytes of the
// CURRENT response body, simulating a network interruption.
func rangeServer(t *testing.T, blob []byte, failAfter *int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := int64(0)
		if rg := r.Header.Get("Range"); rg != "" {
			var err error
			start, err = strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(rg, "bytes="), "-"), 10, 64)
			if err != nil || start > int64(len(blob)) {
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(blob)-1, len(blob)))
			w.Header().Set("Content-Length", strconv.Itoa(len(blob)-int(start)))
			w.WriteHeader(http.StatusPartialContent)
		} else {
			w.Header().Set("Content-Length", strconv.Itoa(len(blob)))
		}
		body := blob[start:]
		if failAfter != nil && *failAfter > 0 && *failAfter < int64(len(body)) {
			if _, err := w.Write(body[:*failAfter]); err != nil {
				t.Fatal(err)
			}
			// Abort the connection mid-body.
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
			return
		}
		if _, err := w.Write(body); err != nil {
			t.Fatal(err)
		}
	}))
}

func testBlob(n int) ([]byte, string) {
	blob := bytes.Repeat([]byte("blunderDB bearoff!"), n)
	sum := sha256.Sum256(blob)
	return blob, hex.EncodeToString(sum[:])
}

func TestResumableDownload_FullThenVerify(t *testing.T) {
	blob, sha := testBlob(10000)
	srv := rangeServer(t, blob, nil)
	defer srv.Close()
	dest := filepath.Join(t.TempDir(), "db.bd")

	if err := resumableDownload(context.Background(), srv.URL, dest, sha, int64(len(blob)), nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || !bytes.Equal(got, blob) {
		t.Fatalf("downloaded file differs (err=%v)", err)
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatal(".part must be gone after a successful install")
	}
}

func TestResumableDownload_ResumesAfterInterruption(t *testing.T) {
	blob, sha := testBlob(20000)
	failAfter := int64(50 << 10) // drop the connection after 50 KiB
	srv := rangeServer(t, blob, &failAfter)
	defer srv.Close()
	dest := filepath.Join(t.TempDir(), "db.bd")

	// First attempt: interrupted mid-body; the .part must survive.
	if err := resumableDownload(context.Background(), srv.URL, dest, sha, int64(len(blob)), nil); err == nil {
		t.Fatal("interrupted download should fail")
	}
	fi, err := os.Stat(dest + ".part")
	if err != nil {
		t.Fatal(".part must be kept after an interruption:", err)
	}
	if fi.Size() == 0 || fi.Size() >= int64(len(blob)) {
		t.Fatalf("unexpected partial size %d", fi.Size())
	}

	// Second attempt: the server now serves fully; the request must be a
	// Range request (verified by the resumed offset landing exactly).
	failAfter = 0
	var sawTotal int64
	if err := resumableDownload(context.Background(), srv.URL, dest, sha, int64(len(blob)),
		func(_, total int64) { sawTotal = total }); err != nil {
		t.Fatal("resume failed:", err)
	}
	if sawTotal != int64(len(blob)) {
		t.Fatalf("progress total = %d, want %d", sawTotal, len(blob))
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, blob) {
		t.Fatal("resumed file differs from the source blob")
	}
}

func TestResumableDownload_CancelKeepsPartThenResumes(t *testing.T) {
	blob, sha := testBlob(20000)
	srv := rangeServer(t, blob, nil)
	defer srv.Close()
	dest := filepath.Join(t.TempDir(), "db.bd")

	// Cancel via context after the first progress callback.
	ctx, cancel := context.WithCancel(context.Background())
	_ = resumableDownload(ctx, srv.URL, dest, sha, int64(len(blob)), func(received, _ int64) {
		if received > 0 {
			cancel()
		}
	})
	// Depending on read timing the download may have completed before the
	// cancellation took effect; both outcomes are legal, but a partial file
	// must never be lost.
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		if _, err := os.Stat(dest + ".part"); err != nil {
			t.Fatal("cancelled download must keep its .part:", err)
		}
		if err := resumableDownload(context.Background(), srv.URL, dest, sha, int64(len(blob)), nil); err != nil {
			t.Fatal("resume after cancel failed:", err)
		}
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, blob) {
		t.Fatal("file differs after cancel + resume")
	}
}

func TestResumableDownload_CompletePartInstallsOffline(t *testing.T) {
	blob, sha := testBlob(5000)
	dest := filepath.Join(t.TempDir(), "db.bd")
	if err := os.WriteFile(dest+".part", blob, 0o644); err != nil {
		t.Fatal(err)
	}
	// No server at all: a complete .part must verify and install offline.
	if err := resumableDownload(context.Background(), "http://127.0.0.1:1/nope", dest, sha, int64(len(blob)), nil); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, blob) {
		t.Fatal("offline install differs")
	}
}

func TestResumableDownload_ChecksumMismatchDiscardsPart(t *testing.T) {
	blob, _ := testBlob(5000)
	srv := rangeServer(t, blob, nil)
	defer srv.Close()
	dest := filepath.Join(t.TempDir(), "db.bd")

	err := resumableDownload(context.Background(), srv.URL, dest, strings.Repeat("0", 64), int64(len(blob)), nil)
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("want checksum error, got %v", err)
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatal("a corrupt .part must be discarded, not resumed forever")
	}
}
