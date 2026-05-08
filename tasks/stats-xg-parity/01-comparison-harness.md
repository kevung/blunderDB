# 01 — Comparison harness

**Goal:** Construire un test Go `TestStatsParity` qui charge chaque JSON de référence (fiche 00), importe le `.xg` et le `.sgf` du match dans deux DB en mémoire, calcule les stats blunderDB, et émet un tableau diff aligné `XG | gnuBG-ref | blunderDB-from-XG | blunderDB-from-SGF` pour chaque métrique. Il `t.Errorf` quand l'écart sort des tolérances courantes.

**Depends on:** 00.

**Does NOT touch:** schéma, formules. Lecture seule côté blunderDB.

**Status: DONE** ✓

## Context

- Les tolérances évoluent fiche par fiche. Avant la fiche 02, on accepte ±45 decisions / ±3.0 PR / ±3.5 pp MWC. La fiche 07 les resserre à ±2 / ±0.1 / ±1 pp.
- Le test `xg_stats_reference_test.go` actuel ne couvre qu'Aachen et compare blunderDB à des baselines blunderDB (pas à XG). Il sera étendu / remplacé par `TestStatsParity` puis retiré (fiche 07).
- Helpers existants : `newTestDB(t)`, `db.ImportXGMatch(file)`, `db.ImportGnuBGMatch(file)`, `db.GetMatchDetailStats(matchID)`.

## Files touched

- `stats_parity_test.go` (créé, à la racine, package `main`)
- `testdata/stats_reference/*.json` (lus seuls, fiche 00)

## Tasks

### 1. Définir les types de chargement

- [x] Dans `stats_parity_test.go`, structs miroirs du JSON (fiche 00) :
  - `refPlayer` couvre les champs XG et gnuBG avec pointeurs (nullable)
  - `refMatch` charge `match_file`, `sgf_file`, `players`, `xg`, `gnubg`, `notes`
- [x] Loader : `loadRefMatch(t, path string) refMatch`.

### 2. Définir la grille de tolérances

- [x] Struct `parityTolerances` avec champ par métrique :
  ```go
  type parityTolerances struct {
      TotalDecisions, CheckerDecisions, DoubleDecisions, TakeDecisions int
      PR, MWCPct, Equity float64
  }
  ```
- [x] `tolPhase01` = `{100, 45, 100, 3, 3.0, 3.5, 0.35}` — valeurs expliquées :
  - `CheckerDecisions=45` : bDB compte les forcés + gaps d'import SGF (jusqu'à 40 d'écart)
  - `DoubleDecisions=100` : bDB compte tous les No Doubles (facteur 2-6× pré-fiche 03)
  - `PR=3.0` : dénominateur gonflé déprime le bDB PR de jusqu'à 2+ pts

### 3. Fonctions utilitaires de diff

- [x] `diffFloat(t, label, want *float64, got, tol float64)` : log + erreur si hors tolérance
- [x] `diffInt(t, label, want *int, got, tol int)` : idem pour les compteurs
- [x] Marker `!!` en début de ligne quand hors tolérance, `  ` sinon

### 4. Subtest par fixture

- [x] Boucle sur les 3 JSON référence
- [x] Import XG si `match_file` existe → `compareXGRef` (contre ref XG) ET `compareGnuBGRef` (contre ref gnuBG)
- [x] Import SGF si `sgf_file` existe → `compareGnuBGRef` (contre ref gnuBG)
- [x] Skip propre si le fichier est absent

### 5. Tableau récapitulatif

- [x] Toutes les valeurs loggées via `t.Logf` lisibles en `go test -v`
- [x] Lignes préfixées `!!` pour identifier visuellement les métriques hors tolérance

### 6. Résultats observés (phase 01)

Écarts actuels blunderDB ↔ XG (Aachen) :

| Métrique | XG P1 | bDB P1 | diff | XG P2 | bDB P2 | diff |
|---|---|---|---|---|---|---|
| total_decisions | 156 | 212 | **+56** | 161 | 252 | **+91** |
| checker_unforced | 138 | 168 | +30 | 144 | 169 | +25 |
| double_decisions | 16 | 42 | +26 | 12 | 78 | **+66** |
| take_decisions | 2 | 2 | 0 | 5 | 5 | 0 |
| PR | 3.13 | 2.28 | −0.85 | 3.07 | 2.08 | −0.99 |
| total_equity | 0.976 | 0.968 | −0.008 | 0.989 | 1.049 | +0.060 |
| total_mwc% | 19.85 | 20.83 | +0.98 | 14.39 | 15.48 | +1.09 |

Tous ces écarts sont **connus et attendus** à ce stade (fiches 02 + 03 les réduiront).

## Acceptance criteria

- [x] `go test -run TestStatsParity ./...` passe avec les tolérances `tolPhase01`.
- [x] Tous les `t.Logf` du diff sont lisibles en `-v` ; structure claire avec marker `!!`.
- [x] Le test couvre les 3 fixtures (skip propre si `.sgf` manquant).
- [x] Le fichier reste ≤ 260 lignes.

## Notes importantes

- `checker_total` (gnuBG) ≠ bDB checker count depuis SGF : certaines positions du SGF n'ont pas de données d'analyse, bDB ne les compte donc pas. Écart jusqu'à 20 positions (test.sgf P2). Ce gap persistera jusqu'à ce que le comptage soit découplé de l'analyse (hors scope immédiat).
- `checker_pr_xg_500` (gnuBG ref) utilise le dénominateur unforced ; bDB utilise toutes les positions → bDB PR structurellement sous-évalué jusqu'à la fiche 04.
- Comparaison XG→gnuBGref = utile pour vérifier la cohérence des données de référence cross-sources.
