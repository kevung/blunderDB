// SPDX-License-Identifier: MIT

package gammonnet

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// The canonical search levels ("instant", "normal", "thorough"): ply,
// filter, prune_k, and — the whole point of issue #25 — the QUALITY that
// pruning costs, attached to the same value instead of living in a comment
// three files away. Before this, ply=2/filter=(0,1,3)/prune_k=12 were
// retyped by hand up to five times across this repo, gammonNet and gammonGo,
// and downstream once shipped a `prune_k = 3` "fast" mode with none of the
// measurement that would have shown it costs seventeen times as much.
//
// gammonNet is the source of truth (ADR-0003 there): `gn_search_level`
// (its `src/gn_search.c`) is the ONE table these numbers are typed into, and
// `data/search_levels.json` its export. `search_levels.json` in this
// directory is a byte-identical copy — the same discipline `met.go`'s
// canonical export already applies to the match equity table (issue #24) —
// verified against `search_levels.sha256` by TestEmbeddedSearchLevelsJSON in
// this package, so a hand-edited or stale copy fails loudly instead of
// quietly disagreeing with what gammonNet ships.
//
//go:embed search_levels.json
var embeddedSearchLevelsJSON []byte

//go:embed search_levels.sha256
var embeddedSearchLevelsSHA256 string

// SearchLevel is one canonical named search shape, quality cost attached.
//
// PruneEquityLoss and its 95% CI are measured
// (gammonNet's docs/mesures/2026-08-26-T3A-regroupement.md, 450 decisions —
// 300 contact, 150 race — at 2-ply filter (0,1,3), pruned search against the
// SAME search unpruned) and are exactly 0 wherever PruneK is 0: nothing is
// lost by a mechanism that is off.
type SearchLevel struct {
	Name                  string
	Ply                   int
	Filter                []int
	PruneK                int
	PruneEquityLoss       float64
	PruneEquityLossCILow  float64
	PruneEquityLossCIHigh float64
}

type searchLevelExport struct {
	Ply               int       `json:"ply"`
	Filter            []int     `json:"filter"`
	PruneK            int       `json:"prune_k"`
	PruneEquityLoss   float64   `json:"prune_equity_loss"`
	PruneEquityLossCI []float64 `json:"prune_equity_loss_ci"`
}

type searchLevelsExport struct {
	Levels map[string]searchLevelExport `json:"levels"`
}

var searchLevels = parseSearchLevels(embeddedSearchLevelsJSON)

func parseSearchLevels(raw []byte) map[string]SearchLevel {
	var export searchLevelsExport
	if err := json.Unmarshal(raw, &export); err != nil {
		// The embedded file is a build asset, not user input: a parse
		// failure here means the copy is corrupt, and no caller of Level()
		// could recover from a canonical level that does not exist.
		panic(fmt.Sprintf("gammonnet: search_levels.json embarqué est invalide : %v", err))
	}
	out := make(map[string]SearchLevel, len(export.Levels))
	for name, entry := range export.Levels {
		level := SearchLevel{
			Name:            name,
			Ply:             entry.Ply,
			Filter:          entry.Filter,
			PruneK:          entry.PruneK,
			PruneEquityLoss: entry.PruneEquityLoss,
		}
		if len(entry.PruneEquityLossCI) == 2 {
			level.PruneEquityLossCILow = entry.PruneEquityLossCI[0]
			level.PruneEquityLossCIHigh = entry.PruneEquityLossCI[1]
		}
		out[name] = level
	}
	return out
}

// Level returns gammonNet's canonical named search level ("instant",
// "normal", "thorough"), or false if name is not one of them — never a
// guessed default standing in for a typo.
func Level(name string) (SearchLevel, bool) {
	level, ok := searchLevels[name]
	return level, ok
}

// mustLevel is Level for this package's own defaults below, where the name
// is a literal this file controls, not caller input: a missing entry is a
// broken embed, and DefaultPruneK failing to compile a value would be a
// worse failure mode than panicking at package init with a name attached.
func mustLevel(name string) SearchLevel {
	level, ok := Level(name)
	if !ok {
		panic(fmt.Sprintf("gammonnet: le niveau canonique %q est absent de l'export embarqué", name))
	}
	return level
}
