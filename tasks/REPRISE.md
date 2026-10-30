# Reprise — 2026-10-30

État de main au 30 octobre, pour reprendre depuis GitHub sur un autre poste.

## État

- Release **0.47.1** coupée le 2026-10-28 (tags 0.47.1, 0.47.0, 0.46.1) ; main a avancé le 30 oct.
  (chantier vision : ADR-0108 à 0111, lot V0, `transcribe --marks`, CORS du serveur média, doc YouTube).
- Nightly verte sur `f208f5060` (2026-10-30) ; non relancée depuis les fusions du 30 oct.
- Issues ouvertes : #664, #842, #845 (vérif. à la main), #846 (poste Windows), #847 (annotation).

## Fusionné depuis le 22 octobre

Les anciennes prochaines étapes sont toutes closes :

| Issue | Sujet |
|---|---|
| #788 | Stats : tournamentReview en lot de joueurs |
| #789 | Stats : garantie documentée (openapi.yaml) sur l'intervalle et l'effectif des cellules |
| #783 | Transcription R3 : transcriptions.import (brouillon depuis un document ou un .mat) |
| #790 | ADR : Nicomaque, package Go de règles de tournoi en fonctions pures partagé gammonGo/blunderDB |
| #791 | Moteur embarqué : relayer les gestes de transcription entre instances (EventsDSN dans Bootstrap) |
| #792 | Transcription : le classement des candidats au bureau attend un jeton HoldCore sous d.mu.RLock |
| #793 | Moteur : parité au bit amd64/arm64 (fusion FMA par le compilateur Go) |
| #760 | Nightly failed: 2026-10-22 (fermée, nightly revenue verte) |

Travaux récents :

- #839 : import .db (total annoncé, analyse sans import, plafond de positions, marque d'étude, annulation propre).
- PR #841 : `training.generateBearoff` / `training.generateEvaluation` (serve).
- Montées de dépendances (modernc.org/sqlite 1.61.0, echo/v4 4.16.1, actions CI).

## Tests

Nightly verte sur `f208f5060` (runs des 29 et 30 octobre).

## Branches

Aucune branche `feat/*` en cours. Les worktrees de mesure `blunderDB-base` et `blunderDB-f16probe`
sont supprimés. PR #809 (fix(domain): boardKey ignore la couleur d'une case vide) : fermée le 30 oct., remplacée par #802 sur main ;
sa partie encore utile (quiz et `duel --play` reconnaissent un coup par le plateau atteint) est l'issue #842.

## Prochaines étapes

1. #664 livre d'ouverture : rapatrier `book.jsonl.gz` de smith quand le calcul est terminé.
2. #842 : quiz et `duel --play` par le plateau atteint (extrait de la PR #809).
3. Chantier vision (`tasks/transcription-vision/README.md` → `chantiers.md`). Lot V0 fait le 30 oct. :
   V0.1 vert (Worker depuis `wails://`), V0.2 rendu (hiérarchie par états = mode normal), V0.3 fusionné
   (#845 reste ouverte pour la vérification à la main sous `make dev`, procédure dans `bancs/03-cors.md`),
   V0.5 outils livrés (#847 : Kevin annote 50 coups), V2.1 `transcribe --marks` fusionné ; H1, H2,
   Dependabot #112 clos ; #846 (LNA WebView2) attend un poste Windows.
   **Suite** : issue V1.0 = accepter l'ADR-0109 (statut, réécriture en place de l'ADR-0082 règle 3,
   index), puis le lot V1 dans l'ordre de `chantiers.md` (V1.1 ∥ V1.2 d'abord). À lancer par
   `/traiter-lot` dans une session **Opus** ; Fable ne sert qu'aux ADR et aux arbitrages.
4. Travail de fond : voir `tasks/BACKLOG.md`.

## Pièges

- Le générateur de `models.ts` (wailsjs) omet `ElsewherePlay` / `ElsewhereNeighbour` : les rétablir à la main.
- Conflits `.po` de `mode_headless` : prendre l'union des msgid, puis `doc-po-update`.
- e2e : port libre, `--workers=1`, `--timeout=40000`.

## Remarques non bloquantes des relectures

- Annuler une copie laisse les transcriptions (match_id NULL), alias et tables d'équité apportés ;
  une copie qui crée une leçon n'a pas de reçu. Le commentaire de `TakeBackCopy` (`db_copy.go`) ne vaut que pour SQLite.
- `stats.timeErrors` : `matchIds: []` = toute la base (convention matchBadges). La liste `IN (...)`
  n'est pas découpée (limite ~32 000 identifiants, comme MatchBadges).
- `prByWindow` par phase fusionne `game_phase` NULL et 0 en un seul « unknown », `Compute` les sépare.
