package direction

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/PileOfCells/backgammon-tournoi/render"
)

// Translating the engine's codes on the Go side (issue #386).
//
// The engine emits no sentence: a match label, a ranking note, a warning are a code and its
// parameters, and blunderDB renders them in nine languages. The panel does it in JavaScript
// (components/direction/labels.js); the standalone display page is written by Go, so it needs
// the same rendering here.
//
// What is NOT duplicated is the strings: both sides read the SAME catalogue — the `direction`
// block of frontend/src/i18n/locales/<lang>.json, handed over as it is. Only the interpolation
// rule is written twice, and `labeler_test.go` holds the two to the same expectations as
// `directionLabels.test.js`.

// Catalog is one language's `direction` block: a tree of strings the host hands over.
type Catalog struct{ tree map[string]any }

// NewCatalog reads a catalogue from the JSON the frontend already loads.
func NewCatalog(blob []byte) (*Catalog, error) {
	var tree map[string]any
	if err := json.Unmarshal(blob, &tree); err != nil {
		return nil, fmt.Errorf("direction: catalogue: %w", err)
	}
	return &Catalog{tree: tree}, nil
}

// t renders one key with its parameters. A missing key returns the empty string and false, so
// every caller decides for itself what to show instead — never a blank.
func (c *Catalog) t(path string, params map[string]any) (string, bool) {
	if c == nil {
		return "", false
	}
	var cur any = c.tree
	for _, seg := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return "", false
		}
		cur, ok = m[seg]
		if !ok {
			return "", false
		}
	}
	s, ok := cur.(string)
	if !ok {
		return "", false
	}
	return interpolate(s, params), true
}

// interpolate replaces {name} by its parameter, exactly as the frontend's $t does. A parameter
// that is not given is left in place: a hole named {score} says what is missing, an empty one
// says nothing.
func interpolate(s string, params map[string]any) string {
	if len(params) == 0 || !strings.ContainsRune(s, '{') {
		return s
	}
	var b strings.Builder
	for {
		i := strings.IndexRune(s, '{')
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		j := strings.IndexRune(s[i:], '}')
		if j < 0 {
			b.WriteString(s)
			return b.String()
		}
		j += i
		name := s[i+1 : j]
		if v, ok := params[name]; ok {
			b.WriteString(s[:i])
			b.WriteString(asString(v))
		} else {
			b.WriteString(s[:j+1])
		}
		s = s[j+1:]
	}
}

func asString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprint(x)
	}
}

// PlayerNamer gives a player's name; the page shows names, never identifiers.
type PlayerNamer func(tournoi.PlayerID) string

// CatalogLabeler renders the engine's codes through a host catalogue. It satisfies
// render.Labeler, which is what the engine's page renderer asks for.
type CatalogLabeler struct {
	C    *Catalog
	Name PlayerNamer
}

var _ render.Labeler = (*CatalogLabeler)(nil)

// NewLabeler builds a Labeler over a catalogue.
func NewLabeler(c *Catalog, name PlayerNamer) *CatalogLabeler {
	if name == nil {
		name = func(id tournoi.PlayerID) string { return string(id) }
	}
	return &CatalogLabeler{C: c, Name: name}
}

// Label renders a structured label, sub-label included.
func (l *CatalogLabeler) Label(x tournoi.Label) string {
	if x.Kind == "" {
		return ""
	}
	sub := ""
	if x.Sub != nil {
		sub = l.Label(*x.Sub)
	}
	out, ok := l.C.t("label."+string(x.Kind), map[string]any{
		"n": x.N, "losses": x.Losses, "match": x.Match, "players": x.Players,
		"spots": x.Spots, "section": l.SectionName(x.Section), "text": x.Text, "sub": sub,
	})
	if !ok {
		// Un code inconnu s'affiche tel quel : une version future du moteur se verra à
		// l'écran plutôt que de laisser un blanc.
		return string(x.Kind)
	}
	return out
}

// Note renders a ranking note.
func (l *CatalogLabeler) Note(n tournoi.Note) string {
	if n.Kind == "" {
		return ""
	}
	sub := ""
	if n.Sub != nil {
		sub = l.Label(*n.Sub)
	}
	out, ok := l.C.t("note."+string(n.Kind), map[string]any{
		"wins": n.Wins, "losses": n.Losses, "lives": n.Lives,
		"section": l.SectionName(n.Section), "sub": sub,
	})
	if !ok {
		return string(n.Kind)
	}
	return out
}

// Reason renders why a proposal is waiting.
func (l *CatalogLabeler) Reason(r tournoi.ReasonCode) string {
	if r == "" {
		return ""
	}
	out, ok := l.C.t("reason."+string(r), nil)
	if !ok {
		return string(r)
	}
	return out
}

// Warn renders a warning code on its own — the page shows the code's sentence, the panel adds
// the facts.
func (l *CatalogLabeler) Warn(w tournoi.WarningCode) string {
	if w == "" {
		return ""
	}
	out, ok := l.C.t("warning."+string(w), nil)
	if !ok {
		return string(w)
	}
	return out
}

// Warning renders a warning WITH its facts, which is what a reader needs to act on it.
func (l *CatalogLabeler) Warning(w tournoi.Warning) string {
	if w.Code == "" {
		return ""
	}
	out, ok := l.C.t("warning."+string(w.Code), map[string]any{
		"match": string(w.Match), "section": l.SectionName(w.Section), "label": l.Label(w.Label),
		"a": l.Name(w.A), "b": l.Name(w.B),
		"expectedA": l.Name(w.ExpectedA), "expectedB": l.Name(w.ExpectedB),
		"length": w.Length, "scoreA": w.ScoreA, "scoreB": w.ScoreB,
	})
	if !ok {
		return string(w.Code)
	}
	return out
}

// SectionName renders a section identifier. A section name is an IDENTIFIER engine-side —
// "main", "conso", "poule:A" — never a label; this is where it becomes readable.
func (l *CatalogLabeler) SectionName(name string) string {
	if name == "" {
		return ""
	}
	if letter, ok := strings.CutPrefix(name, "poule:"); ok {
		if out, found := l.C.t("section.pool", map[string]any{"letter": letter}); found {
			return out
		}
	}
	if letter, ok := strings.CutPrefix(name, "barrage:"); ok {
		if out, found := l.C.t("section.playoff", map[string]any{"letter": letter}); found {
			return out
		}
	}
	out, ok := l.C.t("section."+name, nil)
	if !ok {
		return name
	}
	return out
}

// PhaseName names a phase. The director's own words come first — a phase they named
// "Consolante du dimanche" is called that, in every language.
func (l *CatalogLabeler) PhaseName(p tournoi.PhaseConfig) string {
	if p.Name != "" {
		return p.Name
	}
	out, ok := l.C.t("format."+p.Kind, nil)
	if !ok {
		return p.Kind
	}
	return out
}

// Term renders a word of the rendering itself — a column heading, a table's state. The count
// goes in as {n}, so a language that needs it has it and one that does not ignores it.
func (l *CatalogLabeler) Term(t render.Term, count int) string {
	out, ok := l.C.t("term."+string(t), map[string]any{"n": count})
	if !ok {
		return string(t)
	}
	return out
}
