// SPDX-License-Identifier: MIT

package gammonnet

import (
	"os"
	"regexp"
	"testing"
)

// L'arrondi des tuiles : le garde-fou, et la preuve qu'il garde.
//
// Le test a DEUX VOLETS, et le second est le point. Le volet positif montre que
// roundDownMultiple tient sa postcondition et qu'un parcours tuilé à tuile 6
// reste dans les bornes. Pris seul, il ne prouverait rien : il passerait aussi
// bien sur une tranche surdimensionnée, où le débordement n'aurait simplement
// pas de quoi se voir. Le volet négatif exige donc que la forme masquée
// FAUTIVE meure sur cette même tranche. Si elle survit, ce n'est pas que le
// code est sain, c'est que le harnais ne détecte rien.
//
// gammonNet doit compiler son volet négatif sous AddressSanitizer pour qu'un
// débordement de trois flottants soit visible ; ici la vérification de bornes
// des tranches est toujours active, et le débordement est une panique
// déterministe. C'est la seule chose que ce portage ait de plus facile.

// tiledSum parcourt row par tuiles de tile jusqu'à rounded, puis finit le
// reste une ligne à la fois — la forme exacte d'un noyau tuilé. Panique si
// rounded n'est pas un multiple de tile, ce qui est tout ce que le test veut
// savoir.
func tiledSum(row []float32, rounded, tile int) float32 {
	var s float32
	for j := 0; j < rounded; j += tile {
		for _, v := range row[j : j+tile] {
			s += v
		}
	}
	for j := rounded; j < len(row); j++ {
		s += row[j]
	}
	return s
}

// TestRoundDownMultipleHoldsItsPostcondition — le volet positif.
func TestRoundDownMultipleHoldsItsPostcondition(t *testing.T) {
	for n := -3; n <= 260; n++ {
		for tile := -2; tile <= 40; tile++ {
			got := roundDownMultiple(n, tile)
			if tile <= 0 || n <= 0 {
				if got != 0 {
					t.Fatalf("roundDownMultiple(%d, %d) = %d, veut 0 — une tuile non positive divise par zéro", n, tile, got)
				}
				continue
			}
			if got > n {
				t.Fatalf("roundDownMultiple(%d, %d) = %d dépasse n", n, tile, got)
			}
			// La moitié de la postcondition que le masque viole.
			if got%tile != 0 {
				t.Fatalf("roundDownMultiple(%d, %d) = %d n'est pas un multiple de %d", n, tile, got, tile)
			}
			if n-got >= tile {
				t.Fatalf("roundDownMultiple(%d, %d) = %d laisse un reste de %d, >= la tuile", n, tile, got, n-got)
			}
			// Là où le masque est correct — et il l'est pour toute puissance
			// de deux — les deux formes doivent coïncider, sans quoi le
			// remplacement changerait le découpage des noyaux existants.
			if tile&(tile-1) == 0 && got != n&^(tile-1) {
				t.Fatalf("à tuile %d (puissance de deux), arrondi %d contre masque %d", tile, got, n&^(tile-1))
			}
		}
	}
}

// TestTiledWalkStaysInBoundsAtANonPowerOfTwoTile — le volet positif, en acte :
// une ligne allouée à la taille EXACTE, une tuile qui n'est pas une puissance
// de deux, et le parcours qui va au bout.
func TestTiledWalkStaysInBoundsAtANonPowerOfTwoTile(t *testing.T) {
	// 195 et 6 sont les valeurs du défaut réel (#133) : 195 &^ 5 = 194, qui
	// n'est pas un multiple de 6, et le parcours atteint alors row[192:198].
	const outDim, tile = 195, 6
	row := make([]float32, outDim)
	for i := range row {
		row[i] = 1
	}
	rounded := roundDownMultiple(outDim, tile)
	if rounded != 192 {
		t.Fatalf("roundDownMultiple(%d, %d) = %d, veut 192", outDim, tile, rounded)
	}
	if got := tiledSum(row, rounded, tile); got != outDim {
		t.Fatalf("somme tuilée = %v, veut %d — le reste n'a pas été parcouru", got, outDim)
	}
}

// TestTheMaskedFormReallyDoesOverrun — le volet négatif, et sans lui le volet
// positif ne prouve rien.
func TestTheMaskedFormReallyDoesOverrun(t *testing.T) {
	const outDim, tile = 195, 6
	row := make([]float32, outDim)

	masked := outDim &^ (tile - 1)
	if masked == roundDownMultiple(outDim, tile) {
		t.Fatalf("le masque et l'arrondi rendent tous deux %d : le cas choisi ne distingue plus les deux formes", masked)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("la forme masquée n'a pas débordé : la ligne n'était pas allouée à la taille exacte, " +
				"donc le volet positif ne prouve rien")
		}
	}()
	_ = tiledSum(row, masked, tile)
}

// TestNoKernelRoundsATileWithAMask relit les sources du noyau et de son
// générateur, exactement comme TestGeneratedKernelHasNoFMAAndNoFramePointer
// relit l'assembleur produit : la propriété est un invariant, et elle
// n'apparaît jamais comme une mauvaise réponse dans un banc — elle apparaît
// comme une lecture hors matrice, une fois sur trois, selon l'allocation.
func TestNoKernelRoundsATileWithAMask(t *testing.T) {
	// `x &^ (n - 1)`, avec ou sans espaces et parenthèses, et sa variante
	// écrite `& ^(n - 1)`. Le circonflexe est obligatoire : `n & (n-1)` sans
	// lui n'est pas un arrondi, c'est le test « puissance de deux ».
	mask := regexp.MustCompile(`&\s*\^\s*\(?\s*\w+\s*-\s*1\s*\)?`)

	for _, path := range []string{
		"kernel.go",
		"kernel_go.go",
		"kernel_amd64.go",
		"kernel_noasm.go",
		"network.go",
		"tile.go",
		"_asm/main.go",
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue // kernel_noasm.go n'existe pas sur toutes les cibles
			}
			t.Fatalf("%s : %v", path, err)
		}
		for _, line := range splitLines(string(raw)) {
			if isCommentLine(line) || !mask.MatchString(line) {
				continue
			}
			t.Errorf("%s arrondit une tuile par masque : %q\n"+
				"`n &^ (tile-1)` n'arrondit au multiple inférieur que pour une puissance de deux ; "+
				"à tuile 6 il rend un non-multiple et le noyau lit hors matrice (#133). "+
				"Utiliser roundDownMultiple, ou compter à rebours comme le générateur.", path, line)
		}
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// isCommentLine écarte les lignes de commentaire : ce fichier-ci et tile.go
// CITENT la forme fautive pour l'expliquer, et une interdiction qui empêche de
// nommer ce qu'elle interdit rend la règle indocumentable.
func isCommentLine(line string) bool {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case ' ', '\t':
			continue
		case '/':
			return i+1 < len(line) && (line[i+1] == '/' || line[i+1] == '*')
		case '*':
			return true
		default:
			return false
		}
	}
	return false
}
