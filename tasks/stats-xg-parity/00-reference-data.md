# 00 — Reference data extraction

**Goal:** Produire la vérité-terrain numérique « XG dit X, gnuBG dit Y » pour les 3 matchs appariés disponibles, sous forme de fichiers JSON dans `testdata/stats_reference/`. Ces fichiers seront chargés par le harnais (fiche 01) à chaque exécution de test.

**Depends on:** rien (première phase).

**Does NOT touch:** code de production, schéma, formules. Aucun changement Go ; uniquement de la donnée de test.

**Status: DONE** ✓

## Context

- 3 fixtures appariées (cf. README) : `test.{xg,sgf}`, `charlot1-charlot2_7p_…{xg,sgf}`, et le match Aachen XG-only dont les valeurs XG sont déjà transcrites dans `xg_stats_reference_test.go:28-47`.
- Les `.sgf` gnuBG contiennent déjà les statistiques calculées par gnuBG dans des balises `GS[…]` ; le parser `gnubgparser` les expose dans `Match.Games[i].Statistics` (`gnubgparser/types.go:159-194`). Toutefois, **le converter actuel ne peuple pas ce champ** — les GS tags sont parsés directement depuis le texte brut du SGF par l'outil `cmd/extract_gnubg_stats/main.go`.
- Les `.xg` sont des fichiers binaires ; le match report XG n'est pas lisible sans l'UI XG. Les valeurs XG pour `test.xg` et `charlot1-charlot2.xg` restent `null` en attendant une transcription manuelle.

## Files created

- `testdata/stats_reference/aachen-double-7pt.json` — valeurs XG transcrites depuis `xg_stats_reference_test.go`, gnuBG = null (pas de SGF)
- `testdata/stats_reference/test.json` — gnuBG depuis `test.sgf`, XG = null (binaire)
- `testdata/stats_reference/charlot1-charlot2.json` — gnuBG depuis charlot SGF (comptages seulement — analyse de skill non exécutée), XG = null
- `testdata/stats_reference/SCHEMA.md` — documentation du schéma JSON
- `cmd/extract_gnubg_stats/main.go` — outil Go qui parse les GS tags SGF et sort le JSON agrégé
- `stats_reference_test.go` — `TestStatsReferenceJSON` valide parsabilité + sanity check PR

## Tasks

### 1. Définir le schéma JSON

- [x] Rédiger `testdata/stats_reference/SCHEMA.md` décrivant le format.
- [x] Tous les champs nullables (mettre `null` quand inconnu côté XG ou côté gnuBG).
- [x] Valeurs XG en signe natif positif (magnitudes ; note dans SCHEMA.md).

### 2. Récupérer les valeurs gnuBG depuis les SGF

- [x] Écrire `cmd/extract_gnubg_stats/main.go` : parse les GS tags `M:` et `C:` directement depuis le texte SGF, agrège sur tous les jeux du match, produit le JSON. Format de parsing documenté dans gnubg/sgf.c:WriteStatContext.
- [x] `test.sgf` : stats gnuBG complètes (erreurs non nulles). Sérialisé dans `test.json["gnubg"]`.
- [x] `charlot1-charlot2.sgf` : gnuBG a enregistré les alternatives de coups mais **n'a pas exécuté la passe de statistiques** (skill classification absente — tous les EMG à zéro). Seuls les comptages (unforced, total, cube counts) sont fiables. Documenté dans `notes` du JSON.
- [x] Vérification visuelle des chiffres : cross-check sanity sur les formules (cf. § 4 ci-dessous).
- [x] Sérialisé dans la branche `"gnubg"` du JSON.

**Note sur `anCloseCube`** : ce champ de gnuBG (dénominateur du PR combiné) n'est **pas stocké dans le SGF**. Il est calculé dynamiquement par gnuBG à l'analyse et ne peut pas être reconstruit depuis les GS tags seuls. `close_cube_decisions` et `pr_gnubg` sont donc `null` dans tous les JSON SGF.

### 3. Récupérer les valeurs XG depuis le `.xg`

- [x] Pour Aachen-double : valeurs transcrites depuis `xg_stats_reference_test.go:28-47`.
- [ ] Pour `test.xg` et `charlot1-charlot2.xg` : les fichiers `.xg` sont binaires. Match report non accessible sans l'UI XG. **Action requise** : l'utilisateur peut exporter le match report depuis XG et fournir les valeurs à saisir manuellement dans les JSON.

### 4. Sanity checks

- [x] PR = 500 × |equity_emg| / decisions vérifié pour Aachen : P1=500×0.976/156=3.128≈3.13 ✓, P2=500×0.989/161=3.071≈3.07 ✓
- [x] gnuBG checker_pr_xg_500 vérifié : P1=500×2.152/147=7.32 ✓, P2=500×3.377/127=13.30 ✓
- [x] `TestStatsReferenceJSON` automatise la vérification PR/equity pour tout JSON disposant des valeurs XG.
- [x] La sum anMoves = anTotalMoves pour tous les jeux de test.sgf (vérification interne ✓).
- [ ] Vérification Aachen : gnuBG `checker_unforced` ≈ XG `checker_unforced` (±2) — impossiblee à comparer : pas de SGF pour ce match.

## Acceptance criteria

- [x] 3 JSON valides, parsables (`TestStatsReferenceJSON` vert).
- [x] `SCHEMA.md` aligné sur les fichiers (pas de champ orphelin).
- [x] Au moins un sanity-check (PR vs equity_error / decisions) recalculé et coïncidant.
- [x] Notes documentant toute incohérence XG ↔ gnuBG observée.

## Findings et incohérences détectées

1. **gnuBG `rErrorRateFactor` = 1000, XG = 500.** gnuBG PR ≈ 2 × XG PR pour le même match. Le JSON stocke `checker_pr_xg_500` (facteur 500, comparable XG) ET `checker_pr_gnubg_1000` (facteur 1000, natif gnuBG).

2. **`anCloseCube` absent du SGF.** Le PR combiné gnuBG (checker + cube, dénominateur unforced + close_cube) ne peut pas être reconstitué depuis les GS tags. La fiche 03 (close-cube) ajoutera la détection de ce flag dans blunderDB.

3. **charlot SGF sans skill stats.** Le fichier SGF a été créé par gnuBG 1.08.003 mais le pass d'analyse de statistiques n'a pas été exécuté : tous les `arError*` sont à 0. Les comptages (unforced, total, cube) sont présents et corrects.

4. **Signe des valeurs.** gnuBG stocke les erreurs de checkerplay en positif dans les GS tags (`arErrorCheckerplay[i][0] -= rChequerSkill` lors d'une erreur, où `rChequerSkill < 0`, donc l'accumulation est positive). Les JSON stockent des magnitudes positives dans les deux cas (XG et gnuBG).
