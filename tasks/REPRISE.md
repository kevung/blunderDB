# Reprise — 2026-10-22 (soir)

État laissé par la session autonome blunderdb-e2, pour reprendre depuis GitHub sur un autre poste.
Pas de release, pas de tag.

## Fusionné dans main (poussé, issues fermées)

| Issue | Sujet | Revue |
|---|---|---|
| #772 | Moteur : réseau float16 (.bin16, BGN6) de gammonNet v1.8.1 ; ADR-0104 ; démo régénérée | spec + invariants (Opus) |
| #775 | Échanges entre bases : « Autres bases » devient le poste d'échange (lot 2 : présence d'objet, annulation par reçu, e2e de chaque état) | spec + invariants (Opus), reprise assouplie ×1 |
| #781 | `server.Config.Transcription` / `TranscriptionTTL` (moteur embarqué) | spec (Opus) |
| #782 | `/v1/transcriptions.candidates` : 0-ply sans élagage ni plafond, hors quota ; le panneau du bureau passe par `Database.TranscriptionCandidates` (152 coups au lieu de 12 sur 2-2) | spec + invariants + re-revue (Opus) |
| #784 | État de transcription : `video_source`, repères, durées testés et documentés | spec + invariants (Opus) |
| #785 | `stats.timeErrors` : filtre `matchIds` | spec + invariants (Opus), contrat PG vert |
| #786 | `stats.summary` anonymisé (allow-list) | idem |
| #787 | `prByWindow` découpé par phase | idem |

## Tests

CI `build.yml` de main **verte** sur dcaff7c70 (run 39825966857, 31 jobs success, dont test-os macOS,
golangci-lint, svelte-check, frontend-e2e 336 passés sans flaky). Pour y arriver après la fusion du lot :

- svelte-check : 1253 → 1241 erreurs (plafond `.svelte-check-budget` abaissé à 1241) ; errcheck dans
  `cmd/blunderdb-loadtest/scenario_test.go`.
- macOS (arm64) : TestCrawfordSearchWithCubeIsTheCubelessSearch rouge depuis float16 — la fusion FMA
  de `a*b+c` par le compilateur Go arrondissait différemment les chemins avec et sans cube ; barrières
  `float64(a*b)` dans `cube.go` (reproduit avec GOAMD64=v3 ; amd64 de base et golds inchangés).
  Parité au bit amd64/arm64 du reste du moteur : #793.
- e2e : mocks Wails de `TranscriptionCandidates` manquants (24 échecs) ; vol de focus réel du panneau
  des matchs sur un menu contextuel ouvert dans les 100 ms (`utils/panelFocus.js`).

## Branches restantes

Aucune branche `feat/*` en cours. Worktrees locaux de mesure gardés : `../blunderDB-base`,
`../blunderDB-f16probe` (HEAD détaché, jetables depuis la fusion de #772).

## Prochaines étapes

1. Vérifier la CI de main (build.yml) après ces fusions.
2. Lot gammonGo restant : #788 (tournamentReview en lot), #789 (garantie openapi intervalle/effectif),
   #783 (transcriptions.import, le plus gros), #790 (ADR Nicomaque : ADR d'abord, pas de code).
3. Suites ouvertes pendant le lot : #793 (parité au bit amd64/arm64, FMA), #791 (EventsDSN dans `Bootstrap`, relais des gestes entre
   instances), #792 (le classement des candidats au bureau attend un jeton HoldCore sous `d.mu.RLock` ;
   tests -race de RankAllPlaysWithMET).
4. #664 livre d'ouverture : rapatrier `book.jsonl.gz` de smith quand le calcul est terminé.
5. #760 : fermer après le prochain nightly vert.
6. À la release : `doc/source/historique.rst` (v1.8.1 float16, péremption des analyses v1.7.0 ;
   l.20-22 dit encore « gammonNet v1.7.0 »).

## Remarques non bloquantes des relectures

- Annuler une copie laisse les transcriptions (match_id NULL), alias et tables d'équité apportés ;
  une copie qui crée une leçon n'a pas de reçu. Le commentaire `db_copy.go:143-146` ne vaut que pour SQLite.
- `stats.timeErrors` : `matchIds: []` = toute la base (convention matchBadges) ; signalé à gammonGo.
  La liste `IN (...)` n'est pas découpée (limite ~32 000 identifiants, comme MatchBadges).
- `prByWindow` par phase fusionne `game_phase` NULL et 0 en un seul « unknown », `Compute` les sépare.
