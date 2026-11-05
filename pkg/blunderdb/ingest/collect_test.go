package ingest

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func writeTar(t *testing.T, names ...string) string {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, n := range names {
		if err := tw.WriteHeader(&tar.Header{Name: n, Mode: 0o644, Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "a.tar")
	if err := os.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExtractArchive_RefusesTraversal(t *testing.T) {
	if _, err := ExtractArchive(writeTar(t, "../evil.xg"), t.TempDir(), 1<<20); err == nil {
		t.Fatal("an entry leaving the archive was accepted")
	}
}

func TestExtractArchive_CapsEntries(t *testing.T) {
	names := make([]string, MaxArchiveEntries+1)
	for i := range names {
		names[i] = "x.pdf" // skipped entries count too
	}
	_, err := ExtractArchive(writeTar(t, names...), t.TempDir(), 1<<20)
	if !errors.Is(err, ErrArchiveTooManyEntries) {
		t.Fatalf("err = %v, want ErrArchiveTooManyEntries", err)
	}
}

// rawTarBlock is a ustar header block, written by hand for the types the
// writer refuses to encode.
func rawTarBlock(name string, typeflag byte, size int64) []byte {
	b := make([]byte, 512)
	copy(b, name)
	copy(b[100:], "0000644\x00")
	copy(b[108:], "0000000\x00")
	copy(b[116:], "0000000\x00")
	copy(b[124:], fmt.Sprintf("%011o\x00", size))
	copy(b[136:], "00000000000\x00")
	copy(b[148:], "        ")
	b[156] = typeflag
	copy(b[257:], "ustar\x0000")
	sum := 0
	for _, c := range b {
		sum += int(c)
	}
	copy(b[148:], fmt.Sprintf("%06o\x00 ", sum))
	return b
}

// sparseTar builds a PAX 1.0 sparse tar whose one entry is a hole of realSize
// bytes: a few KiB on disk that the reader unpacks to realSize zeros.
func sparseTar(t *testing.T, name string, realSize int64) string {
	t.Helper()
	var buf bytes.Buffer
	smap := make([]byte, 512)
	copy(smap, "0\n")
	// archive/tar's writer drops GNU.sparse records, so the extended header is
	// written as an entry of its own, which the reader parses back.
	var recs bytes.Buffer
	for k, v := range map[string]string{
		"GNU.sparse.major": "1", "GNU.sparse.minor": "0",
		"GNU.sparse.name": name, "GNU.sparse.realsize": strconv.FormatInt(realSize, 10),
	} {
		rec := " " + k + "=" + v + "\n"
		l := len(rec) + 1
		for len(strconv.Itoa(l))+len(rec) != l {
			l = len(strconv.Itoa(l)) + len(rec)
		}
		recs.WriteString(strconv.Itoa(l) + rec)
	}
	buf.Write(rawTarBlock("PaxHeader/"+name, 'x', int64(recs.Len())))
	buf.Write(recs.Bytes())
	buf.Write(make([]byte, (512-recs.Len()%512)%512))
	buf.Write(rawTarBlock(name, '0', int64(len(smap))))
	buf.Write(smap)
	buf.Write(make([]byte, 1024))
	p := filepath.Join(t.TempDir(), "sparse.tar")
	if err := os.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// zipOf writes a zip of entries name -> declared uncompressed size, deflating
// real bytes of zeros when data is true.
func zipOf(t *testing.T, declared map[string]uint64, zeros map[string]int) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, n := range zeros {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(make([]byte, n)); err != nil {
			t.Fatal(err)
		}
	}
	for name, size := range declared {
		w, err := zw.CreateRaw(&zip.FileHeader{Name: name, Method: zip.Store, UncompressedSize64: size})
		if err != nil {
			t.Fatal(err)
		}
		_ = w
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "a.zip")
	if err := os.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func dirBytes(t *testing.T, dir string) int64 {
	t.Helper()
	var n int64
	_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			if fi, e := d.Info(); e == nil {
				n += fi.Size()
			}
		}
		return nil
	})
	return n
}

func TestArchiveUnpackedSize_CountsSparseEntriesAtTheirRealSize(t *testing.T) {
	const real = 64 << 20
	p := sparseTar(t, "hole.xg", real)
	if fi, _ := os.Stat(p); fi.Size() > 16<<10 {
		t.Fatalf("fixture is %d bytes, want a few KiB", fi.Size())
	}
	got, err := ArchiveUnpackedSize(p)
	if err != nil || got != real {
		t.Fatalf("ArchiveUnpackedSize = %d, %v; want %d", got, err, int64(real))
	}
}

func TestArchiveUnpackedSize_RefusesAnOverflowingZip(t *testing.T) {
	p := zipOf(t, map[string]uint64{"a.xg": 1 << 63, "b.xg": 1 << 63}, nil)
	if _, err := ArchiveUnpackedSize(p); !errors.Is(err, ErrArchiveTooLarge) {
		t.Fatalf("err = %v, want ErrArchiveTooLarge", err)
	}
}

// The write guard holds whatever the archive announces: the bytes written are
// counted, the refusal is clear, and what was written is erased.
func TestExtractArchive_StopsAtTheBudgetWhateverTheArchiveSays(t *testing.T) {
	const budget = 1 << 20
	cases := map[string]string{
		"sparse tar":      sparseTar(t, "hole.xg", 64<<20),
		"zip bomb":        zipOf(t, nil, map[string]int{"zeros.xg": 64 << 20}),
		"zip, lying size": zipOf(t, nil, map[string]int{"a.xg": 2 << 20}),
	}
	for name, archive := range cases {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "files")
			_, err := ExtractArchive(archive, dir, budget)
			if !errors.Is(err, ErrArchiveTooLarge) {
				t.Fatalf("err = %v, want ErrArchiveTooLarge", err)
			}
			if n := dirBytes(t, dir); n != 0 {
				t.Errorf("%d bytes left after the refusal", n)
			}
		})
	}
}
