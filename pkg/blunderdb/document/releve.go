package document

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Format names the file kind in every Relevé, so a JSON file of another kind
// is refused by name rather than half-read.
const Format = "blunderdb-releve"

// Version is the version of the Relevé format this package reads and writes.
// A Relevé of a newer version is refused: its fields would be dropped.
const Version = 1

// ErrInvalid is the refusal of a malformed Relevé. It matches
// storage.ErrInvalid, so the daemon answers it with a 400.
var ErrInvalid = fmt.Errorf("document: invalid relevé: %w", storage.ErrInvalid)

// State is how far a reading of one diagram can be trusted (ADR-0108 rule 5).
type State string

const (
	// Sure — an independent cross-check confirmed the reading: the checkers
	// and the context (cube, dice, player on roll, score) the XGID gives. A
	// reader that does not read the context never claims it.
	Sure State = "sure"
	// ToVerify — the reading holds together but nothing confirmed it.
	ToVerify State = "to-verify"
	// Uncertain — the reading is known to be incomplete or contradicted.
	Uncertain State = "uncertain"
)

// Releve is the reading of one document.
type Releve struct {
	Format   string  `json:"format"`
	Version  int     `json:"version"`
	Document Source  `json:"document"`
	Entries  []Entry `json:"entries"`
}

// Source names the document the positions were read from.
type Source struct {
	Title   string `json:"title"`
	Author  string `json:"author,omitempty"`
	Edition string `json:"edition,omitempty"`
	URL     string `json:"url,omitempty"`
	// SHA256 is the hex SHA-256 of the source file, so a Relevé is matched
	// to the file it was read from; empty for a web page.
	SHA256  string   `json:"sha256,omitempty"`
	Profile *Profile `json:"profile,omitempty"`
}

// Profile is how the document draws its diagrams, fixed on the first one and
// valid for all of them.
type Profile struct {
	// Home is the side of the bottom player's home board: "left" or "right".
	Home string `json:"home,omitempty"`
	// Bottom is the colour of the bottom player: "black" or "white".
	Bottom string `json:"bottom,omitempty"`
	// Pips, Cube and Numbering say in free words where the pip counts and
	// the cube are printed and how the points are numbered.
	Pips      string `json:"pips,omitempty"`
	Cube      string `json:"cube,omitempty"`
	Numbering string `json:"numbering,omitempty"`
}

// Entry is one diagram of the document.
type Entry struct {
	// Rank is the diagram's place in the document, from 1, with no gap.
	Rank    int    `json:"rank"`
	Page    int    `json:"page"`
	Chapter string `json:"chapter,omitempty"`
	// Label is the document's own name for the diagram ("Problem 191").
	Label string `json:"label,omitempty"`
	// Position is the XGID of the diagram: checkers, cube, player on roll,
	// dice, score and match length. It is empty only for an uncertain entry
	// whose diagram could not be read as a legal position: the entry keeps
	// the diagram's place, and the import refuses it until a person fills
	// the position in.
	Position string          `json:"position"`
	State    State           `json:"state"`
	Doubts   []capture.Doubt `json:"doubts,omitempty"`
	Rect     *Rect           `json:"rect,omitempty"`
	// Verified is set when a person has compared the reading with the
	// diagram and accepted it.
	Verified bool `json:"verified"`
}

// Rect is where the diagram sits on its page, as fractions of the page's
// width and height from its top-left corner, so it holds whatever resolution
// the page is rendered at.
type Rect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// CheckersConfirmed reports an entry read by its structure whose checkers
// the printed pip counts confirmed: it is to verify, carries ContextDoubt,
// and its other doubts, if any, are about a cube or dice drawn and not read.
// What is left to verify is the context: cube, dice, player on roll, score.
func (e Entry) CheckersConfirmed() bool {
	if e.State != ToVerify || e.Position == "" {
		return false
	}
	found := false
	for _, d := range e.Doubts {
		switch {
		case d == ContextDoubt:
			found = true
		case !contextField[d.Field] || d.Field == ContextDoubt.Field:
			return false
		}
	}
	return found
}

// Importable reports whether the entry may be imported without forcing.
func (e Entry) Importable() bool { return e.State == Sure || e.Verified }

// Reference is the entry's place in the document as a person cites it:
// "p. 146, Problem 191", or "p. 146" when the diagram has no label.
func (e Entry) Reference() string {
	ref := "p. " + strconv.Itoa(e.Page)
	if l := strings.TrimSpace(e.Label); l != "" {
		ref += ", " + l
	}
	return ref
}

// Decode reads a Relevé and validates it. Unknown fields, trailing data and
// every rule of Validate are refusals.
func Decode(data []byte) (*Releve, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var r Releve
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: data after the relevé", ErrInvalid)
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return &r, nil
}

