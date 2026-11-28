package ingest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
	"github.com/kevung/blunderdb/pkg/blunderdb/pdfattach"
	"github.com/kevung/blunderdb/pkg/blunderdb/progress"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// The paper archive (ADR-0119) on the storage contract, so every mode reads
// and writes it the same way: the desktop and the CLI through Database, the
// daemon through its routes, the batch import through mapFile.

// ArchiveExport is what ExportArchive wrote.
type ArchiveExport struct {
	Matches int `json:"matches"`
	// Codes counts the QR codes, parity codes included.
	Codes int `json:"codes"`
	// Bytes is the size of the coded matches, framing left out.
	Bytes   int           `json:"bytes"`
	Skipped []ArchiveSkip `json:"skipped,omitempty"`
}

// ArchiveSkip is a match an archive could not hold, and why.
type ArchiveSkip struct {
	MatchID int64  `json:"matchId"`
	Reason  string `json:"reason"`
}

// ExportArchive renders matches as an archive page: one match a QR code when
// it fits, a parity code every five, the coded matches embedded. No ids means
// every match of the scope. origin, when set, is the producer's line printed
// at the foot of each sheet. A match the codec cannot hold is skipped and
// named, never written half.
func ExportArchive(ctx context.Context, s storage.Storage, scope string, ids []int64, origin string) ([]byte, ArchiveExport, error) {
	var res ArchiveExport
	if len(ids) == 0 {
		for m, err := range s.Matches().List(ctx, scope, storage.MatchListOpts{}) {
			if err != nil {
				return nil, res, err
			}
			ids = append(ids, m.ID)
		}
	}
	var payloads [][]byte
	var labels []string
	for i, id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, res, err
		}
		progress.Report(ctx, i, len(ids))
		am, label, err := ArchiveMatch(ctx, s, scope, id)
		if err == nil {
			var data []byte
			if data, err = arkqr.Encode(am); err == nil {
				payloads = append(payloads, data)
				labels = append(labels, label)
				res.Bytes += len(data)
				continue
			}
		}
		if errors.Is(err, storage.ErrNotFound) || ctx.Err() != nil {
			return nil, res, fmt.Errorf("match %d: %w", id, err)
		}
		res.Skipped = append(res.Skipped, ArchiveSkip{MatchID: id, Reason: err.Error()})
	}
	if len(payloads) == 0 {
		return nil, res, fmt.Errorf("no match to archive: %w", storage.ErrInvalid)
	}
	progress.Report(ctx, len(ids), len(ids))
	frames, owner, err := arkqr.Frames(payloads)
	if err != nil {
		return nil, res, fmt.Errorf("%w: %w", storage.ErrInvalid, err)
	}
	codes := make([]arkqr.Code, len(frames))
	for i, f := range frames {
		codes[i] = arkqr.Code{Frame: f, Label: codeLabel(f, owner[i], labels)}
	}
	page, err := arkqr.Page(fmt.Sprintf("blunderDB · archive %08X", frames[0].Archive), origin, codes)
	if err != nil {
		return nil, res, err
	}
	res.Matches, res.Codes = len(payloads), len(frames)
	return page, res, nil
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

// ArchiveMatch reads a stored match into what the archive codes, and the
// line printed under its code. It reads the match the way a .mat export and
// the reopening of a draft do, then puts back what a .mat cannot carry: the
// session rules, the full date, the source metadata, the durations, the
// timecodes, the comments and study marks.
func ArchiveMatch(ctx context.Context, s storage.Storage, scope string, id int64) (arkqr.Match, string, error) {
	m, games, moves, err := ReadMatchForMAT(ctx, s, scope, id)
	if err != nil {
		return arkqr.Match{}, "", err
	}
	doc, err := transcript.FromMAT(RenderMAT(m, games, moves))
	if err != nil {
		return arkqr.Match{}, "", err
	}
	RestoreTimecodes(&doc, games, moves)
	h := &doc.Header
	if m.HasJacoby != nil {
		h.Jacoby = *m.HasJacoby
	}
	if m.HasBeaver != nil {
		h.Beaver = *m.HasBeaver
	}
	if !m.MatchDate.IsZero() {
		h.Date = m.MatchDate
	}
	if m.Transcriber != "" {
		h.Transcriber = m.Transcriber
	}
	// Only a URL leaves the library, as in every export.
	h.VideoSource = ""
	if v := issuance.CarriedVideoSource(m.VideoSource); v != nil {
		h.VideoSource = *v
	}
	am := arkqr.Match{
		Comment:       m.Comment,
		Elo:           [2]*float64{m.Player1Elo, m.Player2Elo},
		Experience:    [2]*int{m.Player1Experience, m.Player2Experience},
		EngineVersion: m.EngineVersion,
	}

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
	comments, err := s.Comments().ByPositions(ctx, scope, posIDs)
	if err != nil {
		return arkqr.Match{}, "", err
	}
	positions, err := s.Positions().LoadByIDs(ctx, scope, posIDs)
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
	return am, archiveLabel(m), nil
}

