# 07 — Validation & rollout

**Goal:** Resserrer les tolérances finales, mettre à jour la documentation, le CLI et les libellés UI, et clore la série.

**Depends on:** 06 (donc 02–06 vert).

## Context

- À ce stade, blunderDB devrait reproduire XG / gnuBG à ±2 décisions, ±0.1 PR, ±1 pp MWC sur les 3 fixtures appariées.
- Plusieurs métriques nouvelles ont été ajoutées (Snowie ER, peut-être `mwc_error`). Les libellés et la doc utilisateur n'en parlent pas encore.
- Le test ad-hoc `xg_stats_reference_test.go` est désormais redondant avec `TestStatsParity`.

## Files touched

- `stats_parity_test.go` — tolérances finales.
- `xg_stats_reference_test.go` — suppression ou reroutage vers `TestStatsParity`.
- `doc/source/` (français) — nouvelle section « Stats parity model ».
- `CLI_USAGE.md` — documenter Snowie ER et toute autre sortie nouvelle.
- `frontend/src/components/MatchPanel.svelte`, `TournamentPanel.svelte`, `stats/StatsPanel.svelte` — libellés / colonnes.
- `tasks/stats-xg-parity/README.md` — cocher tous les statuts.
- `CHANGELOG`-équivalent (le projet n'a pas de CHANGELOG.md formel — voir `doc/source/index.rst` qui contient parfois un changelog).

## Tasks

### 1. Resserrage final

- [ ] `tolPhaseFinal = tolerances{Decisions: 2, PR: 0.1, MWC: 1.0, Equity: 0.05, SnowieER: 0.1}`.
- [ ] Lancer `go test -v ./... && go test -v ./tests/...` ; corriger tout écart.
- [ ] Si une fixture refuse de converger (par ex. seuil `is_close_cube` mal calé sur XG), enregistrer le delta dans le JSON `notes` et augmenter sélectivement la tolérance pour la métrique concernée — mais documenter pourquoi.

### 2. Nettoyage

- [ ] Supprimer `xg_stats_reference_test.go` (ou le réduire à un test de régression sur les structures, sans assertions chiffrées).
- [ ] Vérifier que les commentaires « Known intentional differences vs XG » dans le repo sont retirés ou adaptés (ils sont devenus obsolètes : il n'y a plus de différences intentionnelles).
- [ ] Grep `// XG inflates`, `// blunderDB includes forced` etc. et nettoyer.

### 3. Documentation utilisateur

- [ ] `doc/source/` : ajouter `stats_parity.rst` (ou équivalent) en français :
  - Définitions formelles : PR, Snowie ER, MWC loss.
  - Conventions blunderDB ↔ XG ↔ gnuBG.
  - Liste des décisions exclues du PR (forcés, cubes triviaux) et du seuil `0.16`.
  - Caveat : « si vos chiffres divergent de XG, vérifier que vos analyses sont complètes et que la version XG/blunderDB est à jour ».
- [ ] Mettre à jour le sommaire (`index.rst`).

### 4. CLI

- [ ] `CLI_USAGE.md` : section `list --type stats` mise à jour avec exemples montrant Snowie ER.
- [ ] Sortie texte : trier les métriques pour faciliter la lecture (PR, Snowie ER, décisions, MWC dans cet ordre).
- [ ] Sortie JSON : confirmer la présence de `SnowieGlobal` (et `mwc_error` si exposé).

### 5. UI

- [ ] **Match panel** (`MatchPanel.svelte`) : vérifier que les colonnes affichent les nouvelles valeurs alignées XG. Renommer le tooltip si besoin (« PR XG-style — coups forcés et No Double triviaux exclus »). Optionnellement ajouter une ligne « Snowie ER ».
- [ ] **Tournament panel** : idem.
- [ ] **Stats panel** : pareil ; ajouter Snowie ER comme métrique secondaire si l'utilisateur le souhaite (à confirmer en review).
- [ ] Lancer `wails dev`, ouvrir un match XG-importé, vérifier visuellement les chiffres.

### 6. Note de release

- [ ] Mentionner :
  - Bump `DatabaseVersion` (les bases existantes seront migrées au prochain run ; sauvegarder avant ouverture).
  - Changement potentiellement visible : valeurs PR / décisions différentes après mise à jour (alignées sur XG).
  - Nouvelle métrique Snowie ER en CLI (et UI si embarqué).

### 7. Bench rapide

- [ ] `go test -bench=. -benchtime=3x ./...` (s'il y a des benchs stats) ou un timing manuel : importer une grosse base utilisateur (`testdata/tournois/`) et vérifier que `ComputeStats` reste sous 500 ms. Le filtre `statsCountedExpr` ajoute une condition mais devrait être servi par les index `idx_analysis_is_forced` / `idx_analysis_is_close_cube`. Si régression visible, ajouter un index combiné `(decision_type, is_forced, is_close_cube)`.

## Acceptance criteria

- [ ] `go test ./... && go test ./tests/...` vert avec tolérances finales.
- [ ] `wails dev` montre les bons chiffres dans Match / Tournament / Stats.
- [ ] CLI : `./blunderdb list --type stats --format json` retourne `SnowieGlobal` non nul sur une base de test.
- [ ] Doc utilisateur mise à jour, sommaire référence la nouvelle page.
- [ ] README de la série coché à 100 %.
- [ ] Aucun TODO résiduel dans le code des fiches 02-06 (ou s'il y en a, déplacés vers `tasks/FOLLOWUPS.md`).

## Risks

- **Régression perf stats panel.** Les nouvelles colonnes filtrées peuvent perturber les plans d'exécution SQLite. Mitigation : `EXPLAIN QUERY PLAN` sur les requêtes principales ; ajouter un index combiné si nécessaire.
- **Utilisateurs habitués aux anciennes valeurs.** Le PR baisse vers la « bonne » valeur, mais si un utilisateur a fait des captures d'écran historiques, les chiffres ne matcheront plus. Mitigation : documenter clairement dans la note de release.
- **Fixture irréductible.** Si une métrique reste hors tolérance malgré tous les fixes (par ex. seuil close-cube), la documenter en `notes` du JSON ref et accepter une tolérance ad-hoc plutôt que de re-bidouiller le code.
