package gui

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// The interface prints its sheets — a match sheet, a position list, a paper
// archive — to a PDF it saves itself, rendered by the system web engine off
// screen; the PDF carries the page's reimport payload as an attached file,
// so it imports back like the page. The CLI and the server keep the HTML.

// SheetPDFAvailable reports whether this build renders sheets to PDF; when
// it does not, the interface saves the HTML page and opens it in the browser.
func (a *App) SheetPDFAvailable() bool { return sheetPDFAvailable() }

// WriteSheetPDF renders page to an A4 PDF at path, a .pdf the save dialog
// returned.
func (a *App) WriteSheetPDF(path, page string) error {
	if err := checkPDFPath(path); err != nil {
		return err
	}
	return writeSheetPDF(a.renderCtx(), path, []byte(page))
}

// ExportPaperArchivePDF is ExportPaperArchive printed to a PDF at path.
func (a *App) ExportPaperArchivePDF(matchIDs []int64, path string) (*database.PaperArchiveExport, error) {
	if err := checkPDFPath(path); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newGUIError(CodeInvalid, "no database open")
	}
	page, res, err := database.PaperArchivePage(a.db, matchIDs, "")
	if err != nil {
		return nil, err
	}
	if err := writeSheetPDF(a.renderCtx(), path, page); err != nil {
		return nil, err
	}
	return res, nil
}

func (a *App) renderCtx() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

func checkPDFPath(path string) error {
	if !strings.EqualFold(filepath.Ext(path), ".pdf") {
		return newGUIError(CodeInvalid, "not a PDF path: "+path)
	}
	if !filepath.IsAbs(path) {
		return newGUIError(CodeInvalid, "PDF path must be absolute: "+path)
	}
	return nil
}

func writeSheetPDF(ctx context.Context, path string, page []byte) error {
	pdf, err := renderSheetPDF(ctx, page)
	if err != nil {
		return newGUIError(CodeInternal, err.Error())
	}
	if pdf, err = ingest.EmbedArchiveInPDF(pdf, page); err != nil {
		return newGUIError(CodeInternal, err.Error())
	}
	return os.WriteFile(path, pdf, 0o644)
}

// IsBlunderDBArchive reports whether path is a file blunderDB produced that
// carries an archive (ingest.IsArchiveFile): the drop, Ctrl-I and the
// document import send it to the paper archive import.
func (a *App) IsBlunderDBArchive(path string) bool { return ingest.IsArchiveFile(path) }
