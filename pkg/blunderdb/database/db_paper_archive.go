package database

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// PaperArchiveExport is what ExportPaperArchive wrote.
type PaperArchiveExport struct {
	Matches int `json:"matches"`
	// Codes counts the QR codes, parity codes included.
	Codes int `json:"codes"`
	// Bytes is the size of the coded matches, framing left out.
	Bytes   int                `json:"bytes"`
	Skipped []PaperArchiveSkip `json:"skipped,omitempty"`
}

// PaperArchiveSkip is a match an archive could not hold, and why.
type PaperArchiveSkip struct {
	MatchID int64  `json:"matchId"`
	Reason  string `json:"reason"`
}

// ExportPaperArchive writes matches as a printable paper archive (ADR-0119):
// an HTML page of A4 sheets of QR codes, one match a code when it fits, a
// parity code every five, the coded matches embedded as well. No ids means
// the whole library. origin, when set, is the producer's watermark line,
// printed at the foot of each sheet.
func (d *Database) ExportPaperArchive(matchIDs []int64, outPath, origin string) (*PaperArchiveExport, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	ctx := context.Background()
	if len(matchIDs) == 0 {
		for m, err := range d.store.Matches().List(ctx, "", storage.MatchListOpts{}) {
			if err != nil {
				return nil, err
			}
			matchIDs = append(matchIDs, m.ID)
		}
	}
	res := &PaperArchiveExport{}
	var payloads [][]byte
	var labels []string
	for _, id := range matchIDs {
		am, label, err := archivedMatch(ctx, d.store, id)
		if err == nil {
			var data []byte
			if data, err = arkqr.Encode(am); err == nil {
				payloads = append(payloads, data)
				labels = append(labels, label)
				res.Bytes += len(data)
				continue
			}
		}
		if errors.Is(err, storage.ErrNotFound) {
			return nil, fmt.Errorf("match %d: %w", id, err)
		}
		res.Skipped = append(res.Skipped, PaperArchiveSkip{MatchID: id, Reason: err.Error()})
	}
	if len(payloads) == 0 {
		return nil, fmt.Errorf("no match to archive: %w", storage.ErrInvalid)
	}
	frames, owner := arkqr.Frames(payloads)
	codes := make([]arkqr.Code, len(frames))
	for i, f := range frames {
		codes[i] = arkqr.Code{Frame: f, Label: codeLabel(f, owner[i], labels)}
	}
	title := fmt.Sprintf("blunderDB · archive %08X", frames[0].Archive)
	page, err := arkqr.Page(title, origin, codes)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(outPath, page, 0o644); err != nil {
		return nil, err
	}
	res.Matches, res.Codes = len(payloads), len(frames)
	return res, nil
}

func codeLabel(f arkqr.Frame, owner int, labels []string) string {
	if f.Parity {
		lo := f.Index * arkqr.ParityGroup
		return fmt.Sprintf("Parité %d–%d", lo+1, min(f.Total, lo+arkqr.ParityGroup))
	}
	l := fmt.Sprintf("%d/%d · %s", f.Index+1, f.Total, labels[owner])
	if f.Body[1] > 1 {
		l += fmt.Sprintf(" · %d/%d", f.Body[0]+1, f.Body[1])
	}
	return l
}

// archivedMatch reads a stored match into what the archive codes, and the
// line printed under its code. It reads the match the way a .mat export and
// the reopening of a draft do, then puts back what a .mat cannot carry.
func archivedMatch(ctx context.Context, s storage.Storage, id int64) (arkqr.Match, string, error) {
	m, games, moves, err := ingest.ReadMatchForMAT(ctx, s, "", id)
	if err != nil {
		return arkqr.Match{}, "", err
	}
	doc, err := transcript.FromMAT(ingest.RenderMAT(m, games, moves))
	if err != nil {
		return arkqr.Match{}, "", err
	}
	transcription.RestoreTimecodes(&doc, games, moves)
	// Only a URL leaves the library, as in every export.
	if v := issuance.CarriedVideoSource(m.VideoSource); v != nil {
		doc.Header.VideoSource = *v
	}
	am := arkqr.Match{Doc: doc, Comment: m.Comment}

	ann := transcript.Replay(doc, 0)
	var posIDs []int64
	byAction := make([]*domain.Move, len(doc.Actions))
	for i, info := range ann.Actions {
		if info.MoveNumber < 0 || info.GameIndex < 0 || info.GameIndex >= len(games) {
			continue
		}
		for _, mv := range moves[games[info.GameIndex].ID] {
			if mv.MoveNumber == info.MoveNumber {
				byAction[i] = mv
				posIDs = append(posIDs, mv.PositionID)
				break
			}
		}
	}
	comments, err := s.Comments().ByPositions(ctx, "", posIDs)
	if err != nil {
		return arkqr.Match{}, "", err
	}
	positions, err := s.Positions().LoadByIDs(ctx, "", posIDs)
	if err != nil {
		return arkqr.Match{}, "", err
	}
	flagged := map[int64]bool{}
	for _, p := range positions {
		flagged[p.ID] = p.Flagged
	}
	notes := make([]arkqr.Note, len(doc.Actions))
	hasNotes := false
	seen := map[int64]bool{}
	for i, mv := range byAction {
		if mv == nil {
			continue
		}
		a := &doc.Actions[i]
		a.DecisionMS, a.CubeDecisionMS = mv.DecisionMS, mv.CubeDecisionMS
		// A position met twice in the match carries its notes once.
		if seen[mv.PositionID] {
			continue
		}
		seen[mv.PositionID] = true
		n := arkqr.Note{Flagged: flagged[mv.PositionID]}
		for _, c := range comments[mv.PositionID] {
			n.Comments = append(n.Comments, c.Text)
		}
		if len(n.Comments) > 0 || n.Flagged {
			notes[i], hasNotes = n, true
		}
	}
	if hasNotes {
		am.Notes = notes
	}
	am.Doc = doc
	return am, matchLabel(m), nil
}

