package gui

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"regexp"
)

var helpLangRe = regexp.MustCompile(`^[a-z]{2,3}$`)

// HelpBundle returns the in-app help of one language as JSON, inflated from the
// gzip file embedded with the frontend. The WebView calls it when it has no
// DecompressionStream (WebKitGTK < 2.40, WKWebView < 16.4); newer engines
// inflate the same file themselves.
func (a *App) HelpBundle(lang string) (string, error) {
	if !helpLangRe.MatchString(lang) {
		return "", fmt.Errorf("invalid help language %q", lang)
	}
	if a.assets == nil {
		return "", fmt.Errorf("no embedded frontend")
	}
	packed, err := fs.ReadFile(a.assets, "frontend/dist/help/"+lang+".bin")
	if err != nil {
		return "", err
	}
	zr, err := gzip.NewReader(bytes.NewReader(packed))
	if err != nil {
		return "", err
	}
	defer zr.Close()
	plain, err := io.ReadAll(zr)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
