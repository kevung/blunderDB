package gui

import (
	"bytes"
	"compress/gzip"
	"testing"
	"testing/fstest"
)

func TestHelpBundleInflatesEmbeddedFile(t *testing.T) {
	var b bytes.Buffer
	zw := gzip.NewWriter(&b)
	_, _ = zw.Write([]byte(`{"manual":"<p>x</p>"}`))
	_ = zw.Close()
	a := &App{assets: fstest.MapFS{"frontend/dist/help/en.bin": {Data: b.Bytes()}}}

	got, err := a.HelpBundle("en")
	if err != nil || got != `{"manual":"<p>x</p>"}` {
		t.Fatalf("HelpBundle = %q, %v", got, err)
	}
	if _, err := a.HelpBundle("fr"); err == nil {
		t.Error("a missing bundle must fail")
	}
	if _, err := a.HelpBundle("../x"); err == nil {
		t.Error("a path-like language must be refused")
	}
}
