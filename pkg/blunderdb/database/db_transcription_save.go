package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// Saving and exporting a draft (ADR-0045 §2, §3, §8). Plumbing between the
// transcript package, ingest.WriteMatch and the gammonNet batch; the two
// decisions of its own are stated where they are made.

// TranscriptionSaveResult is what a save reports. ToAnalyze is the derived,
// never stored, count of the match's positions without analysis (ADR-0045 §8).
type TranscriptionSaveResult struct {
	MatchID int64 `json:"match_id"`
	// Replaced is false on the first save (the Match was created) and true on
	// every one after it (the same Match was rewritten in place).
	Replaced  bool `json:"replaced"`
	Games     int  `json:"games"`
	Moves     int  `json:"moves"`
	Positions int  `json:"positions"`
	ToAnalyze int  `json:"to_analyze"`
	// Inconsistent: the draft carries an Inconsistency; saved all the same
	// (ADR-0044), flagged for callers that did not warn.
	Inconsistent bool `json:"inconsistent"`
}

// SaveTranscriptionAsMatch materialises the open draft as a Match: the first
// save creates it and posts its id on the document, every save after that
// REPLACES the same Match, id and all (ADR-0045 §2).
//
// The draft stays the source of truth. A replacement is not snapshotted to the
// trash (ADR-0045 §3). The analysis batch is the caller's to start.
func (d *Database) SaveTranscriptionAsMatch(id int64) (*TranscriptionSaveResult, error) {
	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()

	ed, err := d.session(id)
	if err != nil {
		return nil, err
	}

	parts := transcript.Build(ed.Doc)

	// The one refusal: no game at all. ADR-0044 keeps what was written, it
	// does not licence an empty Match polluting statistics and searches.
	if len(parts.Games) == 0 {
		return nil, fmt.Errorf("transcription %d: nothing to save yet", id)
	}

	replace := ed.Doc.Header.MatchID != nil && *ed.Doc.Header.MatchID != 0
	graph := transcriptGraph(parts)
	if replace {
		graph.ReplaceMatchID = *ed.Doc.Header.MatchID
	}

	header := *parts.Match
	header.MatchHash, header.CanonicalHash = transcriptMatchHashes(parts)

	res, err := d.writeTranscribedMatch(context.Background(), graph, header)
	if err != nil {
		return nil, err
	}

	// The match id is posted on the document at the FIRST save and never
	// changes again: it is what makes every later save a replacement rather
	// than a second Match (fonctionnel.md §1.1).
	if !replace {
		matchID := res.MatchID
		ed.Doc.Header.MatchID = &matchID
		if _, err := d.saveTranscription(id, ed.Doc); err != nil {
			return nil, err
		}
	}

	out := &TranscriptionSaveResult{
		MatchID:      res.MatchID,
		Replaced:     res.Replaced,
		Games:        len(parts.Games),
		Positions:    res.SavedPositions,
		Inconsistent: parts.Inconsistent,
	}
	for _, moves := range parts.Moves {
		out.Moves += len(moves)
	}
	if n, err := d.CountMatchPositionsToAnalyze(res.MatchID); err == nil {
		out.ToAnalyze = n
	}
	return out, nil
}

// writeTranscribedMatch is the write itself: one transaction, ingest.WriteMatch,
// and the content hashes stated afterwards.
//
// Afterwards on purpose: WriteMatch dedups on the hashes, which for a save
// would enrich a match the library already holds from XG and hand it to the
// draft. So the graph travels hash-less and ReplaceHeader stamps them in the
// same transaction, for a future IMPORT to dedup against.
func (d *Database) writeTranscribedMatch(ctx context.Context, graph *ingest.MatchGraph, header domain.Match) (ingest.WriteResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var res ingest.WriteResult
	if d.db == nil {
		return res, fmt.Errorf("no database is currently open")
	}

	tx, err := d.store.BeginTx(ctx)
	if err != nil {
		return res, err
	}
	res, err = ingest.WriteMatch(ctx, tx, "", graph, nil)
	if err != nil {
		_ = tx.Rollback()
		return res, err
	}
	header.ID = res.MatchID
	if err := tx.Matches().ReplaceHeader(ctx, "", res.MatchID, &header); err != nil {
		_ = tx.Rollback()
		return res, err
	}
	if err := attachTournament(ctx, tx, res.MatchID, header.TournamentID); err != nil {
		_ = tx.Rollback()
		return res, err
	}
	if err := tx.Commit(); err != nil {
		return res, err
	}
	return res, nil
}

// attachTournament is the header's tournament made true of the Match, in the
// same transaction as the write.
//
// Here, not in the graph: ReplaceHeader does not touch tournament_id, which
// the tournament store owns with its sort order. A save with no tournament
// DETACHES, so clearing the field is not silently ignored.
func attachTournament(ctx context.Context, tx storage.Tx, matchID int64, tournamentID *int64) error {
	if tournamentID != nil && *tournamentID != 0 {
		return tx.Tournaments().AddMatch(ctx, "", *tournamentID, matchID)
	}
	return tx.Tournaments().RemoveMatch(ctx, "", matchID)
}

// transcriptGraph turns what the transcript package returns into the graph
// ingest.WriteMatch writes. It carries no analysis and no comment: the
// analysis is the batch's job afterwards (ADR-0045 §8), and a transcription
// has no notes to attach.
//
// The transcript's own ids are not database ids and are dropped; so are the
// hashes (see writeTranscribedMatch).
func transcriptGraph(parts transcript.Parts) *ingest.MatchGraph {
	g := &ingest.MatchGraph{Match: *parts.Match}
	g.Match.MatchHash, g.Match.CanonicalHash = "", ""

	for _, game := range parts.Games {
		moves := parts.Moves[game.ID]
		positions := parts.Positions[game.ID]

		gg := ingest.GameGraph{Game: *game}
		gg.Game.ID = 0
		gg.Game.MatchID = 0
		for i, mv := range moves {
			mg := ingest.MoveGraph{Move: *mv}
			mg.Move.GameID = 0
			if i < len(positions) {
				pos := positions[i]
				mg.Position = &pos
			}
			gg.Moves = append(gg.Moves, mg)
		}
		g.Games = append(g.Games, gg)
	}
	return g
}

