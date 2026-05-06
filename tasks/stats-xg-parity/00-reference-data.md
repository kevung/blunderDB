# 00 — Reference data extraction

**Goal:** Produire la vérité-terrain numérique « XG dit X, gnuBG dit Y » pour les 3 matchs appariés disponibles, sous forme de fichiers JSON dans `testdata/stats_reference/`. Ces fichiers seront chargés par le harnais (fiche 01) à chaque exécution de test.

**Depends on:** rien (première phase).

**Does NOT touch:** code de production, schéma, formules. Aucun changement Go ; uniquement de la donnée de test.

## Context

- 3 fixtures appariées (cf. README) : `test.{xg,sgf}`, `charlot1-charlot2_7p_…{xg,sgf}`, et le match Aachen XG-only dont les valeurs XG sont déjà transcrites dans `xg_stats_reference_test.go:28-47`.
- Les `.sgf` gnuBG contiennent déjà les statistiques calculées par gnuBG dans des balises `GS[…]` ; le parser `gnubgparser` les expose dans `Match.Games[i].Statistics` (`gnubgparser/types.go:159-194`).
- Les `.xg` contiennent les statistiques XG dans le footer texte du fichier (le « match report »). On peut soit les parser, soit les transcrire à la main depuis l'UI XG ou depuis un PDF utilisateur. Pour la vérité-terrain, **transcription manuelle assumée** — le parser XG n'a pas besoin de comprendre ce footer.

## Files touched

- `testdata/stats_reference/aachen-double-7pt.json` (créé)
- `testdata/stats_reference/test.json` (créé)
- `testdata/stats_reference/charlot1-charlot2.json` (créé)
- `testdata/stats_reference/SCHEMA.md` (créé) — documentation du schéma JSON

## Tasks

### 1. Définir le schéma JSON

- [ ] Rédiger `testdata/stats_reference/SCHEMA.md` décrivant le format. Forme cible :
  ```json
  {
    "match_file": "testdata/test.xg",
    "match_length": 7,
    "players": ["Player1", "Player2"],
    "xg": {
      "player1": {
        "pr": 3.13,
        "snowie_error_rate": 5.21,
        "total_decisions": 156,
        "checker_unforced": 138,
        "double_decisions": 16,
        "take_decisions": 2,
        "pass_decisions": 0,
        "total_errors": 18,
        "total_blunders": 2,
        "total_equity_error_emg": -0.976,
        "checker_equity_error_emg": -0.894,
        "double_equity_error_emg": -0.037,
        "take_equity_error_emg": -0.045,
        "total_mwc_loss_pct": -19.85,
        "checker_mwc_loss_pct": -18.77,
        "double_mwc_loss_pct": -0.42,
        "take_mwc_loss_pct": -0.66
      },
      "player2": { /* same schema */ }
    },
    "gnubg": { /* same schema, populated when SGF available */ },
    "notes": "..."
  }
  ```
- [ ] Tous les champs nullables (mettre `null` quand inconnu côté XG ou côté gnuBG).
- [ ] Valeurs XG en signe natif XG (équités d'erreur négatives, MWC losses négatives).

### 2. Récupérer les valeurs gnuBG depuis les SGF

- [ ] Pour `test.sgf` et `charlot1-charlot2.sgf` : écrire un petit script Go ad-hoc (à mettre dans `cmd/extract_gnubg_stats/main.go` ou comme test `_test.go` qui dump-ifie via `t.Logf`) qui :
  1. Parse le SGF via `gnubgparser`.
  2. Agrège `GameStatistic.Moves.Unforced`, `.Forced`, `.ErrorTotal`, `.ErrorSkill`, `.MissedDouble/WrongDouble/WrongTake/WrongPass` sur tous les jeux du match.
  3. Reconstruit les métriques globales suivant les formules `gnubg/formatgs.c:399-424`.
- [ ] Vérifier visuellement avec un dump gnuBG (par ex. `gnubg-cli` si installé, ou ouverture du SGF dans gnuBG GUI) que les chiffres reconstruits correspondent.
- [ ] Sérialiser dans la branche `"gnubg"` du JSON.

### 3. Récupérer les valeurs XG depuis le `.xg`

- [ ] Pour Aachen-double : reprendre les valeurs déjà transcrites dans `xg_stats_reference_test.go:28-47`.
- [ ] Pour `test.xg` et `charlot1-charlot2.xg` : transcrire le « Match report » de XG (footer texte du `.xg` ou export PDF/HTML XG) à la main. L'utilisateur fournit ces chiffres si non lisibles dans le binaire.
- [ ] Sérialiser dans la branche `"xg"` du JSON.

### 4. Sanity checks

- [ ] Sur chaque JSON : `xg.total_decisions` ≥ `gnubg.total_decisions` (XG semble parfois exclure plus que gnuBG). Vérifier l'hypothèse.
- [ ] `xg.checker_unforced` ≈ `gnubg.unforced[player]` (≤ ±2). Si gros écart, le drapeau forced n'a pas la même définition entre les deux ; documenter dans `notes`.
- [ ] PR = 500 × |total_equity_error_emg| / total_decisions doit reproduire le PR transcrit (à 0.05 près). Si non, l'EMG est probablement déjà signé/normalisé différemment ; documenter.

## Acceptance criteria

- [ ] 3 JSON valides, parsables (`go run testdata/stats_reference/validate.go` ou simple `json.Unmarshal` dans un test).
- [ ] `SCHEMA.md` aligné sur les fichiers (pas de champ orphelin).
- [ ] Au moins un sanity-check (PR vs equity_error / decisions) recalculé à la main pour chaque match et coïncidant.
- [ ] Notes documentant toute incohérence XG ↔ gnuBG observée pendant la transcription.

## Risks

- **Footer XG illisible.** Si le `.xg` ne contient pas le match-report en texte clair, on dépend d'une transcription manuelle de l'UI XG. Mitigation : l'utilisateur a accès à XG, peut fournir une copie d'écran. Le harnais doit pouvoir tourner même si une fixture XG manque (skip avec t.Skip).
- **Stats SGF tronquées.** Si gnuBG n'a pas analysé toutes les positions du match (analyse partielle), les compteurs sont biaisés. Mitigation : vérifier que `Unforced + Forced ≈ total moves attendus` ; sinon, exclure la fixture pour gnuBG.
- **Encodage des joueurs.** XG et gnuBG peuvent inverser player1/player2 selon la perspective. Vérifier sur le `match.player{1,2}_name` après import — la fixture JSON doit nommer explicitement les joueurs et le harnais (fiche 01) doit faire le mapping.
