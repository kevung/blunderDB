// SPDX-License-Identifier: MIT

package gammonnet

// Les largeurs, les tuiles, et l'arrondi qui ne suppose rien — reprise en Go
// de `src/gn_tile.h` de gammonNet.
//
// Un noyau tuilé a besoin du plus grand multiple de `tile` <= n. La forme
// évidente
//
//	rounded := n &^ (tile - 1) // FAUX sauf si tile est une puissance de deux
//
// est juste pour toute puissance de deux et silencieusement hors bornes sinon
// (à tuile 6 et n = 195 elle rend 194 et lit au-delà de la matrice). D'où la
// règle :
//
//   - roundDownMultiple partout où la tuile n'est pas garantie puissance de
//     deux (une division entière par appel de noyau, pas par élément) ;
//   - une assertion de compilation partout où une puissance de deux est
//     supposée.

// Assertion de compilation : EvalBatchWidth est une puissance de deux, car
// le noyau AVX2 s'en sert comme ÉCHELLE d'adressage x86
// (`Mem{Base: actp, Index: jb, Scale: lanes}`, 1, 2, 4 ou 8), et le
// générateur vit dans un autre module. `n & (n-1)` est non nul sinon, et une
// constante uint négative arrête la compilation.
const _ uint = 0 - (EvalBatchWidth & (EvalBatchWidth - 1))

// roundDownMultiple rend le plus grand multiple de tile qui ne dépasse pas n.
//
// Ne suppose rien de tile au-delà d'être positif ; rend 0 pour une tuile ou
// un n non positif (un noyau lent se voit, une lecture hors matrice non).
//
// POSTCONDITION : le résultat est <= n ET un multiple exact de tile.
func roundDownMultiple(n, tile int) int {
	if tile <= 0 || n <= 0 {
		return 0
	}
	return n - n%tile
}
