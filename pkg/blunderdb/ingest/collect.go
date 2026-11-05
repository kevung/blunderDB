package ingest

import (
	"archive/tar"
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// CollectFiles lists the importable files under root, in lexical order: the
// order the pipeline writes in, so match ids do not depend on the file system.
// Without recursive only root's own files are listed. The CLI's folder import
// and the server's batch import both go through it.
func CollectFiles(root string, recursive bool) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && IsImportable(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ErrArchiveTooLarge reports an archive that unpacks past its byte budget.
var ErrArchiveTooLarge = errors.New("archive unpacks to more than the allowed size")

// MaxArchiveEntries bounds the entries an archive may hold, importable or not:
// the byte budget does not stop a million empty entries from exhausting the
// inodes of the spool.
const MaxArchiveEntries = 500_000

// ErrArchiveTooManyEntries reports an archive holding more than MaxArchiveEntries.
var ErrArchiveTooManyEntries = errors.New("archive holds too many entries")

// ExtractArchive unpacks the importable files of a .zip or .tar archive into
// dir and returns how many it wrote. Everything else in the archive — other
// extensions, links, devices — is skipped, and an entry whose name leaves dir
// is refused. The budget counts bytes actually written, never the sizes the
// archive declares, which a hostile archive chooses. Files land in archive
// order: each entry gets its own numbered directory, so CollectFiles reads
// them back in the order the archive held them. An archive that goes over the
// budget, or is refused for any other reason, leaves nothing behind in dir.
func ExtractArchive(archive, dir string, maxBytes int64) (n int, err error) {
	defer func() {
		if err != nil {
			os.RemoveAll(dir)
			n = 0
		}
	}()
	switch strings.ToLower(filepath.Ext(archive)) {
	case ".zip":
		return extractZip(archive, dir, maxBytes)
	case ".tar":
		return extractTar(archive, dir, maxBytes)
	}
	return 0, fmt.Errorf("unsupported archive type %q (want .zip or .tar)", filepath.Ext(archive))
}

// ArchiveUnpackedSize says what archive announces it unpacks to, without
// unpacking it: the logical size of each entry (a sparse tar entry counts its
// holes), summed without overflow. A sum that cannot be held is a refusal. It
// is a first line only — the archive chooses what it announces, and
// ExtractArchive's count of the bytes it writes is the guard.
func ArchiveUnpackedSize(archive string) (int64, error) {
	var total uint64
	add := func(n uint64) error {
		if n > math.MaxInt64 || total > math.MaxInt64-n {
			return ErrArchiveTooLarge
		}
		total += n
		return nil
	}
	if strings.ToLower(filepath.Ext(archive)) == ".zip" {
		zr, err := zip.OpenReader(archive)
		if err != nil {
			return 0, err
		}
		defer zr.Close()
		for _, f := range zr.File {
			if err := add(f.UncompressedSize64); err != nil {
				return 0, err
			}
		}
		return int64(total), nil
	}
	f, err := os.Open(archive)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	tr := tar.NewReader(f)
	for n := 0; ; n++ {
		h, err := tr.Next()
		if err == io.EOF {
			return int64(total), nil
		}
		if err != nil {
			return 0, err
		}
		if n >= MaxArchiveEntries {
			return 0, ErrArchiveTooManyEntries
		}
		if h.Typeflag == tar.TypeReg && h.Size > 0 {
			if err := add(uint64(h.Size)); err != nil {
				return 0, err
			}
		}
	}
}

// unpacker writes entries under dir within a byte budget.
type unpacker struct {
	dir    string
	left   int64
	count  int
	nextID int
	seen   int
}

// entry counts one archive entry against MaxArchiveEntries.
func (u *unpacker) entry() error {
	u.seen++
	if u.seen > MaxArchiveEntries {
		return ErrArchiveTooManyEntries
	}
	return nil
}

func (u *unpacker) put(name string, r io.Reader) error {
	name = filepath.FromSlash(name)
	if !filepath.IsLocal(name) {
		return fmt.Errorf("archive entry %q leaves the archive", name)
	}
	if !IsImportable(name) {
		return nil
	}
	u.nextID++
	dst := filepath.Join(u.dir, fmt.Sprintf("%08d", u.nextID), filepath.Base(name))
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(r, u.left+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if n > u.left {
		return ErrArchiveTooLarge
	}
	u.left -= n
	u.count++
	return nil
}

func extractZip(archive, dir string, maxBytes int64) (int, error) {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return 0, err
	}
	defer zr.Close()
	u := &unpacker{dir: dir, left: maxBytes}
	for _, f := range zr.File {
		if err := u.entry(); err != nil {
			return u.count, err
		}
		if !f.Mode().IsRegular() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return u.count, err
		}
		err = u.put(f.Name, rc)
		rc.Close()
		if err != nil {
			return u.count, err
		}
	}
	return u.count, nil
}

func extractTar(archive, dir string, maxBytes int64) (int, error) {
	f, err := os.Open(archive)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	u := &unpacker{dir: dir, left: maxBytes}
	tr := tar.NewReader(f)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return u.count, nil
		}
		if err != nil {
			return u.count, err
		}
		if err := u.entry(); err != nil {
			return u.count, err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		if err := u.put(h.Name, tr); err != nil {
			return u.count, err
		}
	}
}