func matchLabel(m *domain.Match) string {
	parts := []string{m.Player1Name + " – " + m.Player2Name}
	if !m.MatchDate.IsZero() {
		parts = append(parts, m.MatchDate.Format("2006-01-02"))
	}
	if m.MatchLength > 0 {
		parts = append(parts, fmt.Sprintf("%d pts", m.MatchLength))
	} else {
		parts = append(parts, "money")
	}
	return strings.Join(parts, " · ")
}

// PaperArchiveImport is what ImportPaperArchive read and wrote.
type PaperArchiveImport struct {
	// Codes counts the archive codes read, Repaired those rebuilt from parity.
	Codes    int `json:"codes"`
	Repaired int `json:"repaired"`
	// Imported lists the new matches; Duplicates counts those the library
	// already held, from this archive or from another format.
	Imported   []int64 `json:"imported"`
	Duplicates int     `json:"duplicates"`
	// Missing lists the ranks (from 1) of the codes neither read nor rebuilt.
	Missing []int `json:"missing,omitempty"`
	// Failed lists the matches that could not be read or written, and why.
	Failed []string `json:"failed,omitempty"`
}

// ImportPaperArchive reads paper archives — the exported HTML page, whose
// embedded matches need no code reading, or scans and photographs (PNG,
// JPEG) of its sheets, any number and in any order — and imports their
// matches through the import path of every format: an import batch when the
// caller opened one, and the duplicate check. Nothing about the archive or
// its producer is recorded (ADR-0007).
func (d *Database) ImportPaperArchive(paths []string) (*PaperArchiveImport, error) {
	var frames []arkqr.Frame
	for _, p := range paths {
		fs, err := readArchiveFile(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		frames = append(frames, fs...)
	}
	res := &PaperArchiveImport{Codes: len(frames)}
	if len(frames) == 0 {
		return res, fmt.Errorf("no archive code found: %w", storage.ErrInvalid)
	}
	asm := arkqr.Assemble(frames)
	res.Repaired = asm.Repaired
	for _, ranks := range asm.Missing {
		for _, r := range ranks {
			res.Missing = append(res.Missing, r+1)
		}
	}

	ctx, done := d.beginCancellableImport()
	defer done()
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, data := range asm.Matches {
		am, err := arkqr.Decode(data)
		if err != nil {
			res.Failed = append(res.Failed, fmt.Sprintf("match %d: %v", i+1, err))
			continue
		}
		graph, err := archiveGraph(am)
		if err != nil {
			res.Failed = append(res.Failed, fmt.Sprintf("match %d: %v", i+1, err))
			continue
		}
		id, enriched, err := d.writeImportedMatchEnriched(ctx, graph)
		switch {
		case errors.Is(err, ErrDuplicateMatch), err == nil && enriched:
			res.Duplicates++
		case err != nil:
			if ctx.Err() != nil {
				return res, err
			}
			res.Failed = append(res.Failed, fmt.Sprintf("match %d: %v", i+1, err))
		default:
			res.Imported = append(res.Imported, id)
		}
	}
	return res, nil
}

// IsPaperArchivePage reports whether a file is an exported archive page, so
// the ordinary import can hand it over.
func IsPaperArchivePage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".html" && ext != ".htm" {
		return false
	}
	data, err := os.ReadFile(path)
	return err == nil && arkqr.IsPage(data)
}

func readArchiveFile(path string) ([]arkqr.Frame, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm":
		return arkqr.ReadPage(data)
	default:
		return arkqr.ScanImage(bytes.NewReader(data))
	}
}

// archiveGraph turns an archived match into the graph every import writes:
// the transcription's own construction, with the hashes an import dedups on,
// and the notes put back on the positions they were read from.
func archiveGraph(am arkqr.Match) (*ingest.MatchGraph, error) {
	parts := transcript.Build(am.Doc)
	if len(parts.Games) == 0 {
		return nil, fmt.Errorf("the archived match holds no game: %w", storage.ErrInvalid)
	}
	graph := transcription.MatchGraph(parts)
	// An imported match takes the seats an import gives it, not a draft's.
	graph.KeepSeats = false
	graph.Match.MatchHash, graph.Match.CanonicalHash = transcription.MatchHashes(parts)
	graph.Match.Comment = am.Comment
	if v := am.Doc.Header.VideoSource; v != "" {
		graph.Match.VideoSource = &v
	}
	graph.CommentOrigin = domain.CommentOriginUnknown
	if am.Notes == nil {
		return graph, nil
	}
	ann := transcript.Replay(am.Doc, 0)
	for i, info := range ann.Actions {
		if i >= len(am.Notes) || info.GameIndex < 0 || info.GameIndex >= len(graph.Games) {
			continue
		}
		n := am.Notes[i]
		g := &graph.Games[info.GameIndex]
		for k := range g.Moves {
			mg := &g.Moves[k]
			if mg.Move.MoveNumber != info.MoveNumber {
				continue
			}
			mg.Comments = append(mg.Comments, n.Comments...)
			if n.Flagged && mg.Position != nil {
				mg.Position.Flagged = true
			}
			break
		}
	}
	return graph, nil
}
