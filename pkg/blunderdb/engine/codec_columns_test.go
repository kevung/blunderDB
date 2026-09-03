package engine

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Les fonctions couvertes ici sont celles qui ÉCRIVENT LES COLONNES
// SCALAIRES — les valeurs dénormalisées que la recherche, les statistiques et
// les filtres SQL lisent sans jamais rouvrir le blob. Elles étaient toutes à
// 0 % (fiche C.11, #198), et c'est exactement la famille où le bug
// `CommitImportDatabase` s'était logé : une colonne qui n'est plus écrite ne
// casse aucun test, elle rend simplement des résultats vides.
//
// La règle suivie ici : une colonne se teste contre la valeur qu'elle doit
// contenir (167 pips à l'ouverture, un taux ×100, une équité ×1000), jamais
// contre elle-même.

// openingXGID est la position d'ouverture, dés 3-1 pour le joueur au trait.
const openingXGID = "XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:0:10"

func decodeXGID(t *testing.T, xgid string) domain.Position {
	t.Helper()
	p, err := domain.DecodeXGID(xgid)
	if err != nil {
		t.Fatalf("DecodeXGID(%s): %v", xgid, err)
	}
	return p
}

// TestPipCountsOpeningPosition — 167 de chaque côté est LA valeur de contrôle
// du backgammon ; une erreur d'orientation ou d'indice la rate immédiatement.
func TestPipCountsOpeningPosition(t *testing.T) {
	p := decodeXGID(t, openingXGID)
	pip1, pip2 := PipCounts(p.Board)
	if pip1 != 167 || pip2 != 167 {
		t.Fatalf("PipCounts à l'ouverture = (%d, %d), attendu (167, 167)", pip1, pip2)
	}

	// Un damier vide de tout sauf deux pions sur les points 1 et 24 : le
	// compte de chaque camp est la distance de SON pion à sa propre sortie.
	var b domain.Board
	for i := range b.Points {
		b.Points[i].Color = -1
	}
	b.Points[1] = domain.Point{Color: domain.Black, Checkers: 1}
	b.Points[24] = domain.Point{Color: domain.White, Checkers: 1}
	if got1, got2 := PipCounts(b); got1 != 1 || got2 != 1 {
		t.Fatalf("PipCounts(pion sur 1 / pion sur 24) = (%d, %d), attendu (1, 1)", got1, got2)
	}
}

// TestPopulatePositionColumnsOpening vérifie chaque colonne dérivée contre ce
// que la position d'ouverture contient réellement.
func TestPopulatePositionColumnsOpening(t *testing.T) {
	p := decodeXGID(t, openingXGID)
	c := PopulatePositionColumns(&p)

	if c.Pip1 != 167 || c.Pip2 != 167 || c.PipDiff != 0 {
		t.Errorf("pips = (%d, %d, diff %d), attendu (167, 167, 0)", c.Pip1, c.Pip2, c.PipDiff)
	}
	if c.Dice1 != 3 || c.Dice2 != 1 {
		t.Errorf("dés = (%d, %d), attendu (3, 1)", c.Dice1, c.Dice2)
	}
	if c.Off1 != 0 || c.Off2 != 0 {
		t.Errorf("sorties = (%d, %d), attendu (0, 0)", c.Off1, c.Off2)
	}
	// Deux pions au fond de chaque côté à l'ouverture.
	if c.BackCheckers1 != 2 || c.BackCheckers2 != 2 {
		t.Errorf("pions arrière = (%d, %d), attendu (2, 2)", c.BackCheckers1, c.BackCheckers2)
	}
	if c.NoContact {
		t.Error("NoContact vrai à l'ouverture")
	}
	norm := p.NormalizeForStorage()
	if c.ZobristHash != ZobristHash(&norm) {
		t.Error("ZobristHash de la colonne différent de celui de la position normalisée")
	}
	if c.Occupancy1 == 0 || c.Occupancy2 == 0 {
		t.Errorf("masques d'occupation vides: %#x / %#x", c.Occupancy1, c.Occupancy2)
	}

	// La normalisation est faite PAR la fonction : la même position vue de
	// l'autre côté doit rendre exactement les mêmes colonnes.
	flipped := p
	flipped.PlayerOnRoll = 1 - p.PlayerOnRoll
	if got := PopulatePositionColumns(&flipped); got != c {
		t.Error("les colonnes dépendent de PlayerOnRoll alors qu'elles sont normalisées")
	}
}

