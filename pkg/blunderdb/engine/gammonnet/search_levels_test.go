// SPDX-License-Identifier: MIT

package gammonnet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEmbeddedSearchLevelsJSONMatchesItsChecksumPin holds the embedded copy
// to the checksum gammonNet published with it (search_levels.sha256): a
// hand-edited or stale copy fails here.
func TestEmbeddedSearchLevelsJSONMatchesItsChecksumPin(t *testing.T) {
	sum := sha256.Sum256(embeddedSearchLevelsJSON)
	got := hex.EncodeToString(sum[:])
	want := strings.TrimSpace(embeddedSearchLevelsSHA256)
	if got != want {
		t.Fatalf("search_levels.json ne correspond plus à search_levels.sha256 : "+
			"calculé %s, épinglé %s — les deux doivent être recopiés ensemble "+
			"depuis gammonNet (data/search_levels.json, tools/extract_search_levels.py)",
			got, want)
	}
}

// TestDefaultsDeriveFromTheEmbeddedNormalLevel: the package defaults are the
// "normal" level's own fields, not a second copy.
func TestDefaultsDeriveFromTheEmbeddedNormalLevel(t *testing.T) {
	normal, ok := Level("normal")
	if !ok {
		t.Fatal("le niveau canonique « normal » est absent de l'export embarqué")
	}
	if DefaultPly != normal.Ply {
		t.Errorf("DefaultPly = %d, want %d", DefaultPly, normal.Ply)
	}
	if DefaultPruneK != normal.PruneK {
		t.Errorf("DefaultPruneK = %d, want %d", DefaultPruneK, normal.PruneK)
	}
	if DefaultPruneEquityLoss != normal.PruneEquityLoss {
		t.Errorf("DefaultPruneEquityLoss = %v, want %v", DefaultPruneEquityLoss, normal.PruneEquityLoss)
	}
	cfg := DefaultConfig(normal.Ply)
	for i, want := range normal.Filter {
		if cfg.Filter[i] != want {
			t.Errorf("DefaultConfig(%d).Filter[%d] = %d, want %d (normal.Filter)",
				normal.Ply, i, cfg.Filter[i], want)
		}
	}
}

func TestLevelRefusesAnUnknownName(t *testing.T) {
	if _, ok := Level("fast"); ok {
		t.Fatal(`Level("fast") should not exist — the ungauged prune_k=3 ` +
			"level a downstream consumer once shipped without measurement " +
			"(issue #25) was never a canonical level here")
	}
}

// TestCanonicalFormsAgreeWithGammonNet fails if the embedded copy drifts from
// gammonNet's own `data/search_levels.json`. It runs only when that file is
// found: GAMMONNET_SEARCH_LEVELS_JSON, or a sibling `../gammonNet` checkout.
func TestCanonicalFormsAgreeWithGammonNet(t *testing.T) {
	path := os.Getenv("GAMMONNET_SEARCH_LEVELS_JSON")
	if path == "" {
		// Five levels up reaches this repository's parent, where a sibling
		// gammonNet checkout lives.
		candidates := []string{
			filepath.Join("..", "..", "..", "..", "..", "gammonNet", "data", "search_levels.json"),
			filepath.Join("..", "..", "..", "..", "..", "gammonNet-formes-canoniques", "data", "search_levels.json"),
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
		}
	}
	if path == "" {
		t.Skip("aucun checkout gammonNet trouvé à côté de celui-ci " +
			"(ni GAMMONNET_SEARCH_LEVELS_JSON défini) ; garde ignoré")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture de %s : %v", path, err)
	}
	upstream := parseSearchLevels(raw)
	if len(upstream) == 0 {
		t.Fatalf("%s ne contient aucun niveau — export gammonNet cassé ou vide ?", path)
	}
	for name, want := range upstream {
		got, ok := Level(name)
		if !ok {
			t.Errorf("gammonNet publie le niveau %q, absent de la copie embarquée de blunderDB "+
				"— régénérer pkg/blunderdb/engine/gammonnet/search_levels.json "+
				"(tools/extract_search_levels.py côté gammonNet)", name)
			continue
		}
		if got.Ply != want.Ply {
			t.Errorf("%s: Ply = %d, gammonNet publie %d", name, got.Ply, want.Ply)
		}
		if len(got.Filter) != len(want.Filter) {
			t.Errorf("%s: Filter = %v, gammonNet publie %v", name, got.Filter, want.Filter)
		} else {
			for i := range got.Filter {
				if got.Filter[i] != want.Filter[i] {
					t.Errorf("%s: Filter = %v, gammonNet publie %v", name, got.Filter, want.Filter)
					break
				}
			}
		}
		if got.PruneK != want.PruneK {
			t.Errorf("%s: PruneK = %d, gammonNet publie %d", name, got.PruneK, want.PruneK)
		}
		if math.Abs(got.PruneEquityLoss-want.PruneEquityLoss) > 1e-12 {
			t.Errorf("%s: PruneEquityLoss = %v, gammonNet publie %v",
				name, got.PruneEquityLoss, want.PruneEquityLoss)
		}
		if math.Abs(got.PruneEquityLossCILow-want.PruneEquityLossCILow) > 1e-12 ||
			math.Abs(got.PruneEquityLossCIHigh-want.PruneEquityLossCIHigh) > 1e-12 {
			t.Errorf("%s: PruneEquityLossCI = [%v, %v], gammonNet publie [%v, %v]",
				name, got.PruneEquityLossCILow, got.PruneEquityLossCIHigh,
				want.PruneEquityLossCILow, want.PruneEquityLossCIHigh)
		}
	}
	if len(upstream) != len(searchLevels) {
		t.Errorf("gammonNet publie %d niveaux, la copie embarquée en porte %d — "+
			"un niveau a été ajouté ou retiré d'un côté sans l'autre",
			len(upstream), len(searchLevels))
	}
}

// Sanity on the JSON shape itself, so a future change to the export format
// fails with a clear message rather than a silent zero-value struct.
func TestParseSearchLevelsRejectsGarbage(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("parseSearchLevels(garbage) should panic, not return a zero-value map")
		}
	}()
	parseSearchLevels([]byte("not json"))
	_ = json.Marshal // keep import used if the panic path changes
}
