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

// Version is the version of the Relevé format this package writes. Version 2
// gives each entry two certainties, its checkers and its context, and the
// document a default context a person confirms once (Source.Defaults);
// version 1 gave one state for the whole reading and is still read: Decode
// turns it into version 2. A Relevé of a newer version is refused: its fields
// would be dropped.
const Version = 2

// ErrInvalid is the refusal of a malformed Relevé. It matches
// storage.ErrInvalid, so the daemon answers it with a 400.
var ErrInvalid = fmt.Errorf("document: invalid relevé: %w", storage.ErrInvalid)

// State is how far a reading of one diagram can be trusted (ADR-0108 rule 5).
// In version 2 it is the certainty of an entry's checkers (Entry.Checkers),
// and the state of the entry as a whole follows from its checkers and its
// context (Releve.StateOf).
type State string

const (
	// Sure — an independent cross-check confirmed the reading. An entry is
	// sure only when its checkers are and its context is certified: read,
	// or under the document's defaults once a person confirmed them.
	Sure State = "sure"
	// ToVerify — the reading holds together but nothing confirmed it.
	ToVerify State = "to-verify"
	// Uncertain — the reading is known to be incomplete or contradicted.
	Uncertain State = "uncertain"
)

// ContextState is how an entry's context — cube, dice, player on roll,
// score — is known.
type ContextState string

const (
	// ContextRead — read from the diagram's own print and cross-checked.
	ContextRead ContextState = "read"
	// ContextDefault — not read: the position carries the document's default
	// context (Source.Defaults), certified once a person confirms it.
	ContextDefault ContextState = "default"
	// ContextToVerify — neither read nor under the defaults: a cube or dice
	// drawn and not read, a score not found.
	ContextToVerify ContextState = "to-verify"
	// ContextUncertain — the context read contradicts itself.
	ContextUncertain ContextState = "uncertain"
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
	// Defaults is the context of every entry whose context is "default".
	Defaults *Defaults `json:"defaults,omitempty"`
}

// Defaults is the context a document's diagrams share when they do not print
// it: a person who confirms it once certifies the context of every entry
// under it, and only the exceptions a reader detected (a cube off the
// centre, dice drawn) are left to verify one by one.
type Defaults struct {
	// Context is the XGID after its board: cube, cube owner, player on roll,
	// dice, scores, Crawford, match length, cube limit
	// ("0:0:1:00:0:0:0:0:10": a centred cube, no dice, the bottom player on
	// roll, money play). Every entry under the defaults carries it.
	Context string `json:"context"`
	// Description says it in words, for the person who confirms it.
	Description string `json:"description,omitempty"`
	// Confirmed is set by a person who checked that the document's diagrams
	// follow the defaults.
	Confirmed bool `json:"confirmed"`
}

