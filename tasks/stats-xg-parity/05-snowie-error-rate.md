# 05 — Snowie Error Rate

**Goal:** Calculer le « Snowie Error Rate » selon la formule de `gnubg/formatgs.c:415-424` (mêmes erreurs que PR, mais dénominateur = total moves des deux joueurs, forcés inclus). Sert d'axe de vérification croisée XG ↔ gnuBG ↔ blunderDB ; exposé d'abord en CLI, l'UI viendra en fiche 07 si pertinent.

**Depends on:** 04 (formule PR alignée).

**Does NOT touch:** UI (fait en fiche 07 si décidé).

## Context

- Formule cible (`gnubg/formatgs.c:415-424`) :
  ```c
  int n = psc->anTotalMoves[0] + psc->anTotalMoves[1];
  if (n > 0)
      aasz[i+1] = errorRateMPsnowie(-aaaar[COMBINED][TOTAL][i][NORMALISED] / (float) n);
  ```
- Numérateur identique au numérateur PR (même somme d'erreurs).
- Dénominateur **= nombre total de moves des deux joueurs**, sans filtre forcé/close. C'est ce qui rend Snowie ER plus stable et utilisable comme cross-check entre les outils.
- Snowie ER est la métrique la plus robuste pour valider la formule d'agrégation : XG, gnuBG, et blunderDB doivent converger à 0.1 près si le numérateur (somme des erreurs) est correct, sans être perturbés par les choix de filtre forcé/close.

## Files touched

- `db_stats.go` — `StatsResult.SnowieGlobal`, `MatchDetailStats.SnowieER` par joueur, requêtes ad-hoc.
- `cli.go` — ligne `Snowie ER:` dans `showStats`.
- `stats_parity_test.go` — assertion sur la cohérence Snowie ER blunderDB ≈ Snowie ER gnuBG (référence).

## Tasks

### 1. Étendre les types de résultat

- [ ] Dans `db_stats.go` :
  ```go
  type StatsResult struct {
      …
      SnowieGlobal     float64           `json:"SnowieGlobal"`     // Snowie ER aggregated, both players
      SnowiePerMatch   []MatchSnowieStat `json:"SnowiePerMatch"`   // optional, per match
  }
  type MatchSnowieStat struct { ID int64; Snowie float64 }
  ```
- [ ] Dans `MatchDetailStats` (selon `db_stats.go:GetMatchDetailStats`) : `SnowieER float64` par joueur.

### 2. Calculer le numérateur

- [ ] Numérateur = somme `statsErrExpr` sur **toutes** les décisions du périmètre (pas le filtre `statsCountedExpr`).
- [ ] Attention : Snowie compte le total moves **des deux joueurs** au dénominateur ; on additionne donc les `n_total` des deux joueurs sur la même période.

### 3. Calculer le dénominateur

- [ ] `total_moves(player) = COUNT(*) WHERE player = ?` (sans filtre).
- [ ] Pour `SnowieGlobal` : sommer `total_moves(P1) + total_moves(P2)`.
- [ ] Pour `MatchDetailStats[player].SnowieER` : numérateur du joueur / `(total_moves(P1) + total_moves(P2))`. C'est asymétrique pour un joueur (gnuBG fait pareil — c'est explicite dans la formule).

### 4. Helper `snowieER`

- [ ] Pendant `pr()` :
  ```go
  func snowieER(sumErrMP int64, nMovesBoth int) float64 {
      if nMovesBoth == 0 { return 0 }
      return 500 * float64(sumErrMP) / 1000 / float64(nMovesBoth)
  }
  ```
- [ ] Le facteur 500 est-il identique à PR (mêmes unités EMG normalisées) ? À vérifier — `gnubg/formatgs.c` utilise `errorRateMPsnowie` avec un facteur de 1000 (millipoints). Ajuster si discordance.

### 5. CLI

- [ ] `cli.go showStats` : nouvelle ligne :
  ```
  Snowie Error Rate: <SnowieGlobal> (P1=<er1>, P2=<er2>)
  ```
- [ ] Inclure dans la sortie JSON également (champ `SnowieGlobal`).

### 6. Tests

- [ ] Étendre `stats_parity_test.go` :
  - Vérifier `|blunderDB.snowie − gnubg.snowie| ≤ 0.1` sur les 3 fixtures (référence gnuBG provenant du JSON).
  - Vérifier `|blunderDB.snowie − xg.snowie| ≤ 0.3` quand `xg.snowie_error_rate` est renseigné.
- [ ] Snowie ER doit être > PR (numérateur identique, dénominateur plus grand).

## Acceptance criteria

- [ ] `go test -run TestStatsParity ./...` passe avec assertion Snowie ER aux tolérances ci-dessus.
- [ ] CLI montre Snowie ER en mode text et JSON.
- [ ] Snowie ER blunderDB ≈ Snowie ER gnuBG à 0.1 près sur les fixtures appariées (le fait que les deux convergent confirme que la formule globale d'agrégation est correcte indépendamment des filtres forcé/close).
- [ ] Pas de régression sur les autres métriques.

## Risks

- **Facteur EMG.** Si `errorRateMPsnowie` applique un scale différent du PR (`× 1000` vs `× 500 / 1000`), on aura un décalage d'un facteur 2. Mitigation : test direct contre `gnubg.snowie_error_rate` extrait du SGF ; ajuster jusqu'à correspondance.
- **Asymétrie joueur.** Snowie ER d'un joueur utilise la somme totale des moves des deux joueurs au dénominateur. Documenter dans le code (commentaire de `snowieER`) pour éviter qu'un futur dev « corrige » à tort.
- **Gros périmètre.** `SnowieGlobal` agrégé sur toute la base utilisateur n'a pas de sens immédiat si les périmètres mélangent matchs courts et longs. Décision : on **expose la métrique au niveau match** (via `MatchDetailStats`) et au niveau global filtré (`StatsResult.SnowieGlobal`), mais on ne crée pas de breakdown par tournoi pour Snowie (PR suffit).
