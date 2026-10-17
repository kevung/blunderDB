package openingbook

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Candidate is one play of a reply position, rolled out.
type Candidate struct {
	// Move is the play in blunderDB's exact notation, every hit marked: the
	// name the rollout resolved, so two plays never share it.
	Move string `json:"move"`
	// Equity is normalised equity at the score (money points at money play),
	// the side on roll's (ADR-0019).
	Equity float64 `json:"equity"`
	// StdErr is the standard error of Equity.
	StdErr float64 `json:"std_err"`
	// Games is how many games this play rolled before it stopped: fewer than
	// the line's when the JSD rule dropped it early.
	Games int `json:"games"`
	// JSD is the rollout's own gap to the best in standard deviations of the
	// difference, the quantity its stopping rule reads.
	JSD float64 `json:"jsd"`
}

// Line is the book's unit: one position at one score, either the opening
// move (Reply and OpeningPlay empty) or a reply to an opening play.
type Line struct {
	Opening string `json:"opening"` // the opening roll, "31"
	// OpeningPlay is the opening play the reply follows, a candidate of the
	// opening line at the same score; empty on an opening line.
	OpeningPlay string `json:"opening_play,omitempty"`
	Reply       string `json:"reply,omitempty"` // the reply roll, "52" or "66"; empty on an opening line
	Score       string `json:"score"`           // Score.String: "4-2", "1-4", "money"
	XGID        string `json:"xgid"`            // the position, to roll it again
	// EngineVersion and Signature are the rollout's: two lines with the same
	// signature were computed alike and can be compared.
	EngineVersion string `json:"engine_version"`
	Signature     string `json:"signature"`
	// ExactBearoff says whether the exact bearoff table was used: the
	// signature's "exact bearoff when covered" states a policy, not whether
	// the table was there, and the generator leaves it out on purpose.
	ExactBearoff bool   `json:"exact_bearoff"`
	Seed         uint64 `json:"seed"`
	Games        int    `json:"games"`
	Stop         string `json:"stop"`
	// Candidates are sorted best first.
	Candidates []Candidate `json:"candidates"`
}

// Key is what a Line answers: an opening roll at a score (an opening line,
// Play empty and Reply zero), or a reply roll after the opening play Play.
type Key struct {
	Opening Roll
	Play    string
	Reply   Roll
	Score   Score
}

// OpeningKey is the key of the opening line of opening at s.
func OpeningKey(opening Roll, s Score) Key { return Key{Opening: opening, Score: s} }

// ReplyKey is the key of the line of reply after play, opening at s.
func ReplyKey(opening Roll, play string, reply Roll, s Score) Key {
	return Key{Opening: opening, Play: play, Reply: reply, Score: s}
}

// IsOpening says the key names an opening line.
func (k Key) IsOpening() bool { return k.Reply == Roll{} }

func (k Key) String() string {
	if k.IsOpening() {
		return k.Opening.String() + "@" + k.Score.String()
	}
	return k.Opening.String() + " " + k.Play + " / " + k.Reply.String() + "@" + k.Score.String()
}

// Position is the position the key's line rolls out.
func (k Key) Position() (domain.Position, error) {
	if k.IsOpening() {
		return OpeningPosition(k.Opening, k.Score)
	}
	return ReplyPosition(k.Opening, k.Play, k.Reply, k.Score)
}

// Key reads the line's key back from its fields.
func (l *Line) Key() (Key, error) {
	o, err := ParseRoll(l.Opening)
	if err != nil {
		return Key{}, err
	}
	s, err := ParseScore(l.Score)
	if err != nil {
		return Key{}, err
	}
	if l.Reply == "" {
		if l.OpeningPlay != "" {
			return Key{}, fmt.Errorf("openingbook: opening line %s names an opening play", l.Opening)
		}
		return OpeningKey(o, s), nil
	}
	r, err := ParseRoll(l.Reply)
	if err != nil {
		return Key{}, err
	}
	if l.OpeningPlay == "" {
		return Key{}, fmt.Errorf("openingbook: reply line %s/%s names no opening play", l.Opening, l.Reply)
	}
	return ReplyKey(o, l.OpeningPlay, r, s), nil
}

// The tie rule of the book: a play is level with the best when its gap is
// under tieZ standard deviations of the difference plus tieBias, the bias
// measured of the truncated 0-ply rollout against a full 2-ply one. Refining
// below it would measure the setting, not the play.
const (
	tieZ    = 1.96
	tieBias = 0.005
)

// Ranked is one play read against the best of its line.
type Ranked struct {
	Candidate
	// Gap is the best play's equity minus this one's, ≥ 0.
	Gap float64 `json:"gap"`
	// SigmaDiff is the standard error of Gap, the two estimates taken as
	// independent as the rollout's JSD takes them: common dice correlate
	// them positively, so this overstates it and the tie is conservative.
	SigmaDiff float64 `json:"sigma_diff"`
	// CI95 is the half-width of Gap's 95 % interval.
	CI95 float64 `json:"ci95"`
	// Tied says this play cannot be told from the best: Gap < CI95 + tieBias.
	// The best play is tied with itself.
	Tied bool `json:"tied"`
}