// TestPositionCompactRoundTrip — l'aller-retour que le schéma v2 fait à chaque
// lecture : la colonne `state` compacte plus les colonnes scalaires
// reconstruisent la position.
func TestPositionCompactRoundTrip(t *testing.T) {
	p := decodeXGID(t, "XGID=-b----E-C---eE---c-e----B-:1:1:1:52:3:5:0:7:10")
	norm := p.NormalizeForStorage()
	c := PopulatePositionColumns(&p)

	state := EncodeBoardCompact(norm.Board)
	if !IsCompactState(state) {
		t.Fatalf("EncodeBoardCompact n'a pas produit un état compact: %q", state)
	}
	if IsCompactState("") || IsCompactState(`{"board":{}}`) {
		t.Fatal("IsCompactState accepte un état JSON hérité")
	}

	got := ReconstructPosition(42, state, c.DecisionType, 0, c.Dice1, c.Dice2,
		c.CubeValue, c.CubeOwner, c.Score1, c.Score2, c.HasJacoby, c.HasBeaver)
	if got.ID != 42 {
		t.Errorf("ID = %d, attendu 42", got.ID)
	}
	// Le codec compact ne transporte QUE les cases occupées : une case vide
	// revient à Color 0 là où le décodeur XGID écrit −1. C'est sans effet —
	// tout ce qui lit un damier teste Checkers d'abord, et le hash Zobrist
	// ci-dessous le confirme — mais c'est pourquoi la comparaison porte sur
	// les cases occupées et non sur la structure entière.
	for i := range norm.Board.Points {
		want, have := norm.Board.Points[i], got.Board.Points[i]
		if want.Checkers != have.Checkers {
			t.Fatalf("case %d: %d pions, attendu %d", i, have.Checkers, want.Checkers)
		}
		if want.Checkers > 0 && want.Color != have.Color {
			t.Fatalf("case %d: couleur %d, attendu %d", i, have.Color, want.Color)
		}
	}
	if got.Board.Bearoff != norm.Board.Bearoff {
		t.Errorf("sorties %v, attendu %v", got.Board.Bearoff, norm.Board.Bearoff)
	}
	if EncodeBoardCompact(got.Board) != state {
		t.Error("le damier reconstruit ne se réencode pas à l'identique")
	}
	if got.Cube != norm.Cube || got.Dice != norm.Dice || got.Score != norm.Score {
		t.Errorf("colonnes reconstruites: cube %v dés %v score %v, attendu %v %v %v",
			got.Cube, got.Dice, got.Score, norm.Cube, norm.Dice, norm.Score)
	}
	if ZobristHash(&got) != c.ZobristHash {
		t.Error("la position reconstruite n'a pas le hash de la colonne")
	}

	// Format hérité : le JSON complet est relu, mais les colonnes restent
	// souveraines — c'est la promesse du commentaire de la fonction.
	legacy, err := json.Marshal(norm)
	if err != nil {
		t.Fatal(err)
	}
	fromLegacy := ReconstructPosition(7, string(legacy), c.DecisionType, 0, 6, 6,
		c.CubeValue, c.CubeOwner, c.Score1, c.Score2, c.HasJacoby, c.HasBeaver)
	if EncodeBoardCompact(fromLegacy.Board) != state {
		t.Error("le damier hérité n'est pas relu")
	}
	if fromLegacy.Dice != [2]int{6, 6} {
		t.Errorf("les colonnes ne priment pas sur le JSON hérité: dés %v", fromLegacy.Dice)
	}
}

// sampleAnalysis est une analyse de videau plausible : un double/prise net,
// avec un coup joué qui n'est pas le meilleur.
func sampleAnalysis() *domain.PositionAnalysis {
	err := 0.0345
	return &domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			PlayerWinChances:          72.345,
			PlayerGammonChances:       25.678,
			PlayerBackgammonChances:   2.123,
			OpponentWinChances:        27.655,
			OpponentGammonChances:     4.321,
			OpponentBackgammonChances: 0.234,
			CubefulNoDoubleEquity:     0.6789,
			CubefulNoDoubleError:      0.2123,
			CubefulDoubleTakeEquity:   0.8912,
			CubefulDoublePassEquity:   1.0,
			BestCubeAction:            "Double, Take",
		},
		CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{
				{Move: "13/11 24/23", Equity: 0.1234},
				{Move: "13/8 13/11", Equity: 0.0889, EquityError: &err},
			},
		},
	}
}