func archiveLabel(m *domain.Match) string {
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

// ArchiveGraph turns an archived match into the graph every import writes:
// the transcription's construction, the hashes an import dedups on, the
// source metadata, and the notes put back on the positions they were read
// from. Comments come back of unknown origin: the archive does not say who
// wrote them.
func ArchiveGraph(am arkqr.Match) (*MatchGraph, error) {
	parts := transcript.Build(am.Doc)
	if len(parts.Games) == 0 {
		return nil, fmt.Errorf("the archived match holds no game: %w", storage.ErrInvalid)
	}
	graph := TranscriptGraph(parts)
	// An imported match takes the seats an import gives it, not a draft's.
	graph.KeepSeats = false
	graph.Match.MatchHash, graph.Match.CanonicalHash = TranscriptHashes(parts)
	graph.Match.Comment = am.Comment
	if v := am.Doc.Header.VideoSource; v != "" {
		graph.Match.VideoSource = &v
	}
	graph.Match.Player1Elo, graph.Match.Player2Elo = am.Elo[0], am.Elo[1]
	graph.Match.Player1Experience, graph.Match.Player2Experience = am.Experience[0], am.Experience[1]
	graph.Match.EngineVersion = am.EngineVersion
	// The session rules hold at money play only (ADR-0067).
	if am.Doc.Header.MatchLength == 0 {
		jacoby, beaver := am.Doc.Header.Jacoby, am.Doc.Header.Beaver
		graph.Match.HasJacoby, graph.Match.HasBeaver = &jacoby, &beaver
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

// ReadArchive returns the codes of one archive file: the frames an exported
// page embeds, those a PDF printed from such a page carries as an attached
// file, or the codes an image (PNG, JPEG) shows.
func ReadArchive(data []byte) ([]arkqr.Frame, error) {
	if arkqr.IsPage(data) {
		return arkqr.ReadPage(data)
	}
	if IsPDF(data) {
		var out []arkqr.Frame
		for _, f := range pdfattach.Files(data) {
			if !arkqr.IsPage(f) {
				continue
			}
			fs, err := arkqr.ReadPage(f)
			if err != nil {
				return nil, err
			}
			out = append(out, fs...)
		}
		if len(out) == 0 {
			return nil, ErrNoPDFPayload
		}
		return out, nil
	}
	if t := bytes.TrimSpace(data); len(t) > 0 && t[0] == '<' {
		return nil, arkqr.ErrNoPayload
	}
	return arkqr.ScanImage(bytes.NewReader(data))
}

// ArchiveGraphs rebuilds the matches of the frames read — a lost code from
// its parity — and maps each to its graph. failed names, by rank in the
// archive, the matches that could not be decoded.
func ArchiveGraphs(frames []arkqr.Frame) (graphs []*MatchGraph, asm arkqr.Assembly, failed []string) {
	asm = arkqr.Assemble(frames)
	for i, data := range asm.Matches {
		am, err := arkqr.Decode(data)
		if err == nil {
			var g *MatchGraph
			if g, err = ArchiveGraph(am); err == nil {
				graphs = append(graphs, g)
				continue
			}
		}
		failed = append(failed, fmt.Sprintf("match %d: %v", i+1, err))
	}
	return graphs, asm, failed
}

// ArchivePositions decodes the tranches of position lists an assembly holds,
// in the order read. failed names the tranches that could not be decoded.
func ArchivePositions(asm arkqr.Assembly) (positions []domain.Position, failed []string) {
	for i, data := range asm.Tranches {
		t, err := arkqr.DecodePositions(data)
		if err != nil {
			failed = append(failed, fmt.Sprintf("tranche %d: %v", i+1, err))
			continue
		}
		positions = append(positions, t.Positions...)
	}
	return positions, failed
}

// SaveArchivePositions stores the positions a list's reimport code brought,
// each on its own as the user brought it (storage.SaveIndividual): created
// counts the new ones, known those the library already held.
func SaveArchivePositions(ctx context.Context, store storage.PositionStore, scope string, ps []domain.Position) (created, known int, err error) {
	for i := range ps {
		if err := ctx.Err(); err != nil {
			return created, known, err
		}
		_, isNew, err := storage.SaveIndividual(ctx, store, scope, &ps[i])
		if err != nil {
			return created, known, err
		}
		if isNew {
			created++
		} else {
			known++
		}
	}
	return created, known, nil
}

// mapArchive reads an archive page for the batch pipeline.
func mapArchive(path string) ([]*MatchGraph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !arkqr.IsPage(data) {
		return nil, fmt.Errorf("%w: an HTML file that is not a blunderDB archive page", errUnsupportedFile)
	}
	frames, err := arkqr.ReadPage(data)
	if err != nil {
		return nil, err
	}
	graphs, asm, failed := ArchiveGraphs(frames)
	if len(failed) > 0 || len(asm.Missing) > 0 {
		return nil, fmt.Errorf("archive page: %d matches unreadable, codes missing %v", len(failed), asm.Missing)
	}
	// The batch carries matches only: a printed list goes through the
	// paper archive import, which saves its positions.
	if len(asm.Tranches) > 0 && len(graphs) == 0 {
		return nil, fmt.Errorf("%w: a printed position list, imported as a paper archive", errUnsupportedFile)
	}
	return graphs, nil
}

// ArchiveImporter imports one archive file — a page, a scan, a photograph —
// in one transaction (/v1/imports.archive).
type ArchiveImporter struct{ S storage.Storage }

func (im ArchiveImporter) Import(ctx context.Context, scope string, src Source, prog func(Progress)) (Summary, error) {
	var data []byte
	var err error
	switch {
	case src.Path != "":
		data, err = os.ReadFile(src.Path)
	case src.Reader != nil:
		data, err = io.ReadAll(src.Reader)
	default:
		err = fmt.Errorf("ingest: archive import requires a file")
	}
	if err != nil {
		return Summary{}, err
	}
	frames, err := ReadArchive(data)
	if err != nil {
		return Summary{}, fmt.Errorf("%w: %w", storage.ErrInvalid, err)
	}
	if len(frames) == 0 {
		return Summary{}, fmt.Errorf("%w: no archive code found", storage.ErrInvalid)
	}
	graphs, asm, failed := ArchiveGraphs(frames)
	positions, pfailed := ArchivePositions(asm)
	failed = append(failed, pfailed...)
	if len(failed) > 0 || len(asm.Missing) > 0 {
		return Summary{}, fmt.Errorf("%w: %d matches or tranches unreadable, codes missing %v", storage.ErrInvalid, len(failed), asm.Missing)
	}
	tx, err := im.S.BeginTx(ctx)
	if err != nil {
		return Summary{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var sum Summary
	for _, g := range graphs {
		g.ImportBatchID, g.SkipDuplicates = src.BatchID, src.SkipDuplicates
		res, err := WriteMatch(ctx, tx, scope, g, prog)
		if err != nil {
			return Summary{}, err
		}
		if err := ctx.Err(); err != nil {
			return Summary{}, err
		}
		switch {
		case res.Skipped:
			sum.SkippedDuplicates++
		case res.Enriched:
			sum.Enriched++
		default:
			sum.Matches++
			sum.SavedPositions += res.SavedPositions
			if sum.MatchID == 0 {
				sum.MatchID = res.MatchID
			}
		}
	}
	if sum.ListPositions, sum.ListPositionsKnown, err = SaveArchivePositions(ctx, tx.Positions(), scope, positions); err != nil {
		return Summary{}, err
	}
	if err := tx.Commit(); err != nil {
		return Summary{}, err
	}
	committed = true
	return sum, nil
}

// ErrNoPDFPayload is a PDF that carries no blunderDB archive: not printed by
// blunderDB, or printed without reimport codes.
var ErrNoPDFPayload = fmt.Errorf("%w: this PDF carries no blunderDB archive", arkqr.ErrNoPayload)

// archiveAttachment is the name of the file a printed PDF carries: an
// archive page of its own, importable as it is once extracted.
const archiveAttachment = "blunderdb-archive.html"

// IsPDF reports whether data is a PDF document.
func IsPDF(data []byte) bool {
	return bytes.HasPrefix(bytes.TrimLeft(data, " \t\r\n"), []byte("%PDF-"))
}

// EmbedArchiveInPDF attaches to pdf, printed from page, the frames page
// embeds, so the PDF reimports like the page. A page without frames leaves
// pdf as it is.
func EmbedArchiveInPDF(pdf, page []byte) ([]byte, error) {
	if !arkqr.IsPage(page) {
		return pdf, nil
	}
	frames, err := arkqr.ReadPage(page)
	if err != nil {
		return nil, err
	}
	return pdfattach.Attach(pdf, archiveAttachment, "text/html", []byte(arkqr.EmbedFrames(frames)))
}

// IsArchiveFile reports whether the file at path is one blunderDB produced
// that carries an archive: an exported page, a match sheet or a position
// list saved as HTML with its reimport codes, or the PDF the interface
// printed from one. Every entry that takes a printed or written document
// asks it first, so such a file imports by the archive path.
func IsArchiveFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm", ".pdf":
	default:
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	if arkqr.IsPage(data) {
		return true
	}
	if !IsPDF(data) {
		return false
	}
	frames, err := ReadArchive(data)
	return err == nil && len(frames) > 0
}
