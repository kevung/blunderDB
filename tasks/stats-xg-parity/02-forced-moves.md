# 02 — Forced moves classification

**Goal:** Persister un drapeau `is_forced` pour chaque décision checker afin de pouvoir exclure les coups forcés des compteurs et du PR (alignement XG / gnuBG). Aucune formule de stats n'est encore modifiée — fiche 04 fera le câblage.

**Depends on:** 01 (le harnais doit déjà tourner).

**Does NOT touch:** `db_stats.go`, formules PR/MWC. Pas de modif UI.

## Context

- Référence gnuBG : `gnubg/analysis.c:458-462` accumule l'erreur checker uniquement quand `pmr->ml.cMoves > 1`.
- Côté blunderDB, l'analyse XG/gnuBG/MAT importée stocke `CheckerAnalysis.Moves[]` (`model.go:235`). Pour XG, la liste est exhaustive (XG dump tous les coups candidats) — `len(Moves) <= 1` ⇒ forcé. Pour gnuBG (SGF), pareil. À vérifier sur 2-3 positions par fixture pendant l'implémentation.
- Le parser gnuBG expose déjà des compteurs agrégés `Unforced[2]` / `Forced[2]` (`gnubgparser/types.go:171-172`) — utilisables pour la **validation** par échantillonnage : la somme `is_forced=1 AND decision_type=0` par joueur doit reproduire `Forced[player]` à 0 près.
- Schéma actuel : pas de colonne forcé dans `position` ni dans `analysis` (vérifié par `grep` dans `db.go`).

## Files touched

- `model.go` — bump `DatabaseVersion` (par ex. `2.0.0` → `2.1.0` ; **vérifier la convention de version**).
- `db.go` — ajout colonne `analysis.is_forced` au CREATE TABLE et dans la migration `CheckVersion`.
- `db_analysis.go` — population de `is_forced` dans `populateAnalysisColumns`.
- `migration_test.go` — test que la migration depuis une DB v2.0.0 fonctionne et backfille correctement.
- `stats_parity_test.go` (étendu) — assertion `Forced count` sur les fixtures.

## Tasks

### 1. Schéma + migration

- [ ] Dans `db.go`, ajouter à la définition `CREATE TABLE IF NOT EXISTS analysis` :
  ```sql
  is_forced INTEGER NOT NULL DEFAULT 0
  ```
- [ ] Index : `CREATE INDEX IF NOT EXISTS idx_analysis_is_forced ON analysis(is_forced) WHERE is_forced = 1`. Bénéfique car la majorité des décisions cube (decision_type=1) n'auront pas `is_forced=1` ; l'index partiel filtre rapidement.
- [ ] Étape de migration `2.0.0 → 2.1.0` :
  1. `ALTER TABLE analysis ADD COLUMN is_forced INTEGER NOT NULL DEFAULT 0`.
  2. Backfill : itérer `analysis.id`, décompresser `data`, parser le JSON `PositionAnalysis`, appliquer le prédicat `decision_type=0 AND len(CheckerAnalysis.Moves) <= 1`, `UPDATE analysis SET is_forced = 1 WHERE id = ?`.
  3. Wrapper la boucle dans une transaction explicite + statement préparé.
- [ ] Bump `DatabaseVersion` dans `model.go`.

### 2. Population à l'import

- [ ] Localiser `populateAnalysisColumns` (`db_analysis.go` selon README) ou le path d'écriture des analyses pour XG / gnuBG / MAT / TXT.
- [ ] Ajouter le calcul :
  ```go
  isForced := 0
  if pos.DecisionType == CheckerAction && pa.CheckerAnalysis != nil && len(pa.CheckerAnalysis.Moves) <= 1 {
      isForced = 1
  }
  ```
- [ ] Inclure `is_forced` dans l'INSERT/UPDATE de la ligne `analysis`.

### 3. Cohérence parsers

- [ ] **gnuBG** : ajouter dans le test du harnais une vérification `count(is_forced=1, joueur) == ParsedGameStats.Forced[joueur]` à ±0. Si écart, investiguer le mapping de `pmr->ml.cMoves` côté gnuBG SGF (peut être absent quand l'analyse n'a pas été menée).
- [ ] **XG** : pas d'aggrégation native ; on contrôle indirectement via le harnais 01 (la métrique `checker_unforced` doit reproduire la valeur XG à ±2 après filtre `is_forced=0`).
- [ ] **MAT (Jellyfish)** : le format inclut-il les coups candidats ? À vérifier dans le code racine. Si non, fallback : compter via la quantité de coups dans `Moves`. Si toujours indéterminé, marquer `is_forced=0` par défaut et laisser ces fixtures hors scope du harnais (à documenter dans la fiche 07).

### 4. Test migration

- [ ] `migration_test.go` : créer une DB sample v2.0.0, appliquer la migration, vérifier que :
  - La colonne existe et est NOT NULL.
  - Au moins une position avec `len(Moves)==1` a bien `is_forced=1` après backfill.
  - Le total `is_forced=1 AND decision_type=0` correspond aux comptes attendus pour la fixture (utilisateur peut fixer la valeur attendue à la main).

### 5. Étendre le harnais

- [ ] `stats_parity_test.go` : nouvelle ligne d'assertion `checker_forced_count` (compté via `SELECT COUNT(*) FROM analysis JOIN ... WHERE decision_type=0 AND is_forced=1 AND mv.player = ?`).
- [ ] La métrique « checker_unforced » de la référence (XG) doit maintenant être atteignable, mais **les compteurs affichés par blunderDB ne changent pas encore** (formule pas mise à jour).

## Acceptance criteria

- [ ] `go test ./...` vert.
- [ ] Migration testée : DB v2.0.0 → v2.1.0 sans perte de données, colonne backfillée.
- [ ] Sur les 3 fixtures, `count(is_forced=0, decision_type=0, player)` correspond à `xg.checker_unforced` à ±2 (le filtre est conceptuellement bon).
- [ ] Sur la fixture `test.sgf`, validation croisée gnuBG : `count(is_forced=1, decision_type=0, player)` = `gnubg.forced[player]`.
- [ ] Aucune régression sur `xg_stats_reference_test.go` (les chiffres affichés sont inchangés à ce stade).

## Risks

- **Backfill long.** Sur grosses bases utilisateur (~ 100k positions), décompresser+parser chaque blob JSON peut prendre quelques minutes. Mitigation : transaction unique + `PRAGMA journal_mode=MEMORY` durant la migration ; messages d'avancement dans le log app.
- **Faux négatifs sur XG.** Certaines positions XG peuvent n'avoir qu'un seul coup analysé sans qu'il soit forcé (par ex. analyse tronquée). Mitigation : vérifier sur un échantillon que `len(Moves)==1 ⇒ position effectivement forcée` (rejouer le coup à la main sur 5 cas). Si ce n'est pas robuste, basculer la détection vers un compte de coups légaux calculé par un générateur de mouvements à l'import (gros effort — out-of-scope ici, à escalader).
- **Modification de hash canonique.** L'ajout de `is_forced` est sur `analysis`, pas sur `position` — pas d'impact sur le hash canonique de dedup (vérifié dans `model.go`). Mais à confirmer en relisant `canonical_hash_test.go`.