// Encode validates the Relevé and writes it as indented JSON.
func (r *Releve) Encode() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// Validate checks the whole Relevé and reports the first fault.
func (r *Releve) Validate() error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
	}
	if r.Format != Format {
		return bad("format %q, want %q", r.Format, Format)
	}
	if r.Version < 1 || r.Version > Version {
		return bad("version %d, this program reads version %d", r.Version, Version)
	}
	d := r.Document
	if strings.TrimSpace(d.Title) == "" {
		return bad("the document has no title")
	}
	if d.SHA256 != "" {
		if b, err := hex.DecodeString(d.SHA256); err != nil || len(b) != 32 {
			return bad("sha256 %q is not 64 hexadecimal digits", d.SHA256)
		}
	}
	if p := d.Profile; p != nil {
		if p.Home != "" && p.Home != "left" && p.Home != "right" {
			return bad("profile home %q, want left or right", p.Home)
		}
		if p.Bottom != "" && p.Bottom != "black" && p.Bottom != "white" {
			return bad("profile bottom %q, want black or white", p.Bottom)
		}
	}
	if len(r.Entries) == 0 {
		return bad("no entry")
	}
	for i, e := range r.Entries {
		if e.Rank != i+1 {
			return bad("entry %d has rank %d: ranks run from 1 with no gap", i+1, e.Rank)
		}
		if err := e.validate(); err != nil {
			return bad("rank %d: %v", e.Rank, err)
		}
		if i > 0 && e.Page < r.Entries[i-1].Page {
			return bad("rank %d is on page %d, before rank %d on page %d: entries follow the document", e.Rank, e.Page, i, r.Entries[i-1].Page)
		}
	}
	return nil
}

func (e Entry) validate() error {
	if e.Page < 1 {
		return fmt.Errorf("page %d", e.Page)
	}
	switch e.State {
	case Sure, ToVerify, Uncertain:
	default:
		return fmt.Errorf("state %q, want sure, to-verify or uncertain", e.State)
	}
	if e.State == Sure && len(e.Doubts) > 0 {
		return fmt.Errorf("a sure entry has no doubt")
	}
	for _, d := range e.Doubts {
		if strings.TrimSpace(d.Field) == "" {
			return fmt.Errorf("a doubt names no field")
		}
	}
	if r := e.Rect; r != nil {
		if r.X < 0 || r.Y < 0 || r.W <= 0 || r.H <= 0 || r.X+r.W > 1 || r.Y+r.H > 1 {
			return fmt.Errorf("rect %+v is not inside the page (fractions of its size)", *r)
		}
	}
	if e.Position == "" && e.State == Uncertain && !e.Verified {
		return nil // a diagram that could not be read: listed, never importable as is
	}
	_, err := ParsePosition(e.Position)
	return err
}

// ParsePosition reads an entry's XGID strictly: domain.DecodeXGID forgives
// missing or out-of-range fields, which a Relevé must not, since it is the
// record a person verifies.
func ParsePosition(xgid string) (domain.Position, error) {
	s := strings.TrimPrefix(strings.TrimSpace(xgid), "XGID=")
	f := strings.Split(s, ":")
	if len(f) != 10 {
		return domain.Position{}, fmt.Errorf("position %q: an XGID has 10 fields, got %d", xgid, len(f))
	}
	board := f[0]
	if len(board) == 26 {
		if c := board[0]; c != '-' && (c < 'a' || c > 'o') {
			return domain.Position{}, fmt.Errorf("position %q: only player O stands on bar 0", xgid)
		}
		if c := board[25]; c != '-' && (c < 'A' || c > 'O') {
			return domain.Position{}, fmt.Errorf("position %q: only player X stands on bar 25", xgid)
		}
	}
	num := make([]int, 10)
	for i := 1; i < 10; i++ {
		if i == 4 {
			continue
		}
		n, err := strconv.Atoi(f[i])
		if err != nil {
			return domain.Position{}, fmt.Errorf("position %q: field %d is not a number", xgid, i)
		}
		num[i] = n
	}
	switch {
	case num[1] < 0 || num[1] > 10:
		return domain.Position{}, fmt.Errorf("position %q: cube exponent %d", xgid, num[1])
	case num[2] < -1 || num[2] > 1:
		return domain.Position{}, fmt.Errorf("position %q: cube owner %d", xgid, num[2])
	case num[3] != 1 && num[3] != -1:
		return domain.Position{}, fmt.Errorf("position %q: player on roll %d, want 1 or -1", xgid, num[3])
	case num[8] < 0 || num[5] < 0 || num[6] < 0:
		return domain.Position{}, fmt.Errorf("position %q: negative score or match length", xgid)
	case num[8] > 0 && (num[5] >= num[8] || num[6] >= num[8]):
		return domain.Position{}, fmt.Errorf("position %q: a score reaches the match length", xgid)
	case num[8] == 0 && (num[5] != 0 || num[6] != 0):
		return domain.Position{}, fmt.Errorf("position %q: a score in money play", xgid)
	}
	if dice := f[4]; dice != "00" {
		if len(dice) != 2 || dice[0] < '1' || dice[0] > '6' || dice[1] < '1' || dice[1] > '6' {
			return domain.Position{}, fmt.Errorf("position %q: dice %q, want 00 or two faces 1-6", xgid, dice)
		}
	}
	pos, err := domain.DecodeXGID(s)
	if err != nil {
		return domain.Position{}, fmt.Errorf("position %q: %w", xgid, err)
	}
	if pos.Board.Bearoff[domain.Black] == 15 || pos.Board.Bearoff[domain.White] == 15 {
		return domain.Position{}, fmt.Errorf("position %q: a player has no checker left", xgid)
	}
	return pos, nil
}
