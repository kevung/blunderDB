# 04 — PR denominator alignment

**Goal:** Faire passer la formule PR (et les compteurs de décisions affichés) sur le dénominateur XG/gnuBG : `n_unforced_checker + n_close_cube`. Tout le câblage front + CLI dépend de cette fiche pour montrer les bons chiffres.

**Depends on:** 02 (`is_forced`) et 03 (`is_close_cube`).

**Does NOT touch:** Snowie ER (fiche 05), MWC (fiche 06).

## Context

- Formule actuelle (`db_stats.go:129-134`) :
  ```go
  return 500 * float64(sumErrMP) / 1000 / float64(nDecisions)
  ```
  où `nDecisions` est `COUNT(*)` sans filtre `is_forced` ni `is_close_cube`.
- Formule cible alignée XG/gnuBG :
  ```
  PR = 500 × (Σ errMP_unforced_checker + Σ errMP_close_cube) / 1000 /
       (n_unforced_checker + n_close_cube)
  ```
- Numérateur : la somme des erreurs pour les décisions filtrées seulement (déjà aligné en réalité, car les coups forcés ont `errMP = 0` et les No Double évidents aussi — mais à confirmer empiriquement sur les fixtures).
- Effet attendu sur Aachen : décisions P1 passe de 212 → ~156 (XG), PR P1 de 2.28 → ~3.13.

## Files touched

- `db_stats.go` — `pr()`, `statsErrExpr` inchangée mais nouvelle constante `statsCountedExpr`, requêtes de `ComputeStats`, `populateMatchStats`, `populateTournamentStats`, `GetMatchDetailStats`, `rolling*`.
- `cli.go` — `showStats` : pas de changement d'affichage, mais vérifier que les compteurs imprimés utilisent les nouvelles requêtes.
- `stats_parity_test.go` — resserrer la tolérance PR à ±0.2 et la tolérance décisions à ±5.
- `xg_stats_reference_test.go` — basculer les baselines vers les valeurs XG (la référence change, ce n'est plus un test de régression interne mais un test d'alignement). Possible suppression au profit de `TestStatsParity`.

## Tasks

### 1. Définir le filtre comptable partagé

- [x] Nouvelle constante dans `db_stats.go` :
  ```go
  // statsCountedExpr is the SQL predicate selecting the decisions that count
  // toward PR / decision tallies (XG + gnuBG semantics). Forced checker plays
  // and non-close cube decisions are excluded.
  const statsCountedExpr = "((p.decision_type = 0 AND a.is_forced = 0) OR (p.decision_type = 1 AND a.is_close_cube = 1))"
  ```
- [x] Réfléchir : ce filtre doit-il être dans `buildStatsWhereClause` (toujours appliqué) ou être une option ? **Décision : toujours appliqué** ; un filtre `IncludeAll` peut être ajouté plus tard si besoin de retrouver les chiffres « bruts ». Pas dans cette fiche.

### 2. Mise à jour des requêtes

- [x] `ComputeStats` :
  - PR global : ajouter `AND statsCountedExpr` à la WHERE via `buildStatsWhereClause`.
  - PR par tournoi / match : idem.
  - `Totals.NumDecisions` : reflète le compte filtré.
  - Histogramme d'erreurs : filtre appliqué (coups forcés à err=0 exclus du bucket [0, 100[).
  - `TopBlunders` : pas de filtre (un blunder reste un blunder, même sur un coup non-comptable).
- [x] `populateMatchStats`, `populateTournamentStats`, `GetMatchDetailStats` : filtre appliqué.
- [x] Rolling PR : filtre appliqué dans la sub-query (via `buildStatsWhereClause`).

### 3. Adapter `pr()`

- [x] Le helper `pr(sumErrMP, nDecisions)` reste **inchangé** ; le filtrage se fait dans la requête, pas dans la fonction.
- [x] Renommage optionnel non réalisé (ne pas inflater le diff).

### 4. Mettre à jour `MatchDetailStats`

- [x] Champs `TotalDecisions`, `CheckerDecisions`, `DoubleDecisions`, `TakeDecisions` reflètent les compteurs filtrés : checker = unforced ; double = close non-take/pass ; take/pass = inchangés.
- [x] Pas de nouveau champ pour les chiffres bruts à cette phase.

### 5. Tests

- [x] `stats_parity_test.go` : `tolPhase04{Decisions: 5, CheckerDecisions: 10, PR: 0.2, MWCPct: 3.5, Equity: 0.5}`.
- [x] Sur Aachen, P1 : décisions=159 (XG=156, Δ=3 ≤ 5), PR=3.04 (XG=3.13, Δ=0.09 ≤ 0.2). ✓
- [x] `xg_stats_reference_test.go` : baselines remplacées par les valeurs XG. Décision de retrait reportée à fiche 07.
- [x] Fixtures `db_stats_test.go` et `db_stats_drilldown_test.go` : `is_close_cube=1` pour les lignes cube.

## Acceptance criteria

- [x] `go test -run TestStatsParity ./...` passe avec `Decisions:5`, `PR:0.2` sur les 3 fixtures.
- [x] Pas de régression `go test ./...`.
- [ ] Le CLI `./blunderdb list --type stats` montre un PR cohérent avec XG sur les fixtures. *(validation manuelle, non bloquante)*
- [x] `MatchDetailStats` retourne les compteurs filtrés.

## Résultats observés (Aachen 7pt, bDB vs XG)

| Métrique | XG | bDB | Δ |
|---|---|---|---|
| Total décisions P1/P2 | 156 / 161 | 159 / 165 | 3 / 4 |
| Checker (unforced) P1/P2 | 138 / 144 | 142 / 150 | 4 / 6 |
| Double (close) P1/P2 | 16 / 12 | 15 / 10 | 1 / 2 |
| PR P1/P2 | 3.13 / 3.07 | 3.04 / 3.18 | 0.09 / 0.11 |
| MWC loss P1/P2 | 19.85 % / 14.39 % | 20.83 % / 15.48 % | 0.98 / 1.09 |

Écarts résiduels : edge cases de classification forced (≤6 checker) + bruit moteur d'analyse (cross-engine rounding < 1 %).

## Risks

- **TopBlunders pollués.** Si on filtre les TopBlunders, on risque de cacher un coup forcé qui aurait quand même une grosse erreur (impossible mathématiquement mais bug possible si `is_forced` mal positionné). Mitigation : **ne pas filtrer TopBlunders** + assertion test que `len(TopBlunders) > 0` sur les fixtures.
- **Histogramme déformé.** Idem, à arbitrer. **Décision proposée : filtrer**, pour aligner avec XG ; sinon le bucket [0, blunderThreshold[ explose.
- **Effet domino sur le panneau Stats global.** Le panneau stats agrège sur toute la base utilisateur ; le changement de PR sera visible et pourra surprendre. Mitigation : prévoir une note de release (fiche 07).
- **Numérateur asymétrique.** Hypothèse : tous les coups forcés ont `errMP = 0`. Si certains ont `errMP > 0` (analyse partielle, bug), le numérateur courant sur-compte. Vérifier empiriquement : `SELECT SUM(err) WHERE is_forced = 1 AND decision_type = 0` doit être ≈ 0. Si non, logger.
