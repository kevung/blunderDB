package rollout

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// EngineVersion names this rollout as a Configuration (CONTEXT.md): the
// rollout procedure on top of the gammonNet Configuration it plays with.
// A change to the procedure that moves a number (dice, variance reduction,
// cube policy, leaf valuation) bumps the first part; the second follows the
// network.
const EngineVersion = "blunderDB rollout v1 / " + gammonnet.EngineVersion

// DefaultSeed is the seed a rollout uses when none is named, so the same
// command on the same position prints the same numbers.
const DefaultSeed uint64 = 0x5EED_B10D_E7DB

// maxGames bounds a request: beyond it a rollout is a background job of
// hours, which this entry point is not.
const maxGames = 36 * 36 * 36 * 16

// Settings are a rollout's parameters. Every field but Workers enters the
// result: two rollouts with equal Settings (Workers aside) on the same
// position print the same numbers, bit for bit.
type Settings struct {
	// Truncation is how many half-moves each game plays before its position
	// is valued by the engine; 0 plays every game to its end. A game that
	// reaches a bearoff the exact table covers stops there whatever this says.
	Truncation int `json:"truncation"`
	// MinGames is how many games every candidate plays before the JSD rule
	// may stop it; MaxGames is the most any candidate plays. Both are
	// multiples of 36, which keeps the first roll stratified.
	MinGames int `json:"min_games"`
	MaxGames int `json:"max_games"`
	// JSDLimit stops a candidate once its gap to the best, in standard
	// deviations of the difference, reaches it; 0 never stops early.
	JSDLimit float64 `json:"jsd_limit"`
	// Ply is the gammonNet depth of every decision inside a game — plays,
	// cube actions — and of the truncation leaf.
	Ply int `json:"ply"`
	// Candidates is how many plays a checker rollout rolls when none is
	// named, best first at max(Ply, 2).
	Candidates int `json:"candidates"`
	// Seed fixes every die of every game.
	Seed uint64 `json:"seed"`
	// Workers is the number of games played at once; 0 means one per core.
	// It changes the time, never the numbers. Ignored when Options.Exec is
	// set: the caller's Exec decides how many games run at once.
	Workers int `json:"workers"`
}

// Fast is the « Rapide » preset: a first sort, to confirm a hint or drop a
// clear loser — truncated at 7 half-moves, 216 games, stop at JSD 3 after 108.
func Fast() Settings {
	return Settings{Truncation: 7, MinGames: 108, MaxGames: 216, JSDLimit: 3, Ply: 0, Candidates: 5, Seed: DefaultSeed}
}

// Standard is the « Standard » preset: 1296 games (every ordered pair of
// opening rolls once), stop at JSD 3 after 324, truncated at 11 half-moves
// as gnubg's default rollout is.
func Standard() Settings {
	return Settings{Truncation: 11, MinGames: 324, MaxGames: 1296, JSDLimit: 3, Ply: 0, Candidates: 5, Seed: DefaultSeed}
}

// Preset returns a named preset: "fast" (rapide) or "standard". A custom
// (libre) rollout is any Settings value.
func Preset(name string) (Settings, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "fast", "rapide":
		return Fast(), true
	case "standard":
		return Standard(), true
	}
	return Settings{}, false
}

// Validate refuses settings no rollout can honour.
func (s Settings) Validate() error {
	switch {
	case s.MaxGames < 1 || s.MaxGames > maxGames:
		return fmt.Errorf("rollout: max games %d outside 1..%d", s.MaxGames, maxGames)
	case s.MinGames < 0 || s.MinGames > s.MaxGames:
		return fmt.Errorf("rollout: min games %d outside 0..%d", s.MinGames, s.MaxGames)
	case s.MaxGames%36 != 0 || s.MinGames%36 != 0:
		// The first roll is stratified over 36 games: any other count
		// leaves some opening rolls over-represented.
		return fmt.Errorf("rollout: min games %d and max games %d must be multiples of 36", s.MinGames, s.MaxGames)
	case s.Truncation < 0:
		return fmt.Errorf("rollout: truncation %d is negative", s.Truncation)
	case s.JSDLimit < 0:
		return fmt.Errorf("rollout: JSD limit %g is negative", s.JSDLimit)
	case s.Ply < 0 || s.Ply > gammonnet.MaxPly:
		return fmt.Errorf("rollout: ply %d outside 0..%d", s.Ply, gammonnet.MaxPly)
	case s.Candidates < 0:
		return fmt.Errorf("rollout: candidates %d is negative", s.Candidates)
	case s.Workers < 0:
		return fmt.Errorf("rollout: workers %d is negative", s.Workers)
	}
	return nil
}

