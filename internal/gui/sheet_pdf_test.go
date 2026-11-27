package gui

import "testing"

// A sheet goes only to an absolute .pdf path, the one the save dialog gave.
func TestWriteSheetPDF_RefusesAPathThatIsNotAPDF(t *testing.T) {
	a := &App{}
	for _, p := range []string{"sheet.pdf", "/tmp/sheet.html", "/tmp/sheet"} {
		if err := a.WriteSheetPDF(p, "<html></html>"); err == nil {
			t.Errorf("WriteSheetPDF(%q) accepted", p)
		}
	}
}