// StructureDefaults is the context a reader that does not read it writes: a
// centred cube, no dice, the bottom player on roll, money play.
var StructureDefaults = Defaults{
	Context:     "0:0:1:00:0:0:0:0:10",
	Description: "centred cube, no dice, the bottom player on roll, money play",
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
	Position string `json:"position"`
	// Checkers is the certainty of the checkers' reading.
	Checkers State `json:"checkers,omitempty"`
	// Complete is a reading of fifteen checkers a side with no doubt that no
	// cross-check confirmed: right on every measure so far, it is a state
	// between to verify and sure that an import accepts as a whole only on
	// request (ImportOptions.AcceptComplete). A position with checkers borne
	// off is never complete.
	Complete bool `json:"complete,omitempty"`
	// Context is how the cube, dice, player on roll and score are known.
	Context ContextState `json:"context,omitempty"`
	// State is the one certainty of a version 1 entry; Decode turns it into
	// Checkers and Context, and a version 2 entry has none.
	State  State           `json:"state,omitempty"`
	Doubts []capture.Doubt `json:"doubts,omitempty"`
	Rect   *Rect           `json:"rect,omitempty"`
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

// checkersConfirmedV1 is a version 1 entry read by its structure whose
// checkers the printed pips confirmed: to verify, with ContextDoubt and no
// doubt but about a cube or dice drawn and not read.
func (e Entry) checkersConfirmedV1() bool {
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

// ContextCertified reports whether the entry's context is read, or under
// the document's defaults once a person confirmed them.
func (r *Releve) ContextCertified(e Entry) bool {
	switch e.Context {
	case ContextRead:
		return true
	case ContextDefault:
		return r.Document.Defaults != nil && r.Document.Defaults.Confirmed
	}
	return false
}

// StateOf is the state of the entry as a whole: uncertain when its checkers
// or its context are, or when it has no position; sure only when its
// checkers are and its context is certified; to verify otherwise.
func (r *Releve) StateOf(e Entry) State {
	if r.Version < 2 {
		return e.State
	}
	switch {
	case e.Position == "" || e.Checkers == Uncertain || e.Context == ContextUncertain:
		return Uncertain
	case e.Checkers == Sure && r.ContextCertified(e):
		return Sure
	}
	return ToVerify
}

// CheckersConfirmed reports an entry whose checkers a cross-check confirmed
// and whose context alone is left to verify.
func (r *Releve) CheckersConfirmed(e Entry) bool {
	if r.Version < 2 {
		return e.checkersConfirmedV1()
	}
	return e.Position != "" && e.Checkers == Sure && r.StateOf(e) == ToVerify
}

// CompleteCertified reports an entry that is neither sure nor verified but
// whose fifteen checkers a side were read with no doubt, its context
// certified: what ImportOptions.AcceptComplete accepts as a whole.
func (r *Releve) CompleteCertified(e Entry) bool {
	return r.Version >= 2 && e.Complete && !e.Verified && r.StateOf(e) == ToVerify &&
		e.Checkers == ToVerify && r.ContextCertified(e)
}

// Importable reports whether the entry may be imported without forcing:
// sure or verified, or complete with its context certified when
// acceptComplete.
func (r *Releve) Importable(e Entry, acceptComplete bool) bool {
	return e.Verified || r.StateOf(e) == Sure || (acceptComplete && r.CompleteCertified(e))
}

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
// every rule of Validate are refusals. A version 1 Relevé is returned as
// version 2 (Upgrade).
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
	return r.Upgrade(), nil
}

// Upgrade returns a valid Relevé as version 2: r itself when it is already,
// else a copy whose entries carry two certainties. A version 1 sure entry
// has its checkers sure and its context read. An entry a structure reader
// left with ContextDoubt keeps its checkers confirmed when they were, and
// its context is under StructureDefaults, unconfirmed, when its position
// carries them and no cube or dice was drawn. Any other entry has its
// checkers in its old state and its context to verify: version 1 did not
// say which part of the reading a doubt was about.
func (r *Releve) Upgrade() *Releve {
	if r.Version >= 2 {
		return r
	}
	u := *r
	u.Version = Version
	u.Entries = make([]Entry, len(r.Entries))
	defaults := false
	for i, e := range r.Entries {
		v1 := e.State
		confirmed := e.checkersConfirmedV1()
		e.State = ""
		e.Checkers, e.Context = v1, ContextToVerify
		switch {
		case v1 == Sure:
			e.Context = ContextRead
		case hasDoubt(e.Doubts, ContextDoubt):
			if confirmed {
				e.Checkers = Sure
			}
			if !drawnContext(e.Doubts) && positionContext(e.Position) == StructureDefaults.Context {
				e.Context = ContextDefault
				defaults = true
			}
		}
		u.Entries[i] = e
	}
	if defaults && u.Document.Defaults == nil {
		d := StructureDefaults
		u.Document.Defaults = &d
	}
	return &u
}

func hasDoubt(ds []capture.Doubt, want capture.Doubt) bool {
	for _, d := range ds {
		if d == want {
			return true
		}
	}
	return false
}

// drawnContext reports a doubt about a cube or dice drawn and not read.
func drawnContext(ds []capture.Doubt) bool {
	for _, d := range ds {
		if contextField[d.Field] && d != ContextDoubt {
			return true
		}
	}
	return false
}

// positionContext is the XGID after its board, "" when there is none.
func positionContext(xgid string) string {
	s := strings.TrimPrefix(strings.TrimSpace(xgid), "XGID=")
	if _, ctx, ok := strings.Cut(s, ":"); ok {
		return ctx
	}
	return ""
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
		return bad("version %d, this program reads versions 1 to %d", r.Version, Version)
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
	if df := d.Defaults; df != nil {
		if r.Version < 2 {
			return bad("version 1 has no defaults")
		}
		if _, err := ParsePosition("XGID=" + startingBoard + ":" + df.Context); err != nil {
			return bad("defaults context %q is not the end of an XGID after its board", df.Context)
		}
	}
	if len(r.Entries) == 0 {
		return bad("no entry")
	}
	for i, e := range r.Entries {
		if e.Rank != i+1 {
			return bad("entry %d has rank %d: ranks run from 1 with no gap", i+1, e.Rank)
		}
		if err := r.validateEntry(e); err != nil {
			return bad("rank %d: %v", e.Rank, err)
		}
		if i > 0 && e.Page < r.Entries[i-1].Page {
			return bad("rank %d is on page %d, before rank %d on page %d: entries follow the document", e.Rank, e.Page, i, r.Entries[i-1].Page)
		}
	}
	return nil
}

// startingBoard is a legal board a defaults context is checked on.
const startingBoard = "-b----E-C---eE---c-e----B-"

func (r *Releve) validateEntry(e Entry) error {
	if e.Page < 1 {
		return fmt.Errorf("page %d", e.Page)
	}
	for _, d := range e.Doubts {
		if strings.TrimSpace(d.Field) == "" {
			return fmt.Errorf("a doubt names no field")
		}
	}
	if r.Version < 2 {
		if e.Checkers != "" || e.Context != "" || e.Complete {
			return fmt.Errorf("version 1 has no checkers, context or complete: it has one state")
		}
		switch e.State {
		case Sure, ToVerify, Uncertain:
		default:
			return fmt.Errorf("state %q, want sure, to-verify or uncertain", e.State)
		}
		if e.State == Sure && len(e.Doubts) > 0 {
			return fmt.Errorf("a sure entry has no doubt")
		}
	} else if err := r.validateCertainties(e); err != nil {
		return err
	}
	if r := e.Rect; r != nil {
		if r.X < 0 || r.Y < 0 || r.W <= 0 || r.H <= 0 || r.X+r.W > 1 || r.Y+r.H > 1 {
			return fmt.Errorf("rect %+v is not inside the page (fractions of its size)", *r)
		}
	}
	if e.Position == "" && r.StateOf(e) == Uncertain && !e.Verified {
		return nil // a diagram that could not be read: listed, never importable as is
	}
	_, err := ParsePosition(e.Position)
	return err
}

// validateCertainties holds a version 2 entry's checkers and context to what
// they claim: sure checkers carry no doubt but about the context, a context
// read carries none about it, a context under the defaults carries them.
func (r *Releve) validateCertainties(e Entry) error {
	if e.State != "" {
		return fmt.Errorf("state %q: version 2 has none, the state follows from checkers and context", e.State)
	}
	switch e.Checkers {
	case Sure, ToVerify, Uncertain:
	default:
		return fmt.Errorf("checkers %q, want sure, to-verify or uncertain", e.Checkers)
	}
	switch e.Context {
	case ContextRead, ContextDefault, ContextToVerify, ContextUncertain:
	default:
		return fmt.Errorf("context %q, want read, default, to-verify or uncertain", e.Context)
	}
	for _, d := range e.Doubts {
		switch {
		case !contextField[d.Field] && (e.Checkers == Sure || e.Complete):
			return fmt.Errorf("a doubt on %q: checkers sure or complete have none", d.Field)
		case contextField[d.Field] && e.Context == ContextRead:
			return fmt.Errorf("a doubt on %q: a context read has none", d.Field)
		case contextField[d.Field] && d != ContextDoubt && e.Context == ContextDefault:
			return fmt.Errorf("a doubt on %q: an entry under the defaults has a cube and dice as they say", d.Field)
		}
	}
	if e.Complete && e.Checkers == Uncertain {
		return fmt.Errorf("uncertain checkers are not complete")
	}
	if e.Context == ContextDefault && e.Position != "" {
		if r.Document.Defaults == nil {
			return fmt.Errorf("context default, but the document has no defaults")
		}
		if got := positionContext(e.Position); got != r.Document.Defaults.Context {
			return fmt.Errorf("context default, but the position ends %q, not the defaults %q", got, r.Document.Defaults.Context)
		}
	}
	return nil
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
