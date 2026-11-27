//go:build !(linux && (dev || production))

package gui

import (
	"context"
	"errors"
)

// Only WebKitGTK renders sheets to PDF here; elsewhere the interface keeps
// the HTML page, opened in the browser, and this is never reached.
func sheetPDFAvailable() bool { return false }

func renderSheetPDF(context.Context, []byte) ([]byte, error) {
	return nil, errors.New("PDF rendering is not available in this build")
}
