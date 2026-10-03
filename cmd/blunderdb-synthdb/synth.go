package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// maxScoreShift bounds the away-score offset a variant adds. The Zobrist
// hash clamps each away score to 63, so a fixture's longest match plus the
// shift must stay below it or variants would collapse onto the same rows.
const maxScoreShift = 40

// Options drive one generation run.
type Options struct {
	Fixtures  []string // .xg match files, match play only (money files are skipped)
	Positions int      // stop once at least this many positions were written
	Seed      uint64
	Progress  func(matches, positions int)
}

// Result reports what a run wrote.
type Result struct {
	Matches   int
	Positions int // positions the writes reported saved, before cross-match dedup
	Skipped   []string
}

type fixture struct {
	path     string
	maxAway  int
	baseHash string
}

// Generate writes synthetic matches into store until opts.Positions is
// reached. Each match is a committed fixture re-mapped from disk, with
// fictional player names, event and date drawn from the seed, and both
// players' away scores shifted by a per-variant offset: score is part of a
// position's Zobrist identity, so a shifted variant lands on new position
// rows instead of deduplicating onto the fixture's. Dice are left alone —
// changing them would make the recorded moves illegal.
func Generate(ctx context.Context, store storage.Storage, opts Options) (Result, error) {
	var res Result
	fixtures, skipped, err := loadFixtures(opts.Fixtures)
	if err != nil {
		return res, err
	}
	res.Skipped = skipped
	if len(fixtures) == 0 {
		return res, fmt.Errorf("no match-play .xg fixture among %d files", len(opts.Fixtures))
	}
	rng := rand.New(rand.NewPCG(opts.Seed, opts.Seed^0x9e3779b97f4a7c15))
	base := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

	for k := 0; res.Positions < opts.Positions; k++ {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		fx := fixtures[k%len(fixtures)]
		variant := k / len(fixtures)
		width := maxScoreShift + 1
		d1, d2 := variant%width, (variant/width)%width
		if variant >= width*width {
			return res, fmt.Errorf("fixtures exhausted after %d matches: add fixtures or lower -positions", k)
		}

		graph, err := ingest.MapXG(fx.path)
		if err != nil {
			return res, fmt.Errorf("%s: %w", fx.path, err)
		}
		p1 := rng.IntN(len(firstNames) * len(lastNames))
		p2 := (p1 + 1 + rng.IntN(len(firstNames)*len(lastNames)-1)) % (len(firstNames) * len(lastNames))
		perturb(graph, variant, d1, d2, playerName(p1), playerName(p2),
			fmt.Sprintf("Synthetic Open %02d", rng.IntN(60)+1),
			base.AddDate(0, 0, rng.IntN(3650)), fx.baseHash)

		tx, err := store.BeginTx(ctx)
		if err != nil {
			return res, err
		}
		wr, err := ingest.WriteMatch(ctx, tx, "", graph, nil)
		if err != nil {
			_ = tx.Rollback()
			return res, fmt.Errorf("match %d (%s): %w", k, filepath.Base(fx.path), err)
		}
		if err := tx.Commit(); err != nil {
			return res, err
		}
		res.Matches++
		res.Positions += wr.SavedPositions
		if opts.Progress != nil {
			opts.Progress(res.Matches, res.Positions)
		}
	}
	return res, nil
}

func loadFixtures(paths []string) ([]fixture, []string, error) {
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	var out []fixture
	var skipped []string
	for _, p := range sorted {
		g, err := ingest.MapXG(p)
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s: %v", p, err))
			continue
		}
		if g.Match.MatchLength <= 0 {
			skipped = append(skipped, p+": money game, no score to shift")
			continue
		}
		maxAway := 0
		for _, gm := range g.Games {
			for _, mv := range gm.Moves {
				if mv.Position == nil {
					continue
				}
				maxAway = max(maxAway, mv.Position.Score[0], mv.Position.Score[1])
			}
		}
		if maxAway+maxScoreShift > 63 {
			skipped = append(skipped, fmt.Sprintf("%s: %d-away exceeds the hash's score range once shifted", p, maxAway))
			continue
		}
		out = append(out, fixture{path: p, maxAway: maxAway, baseHash: g.Match.MatchHash})
	}
	return out, skipped, nil
}

// perturb rewrites graph in place into variant number variant of its fixture.
func perturb(graph *ingest.MatchGraph, variant, d1, d2 int, p1, p2, event string, date time.Time, baseHash string) {
	m := &graph.Match
	m.Player1Name, m.Player2Name = p1, p2
	m.Event, m.TournamentName = event, ""
	m.Location, m.Round, m.Comment = "", "", ""
	m.MatchDate = date
	m.MatchLength += int32(max(d1, d2))
	m.FilePath = fmt.Sprintf("synthetic/%s#%d", filepath.Base(m.FilePath), variant)
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%s|%s", baseHash, variant, p1, p2)))
	m.MatchHash = "synth-" + hex.EncodeToString(sum[:16])
	m.CanonicalHash = m.MatchHash
	for gi := range graph.Games {
		for mi := range graph.Games[gi].Moves {
			pos := graph.Games[gi].Moves[mi].Position
			if pos == nil {
				continue
			}
			if pos.Score[0] > 0 {
				pos.Score[0] += d1
			}
			if pos.Score[1] > 0 {
				pos.Score[1] += d2
			}
		}
	}
}

// Fictional names only: the generated base ships in benchmarks and CI
// artefacts, and must never carry a real player's name.
var firstNames = strings.Fields("Alix Bastien Céleste Dorian Elsa Fabien Gaëlle Hugo Inès Jules Katia Léon Maëlle Nils Ophélie Pablo Quentin Rosalie Sacha Timothée Ugo Victoire Wanda Xavier Yasmine Zacharie")
var lastNames = strings.Fields("Aubépine Bruyère Coquelicot Digitale Églantine Fougère Gentiane Houblon Iris Jacinthe Lavande Mélilot Narcisse Orchis Pervenche Renoncule Sauge Trèfle Valériane Verveine")

func playerName(i int) string {
	return firstNames[i%len(firstNames)] + " " + lastNames[i/len(firstNames)%len(lastNames)]
}
