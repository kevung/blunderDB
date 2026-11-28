package pdfattach

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"testing"
)

// minimalPDF builds a one-page PDF with a classic cross-reference table;
// catalogExtra goes inside the catalog dictionary.
func minimalPDF(catalogExtra string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R" + catalogExtra + " >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] >>",
	}
	offs := make([]int, len(objs))
	for i, o := range objs {
		offs[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	x := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, o := range offs {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R /ID [<00ff> <00ff>] >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, x)
	return b.Bytes()
}

func TestAttachThenFilesRoundTrip(t *testing.T) {
	src := minimalPDF("")
	payload := []byte("<script type=\"x\">\nAAAA\n</script>\n")
	out, err := Attach(src, "a (b).html", "text/html", payload)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out, src) {
		t.Fatal("the original bytes must be kept: Attach is an incremental update")
	}
	files := Files(out)
	if len(files) != 1 || !bytes.Equal(files[0], payload) {
		t.Fatalf("Files = %q, want the payload", files)
	}
	for _, want := range []string{"/Prev ", "/EmbeddedFiles", "/Subtype /text#2Fhtml", `(a \(b\).html)`, "/ID [<00ff> <00ff>]"} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("update lacks %q", want)
		}
	}
	if len(Files(src)) != 0 {
		t.Error("a PDF without attachment yields none")
	}
}

func TestFilesReadsAFlateStream(t *testing.T) {
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	_, _ = w.Write([]byte("hello"))
	w.Close()
	pdf := []byte(fmt.Sprintf("%%PDF-1.7\n7 0 obj\n<< /Type /EmbeddedFile /Filter /FlateDecode /Length 8 0 R >>\nstream\r\n%s\r\nendstream\nendobj\n", z.Bytes()))
	files := Files(pdf)
	if len(files) != 1 || string(files[0]) != "hello" {
		t.Fatalf("Files = %q", files)
	}
}

func TestAttachRefusesWhatItCannotRewrite(t *testing.T) {
	if _, err := Attach(minimalPDF(" /Names 9 0 R"), "x", "text/plain", nil); !errors.Is(err, ErrUnsupported) {
		t.Errorf("a catalog that names files: err = %v", err)
	}
	if _, err := Attach([]byte("%PDF-1.4\nnothing"), "x", "text/plain", nil); !errors.Is(err, ErrUnsupported) {
		t.Errorf("no startxref: err = %v", err)
	}
}

func TestAttachAfterACrossReferenceStream(t *testing.T) {
	var b bytes.Buffer
	b.WriteString("%PDF-1.5\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	x := b.Len()
	b.WriteString("5 0 obj\n<< /Type /XRef /Size 6 /Root 1 0 R /W [1 2 1] /Length 0 >>\nstream\n\nendstream\nendobj\n")
	fmt.Fprintf(&b, "startxref\n%d\n%%%%EOF\n", x)
	out, err := Attach(b.Bytes(), "f.txt", "text/plain", []byte("data"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("6 0 obj")) || !bytes.Contains(out, []byte(fmt.Sprintf("/Prev %d", x))) {
		t.Error("the new objects number from /Size and chain to the stream")
	}
	if f := Files(out); len(f) != 1 || string(f[0]) != "data" {
		t.Errorf("Files = %q", f)
	}
}
