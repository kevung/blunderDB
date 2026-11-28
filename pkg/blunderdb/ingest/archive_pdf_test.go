package ingest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
)

// onePagePDF is the smallest PDF Attach rewrites.
func onePagePDF() []byte {
	head := "%PDF-1.4\n"
	o1 := "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n"
	o2 := "2 0 obj\n<< /Type /Pages /Kids [] /Count 0 >>\nendobj\n"
	x := len(head) + len(o1) + len(o2)
	return []byte(fmt.Sprintf("%s%s%sxref\n0 3\n0000000000 65535 f \n%010d 00000 n \n%010d 00000 n \ntrailer\n<< /Size 3 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		head, o1, o2, len(head), len(head)+len(o1), x))
}

// A PDF printed from a page that embeds frames reads back as that page.
func TestReadArchive_PDFCarriesThePagePayload(t *testing.T) {
	frames, _, err := arkqr.Frames([][]byte{[]byte("a coded match"), []byte("another one")})
	if err != nil {
		t.Fatal(err)
	}
	page := []byte("<html><body>" + arkqr.EmbedFrames(frames) + "</body></html>")
	pdf, err := EmbedArchiveInPDF(onePagePDF(), page)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadArchive(pdf)
	if err != nil || len(got) != len(frames) {
		t.Fatalf("ReadArchive = %d frames, %v; want %d", len(got), err, len(frames))
	}
	for i := range got {
		if string(got[i].Bytes()) != string(frames[i].Bytes()) {
			t.Fatalf("frame %d differs", i)
		}
	}
}

func TestReadArchive_PDFWithoutPayload(t *testing.T) {
	if _, err := ReadArchive(onePagePDF()); !errors.Is(err, ErrNoPDFPayload) || !errors.Is(err, arkqr.ErrNoPayload) {
		t.Fatalf("err = %v, want ErrNoPDFPayload", err)
	}
	// A page without frames prints to a PDF left as it is.
	pdf, err := EmbedArchiveInPDF(onePagePDF(), []byte("<html></html>"))
	if err != nil || string(pdf) != string(onePagePDF()) {
		t.Fatalf("EmbedArchiveInPDF changed a PDF with nothing to carry: %v", err)
	}
}

// The one question every document entry asks: a file blunderDB produced,
// HTML or PDF, codes included.
func TestIsArchiveFile(t *testing.T) {
	frames, _, err := arkqr.Frames([][]byte{[]byte("a coded match")})
	if err != nil {
		t.Fatal(err)
	}
	page := []byte("<html><body>" + arkqr.EmbedFrames(frames) + "</body></html>")
	ours, err := EmbedArchiveInPDF(onePagePDF(), page)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	files := map[string][]byte{
		"sheet.html": page, "sheet.pdf": ours, "book.pdf": onePagePDF(),
		"notes.html": []byte("<html></html>"), "page.txt": page,
	}
	want := map[string]bool{"sheet.html": true, "sheet.pdf": true}
	for name, data := range files {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		if got := IsArchiveFile(p); got != want[name] {
			t.Errorf("IsArchiveFile(%s) = %v", name, got)
		}
	}
	if IsArchiveFile(filepath.Join(dir, "absent.pdf")) {
		t.Error("a missing file is not one")
	}
}
