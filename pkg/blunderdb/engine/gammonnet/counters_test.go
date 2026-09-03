// SPDX-License-Identifier: MIT

package gammonnet

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Les compteurs (Counters, BatchFill, CubeValuations, ResetCounters) et
// KernelName sont la SURFACE DE SONDE : rien hors de ce paquet ne les appelle,
// et #198 les garde pour cette raison — mais seuls les fichiers de mesure, tous
// derrière une variable d'environnement, les touchaient, si bien qu'ils étaient
// à 0 % sur chaque exécution ordinaire. Un compteur faux ne casse rien : il
// rend une mesure fausse, ce qui est pire.
//
// Ce test tourne toujours et coûte une recherche 0-ply.

func countersTestSearcher(t *testing.T, workers int) (*Searcher, Position) {
	t.Helper()
	dp, err := domain.DecodeXGID(openingXGID)
	if err != nil {
		t.Fatalf("DecodeXGID: %v", err)
	}
	dp.PlayerOnRoll = domain.White
	p, err := FromDomain(&dp)
	if err != nil {
		t.Fatalf("FromDomain: %v", err)
	}
	s, err := NewSearcher(DefaultConfig(0))
	if err != nil {
		t.Fatalf("NewSearcher: %v", err)
	}
	if workers > 1 {
		s = s.WithWorkers(workers)
	}
	return s, p
}

func TestSearcherCountersCountAndReset(t *testing.T) {
	s, p := countersTestSearcher(t, 1)

	if evals, prune, hits := s.Counters(); evals != 0 || prune != 0 || hits != 0 {
		t.Fatalf("un chercheur neuf compte déjà %d/%d/%d", evals, prune, hits)
	}
	if filled, slotted := s.BatchFill(); filled != 0 || slotted != 0 {
		t.Fatalf("un chercheur neuf a déjà rempli %d/%d voies", filled, slotted)
	}
	if n := s.CubeValuations(); n != 0 {
		t.Fatalf("un chercheur neuf a déjà valué %d fois le videau", n)
	}

	pos := p
	if _, ok, err := s.BestPlay(&pos, 3, 1); err != nil || !ok {
		t.Fatalf("BestPlay: ok=%v err=%v", ok, err)
	}

	evals, _, _ := s.Counters()
	if evals == 0 {
		t.Fatal("aucune évaluation comptée après une recherche")
	}
	filled, slotted := s.BatchFill()
	if filled == 0 || slotted < filled {
		t.Fatalf("remplissage de lot incohérent: %d remplies pour %d voies", filled, slotted)
	}
	if slotted%uint64(EvalBatchWidth) != 0 {
		t.Errorf("les voies réservées (%d) ne sont pas un multiple de la largeur de lot (%d)",
			slotted, EvalBatchWidth)
	}

	// Une seconde recherche identique n'évalue plus rien de neuf : le cache
	// répond. C'est ce que « un succès de cache est une évaluation qui n'a pas
	// eu lieu » veut dire, et c'est vérifiable.
	before, _, hitsBefore := s.Counters()
	pos = p
	if _, ok, err := s.BestPlay(&pos, 3, 1); err != nil || !ok {
		t.Fatalf("seconde BestPlay: ok=%v err=%v", ok, err)
	}
	after, _, hitsAfter := s.Counters()
	if after < before {
		t.Fatalf("le compteur d'évaluations a reculé: %d → %d", before, after)
	}
	if hitsAfter <= hitsBefore {
		t.Errorf("aucun succès de cache sur la même position rejouée (%d → %d)", hitsBefore, hitsAfter)
	}

	s.ResetCounters()
	evals, prune, hits := s.Counters()
	filled, slotted = s.BatchFill()
	if evals != 0 || prune != 0 || hits != 0 || filled != 0 || slotted != 0 || s.CubeValuations() != 0 {
		t.Fatalf("après ResetCounters: %d/%d/%d, lot %d/%d, videau %d",
			evals, prune, hits, filled, slotted, s.CubeValuations())
	}
}

// TestSearcherCountersIncludeWorkers : un chercheur parallèle fait l'essentiel
// de son travail dans ses ouvriers, donc un compteur qui ne les additionnait
// pas rétrécirait à mesure qu'on ajoute des cœurs — l'inverse de ce qu'une
// sonde de coût sert à mesurer.
func TestSearcherCountersIncludeWorkers(t *testing.T) {
	s, p := countersTestSearcher(t, 4)
	pos := p
	if _, ok, err := s.BestPlay(&pos, 3, 1); err != nil || !ok {
		t.Fatalf("BestPlay: ok=%v err=%v", ok, err)
	}
	if evals, _, _ := s.Counters(); evals == 0 {
		t.Fatal("une recherche parallèle ne compte aucune évaluation")
	}
	s.ResetCounters()
	if evals, _, _ := s.Counters(); evals != 0 {
		t.Fatalf("ResetCounters laisse %d évaluations dans les ouvriers", evals)
	}
}

// TestKernelNameNamesARealPath : la sonde imprime ce nom à côté de chaque
// chronométrage, donc il doit nommer une voie qui existe.
func TestKernelNameNamesARealPath(t *testing.T) {
	name := KernelName()
	if name == "" || name == "invalid" {
		t.Fatalf("KernelName() = %q", name)
	}
	if err := kernelError(); err != nil {
		t.Fatalf("kernelError() = %v alors que le noyau se nomme %q", err, name)
	}
	if !strings.EqualFold(name, "go") && !strings.EqualFold(name, "avx2") && !strings.EqualFold(name, "neon") {
		t.Errorf("noyau inattendu %q — le sélecteur ne connaît que go/avx2/neon", name)
	}
}