// TestPopulateAnalysisColumns vérifie les échelles (taux ×100, équités ×1000)
// et l'appariement de coup par NormalizeMove — deux parties du même coup dans
// l'ordre inverse sont le même coup.
func TestPopulateAnalysisColumns(t *testing.T) {
	a := sampleAnalysis()
	c := PopulateAnalysisColumns(a, "13/11 24/23", "Double")

	if c.BestCubeAction != "Double, Take" {
		t.Errorf("BestCubeAction = %q", c.BestCubeAction)
	}
	if c.Player1WinRate != 7235 || c.Player2WinRate != 2766 {
		t.Errorf("taux de gain = (%d, %d), attendu (7235, 2766)", c.Player1WinRate, c.Player2WinRate)
	}
	if c.Player1GammonRate != 2568 || c.Player1BackgammonRate != 212 {
		t.Errorf("taux gammon/backgammon = (%d, %d)", c.Player1GammonRate, c.Player1BackgammonRate)
	}
	// "13/11 24/23" est le meilleur coup ici, il n'a pas d'erreur ; celle
	// remplie est celle du second, retrouvé malgré l'ordre inverse de ses
	// parties et l'espace final.
	c2 := PopulateAnalysisColumns(a, "13/11 24/23 ", "Double")
	if c2.BestMoveEquityError != 0 {
		t.Errorf("le meilleur coup porte une erreur: %d", c2.BestMoveEquityError)
	}
	c3 := PopulateAnalysisColumns(a, "13/11 13/8", "Double")
	if c3.BestMoveEquityError != 35 {
		t.Errorf("BestMoveEquityError = %d, attendu 35 (0,0345 arrondi au millipoint)", c3.BestMoveEquityError)
	}

	// Double alors que le meilleur est Double/Take : erreur nulle.
	if c.CubeError != 0 {
		t.Errorf("CubeError = %d pour l'action optimale", c.CubeError)
	}
	// Ne pas doubler coûte 0,8912 − 0,6789 = 0,2123 → 212 millipoints.
	nd := PopulateAnalysisColumns(a, "", "No Double")
	if nd.CubeError != 212 {
		t.Errorf("CubeError(No Double) = %d, attendu 212", nd.CubeError)
	}

	if c.IsForced != 0 {
		t.Errorf("IsForced = %d avec deux coups légaux", c.IsForced)
	}
	forced := sampleAnalysis()
	forced.CheckerAnalysis.Moves = forced.CheckerAnalysis.Moves[:1]
	if got := PopulateAnalysisColumns(forced, "", "").IsForced; got != 1 {
		t.Errorf("IsForced = %d avec un seul coup légal", got)
	}

	if got := PopulateAnalysisColumns(nil, "13/11", "Take"); got != (AnalysisColumns{}) {
		t.Errorf("une analyse nulle rend %+v, attendu la structure vide", got)
	}
}

// TestPopulateAnalysisColumnsCheckerOnly : sans analyse de videau, les taux
// viennent du MEILLEUR coup, pas du coup joué.
func TestPopulateAnalysisColumnsCheckerOnly(t *testing.T) {
	a := &domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{
				{Move: "8/5 6/5", PlayerWinChance: 54.321, OpponentWinChance: 45.679},
				{Move: "24/21 13/11", PlayerWinChance: 48.0, OpponentWinChance: 52.0},
			},
		},
	}
	c := PopulateAnalysisColumns(a, "24/21 13/11", "")
	if c.Player1WinRate != 5432 || c.Player2WinRate != 4568 {
		t.Errorf("taux = (%d, %d), attendu ceux du meilleur coup (5432, 4568)",
			c.Player1WinRate, c.Player2WinRate)
	}
}

