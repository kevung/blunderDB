# Reprise — 2026-11-20

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
- Issues ouvertes : #961 (bibliothèque, trois cases restantes), #952 (portée par #959), #846
  (poste Windows), #858 et #870 (vision).

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

## En cours — #959, suite de #952 (brouillon, branche `perf/952-suite`)

- Les quatre cases sont faites : champs de liste des positions, index couvrant du tri par pips
  (migration 071), filtre de match × colonne rare, **résultat de match conservé** (migration 072,
  étape SQLite 2.58 → 2.59, GUI, CLI, doc en 9 langues). `feat/champs-liste` y est fusionnée ;
  les branches `perf/952-resultat-wip*` sont supprimées.
- Une base SQLite ouverte en lecture seule sans migration lit un résultat inconnu au lieu
  d'échouer sur la colonne absente.
- **Plancher de lecture 2.59.0, monté par l'écriture** (ADR-0114, 13b9d4426) : `MinReaderVersion`
  reste 2.50.0 (ADR-0113 §6 : jamais de montée à l'ouverture) ; `MatchResultReaderVersion` vaut
  2.59.0 et le plancher y monte dans la transaction qui écrit un résultat connu (`Save`,
  `Reinstate`, `ReplaceHeader` des deux backends, export SQLite). main refusionnée, suite
  PostgreSQL complète verte en local, CI de la PR verte.
- **Deux points validés le 20** (rien à changer dans la branche) :
  - le rôle `blunderdb_loader` reçoit `INSERT, UPDATE` sur `metadata` (il n'avait que `SELECT`),
    comme le rôle du démon ; sans quoi `bulkload` échoue à monter le plancher. Suivi possible, non
    exigé : une fonction `SECURITY DEFINER` qui ne fait que monter `min_reader_version` ;
  - une base migrée de 2.58 garde ses résultats rétro-calculés **sans** monter le plancher
    (ADR-0114 §3) : un binaire 2.58 peut encore y importer des matchs au résultat inconnu, jusqu'au
    premier résultat écrit par un binaire 2.59.
- **Mesures au banc faites le 20**, reportées dans le corps de la PR (rapport :
  `~/bmab-banc/RAPPORT-959.md` sur mochy). (A) main `48013308e` contre (B) `13b9d4426`, base
  pleine, p50 / p95 en ms :
  - tri par pips ASC sur une structure : 1 207 / 2 315 → 296 / 518 ; **seuil p95 ≤ 250 non tenu**
    (plan voulu, sans Sort ; reste le filtre de structure lu ligne à ligne dans l'index). DESC
    6,9 / 50 : tenu ;
  - filtre de match × colonne rare : tenu quand la colonne est sous la borne de 10 000 positions
    (p95 82 à 114 contre 220 à 809) ; **inchangé au-dessus** (p95 jusqu'à 6,5 s), la PR retombant
    sur l'ancien énoncé ;
  - aucune régression des filtres seuls ; un passage lent du filtre d'erreur aux réglages de
    production ne s'est pas reproduit (B, A, B égaux), cause non trouvée ;
  - champs de liste : + 0,6 ms au p50 d'une page ; Q11 et Q12 à 50 clients (dû depuis #911) : pas
    de régression ; migration 071 : 45 s et 2 × 935 Mo ; 072 : 1 à 2 s.
- **Banc laissé** en (B), schéma 072, réglages levés ; sauvegarde d'avant montée :
  `~/bmab-banc/sauvegarde/pgdata-nvme-0.55.0-avant-959-20261120`. `/` de mochy : 8,6 Go libres.
- **Reste avant fusion** : décider si les deux seuils non tenus bloquent la fusion ou partent en
  suivi (la PR ne dégrade rien et gagne partout où elle agit) ; puis sortir du brouillon,
  fusionner, fermer #952.

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

1. #959 : trancher sur les deux seuils non tenus (tri par pips ASC, colonne au-dessus de la
   borne), fusion ; fermer #952.
2. #961 : tri des matchs par PR, origine d'un tournoi, critères des collections vivantes (la
   version qui les porte demande l'accord du propriétaire).
3. Vision : campagne matériels, voir son `ETAT.md`.
4. #846 (LNA WebView2) : attend un poste Windows.
5. Travail de fond : voir `tasks/BACKLOG.md`.

## Pièges

- Conflits `.po` de `mode_headless` : prendre l'union des msgid, puis `doc-po-update`.
- e2e : port libre, `--workers=1`, `--timeout=40000`.
