# Transcription de matchs — plan de développement

Quatre lots, chacun livrable et utilisable seul, découpés en issues **verticales** : une
issue traverse Go → binding Wails → Svelte → tests → doc/`.po` et laisse l'application
utilisable. Format d'une issue : [issues/README.md](issues/README.md). Efforts : S ≤ ½ j,
M ≤ 2-3 j, L = chantier. Une issue = une branche = un worktree = une PR (`CLAUDE.md`).

## Lot 0 — fondations (rien de visible, tout s'y appuie)

| Issue | Titre | Effort | Dépend de |
|---|---|---|---|
| [T0.1](issues/T0.1-adr-glossaire.md) | ADR-0044, ADR-0045, glossaire, amendement 0037 (#332) | S | — (cette branche) |
| [T0.2](issues/T0.2-paquet-transcript.md) | Paquet `transcript` : document, Actions, Replay, Incohérences, parties du match, tests sur `.mat` réels (#333) | L | T0.1 |
| [T0.3](issues/T0.3-schema-2-21-0.md) | Table `transcription`, 2.21.0 dans les trois copies, `demo.db`, étape d'export (#334) | M | — |
| [T0.4](issues/T0.4-mat-entetes-resignation.md) | `RenderMAT` : en-têtes Site/Round/EventDate/Transcriber ; partie finie sans dernier coup (#335) | S | — |
| [T0.5](issues/T0.5-writematch-remplacement.md) | `ingest.WriteMatch` en mode remplacement (même `id`, purge, pas de Trash) (#336) | M | — |
| [T0.6](issues/T0.6-analyse-ciblee-shutdown.md) | Lot gammonNet ciblé sur un match ; `shutdown` annule les lots (#337) | S | — |
| [T0.7](issues/T0.7-importeurs-sentinelle-crawford.md) | Issue jumelle : les importeurs écrivent la sentinelle Crawford ; `repair` rehache (#338) | M | — (indépendante) |

Livrable du lot : `go test ./pkg/blunderdb/transcript/...` vert sur les `.mat` du dépôt ;
un `.mat` rejoué et re-rendu à l'identique ; la base à 2.21.0.

## Lot 1 — transcrire un match au clavier

| Issue | Titre | Effort | Dépend de |
|---|---|---|---|
| [T1.1](issues/T1.1-onglet-mode-store.md) | Onglet, mode `TRANSCRIBE`, store, bindings lister/créer/ouvrir/fermer, `Ctrl+Maj+T`, `transcribe`/`tr` (#339) | M | T0.2, T0.3 |
| [T1.2](issues/T1.2-creation-ouverture.md) | Création (longueur, argent + Jacoby) et ouverture (dé J1, dé J2, égalité) (#340) | S | T1.1 |
| [T1.3](issues/T1.3-des-candidats.md) | Dés au clavier, liste 0-ply, présélection, `j`/`k`, « un chiffre valide », Retour, Entrée, danse (#341) | M | T1.2 |
| [T1.4](issues/T1.4-videau-fin-de-partie.md) | `d`/`t`/`p`, redouble, fin de partie par passe ou sortie, score, Crawford dérivé, partie suivante, fin de match (#342) | M | T1.3 |
| [T1.5](issues/T1.5-resignation.md) | Résignation `r` + niveau ; match inachevé (#343) | S | T1.4 |
| [T1.6](issues/T1.6-transcript-deux-colonnes.md) | Transcript deux colonnes, Cursor-cellule, `h`/`l`, décorations d'Incohérence, parties repliables, texte `.mat` copiable (#344) | M | T1.3 |
| [T1.7](issues/T1.7-correction-replay.md) | Correction en place, `i`/`a`/`x`/`s`, `Ctrl+Z`/`Ctrl+Maj+Z`, Replay, saut à la première Incohérence (#345) | M | T1.6 |
| [T1.8](issues/T1.8-brouillon-durable.md) | Écriture après chaque Action, reprise après plantage, ligne « brouillon » dans le panneau Match (#346) | M | T1.3, T0.3 |
| [T1.9](issues/T1.9-enregistrer-exporter.md) | Enregistrer/remplacer, lot d'analyse ciblé, export `.mat` avec avertissement, fermer le brouillon (#347) | M | T1.7, T0.4, T0.5, T0.6 |
| [T1.10](issues/T1.10-documentation.md) | `manuel.rst`, `raccourcis.rst`, `cmd_mode.rst`, huit `.po`, `make help` (#348) | M | T1.9 |

Livrable du lot : un match en 7 points transcrit au clavier depuis une feuille de score,
corrigé, enregistré, analysé, exporté en `.mat` que gnubg importe sans « Invalid move » ;
budgets 4.1–4.3 de [ux.md](ux.md) tenus à la main. **La documentation fait partie du lot.**

## Lot 2 — la souris et le plateau

| Issue | Titre | Effort | Dépend de |
|---|---|---|---|
| [T2.1](issues/T2.1-triangle-des-jets.md) | Triangle des 21 jets (prototype d'abord) ; ouverture à la souris (#349) | S | T1.3 |
| [T2.2](issues/T2.2-filtre-par-point.md) | Filtre des candidats par clic sur un point de départ (#350) | S | T1.3 |
| [T2.3](issues/T2.3-coup-au-plateau.md) | Coup joué sur le plateau via `quizPlay.js`, dés déduits (#351) | M | T1.3 |
| [T2.4](issues/T2.4-coup-illegal.md) | Coup illégal : déplacement libre puis « ce plateau est le coup » ; saisie texte (#352) | M | T2.3 |
| [T2.5](issues/T2.5-videau-souris-menu.md) | Clic sur le videau, rangée D/T/P/R, menu contextuel du Transcript (#353) | S | T1.7 |

Livrable : le même match transcrit à la souris seule ; budgets souris de ux.md tenus.

## Lot 3 — l'entour

| Issue | Titre | Effort | Dépend de |
|---|---|---|---|
| [T3.1](issues/T3.1-metadonnees-tournoi.md) | Volet métadonnées complet, autocomplétion des joueurs, tournoi, transcripteur, inversion des joueurs (#354) | M | T1.9 |
| [T3.2](issues/T3.2-changement-longueur.md) | Changement de longueur en cours de transcription (#355) | S | T1.7 |
| [T3.3](issues/T3.3-reprise-analyse.md) | Reprise de l'analyse à la réouverture de la base (#356) | S | T1.9 |
| [T3.4](issues/T3.4-cli-transcribe.md) | `blunderdb transcribe` : rejouer, vérifier, rendre (#357) | M | T0.2 |
| [T3.5](issues/T3.5-e2e-budgets.md) | Specs Playwright qui comptent les gestes (`frontend/tests/e2e/`) ; test Go de latence du Replay (#358) | M | T2.5 |

## Ordre et parallélisme

```
T0.1 ─┬─ T0.2 ──────────────┬─ T1.1 ─ T1.2 ─ T1.3 ─┬─ T1.4 ─ T1.5
      │                     │                      ├─ T1.6 ─ T1.7 ─┬─ T1.9 ─ T1.10
      ├─ T0.3 ──────────────┘                      ├─ T1.8 ────────┘
      ├─ T0.4 ─┐                                   ├─ T2.1  T2.2  T2.3 ─ T2.4
      ├─ T0.5 ─┼──────────────────────────────── T1.9                  T2.5 ─ T3.5
      ├─ T0.6 ─┘                                   └─ T3.2   T3.1  T3.3
      └─ T0.7 (indépendante)                        T3.4 (dès T0.2)
```

Trois files parallèles possibles dès le lot 0 : le paquet (T0.2), le schéma (T0.3), les
correctifs d'import/export (T0.4–T0.7). Au lot 1, T1.6 et T1.8 avancent en parallèle de T1.4.

## Ce que le plan ne contient pas, et pourquoi

- **Rouvrir un Match importé comme brouillon** : demande un chemin « Match → document » que
  T3.4 (`transcribe --check` d'un `.mat`) préfigure ; backlog après le lot 3.
- **Match sans règle de Crawford** : backlog (décision du grill).
- **Résignation lue du SGF à l'import** (`ingest/gnubg.go:133` la jette) : hors sujet,
  à noter au `BACKLOG.md`.
- **Panneau Match en deux colonnes** : `TranscriptView.svelte` le permettra ; c'est une
  décision d'ergonomie du panneau Match, pas de ce plan.

## Risques nommés

| Risque | Parade |
|---|---|
| `Ctrl+S` pris globalement par « sauver la position » | vérifier `isAlwaysGlobal` au T1.9 ; sinon `Ctrl+Entrée` |
| `r` est « position au hasard » hors panneau | le panneau a le focus ; la résignation demande un second chiffre et `Échap` annule |
| 0-ply sur un double ouvert (> 100 coups) | mesuré < 1 ms par coup ; test de latence T3.5 |
| Prix documentaire (≈ 75 msgid × 8) | accepté comme section ; aucune page nouvelle |
| Le remplacement casse un pointeur externe sur `game.id`/`move.id` | ces ids ne sont référencés que par `move_analysis` (cascade) ; `last_visited_position` est un index de position, revalidé à l'ouverture |
