# 06 — MWC cross-check

**Goal:** Re-valider le calcul de MWC loss après le resserrage du dénominateur PR (fiche 04) et l'ajout de Snowie ER (fiche 05). Si l'écart résiduel avec XG dépasse 1 pp sur les 3 fixtures, investiguer la cause et corriger.

**Depends on:** 04 + 05.

**Does NOT touch:** PR (fixé en 04). Idéalement aucun changement requis si l'écart MWC résiduel s'absorbe avec le filtre `statsCountedExpr`.

## Context

- État actuel de MWC (`db_met.go:435-447`) : conversion EMG→MWC post-hoc par décision, en s'appuyant sur le score de la position et le cube value au moment de la décision.
- État actuel de la précision : ~ 1 pp d'écart vs XG sur Aachen (P1 : -19.85 vs -20.83 ; P2 : -14.39 vs -15.48). L'écart pourrait venir :
  - des No Double sur-comptés (résolu par fiche 03) ;
  - des coups forcés à err=0 qui contribuent 0 mais inflatent le compteur (résolu par fiche 04, sans effet sur MWC car la somme est inchangée) ;
  - d'un décalage de score (l'erreur d'un coup est encourue *avant* de le jouer ; le score utilisé pour la conversion doit être celui *avant* le coup) ;
  - d'une convention de signe sur le « player favori » (gnuBG `eq2mwc` est pondéré par la perspective du joueur).
- Référence gnuBG : `gnubg/analysis.c:1449-1464` calcule `rCost = eq2mwc(rSkill, &ci) - eq2mwc(0.0f, &ci)` *par décision* puis somme — pas de conversion post-hoc.

## Files touched

- `db_met.go` — refonte éventuelle de `ConvertEMGLossToMWCLoss` ou ajout d'une voie alternative.
- `db_analysis.go` — si nécessaire, matérialiser `mwc_error` à l'import (nouvelle colonne).
- `db_stats.go` — adapter les requêtes MWC pour SUM-er la colonne directement plutôt que reconvertir en Go.
- Migration éventuelle (nouveau bump `DatabaseVersion` si on ajoute une colonne).

## Tasks

### 1. Mesure post-fix

- [ ] Lancer `go test -run TestStatsParity -v ./...` et noter les écarts MWC sur les 3 fixtures, par joueur, après les fiches 04+05.
- [ ] **Si l'écart est ≤ 1 pp partout :** la fiche se conclut par un test resserré (`MWC: 1.0`) et un mini-changelog. Stop ici.
- [ ] **Si l'écart > 1 pp :** continuer en tâche 2.

### 2. Diagnostic

- [ ] Ajouter un mode debug à `ConvertEMGLossToMWCLoss` (drapeau `verbose`) qui logge par décision :
  ```
  pos=<id> player=<+/-1> cube=<v> score=(p1,p2) errMP=<x> mwcWin=<m> mwcLose=<m> contrib=<c>
  ```
- [ ] Comparer 5 décisions clés (les plus contributrices au MWC) entre blunderDB et gnuBG (en re-tournant le SGF dans gnuBG GUI ou via le harnais).
- [ ] Identifier la divergence : score, cube exponent, signe player, interaction crawford…

### 3. Si la cause est un score décalé

- [ ] gnuBG calcule MWC à la position où la décision est prise (avant le coup). blunderDB utilise potentiellement le score d'une position post-coup. Vérifier :
  - dans `db_met.go:435-447` quel score est passé (`p.score_1`, `p.score_2` qui sont des away-scores) ;
  - les commentaires laissent entendre que `currentScore = matchLength - awayScore` est le bon mapping (commit `ce025484`).
- [ ] Corriger si écart trouvé.

### 4. Si la cause est plus structurelle (par-décision MWC requise)

- [ ] Ajouter colonne `analysis.mwc_error INTEGER` (millipoints × 1000 par exemple, ou stockage flottant — réfléchir aux unités).
- [ ] Calculer à l'import via `gnuBGGetME` (déjà disponible) : `mwc_error = eq2mwc_loss(equity_error, position_state)`.
- [ ] Migration : backfill via lecture du blob `data` + état position.
- [ ] Mise à jour des requêtes MWC dans `db_stats.go` pour faire `SUM(a.mwc_error)` directement.
- [ ] Bump `DatabaseVersion`.

### 5. Tests

- [ ] Resserrer `tolerances.MWC` à `1.0` pp (objectif fiche 07 : 0.5 si possible).
- [ ] Sur les 3 fixtures, MWC blunderDB ≈ MWC XG à 1 pp.

## Acceptance criteria

- [ ] `go test -run TestStatsParity ./...` passe avec `MWC ≤ 1.0 pp`.
- [ ] Si une nouvelle colonne `mwc_error` est ajoutée : migration testée + backfill validé.
- [ ] Aucune régression PR / Snowie ER (les fiches précédentes restent vertes).

## Risks

- **Nouveau bump de schéma.** Si on matérialise `mwc_error`, c'est un troisième bump après 02+03. Assumable, mais grouper si possible avec 02/03 si on ré-exécute les phases.
- **Sensibilité au crawford / post-crawford.** `gnuBGGetME` doit recevoir le bon flag crawford. Vérifier dans `populateAnalysisColumns` le passage de `pos.HasJacoby`/contexte crawford. À documenter.
- **MWC = `eq2mwc(skill) − eq2mwc(0)`.** Le code blunderDB fait `(emg/1000) × (mwcWin − mwcLose) / 2` qui suppose une linéarité approximative. Pour les positions « compressées » (proches de 0% ou 100% MWC), la linéarité casse. Si l'écart résiduel se concentre sur ces positions, c'est la signature du problème ; remplacer par la formule par-décision.