// DepthLabel is the AnalysisDepth a stored rollout carries. It contains
// "Rollout", which domain.AnalysisDepthRank places above every ply — and
// must not end in "-ply", which that rank would read as a depth.
func (s Settings) DepthLabel() string {
	trunc := "untruncated"
	if s.Truncation > 0 {
		trunc = fmt.Sprintf("truncated %d", s.Truncation)
	}
	return fmt.Sprintf("Rollout %d games (%s, %s)", s.MaxGames, gammonnet.DepthLabel(s.Ply), trunc)
}

// Signature is the full line a rollout of the Candidates best plays is
// reproduced from: everything that moves a number, in one place.
func (s Settings) Signature() string { return s.SignatureFor(nil) }

// SignatureFor is the Signature of a rollout of the plays moves names, or of
// the Candidates best when moves is empty. Which plays are rolled is part of
// it: two rollouts of different plays are two Configurations, never a longer
// and a shorter series of one. Neither the order moves are named in nor the
// spacing inside a name is.
func (s Settings) SignatureFor(moves []string) string {
	plays := fmt.Sprintf("candidates %d best", s.Candidates)
	if len(moves) > 0 {
		norm := make([]string, len(moves))
		for i, m := range moves {
			norm[i] = strings.Join(strings.Fields(m), " ")
		}
		plays = "candidates {" + strings.Join(slices.Compact(slices.Sorted(slices.Values(norm))), ", ") + "}"
	}
	return s.signature(plays)
}

// CubeSignature is the Signature of a rollout of a cube decision: its two
// branches are fixed, so the number of candidates moves nothing.
func (s Settings) CubeSignature() string { return s.signature("cube decision") }

// SignatureAt is the Signature a rollout of pos with s, no play named,
// carries: the cube's when pos has no dice, the best plays' otherwise.
func (s Settings) SignatureAt(pos *domain.Position) string {
	if hasDice(pos) {
		return s.Signature()
	}
	return s.CubeSignature()
}

func (s Settings) signature(plays string) string {
	trunc := "none"
	if s.Truncation > 0 {
		trunc = fmt.Sprintf("%d half-moves", s.Truncation)
	}
	stop := "off"
	if s.JSDLimit > 0 {
		stop = fmt.Sprintf("JSD >= %g after %d games", s.JSDLimit, s.MinGames)
	}
	return fmt.Sprintf("%s; cubeful; %s plays, cube and leaves; %s; variance reduction 1-ply; "+
		"common quasi-random dice (2 plies); seed %d; games <= %d; stop %s; truncation %s; exact bearoff when covered",
		EngineVersion, gammonnet.DepthLabel(s.Ply), plays, s.Seed, s.MaxGames, stop, trunc)
}

// ParseSpec reads settings from one line of text, the form `analyze --rollout`,
// the /v1 routes and the MCP tool share: an optional preset name first
// ("fast"/"rapide", "standard", or "custom"/"libre", which starts from fast),
// then key=value overrides — games, min-games, truncation, jsd, ply,
// candidates, seed — separated by commas or spaces. "games" lowers min-games
// with it, as the rollout command's --games does. The result is validated.
func ParseSpec(spec string) (Settings, error) {
	fields := strings.FieldsFunc(spec, func(r rune) bool { return r == ',' || r == ' ' || r == ';' })
	s := Fast()
	for i, f := range fields {
		key, value, isPair := strings.Cut(f, "=")
		if !isPair {
			if i != 0 {
				return Settings{}, fmt.Errorf("rollout: %q: a preset name comes first", f)
			}
			switch strings.ToLower(f) {
			case "custom", "libre":
				continue
			}
			p, ok := Preset(f)
			if !ok {
				return Settings{}, fmt.Errorf("rollout: unknown preset %q (fast, standard, custom)", f)
			}
			s = p
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "jsd" {
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return Settings{}, fmt.Errorf("rollout: jsd=%q: %w", value, err)
			}
			s.JSDLimit = v
			continue
		}
		if key == "seed" {
			v, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return Settings{}, fmt.Errorf("rollout: seed=%q: %w", value, err)
			}
			s.Seed = v
			continue
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			return Settings{}, fmt.Errorf("rollout: %s=%q: %w", key, value, err)
		}
		switch key {
		case "games":
			s.MaxGames = v
			s.MinGames = min(s.MinGames, v)
		case "min-games", "min_games":
			s.MinGames = v
		case "truncation":
			s.Truncation = v
		case "ply":
			s.Ply = v
		case "candidates":
			s.Candidates = v
		default:
			return Settings{}, fmt.Errorf("rollout: unknown setting %q (games, min-games, truncation, jsd, ply, candidates, seed)", key)
		}
	}
	if err := s.Validate(); err != nil {
		return Settings{}, err
	}
	return s, nil
}
