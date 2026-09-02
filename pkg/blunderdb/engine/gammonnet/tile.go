// SPDX-License-Identifier: MIT

package gammonnet

// Les largeurs, les tuiles, et l'arrondi qui ne suppose rien.
//
// POURQUOI CE FICHIER EXISTE — ET LE BUG VENAIT D'ICI.
//
// Un noyau tuilé parcourt sa dimension par tuiles pleines puis finit le reste à
// la main. Il lui faut donc le plus grand multiple de `tile` qui ne dépasse pas
// `n`. La façon évidente de l'écrire est
//
//	rounded := n &^ (tile - 1) // FAUX sauf si tile est une puissance de deux
//
// et elle est fausse de la pire manière : elle est CORRECTE pour toutes les
// tuiles que le code emploie aujourd'hui, et silencieusement hors bornes pour
// la première qui n'en est pas une. Ce dépôt a livré exactement cette ligne
// (#133) : à tuile 6 et n = 195 elle rend 194 — qui n'est pas un multiple de 6
// — donc la boucle `for j := 0; j < rounded; j += tile` atteint j = 192 et lit
// les lignes de poids 192 à 197 dans une matrice qui en a 195. Les tests ne
// l'ont pas vu parce que la tuile valait 4 quand ils ont été écrits.
//
// Le défaut a été corrigé en changeant de forme — le générateur d'assembleur
// (`_asm/main.go`) compte les sorties restantes à rebours et n'arrondit plus du
// tout — mais SANS ASSERTION : rien n'empêchait de réintroduire le masque.
// gammonNet a écrit le garde-fou que ce dépôt n'avait pas (`src/gn_tile.h`,
// T90) ; ce fichier est sa reprise en Go, en application de l'ADR-0003.
//
// La règle que ce fichier fait tenir :
//
//   - roundDownMultiple partout où la tuile n'est PAS GARANTIE puissance de
//     deux. Le coût est une division entière, payée UNE FOIS par appel de
//     noyau, pas une fois par élément.
//   - une assertion de compilation partout où une puissance de deux EST
//     supposée, pour que l'hypothèse soit dite au compilateur et pas seulement
//     au lecteur.
//
// Un commentaire disant « la tuile doit être une puissance de deux » n'est pas
// un garde-fou. Une compilation qui s'arrête en est un.

// Assertion de compilation : EvalBatchWidth est une puissance de deux.
//
// L'hypothèse est réelle et elle est matérielle : le noyau AVX2 adresse la
// colonne d'activations avec `Mem{Base: actp, Index: jb, Scale: lanes}`, et
// l'échelle d'un mode d'adressage x86 ne peut valoir que 1, 2, 4 ou 8. Une
// largeur de lot qui cesserait d'être une puissance de deux ne serait pas
// « moins rapide », elle serait ingénérable — et le générateur vit dans un
// autre module, donc son échec n'arriverait pas ici.
//
// Comment ça marche : `n & (n-1)` vaut 0 pour une puissance de deux et autre
// chose sinon ; la soustraction en fait alors une constante NÉGATIVE, et une
// constante négative de type uint arrête la compilation sur
// « constant -N overflows uint ».
const _ uint = 0 - (EvalBatchWidth & (EvalBatchWidth - 1))

// roundDownMultiple rend le plus grand multiple de tile qui ne dépasse pas n.
//
// Ne suppose rien de tile au-delà d'être positif. Rend 0 pour une tuile ou un n
// non positif plutôt que de diviser par zéro : un noyau qui boucle alors zéro
// fois et tombe entièrement dans sa queue scalaire est lent, et la lenteur est
// un mode de défaillance qu'on voit — contrairement à une lecture au-delà de la
// fin d'une matrice.
//
// POSTCONDITION, et c'est tout l'objet de la fonction : le résultat est <= n
// ET un multiple exact de tile. `n &^ (tile-1)` satisfait la première et viole
// la seconde, ce qui est précisément pourquoi il déborde.
func roundDownMultiple(n, tile int) int {
	if tile <= 0 || n <= 0 {
		return 0
	}
	return n - n%tile
}
