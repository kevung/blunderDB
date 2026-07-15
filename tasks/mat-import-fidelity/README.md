# Fidélité des coups importés pour l'export `.mat` (XG & BGF)

Faire en sorte qu'un match exporté en `.mat` (feature GUI+CLI livrée) s'importe **sans erreur dans le vrai gnuBG**, quelle que soit la source du match. Aujourd'hui c'est le cas pour les matchs d'origine **gnuBG** (0 coup invalide) mais **pas** pour ceux importés depuis **XG** (`.xg`) ni **BGF** (`.bgf`).

Le blocage n'est **pas** dans l'exportateur (`pkg/blunderdb/ingest/mat_export.go`), qui rend fidèlement ce qui est stocké. Il est dans la **donnée de coups produite à l'import** : elle est dégénérée/incomplète pour certains coups, et le vrai gnuBG — contrairement à notre `gnubgparser` — **valide la légalité** des coups à l'import et les rejette.

## Symptôme

`gnubg -q -t -c cmds.txt` (avec `import mat "f.mat"` puis compter `grep -c "Invalid move"`) sur un export :

| Source du match | Coups « Invalid » gnuBG |
|---|---|
| gnuBG (`.mat`/`.sgf`) | **0** ✅ |
| XG (`.xg`) | 5 — tous `1/1` |
| BGF (`.bgf`) | 5 — `3/2`, `4/2`, `6/5`, … |

## Cause 1 — XG ne consigne pas le coup final gagnant

Le **dernier coup** de certaines parties (le bearoff final qui gagne la partie) n'est pas enregistré par XG. `xgparser` renvoie un `PlayedMove` `[0,0,…]`, converti en `1/1` (de l'as vers l'as, distance nulle) par `convertXGMoveToStringWithHits` (`pkg/blunderdb/ingest/xgmap.go`). gnuBG rejette `1/1`.

**Vérifié** (dump de `test.xg`) : les 3 cas sont toujours `IS_LAST=true` (dernier coup de la partie), dés variés (3-3, 2-1, 5-3). Ex. `g2 m88` : joueur 2, 3-3, `RAW=[1 1 1 1 1 1 1 1]`, position réelle = pions sur les points 2/3 → vrai coup `3/off 2/off(3)`, **absent des données**.

Pire : la **position stockée** pour ce coup est elle aussi inexploitable (`domain.LegalMoves` renvoie vide, `player_on_roll` incohérent) — on ne peut donc pas reconstruire le coup depuis la position.

**Piste de correction (import) :** dans `xgmap.go`, détecter le placeholder du coup final (from==to / `[0,0]`) et **reconstruire** le bearoff légal depuis la position + dés au moment de l'import (réutiliser `domain.LegalMoves`), en stockant un `checker_move` légal et une position correcte. À défaut, marquer explicitement le coup comme « non enregistré » d'une façon que l'export sait traduire en case gnuBG-valide.

## Cause 2 — BGF `green=7` : coups partiels + plateau non avancé

BGBlitz marque un dé injouable avec `green=7` (dé = 7, valeur sentinelle) et stocke un coup **partiel** (un seul dé joué : `3/2` pour un jet 1-5). De plus `bgf.go:122` **saute la mise à jour du plateau** (`bgfApplyCheckerMove`) pour ces coups → les positions stockées **en aval** sont incohérentes.

gnuBG rejette ces partiels (il calcule le 2ᵉ dé comme jouable). Reste à trancher, en lisant le format BGF, **si `green=7` signifie** :
- (a) un vrai coup partiel où le 2ᵉ dé est réellement injouable (alors gnuBG devrait l'accepter — enquêter sur une éventuelle divergence de plateau), ou
- (b) un pseudo-coup « analysis-only » à **ne pas** émettre dans la transcription (alors : l'omettre proprement), ou
- (c) une donnée BGBlitz erronée (le 2ᵉ dé était jouable mais non consigné → reconstruire).

Le fait que `bgf.go` saute déjà la mise à jour du plateau suggère (b), mais à confirmer sur le format (`bgfparser` expose aussi `moveData["moveAnalysis"][played].move`, potentiellement plus complet — à décoder et comparer).

## Pourquoi corriger à l'import, pas à l'export

Une **reconstruction consciente du plateau côté export** a été tentée puis **revertée** (worktree `fix/mat-export-move-data`, non mergé) :
- elle échoue pour XG car la **position stockée du coup final est dégénérée** (rien à reconstruire) ;
- reproduire fidèlement la sémantique de plateau `.mat` de gnuBG (quel joueur va dans quel sens, miroir perspective-joueur ↔ absolue) est très piégeux : une correspondance légèrement fausse fait diverger le plateau tracké de la relecture gnuBG et produit *plus* de coups invalides.

Corriger à l'**import** (stocker le vrai coup + une position correcte) résout le problème à la racine et bénéficie aussi à l'affichage/l'analyse, pas seulement au `.mat`.

## Non-bug confirmé

`domain.LegalMoves` (`pkg/blunderdb/domain/moves.go`) est **correct** : il dédoublonne les coups par **plateau résultant**, donc deux notations différentes pour le même plateau (ex. `1/off 5/1` vs `4/off 5/4`) fusionnent. Tout appariement doit comparer les **plateaux résultants**, pas les chaînes de notation.

## Outillage de vérification

- `gnubg` installé en `/usr/local/bin/gnubg`.
- Import + comptage : `printf 'set output error stdout\nimport mat "f.mat"\nquit\n' > c.txt ; gnubg -q -t -c c.txt 2>&1 | grep -c "Invalid move"`.
- Format de référence : `import mat "f.mat"` puis `export match mat "ref.mat"` réexporte le `.mat` tel que gnuBG l'écrit (comparer coup à coup).
- Fixtures : `testdata/test.xg` (= même match que `test.mat`, dédupliqué à l'import), `testdata/TachiAI_V_player_Nov_2__2025__16_55.bgf`.

## État livré (mergé `main`)

- Feature export `.mat` (GUI + CLI), nom de fichier, batch CLI.
- En-tête money game (`0 point match`) + `gnubgparser v1.3.0`.
- Danse rendue en case `.mat` vide (plus le littéral « Cannot Move »).
