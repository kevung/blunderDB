# Reprise — 2026-11-19

État de main au 19 novembre, pour reprendre depuis GitHub sur un autre poste.

## État

- Release **0.57.0** coupée le 2026-11-16. Depuis la reprise du 10 nov. (détail :
  `doc/source/historique.rst`) :
  - **0.53.1** à **0.54.0** : gestes de nicomaque 0.9 servis par `directions.*`,
    `directions.simulate`, export sans provenance, Go 1.27.4.
  - **0.55.0** : recherche lue page à page au curseur ; import rapide d'un nouveau tenant dans une
    grande base PostgreSQL.
  - **0.56.0** : la relation « proche » compte des jets légaux (`like<n>`, `like*` retiré) ;
    `directions.confirmProposals` ; sonde bornée structure × filtre, index des coups chronométrés
    (migration 068) ; les statistiques de match survivent à un import.
  - **0.57.0** : collage d'un identifiant GNU Backgammon ; blot en SQL (migration 069), erreur par
    deux branches bornées, joueur / adversaire / tournoi / longueur par marche bornée.
- Sur main après 0.57.0, sans release (`DatabaseVersion` **2.58.0**) : trace des chances de gagner
  le match (`stats.matchWinTrace`, première case de #961), évaluations de pions gardées par source
  (ADR-0113, étape 2.57 → 2.58), graphe des durées de décision, nicomaque v0.10.0, bornes de
  `directions.simulate`, plateau de fin de partie lu par son index (#914, fusionnée le 19).
- Issues ouvertes : #961 (bibliothèque, trois cases restantes), #952 (portée par #959), #935
  (nightly), #846 (poste Windows), #858 et #870 (vision).

## CI

- Deux rouges de main corrigés le 19 : `EquityToMWC` fusionnait un produit en FMA sur arm64 et
  amd64-v3 (`TestEngineEmitsNoFusedMultiplyAdd`) ; `TestDirectionSimulate_ThirdAtOnceIsBusy`
  attendait sans fin un chevauchement que les bornes de la simulation ne garantissent plus.
- Restaient `TestSimulate_UsefulOrTooHeavy` et `TestSimulate_WorkWithinTheBounds` (`hostile-smoke`,
  `test (rest)`), repris par #962 et #963 : **vérifier que le run de 2faadf91e est vert**.
- **Nightly rouge chaque jour depuis le 13 nov. (#935)**, non diagnostiqué :
  `TestImportStoredRowsAreStable` (PostgreSQL) lit 26 lignes `match_stats` stockées contre 22
  enregistrées (942 contre 792 pour `match_stats_cell`, 6 contre 4 pour `match_stats_position`) —
  à rapprocher des statistiques conservées à l'import (0.56.0) : empreinte à ré-épingler, ou
  lignes en trop. Le lien gnu.org du linkcheck est une panne réseau.

## En cours — #959, suite de #952 (brouillon, branche `perf/952-suite`, 2c63d7584)

- Les quatre cases sont faites : champs de liste des positions, index couvrant du tri par pips
  (migration 071), filtre de match × colonne rare, **résultat de match conservé** (migration 072,
  étape SQLite 2.58 → 2.59, GUI, CLI, doc en 9 langues). `feat/champs-liste` y est fusionnée ;
  `perf/952-resultat-wip` et `-wip-2` sont le doublon abandonné (branches à supprimer).
- Une base SQLite ouverte en lecture seule sans migration lit un résultat inconnu au lieu
  d'échouer sur la colonne absente.
- **À arbitrer avant fusion** : `MinReaderVersion` est resté à 2.50.0, donc un binaire 2.50 à
  2.58 écrit encore dans une base 2.59 ; les matchs qu'il importe gardent un résultat NULL que
  rien ne recalcule. Monter le plancher, ou recalculer à la lecture.
- **À faire avant fusion** : la suite PostgreSQL complète (seul un sous-ensemble ciblé a tourné en
  local) ; `hostile-smoke` de la PR, rouge avant #962/#963 — refusionner main ; toutes les mesures
  « à mesurer » du corps de la PR, au banc gammonGo base pleine.
- **Mesure toujours due depuis #911** : Q11 et Q12 à 50 clients sur la pile du banc.

## Chantier vision — en pause, bloqué hors dépôt

Fait jusqu'au 2 nov. : lot V0, V1.0, lot V1 livré (noyau Worker, géométrie, automates,
calibration, ligne de temps, dHash de l'overlay, doc) ; la vision est exclue des versions
publiées (857d10d17).

- **V1.7 (#858) rouge** : détection census F1 0,046 contre les Repères (match 285), gain de
  navigation 2,4 % ; après #868 et réglage par rejeu, 0,37 pour un hasard de 0,28.
- **Détecteur de dés appris (#870)**, insensible à la couleur, trois matériels (v1/v2 BMAB Paris,
  v3 match 286, v4 match 287) : la détection dépasse le hasard sur matériel non vu avec deux
  matériels à l'entraînement (F1 0,54–0,60 contre 0,35–0,38) ; la **lecture des valeurs ne suit
  pas** (8–22 % de paires exactes) ; 4–6 faux positifs/min. Pas de révision des ADR-0109/0110,
  pas de fusion V1/V2.
- **Avant de trancher** : un 4ᵉ matériel, le lecteur mesuré seul sur des dés bien localisés,
  plus de jets étiquetés en valeurs sur v3 et v4.
- Prototypes (`~/src/vision-proto/`) et notes de banc (`~/bkgm/bancs/`) : accessibles de
  nouveau ; la campagne `~/bkgm/bancs/campagne-materiels/ETAT.md` fait foi (fusion pions + dés).

## Prochaines étapes

1. CI : run de main, puis le nightly (#935).
2. #959 : arbitrage `MinReaderVersion`, suite PostgreSQL, mesures au banc, fusion ; fermer #952.
3. #961 : tri des matchs par PR, origine d'un tournoi, critères des collections vivantes (la
   version qui les porte demande l'accord du propriétaire).
4. Vision : campagne matériels, voir son `ETAT.md`.
5. #846 (LNA WebView2) : attend un poste Windows.
6. Travail de fond : voir `tasks/BACKLOG.md`.

## Pièges

- Conflits `.po` de `mode_headless` : prendre l'union des msgid, puis `doc-po-update`.
- e2e : port libre, `--workers=1`, `--timeout=40000`.
