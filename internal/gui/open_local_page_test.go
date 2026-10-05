package gui

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenLocalPage_Validation(t *testing.T) {
	dir := t.TempDir()
	page := filepath.Join(dir, "index.html")
	if err := os.WriteFile(page, []byte("<html></html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	txt := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(txt, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "dir.html")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.html")
	linked := runtime.GOOS != "windows" && os.Symlink(page, link) == nil

	var opened []string
	orig := openFile
	openFile = func(p string) error { opened = append(opened, p); return nil }
	t.Cleanup(func() { openFile = orig })

	a := &App{}
	cases := []struct {
		name string
		path string
		ok   bool
		skip bool
	}{
		{"html file", page, true, false},
		{"wrong extension", txt, false, false},
		{"missing", filepath.Join(dir, "absent.html"), false, false},
		{"directory", sub, false, false},
		{"relative", "index.html", false, false},
		{"symlink", link, false, !linked},
	}
	for _, c := range cases {
		if c.skip {
			continue
		}
		before := len(opened)
		err := a.OpenLocalPage(c.path)
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok=%v", c.name, err, c.ok)
		}
		if got := len(opened) - before; (got == 1) != c.ok {
			t.Errorf("%s: openFile called %d times", c.name, got)
		}
	}
}
