# Reprise — 2026-11-20 (soir)

État de main au 20 novembre, pour reprendre depuis GitHub sur un autre poste.

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
- Le 20 : **#959 fusionnée** (#952 fermée ; champs de liste, index couvrant du tri, filtre de
  match × colonne rare, résultat de match stocké, schéma 2.59.0, ADR-0114) puis **origine d'un
  tournoi** (schéma **2.60.0**, PG 073 ; #961 fermée, ses quatre cases sur main). Les deux seuils
  non tenus au banc partent en suivi : **#965**.
- Issues ouvertes : #965 (seuils de #959), #846 (poste Windows), #858 et #870 (vision).

## CI

- **main vert** (run de 48013308e, le 19) après quatre corrections du jour : `EquityToMWC`
  fusionnait un produit en FMA sur arm64 et amd64-v3 ; `TestDirectionSimulate_ThirdAtOnceIsBusy`
  attendait sans fin un chevauchement ; les gardes d'horloge de la simulation (#962, #963) ;
  réimporter une source de pions déjà tenue réécrivait ses analyses (ci-dessous).
- **Nightly (#935)**, rouge du 13 au 18, deux causes sur `TestImportStoredRowsAreStable` : des
  statistiques de match en plus, voulues, empreinte déjà ré-épinglée ; puis, depuis les sources de
  pions, un réimport de `.xg` ou `.sgf` qui réécrivait `analysis_data` et annonçait des analyses
  « approfondies ». `MergeCheckerSources` garde la source tenue quand l'évaluation est la même
  (`TestReimportOfAStoredSourceWritesNothing`, hors nightly). Nightly vert le 19 au soir, **#935
  fermée**. Le lien gnu.org du linkcheck est une panne passagère (job consultatif).
- Proposé, non fait : `TestImportStoredRowsAreStable` dans le job `test-postgres` de `build.yml`
  (≈ 35 s), qui aurait signalé les deux rouges dès la PR. Vu, laissé : le tri de la vue fusionnée
  est instable (`sort.Slice` dans `rankCheckerMoves`).

## #959 — fusionnée le 20

Détail des mesures au banc : corps de la PR #959 et `~/bmab-banc/RAPPORT-959.md` sur mochy. Banc
laissé en schéma 072, réglages levés ; sauvegarde d'avant montée :
`~/bmab-banc/sauvegarde/pgdata-nvme-0.55.0-avant-959-20261120`. Suites : #965 (tri par pips ASC
sur structure, p95 518 ms pour ≤ 250 ; filtre de match × colonne rare au-dessus de la borne).
Suivi possible, non exigé : une fonction `SECURITY DEFINER` qui ne monte que
`min_reader_version`, au lieu de `INSERT, UPDATE` sur `metadata` pour `blunderdb_loader`.

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

1. Release portant 2.58 → 2.60 (sources de pions, résultat de match, origine des tournois, #961) :
   demande l'accord du propriétaire.
2. #965 : les deux seuils non tenus de #959.
3. Vision : campagne matériels, voir son `ETAT.md`.
4. #846 (LNA WebView2) : attend un poste Windows.
5. Travail de fond : voir `tasks/BACKLOG.md`.

## Pièges

- Conflits `.po` de `mode_headless` : prendre l'union des msgid, puis `doc-po-update`.
- e2e : port libre, `--workers=1`, `--timeout=40000`.