// Rank reads every candidate of the line against the best, best first.
// Every tied play is a right answer to « which play at this score? ».
func (l *Line) Rank() []Ranked {
	if len(l.Candidates) == 0 {
		return nil
	}
	cs := append([]Candidate(nil), l.Candidates...)
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].Equity > cs[j].Equity })
	best := cs[0]
	out := make([]Ranked, len(cs))
	for i, c := range cs {
		r := Ranked{Candidate: c, Gap: best.Equity - c.Equity}
		if i > 0 {
			r.SigmaDiff = math.Sqrt(best.StdErr*best.StdErr + c.StdErr*c.StdErr)
		}
		r.CI95 = tieZ * r.SigmaDiff
		r.Tied = i == 0 || r.Gap < r.CI95+tieBias
		out[i] = r
	}
	return out
}

// Book is a set of lines, one per key.
type Book struct {
	lines map[Key]Line
}

// Lookup returns the line of a key.
func (b *Book) Lookup(k Key) (Line, bool) {
	if b == nil {
		return Line{}, false
	}
	l, ok := b.lines[k]
	return l, ok
}

// Len is the number of lines.
func (b *Book) Len() int {
	if b == nil {
		return 0
	}
	return len(b.lines)
}

// Lines returns the lines in key order: opening, reply, score.
func (b *Book) Lines() []Line {
	keys := make([]Key, 0, b.Len())
	for k := range b.lines {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keyLess(keys[i], keys[j]) })
	out := make([]Line, len(keys))
	for i, k := range keys {
		out[i] = b.lines[k]
	}
	return out
}

func keyLess(a, b Key) bool {
	switch {
	case a.Opening != b.Opening:
		return rollLess(a.Opening, b.Opening)
	case a.Play != b.Play:
		return a.Play < b.Play
	case a.Reply != b.Reply:
		return rollLess(a.Reply, b.Reply)
	case a.Score.Replier != b.Score.Replier:
		return a.Score.Replier < b.Score.Replier
	}
	return a.Score.Opener < b.Score.Opener
}

func rollLess(a, b Roll) bool {
	if a.Hi != b.Hi {
		return a.Hi < b.Hi
	}
	return a.Lo < b.Lo
}

// Load reads JSON Lines, strictly: a line that does not parse, or a key seen
// twice, is an error. The embedded book is compiled, never torn.
func Load(r io.Reader) (*Book, error) { return readLines(r, false) }

// Compile reads a run's file, which may hold a key twice when two runs
// shared it, and returns its book, the later line of a key kept.
func Compile(r io.Reader) (*Book, error) { return readLines(r, true) }

func readLines(r io.Reader, laterWins bool) (*Book, error) {
	b := &Book{lines: map[Key]Line{}}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	n := 0
	for sc.Scan() {
		n++
		raw := bytes.TrimSpace(sc.Bytes())
		if len(raw) == 0 {
			continue
		}
		var l Line
		if err := json.Unmarshal(raw, &l); err != nil {
			return nil, fmt.Errorf("openingbook: line %d: %w", n, err)
		}
		k, err := l.Key()
		if err != nil {
			return nil, fmt.Errorf("openingbook: line %d: %w", n, err)
		}
		if _, dup := b.lines[k]; dup && !laterWins {
			return nil, fmt.Errorf("openingbook: line %d: %s appears twice", n, k)
		}
		b.lines[k] = l
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("openingbook: %w", err)
	}
	return b, nil
}

// Write writes the book as JSON Lines in key order, the form Load reads and
// the embedded file holds.
func (b *Book) Write(w io.Writer) error {
	enc := lineEncoder(w)
	for _, l := range b.Lines() {
		if err := enc.Encode(l); err != nil {
			return err
		}
	}
	return nil
}

// lineEncoder writes one Line per line, its signature's "<=" left as typed.
func lineEncoder(w io.Writer) *json.Encoder {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc
}

//go:embed book.jsonl.gz
var embedded []byte

var (
	defaultOnce sync.Once
	defaultBook *Book
	defaultErr  error
)

// Default is the book embedded in the binary; empty until it is computed.
func Default() (*Book, error) {
	defaultOnce.Do(func() {
		zr, err := gzip.NewReader(bytes.NewReader(embedded))
		if errors.Is(err, io.EOF) {
			defaultBook = &Book{lines: map[Key]Line{}}
			return
		}
		if err != nil {
			defaultErr = fmt.Errorf("openingbook: embedded book: %w", err)
			return
		}
		defer func() { _ = zr.Close() }()
		defaultBook, defaultErr = Load(zr)
	})
	return defaultBook, defaultErr
}
