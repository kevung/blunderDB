// SPDX-License-Identifier: MIT

package gammonnet

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// The canonical search levels ("instant", "normal", "thorough"): ply,
// filter, prune_k and the QUALITY pruning costs, attached to the same value.
//
// gammonNet is the source of truth: `gn_search_level` (src/gn_search.c) is
// the one table, `data/search_levels.json` its export. The copy here is
// byte-identical, checked against `search_levels.sha256` by
// TestEmbeddedSearchLevelsJSON, so a stale or hand-edited copy fails loudly.
//
//go:embed search_levels.json
var embeddedSearchLevelsJSON []byte

//go:embed search_levels.sha256
var embeddedSearchLevelsSHA256 string

// SearchLevel is one canonical named search shape, quality cost attached.
//
// PruneEquityLoss and its 95% CI are measured upstream (pruned against the
// same search unpruned, 450 decisions) and are exactly 0 where PruneK is 0.
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

// mustLevel is Level for this package's own defaults, where the name is a
// literal: a missing entry is a broken embed, so it panics at init.
func mustLevel(name string) SearchLevel {
	level, ok := Level(name)
	if !ok {
		panic(fmt.Sprintf("gammonnet: le niveau canonique %q est absent de l'export embarqué", name))
	}
	return level
}
