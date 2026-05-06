# Stats parity blunderDB ↔ eXtremeGammon — Task Sheets

Aligner les statistiques de match (PR, MWC loss, nombre de décisions, blunders) calculées par blunderDB sur celles fournies par eXtremeGammon (référence utilisateur) en s'appuyant sur le code source de gnuBG (`/gnubg/`) et sur des matchs disponibles dans plusieurs formats (`.xg`, `.sgf`, `.mat`).

Les valeurs XG sont la vérité-terrain. gnuBG sert de référence ouverte (formules vérifiables dans `gnubg/formatgs.c`, `gnubg/analysis.c`, `gnubg/eval.c`) et de second axe de comparaison.

## Diagnostic résumé

Le test `xg_stats_reference_test.go` documente déjà l'écart sur le match Aachen 7pt (Squire/Jørgensen vs Harmand/Unger) :

| Métrique | XG | blunderDB |
|---|---|---|
| Total décisions P1/P2 | 156 / 161 | 212 / 252 |
| Checker (Unforced) | 138 / 144 | 168 / 169 |
| Doubling decisions | 16 / 12 | 42 / 78 |
| PR | 3.13 / 3.07 | 2.28 / 2.08 |
| MWC loss | -19.85 % / -14.39 % | -20.83 % / -15.48 % |

Deux causes principales identifiées :

1. **blunderDB ne distingue pas les décisions forcées.** XG / gnuBG excluent du dénominateur PR les checker plays à un seul coup légal (`pmr->ml.cMoves > 1` dans `gnubg/analysis.c:458`).
2. **blunderDB compte tous les No Double comme décisions cube.** XG / gnuBG ne comptent que les décisions cube *proches* (`isCloseCubedecision`, seuil 0.16 d'équité dans `gnubg/eval.c:5088`).

L'écart résiduel sur MWC (~1 pp) est probablement consécutif aux deux points ci-dessus ; il sera re-validé une fois (1) et (2) corrigés.

## Ordre d'exécution

Dépendances strictes top-down. Ne pas démarrer une fiche avant la précédente.

| # | Fiche | Objet | Dépend de |
|---|---|---|---|
| 00 | [reference-data.md](00-reference-data.md) | Extraire chiffres XG + gnuBG par match | — |
| 01 | [comparison-harness.md](01-comparison-harness.md) | Test Go `TestStatsParity` (XG vs gnuBG vs blunderDB) | 00 |
| 02 | [forced-moves.md](02-forced-moves.md) | Colonne `is_forced` + détection import + migration | 01 |
| 03 | [close-cube.md](03-close-cube.md) | Colonne `is_close_cube` + prédicat 0.16 + migration | 01 |
| 04 | [pr-denominator.md](04-pr-denominator.md) | Nouvelle formule PR (dénominateur unforced + close) | 02 + 03 |
| 05 | [snowie-error-rate.md](05-snowie-error-rate.md) | Snowie ER (numérateur identique, dénominateur = total moves) | 04 |
| 06 | [mwc-cross-check.md](06-mwc-cross-check.md) | Re-validation MWC, fix résiduel si >1 pp | 04 + 05 |
| 07 | [validation-rollout.md](07-validation-rollout.md) | Tolérances finales, doc, CLI, libellés UI | 06 |

## Working rules

- Chaque fiche reste ≤ 500 lignes (règle CLAUDE.md). Si une fiche grossit, la scinder.
- **Une fiche = une PR.** Ne pas mélanger les phases.
- **Aucune formule n'est modifiée avant la fiche 04.** Les fiches 02 et 03 ajoutent uniquement des colonnes ; elles ne changent pas les chiffres affichés. Cela permet d'évaluer la corrélation des nouveaux flags avec les comptes XG/gnuBG sans casser les baselines.
- À la fin de chaque fiche : `go test ./... && go test ./tests/...` doit être vert. Le harnais `TestStatsParity` (fiche 01) garde les tolérances courantes ; on les resserre à chaque phase.
- Mettre à jour le checkbox-statut dans chaque fiche au fur et à mesure ; un repreneur doit pouvoir savoir où on en est en lisant la fiche.

## Rollback strategy

- 00–01 : additif (nouveaux fichiers de test/référence). Revert = `git revert`.
- 02–03 : migrations additives (colonnes `NOT NULL DEFAULT 0`). Si un backfill est faux, *forward-only* — ajouter une étape `2.x.0 → 2.x.1` qui re-backfille. **Ne pas dropper de colonnes.**
- 04–05 : changement de formule (effet visible utilisateur). Revert possible via `git revert`, schéma intact.
- 06 : potentiellement nouvelle colonne `mwc_error` (si nécessaire). Même règle que 02–03.
- 07 : doc + libellés. Trivialement révertable.

## Statut courant

- [x] 00 — reference-data
- [x] 01 — comparison-harness
- [x] 02 — forced-moves
- [ ] 03 — close-cube
- [ ] 04 — pr-denominator
- [ ] 05 — snowie-error-rate
- [ ] 06 — mwc-cross-check
- [ ] 07 — validation-rollout

## Fixtures appariées disponibles

| Match | Formats | Chemin |
|---|---|---|
| `test` | `.xg`, `.sgf`, `.mat` | `testdata/test.{xg,sgf,mat}` |
| `charlot1-charlot2_7p_2025-11-08-2305` | `.xg`, `.sgf`, `.mat` | `testdata/charlot1-charlot2_7p_2025-11-08-2305.{xg,sgf,mat}` |
| `Aachen-Squire/Jørgensen-Harmand/Unger 7pt` | `.xg` seul, valeurs XG transcrites | `testdata/2024-08-10-Aachen-…/double/` |

Les fichiers `.sgf` (gnuBG) embarquent les statistiques pré-calculées dans le bloc `GameStatistic` du parser ; le parser `gnubgparser/types.go:159-194` extrait déjà `Unforced[2]`, `Forced[2]`, `MissedDouble`, `WrongDouble`, `WrongTake`, `WrongPass`, mais ces données ne sont actuellement consommées nulle part.

## Références gnuBG (à citer dans les fiches)

- `gnubg/formatgs.c:399-409` — formule PR (« Error rate per decision »).
- `gnubg/formatgs.c:415-424` — formule Snowie Error Rate.
- `gnubg/analysis.c:458-462` — accumulation checker, exclusion des forcés (`cMoves > 1`).
- `gnubg/analysis.c:1430-1474` — `getMWCFromError`, dérivation `COMBINED[PERMOVE]`.
- `gnubg/analysis.c:1449-1464` — accumulation MWC error par décision (`eq2mwc`).
- `gnubg/eval.c:5088-5100` — prédicat `isCloseCubedecision` (seuil 0.16).