// transcriptMatchHashes are the two content hashes of a transcribed match.
//
// The canonical one is the SAME scheme as the importers
// (computeCanonicalMatchHashFromXG and twins), so a later XG/GnuBG import of
// the same match enriches instead of duplicating. The format-specific one
// hashes the plays as typed and changes whenever the document does.
func transcriptMatchHashes(parts transcript.Parts) (matchHash, canonicalHash string) {
	m := parts.Match

	var b strings.Builder
	p1 := strings.TrimSpace(strings.ToLower(m.Player1Name))
	p2 := strings.TrimSpace(strings.ToLower(m.Player2Name))
	fmt.Fprintf(&b, "transcript1:%s|%s|%d|", p1, p2, m.MatchLength)
	for gi, game := range parts.Games {
		fmt.Fprintf(&b, "g%d:%d,%d,%d,%d|", gi,
			game.InitialScore[0], game.InitialScore[1], game.Winner, game.PointsWon)
		for mi, mv := range parts.Moves[game.ID] {
			fmt.Fprintf(&b, "m%d:%s,%d,d%d%d,p%s|", mi, mv.MoveType, mv.Player,
				mv.Dice[0], mv.Dice[1], mv.CheckerMove+mv.CubeAction)
		}
	}
	matchHash = sha256Hex(b.String())

	var c strings.Builder
	if p1 > p2 {
		p1, p2 = p2, p1
	}
	fmt.Fprintf(&c, "canonical2:%s|%s|%d|%d|", p1, p2, m.MatchLength, len(parts.Games))
	for gi, game := range parts.Games {
		fmt.Fprintf(&c, "g%d|", gi)
		dice := 0
		for _, mv := range parts.Moves[game.ID] {
			if dice >= transcriptCanonicalDicePerGame {
				break
			}
			if mv.MoveType != "checker" {
				continue
			}
			d1, d2 := mv.Dice[0], mv.Dice[1]
			if d1 > d2 {
				d1, d2 = d2, d1
			}
			fmt.Fprintf(&c, "d%d%d|", d1, d2)
			dice++
		}
	}
	return matchHash, sha256Hex(c.String())
}

// transcriptCanonicalDicePerGame mirrors ingest's maxCanonicalDicePerGame,
// which is unexported there. The two must state the same number or a
// transcribed match stops hashing like its imported twin — which is the whole
// point of the canonical hash.
const transcriptCanonicalDicePerGame = 10

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// SuggestTranscriptionMatFilename is SuggestMatFilename for a draft: the name
// the export dialog opens on, built by the same helper from the match the
// document would produce, so a draft and the Match it was saved as suggest
// the same file.
func (d *Database) SuggestTranscriptionMatFilename(id int64) (string, error) {
	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()

	ed, err := d.session(id)
	if err != nil {
		return "", err
	}
	m, _, _ := transcript.MatchParts(ed.Doc)
	return ingest.SuggestMATFilename(m), nil
}

// ExportTranscriptionMAT writes the open draft as a .mat file, rendering
// first so a failure leaves no truncated file. Exported AS TYPED, illegal
// plays included (ADR-0044); nothing is written to the library.
func (d *Database) ExportTranscriptionMAT(id int64, outputPath string) error {
	text, err := d.TranscriptionMAT(id)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(text), 0o644)
}

// TranscriptionAnalysisResume is the fact ADR-0045 §8 refuses to store,
// recounted instead: the last draft that produced a Match, and how many of
// its positions still lack analysis.
type TranscriptionAnalysisResume struct {
	// TranscriptionID is the draft that produced the Match, and Label what the
	// drafts list shows for it, so the offer can name what it is about.
	TranscriptionID int64  `json:"transcription_id"`
	MatchID         int64  `json:"match_id"`
	Label           string `json:"label"`
	// ToAnalyze is CountMatchPositionsToAnalyze at the moment of asking, and
	// never anything else: the figure the targeted batch is about to work
	// through.
	ToAnalyze int `json:"to_analyze"`
}

// PendingTranscriptionAnalysis reports whether the last SAVED draft's Match
// has positions left to analyse, and returns nil when it has none — which is
// also what a library with no saved draft at all answers.
//
// Asked on open; ignoring the offer stores nothing. Scoped to one match on
// purpose, not the library's imported positions. "Last saved" is the most
// recently updated draft with a match id (ListTranscriptions' order); a
// deleted Match has its column NULLed by the schema.
func (d *Database) PendingTranscriptionAnalysis() (*TranscriptionAnalysisResume, error) {
	drafts, err := d.ListTranscriptions()
	if err != nil {
		return nil, err
	}
	for _, draft := range drafts {
		if draft.MatchID == 0 {
			continue
		}
		// CountMatchPositionsToAnalyze takes the read lock itself; taking it
		// around this loop as well would deadlock the moment a writer queued
		// up between the two (sync.RWMutex does not admit nested readers).
		n, err := d.CountMatchPositionsToAnalyze(draft.MatchID)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, nil
		}
		return &TranscriptionAnalysisResume{
			TranscriptionID: draft.ID,
			MatchID:         draft.MatchID,
			Label:           draft.Label,
			ToAnalyze:       n,
		}, nil
	}
	return nil, nil
}
