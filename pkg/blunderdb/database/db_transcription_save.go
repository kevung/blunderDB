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

// =====================================================================
// Saving a draft, exporting it, and being done with it (T1.9)
// =====================================================================
//
// fonctionnel.md §4 and §5, ADR-0045 §2, §3 and §8. Like db_transcription.go
// this file is PLUMBING: the document is the transcript package's business,
// the writing is ingest.WriteMatch's, the analysis is
// db_gammonnet_batch.go's. What is here is the join, and the two decisions
// the join has to take on its own — both stated below, where they are made.

// TranscriptionSaveResult is what a save reports: the Match it produced, and
// what the panel needs to say next. ToAnalyze is the count ADR-0045 §8 asks
// the panel to derive rather than store — the positions of the saved match
// that have no analysis at all, which is exactly what the targeted batch is
// about to work through.
type TranscriptionSaveResult struct {
	MatchID int64 `json:"match_id"`
	// Replaced is false on the first save (the Match was created) and true on
	// every one after it (the same Match was rewritten in place).
	Replaced  bool `json:"replaced"`
	Games     int  `json:"games"`
	Moves     int  `json:"moves"`
	Positions int  `json:"positions"`
	ToAnalyze int  `json:"to_analyze"`
	// Inconsistent restates what the panel already knows from the annotated
	// document: the draft carries at least one Inconsistency. The save was
	// made all the same — nothing is refused (ADR-0044) — and the field is
	// here so a caller that did not warn (the CLI, a test) still can.
	Inconsistent bool `json:"inconsistent"`
}

// SaveTranscriptionAsMatch materialises the open draft as a Match: the first
// save creates it and posts its id on the document, every save after that
// REPLACES the same Match, id and all (ADR-0045 §2).
//
// The draft stays open and stays the source of truth. Nothing is snapshotted:
// a replacement is housekeeping, not a deletion, and replacing one match
// twenty times during a review must not leave twenty copies in the trash
// (ADR-0045 §3).
//
// It does NOT start the analysis batch — that is the caller's, because the
// batch is a background job with a progress bar and a cancel button, and this
// package has neither. The count it hands back is what the caller starts it
// on.
func (d *Database) SaveTranscriptionAsMatch(id int64) (*TranscriptionSaveResult, error) {
	d.transcriptMu.Lock()
	defer d.transcriptMu.Unlock()

	ed, err := d.session(id)
	if err != nil {
		return nil, err
	}

	parts := transcript.Build(ed.Doc)

	// The one thing a save refuses, and it is not an Inconsistency: a document
	// with no game at all. ADR-0044's "nothing is refused" is about what the
	// user wrote down being kept as written — it is not a licence to file an
	// empty Match in the library, which would then count in the statistics and
	// answer searches with nothing at all.
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
// Afterwards, and that is the decision this function exists for. WriteMatch
// reads MatchHash and CanonicalHash to answer "has this match arrived here
// before?" — the right question for an IMPORT and the wrong one for a save: a
// transcription of a match the library already holds from XG would be silently
// enriched into it, the draft would come to own someone else's match, and the
// next save would rewrite that match's games from the draft. So the graph
// travels with the hashes empty (nothing to find, the match is created or
// replaced as asked) and ReplaceHeader states them on the row a moment later,
// inside the same transaction. They are then what fonctionnel.md §4.2 says
// they are: the content identity a future IMPORT of this match deduplicates
// against, never the identity of the transcribed Match, which is its id.
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
// It is stated HERE and not in the graph because ReplaceHeader does not touch
// `tournament_id`: the column belongs to the tournament's own ordering
// (tournament_sort_order goes with it), and the store that owns both is the
// one asked. A save with no tournament DETACHES, so that clearing the field in
// the metadata pane is a change like any other rather than one the next save
// silently ignores — the attachment is decided at the save (fonctionnel.md
// §1.1), and the draft is what decides it.
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
// The ids the transcript package numbers its games with are its own — they
// index Moves and Positions, they are not database ids — so they are dropped
// here; WriteMatch assigns the real ones. The hashes are dropped too, for the
// reason writeTranscribedMatch states.
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
// The canonical one is deliberately the SAME scheme the importers compute
// (ingest/xg.go's computeCanonicalMatchHashFromXG and its twins): players
// sorted and lower-cased, the length, the number of games, then the first ten
// rolls of each game with each roll's dice sorted. A match transcribed here
// and later imported from an XG or a GnuBG file therefore hashes the same,
// and the import enriches instead of filing a second copy — which is what
// "they serve the deduplication of imports" means (fonctionnel.md §4.2).
//
// The format-specific one is this format's own: the plays as a transcript
// writes them, which no importer spells the same way. Its purpose is to
// change whenever the document changes, which it does — it is recomputed and
// rewritten at every save.
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

// ExportTranscriptionMAT writes the open draft as a .mat file. It renders
// first and writes second, like ExportMatchMAT, so a failure leaves no
// truncated file behind.
//
// The draft is exported AS TYPED. An illegal play goes out as it was played —
// gnubg and XG will flag it "Invalid move" and diverge from there, which is
// what the caller warns about, and the export is never refused (ADR-0044,
// fonctionnel.md §5). Nothing here is written to the library: an export is not
// a save.
func (d *Database) ExportTranscriptionMAT(id int64, outputPath string) error {
	text, err := d.TranscriptionMAT(id)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(text), 0o644)
}

// TranscriptionAnalysisResume is the fact ADR-0045 §8 refuses to store: the
// last draft that produced a Match, and how many of that Match's positions
// still carry no analysis. A batch cut short by a close leaves nothing behind
// — no flag, no journal, no `.ckpt` — so "the analysis never finished" is not
// a state to be read back but a count to be taken again, which is what this
// is.
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
// It is what the status bar asks when a database is opened (fonctionnel.md
// §4, "Reprise de l'analyse"), and it is deliberately a QUESTION rather than a
// reminder someone left: ignoring the offer stores nothing, so reopening the
// database asks again for as long as positions are missing, and finishing the
// batch makes the offer disappear on its own.
//
// The scope is one match on purpose. The library-wide catch-up already exists
// in the settings, and it is not what is wanted here: a user who transcribed
// one match must not be handed the thousands of imported positions he never
// asked to have analysed.
//
// "The last saved draft" is the most recently updated draft that carries a
// match id — ListTranscriptions is already ordered that way. A draft whose
// Match was deleted has had its column set back to NULL by the schema, so it
// is not a candidate, and no stale match id is ever counted.
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