// TestComputeIsCloseCube couvre le prédicat gnubg isCloseCubedecision, y
// compris l'écrêtage de rDouble à +1 qui rend « close » toute décision où
// doubler dépasse le point.
func TestComputeIsCloseCube(t *testing.T) {
	dca := func(nd, dt, dp float64, best string) *domain.DoublingCubeAnalysis {
		return &domain.DoublingCubeAnalysis{
			CubefulNoDoubleEquity:   nd,
			CubefulDoubleTakeEquity: dt,
			CubefulDoublePassEquity: dp,
			BestCubeAction:          best,
		}
	}
	tests := []struct {
		name   string
		dca    *domain.DoublingCubeAnalysis
		played string
		want   int64
	}{
		{"une réponse est toujours close (take)", nil, "Take", 1},
		{"une réponse est toujours close (pass)", nil, "Pass", 1},
		{"pas d'analyse de videau", nil, "Double", 0},
		{"écart sous le seuil", dca(0.50, 0.45, 1.0, "No Double"), "Double", 1},
		{"écart au-dessus du seuil", dca(0.50, 0.20, 1.0, "No Double"), "Double", 0},
		{"double/pass avec un DT écrêté", dca(0.90, 1.40, 1.0, "Double, Pass"), "Double", 1},
		{"action inconnue: le maximum sert d'optimum", dca(0.10, 0.50, 1.0, "?"), "Double", 0},
	}
	for _, tt := range tests {
		if got := ComputeIsCloseCube(tt.dca, tt.played); got != tt.want {
			t.Errorf("%s: ComputeIsCloseCube = %d, attendu %d", tt.name, got, tt.want)
		}
	}
}

// TestRoundAnalysisForStorage : les taux à 0,01 %, les équités au millipoint,
// et le pointeur d'erreur arrondi sans être perdu.
func TestRoundAnalysisForStorage(t *testing.T) {
	RoundAnalysisForStorage(nil) // ne doit pas paniquer

	if got := RoundToMillipoint(0.12345); got != 0.123 {
		t.Errorf("RoundToMillipoint(0,12345) = %v", got)
	}
	if got := RoundToHundredthPercent(72.3456); got != 72.35 {
		t.Errorf("RoundToHundredthPercent(72,3456) = %v", got)
	}

	err := 0.0345678
	a := &domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			PlayerWinChances:      72.34567,
			CubefulNoDoubleEquity: 0.6789123,
			WrongTakePercentage:   12.34567,
		},
		AllCubeAnalyses: []domain.DoublingCubeAnalysis{{OpponentWinChances: 27.65432}},
		CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{{
				Equity: 0.1234567, EquityError: &err, PlayerWinChance: 54.32109,
			}},
		},
	}
	RoundAnalysisForStorage(a)

	if a.DoublingCubeAnalysis.PlayerWinChances != 72.35 {
		t.Errorf("taux non arrondi: %v", a.DoublingCubeAnalysis.PlayerWinChances)
	}
	if a.DoublingCubeAnalysis.CubefulNoDoubleEquity != 0.679 {
		t.Errorf("équité non arrondie: %v", a.DoublingCubeAnalysis.CubefulNoDoubleEquity)
	}
	if a.DoublingCubeAnalysis.WrongTakePercentage != 12.35 {
		t.Errorf("pourcentage non arrondi: %v", a.DoublingCubeAnalysis.WrongTakePercentage)
	}
	if a.AllCubeAnalyses[0].OpponentWinChances != 27.65 {
		t.Errorf("AllCubeAnalyses non arrondi: %v", a.AllCubeAnalyses[0].OpponentWinChances)
	}
	m := a.CheckerAnalysis.Moves[0]
	if m.Equity != 0.123 || m.PlayerWinChance != 54.32 {
		t.Errorf("coup non arrondi: équité %v, taux %v", m.Equity, m.PlayerWinChance)
	}
	if m.EquityError == nil || *m.EquityError != 0.035 {
		t.Errorf("erreur d'équité: %v", m.EquityError)
	}
	if &err == m.EquityError {
		t.Error("l'arrondi a écrit à travers le pointeur d'origine")
	}
}

