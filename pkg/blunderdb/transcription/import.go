package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// ErrNothingToImport answers a .mat from which no Action could be read: a
// draft made of it would be an empty one, which Create already gives, and a
// file that says nothing is more likely the wrong file.
var ErrNothingToImport = errors.New("transcription: the .mat holds no game")

// ImportMAT opens a new draft on a Jellyfish/gnubg .mat text: the document
// the replay of `transcribe --mat` reads, written as a draft rather than
// finished, so its Inconsistencies are corrected before a Match exists. The
// draft owns no Match yet; finishing it writes a new one.
func (s *Service) ImportMAT(ctx context.Context, scope, text string) (*State, error) {
	doc, err := transcript.FromMAT(text)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", storage.ErrInvalid, err)
	}
	if len(doc.Actions) == 0 {
		return nil, fmt.Errorf("%w: %w", storage.ErrInvalid, ErrNothingToImport)
	}
	return s.importDocument(ctx, scope, doc)
}

// ImportDocument opens a new draft on a transcription document — the shape a
// draft is stored in (`header`, `actions`, optionally `format_version` and
// `next_score`). A document of an earlier format is converted as a stored
// draft is; one without a format is read as the current one, as materialize
// reads its Actions. What the rule machine finds wrong in the Actions is kept
// and marked, as in any draft (ADR-0045 rule 5); what no replay can mean — an
// unknown kind, a side or a die out of range, a format from a later version —
// refuses the whole document and writes nothing.
func (s *Service) ImportDocument(ctx context.Context, scope string, doc transcript.Document) (*State, error) {
	doc, err := prepareImport(doc)
	if err != nil {
		return nil, err
	}
	return s.importDocument(ctx, scope, doc)
}

// Import sniffs text: a JSON object is a transcription document, anything
// else a .mat. It serves the callers that take a file without saying which.
// A byte-order mark, which some editors write at the head of a file, is not
// part of either format.
func (s *Service) Import(ctx context.Context, scope, text string) (*State, error) {
	text = strings.TrimPrefix(text, "\uFEFF")
	if strings.HasPrefix(strings.TrimSpace(text), "{") {
		doc, err := DecodeDocument([]byte(text))
		if err != nil {
			return nil, err
		}
		return s.ImportDocument(ctx, scope, doc)
	}
	return s.ImportMAT(ctx, scope, text)
}

// importDocument writes doc as a new draft. The header gets the defaults a
// new draft gets, and loses the Match id: an imported document comes from
// elsewhere and owns nothing here.
func (s *Service) importDocument(ctx context.Context, scope string, doc transcript.Document) (*State, error) {
	if err := replayable(doc); err != nil {
		return nil, err
	}
	doc.Header = s.draftHeader(ctx, scope, doc.Header)
	doc.Cursor = len(doc.Actions)
	return s.insert(ctx, scope, doc)
}

// DecodeDocument reads a transcription document in JSON, and nothing after
// it. A failure is a storage.ErrInvalid.
func DecodeDocument(raw []byte) (transcript.Document, error) {
	var doc transcript.Document
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&doc); err != nil {
		return doc, fmt.Errorf("%w: transcription document: %w", storage.ErrInvalid, err)
	}
	if dec.More() {
		return doc, fmt.Errorf("%w: transcription document: trailing data after the document", storage.ErrInvalid)
	}
	return doc, nil
}

// prepareImport brings a posted document to the current format and checks
// what can be checked without a replay.
func prepareImport(doc transcript.Document) (transcript.Document, error) {
	switch {
	case doc.FormatVersion > transcript.FormatVersion:
		return doc, fmt.Errorf("%w: transcription document format %d, this version reads up to %d",
			storage.ErrInvalid, doc.FormatVersion, transcript.FormatVersion)
	case doc.FormatVersion < 0:
		return doc, fmt.Errorf("%w: transcription document format %d", storage.ErrInvalid, doc.FormatVersion)
	case doc.FormatVersion == 0:
		doc.FormatVersion = transcript.FormatVersion
	}
	doc = transcript.Upgrade(doc)
	return doc, checkImport(doc)
}

// checkImport refuses what a replay cannot give a meaning to. The rules
// themselves are not checked: a draft keeps an illegal play and marks it.
func checkImport(doc transcript.Document) error {
	h := doc.Header
	if h.MatchLength < 0 {
		return invalidImport(-1, "the match length is %d", h.MatchLength)
	}
	if h.MaxCube < 0 {
		return invalidImport(-1, "the cube ceiling is %d", h.MaxCube)
	}
	if ns := doc.NextScore; ns != nil && (ns[0] < 0 || ns[1] < 0) {
		return invalidImport(-1, "the next score %v is negative", *ns)
	}
	for i, a := range doc.Actions {
		if a.Side != domain.Black && a.Side != domain.White {
			return invalidImport(i, "the side is %d, not player 1 or 2", a.Side)
		}
		if a.Score != nil && (a.Score[0] < 0 || a.Score[1] < 0) {
			return invalidImport(i, "the declared score %v is negative", *a.Score)
		}
		switch a.Kind {
		case transcript.KindChecker, transcript.KindDance, transcript.KindUnrecorded:
			if a.Dice[0] < 1 || a.Dice[0] > 6 || a.Dice[1] < 1 || a.Dice[1] > 6 {
				return invalidImport(i, "the roll %v is not two dice", a.Dice)
			}
			for _, st := range a.Steps {
				if !onBoard(st.From) || (!onBoard(st.To) && st.To != domain.Off) {
					return invalidImport(i, "the step %d/%d is off the board", st.From, st.To)
				}
			}
		case transcript.KindResign:
			if a.Level < 1 || a.Level > 3 {
				return invalidImport(i, "a resignation is worth 1 to 3 games, not %d", a.Level)
			}
		case transcript.KindDouble, transcript.KindTake, transcript.KindPass, transcript.KindForfeit:
		default:
			return invalidImport(i, "unknown action kind %q", a.Kind)
		}
	}
	return nil
}

// onBoard is a point a step may leave from: a point or a bar. A step may also
// end at domain.Off, a checker borne off.
func onBoard(pt int) bool { return pt >= 0 && pt <= domain.NumPoints+1 }

// replayable is the last guard before a document becomes a draft: one that
// the replay cannot go through would be a draft nobody can open.
func replayable(doc transcript.Document) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: the transcription document cannot be replayed: %v", storage.ErrInvalid, r)
		}
	}()
	transcript.Replay(doc, 0)
	return nil
}

func invalidImport(rank int, format string, args ...any) error {
	return &RefusedAction{Rank: rank, Refusal: &transcript.Refusal{
		Kind:   transcript.RefusedRecord,
		Detail: fmt.Sprintf(format, args...),
	}}
}