// TestNormalizeMove : deux écritures du même coup se comparent égales, un
// coup différent non.
func TestNormalizeMove(t *testing.T) {
	if NormalizeMove("5/2 5/4") != NormalizeMove("5/4  5/2 ") {
		t.Error("le même coup dans deux ordres ne se normalise pas pareil")
	}
	if NormalizeMove("5/2 5/4") == NormalizeMove("5/2 5/3") {
		t.Error("deux coups différents se normalisent pareil")
	}
	if NormalizeMove("") != "" {
		t.Error("NormalizeMove(\"\") n'est pas vide")
	}
}

// TestRecompressAnalysisData couvre le chemin de mise à niveau opportuniste
// (import de .db natif, passe de Vacuum) : du JSON brut monte en zstd, du
// zstd ne bouge pas, et le contenu traverse intact.
func TestRecompressAnalysisData(t *testing.T) {
	raw, err := json.Marshal(&domain.PositionAnalysis{XGID: "XGID=test", Player1: "Alice"})
	if err != nil {
		t.Fatal(err)
	}

	if NeedsRecompression(nil) {
		t.Error("NeedsRecompression(nil) est vrai")
	}
	if !NeedsRecompression(raw) {
		t.Error("du JSON brut n'a pas besoin d'être recompressé ?")
	}

	up, err := RecompressAnalysisData(raw)
	if err != nil {
		t.Fatalf("RecompressAnalysisData: %v", err)
	}
	if NeedsRecompression(up) {
		t.Error("le résultat de la recompression a encore besoin d'être recompressé")
	}
	back, err := DecodeAnalysisFromStorage(up)
	if err != nil {
		t.Fatalf("DecodeAnalysisFromStorage: %v", err)
	}
	if back.XGID != "XGID=test" || back.Player1 != "Alice" {
		t.Errorf("contenu perdu à la recompression: %+v", back)
	}

	// Idempotence : une seconde passe ne réécrit rien.
	again, err := RecompressAnalysisData(up)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != len(up) {
		t.Error("la recompression d'un blob déjà courant l'a réécrit")
	}

	empty, err := RecompressAnalysisData(nil)
	if err != nil || empty != nil {
		t.Errorf("RecompressAnalysisData(nil) = %v, %v", empty, err)
	}
}

// TestBearoffIndexAndRollDistribution couvre les deux fonctions que
// engine/race adresse avec la même indexation combinatoire.
func TestBearoffIndexAndRollDistribution(t *testing.T) {
	// L'indexation est injective sur les arrangements d'un même nombre de
	// pions : c'est LA propriété dont dépendent les deux bases bearoff.
	seen := map[int][6]int{}
	var walk func(rest, point int, acc [6]int)
	walk = func(rest, point int, acc [6]int) {
		if point == 6 {
			if rest != 0 {
				return
			}
			idx := BearoffIndex(acc, 6, 6)
			if prev, dup := seen[idx]; dup {
				t.Fatalf("index %d partagé par %v et %v", idx, prev, acc)
			}
			seen[idx] = acc
			return
		}
		for k := 0; k <= rest; k++ {
			acc[point] = k
			walk(rest-k, point+1, acc)
		}
	}
	walk(6, 0, [6]int{})
	if len(seen) == 0 {
		t.Fatal("aucun arrangement énuméré")
	}

	probs, err := RollDistribution([6]int{2, 2, 2, 2, 2, 2})
	if err != nil {
		t.Fatalf("RollDistribution: %v", err)
	}
	var sum float64
	for _, p := range probs {
		if p < 0 {
			t.Fatalf("probabilité négative dans la distribution: %v", p)
		}
		sum += p
	}
	if math.Abs(sum-1.0) > 1e-6 {
		t.Errorf("la distribution somme à %v, attendu 1", sum)
	}
	// Douze pions ne sortent jamais en moins de trois lancers.
	for n := 0; n < 3; n++ {
		if probs[n] != 0 {
			t.Errorf("P(sortie en %d lancers) = %v, attendu 0", n, probs[n])
		}
	}

	if _, err := RollDistribution([6]int{-1, 0, 0, 0, 0, 0}); err == nil {
		t.Error("un nombre de pions négatif est accepté")
	}
	if _, err := RollDistribution([6]int{15, 15, 0, 0, 0, 0}); err == nil {
		t.Error("trop de pions est accepté")
	}
}
