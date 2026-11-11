# Reprise — 2026-11-10

État de main au 10 novembre, pour reprendre depuis GitHub sur un autre poste.

## État

- Release **0.53.0** coupée le 2026-11-09 (`DatabaseVersion` 2.57.0). Depuis la reprise du 6 nov. :
  - **0.52.0** : gammonNet v1.9.0 (réseau 768×768 par défaut, le 4 couches embarqué à côté),
    chargement hors ligne de la bibliothèque publique par COPY (rôle chargeur confiné au tenant 0),
    réparation d'une vidéo `.webm`/`.mkv` mal multiplexée, abandon du match en transcription,
    synchronisation après un 402, livre d'ouverture entier au préréglage `standard`.
  - **0.52.1** : correctif de CI (modules babel des PDF sous ubuntu-26.04).
  - **0.53.0** : le répondant à un double peut abandonner la partie ou le match ; sous PostgreSQL,
    les jointures de la recherche, des statistiques, d'Anki, de l'entraînement et de la
    maintenance nomment leur tenant (clés à `tenant_id` en tête depuis la migration 066).
- Sur main après 0.53.0, sans release : un fichier déjà importé est reconnu par ses octets avant
  décodage (#909, migration 067) ; la règle du plus grand dé regarde le dé consommé ; le parcours
  d'une recherche à filtre de match lourd réécrit le prédicat au lieu de lister les parties
  (#911, ferme #910).
- Issues ouvertes : #912 (test ETag intermittent), #846 (poste Windows), #858 et #870 (vision).

## En cours — performance de la recherche sous PostgreSQL (banc gammonGo, 20 M positions)

- **#914** (brouillon, rebasée sur main le 10 nov.) : plateau de fin de partie, l'occupation lue
  par son index quand le pip la réduit (Q31). À relire : le pire cas du nouveau seuil, 591 ms à
  froid pour une occupation de 4 997 lignes au pip épars.
- **Mesure due après #911** : rejouer Q11 et Q12 à 50 clients sur la pile du banc ; #910 s'est
  fermée à la fusion sans elle.
- **#913** (horloge injectable pour l'ETag, répond à #912) : le job `test (rest)` a été annulé à
  la limite de 25 min ; le relancer, et diagnostiquer un blocage s'il la dépasse encore.
- Dependabot #903 (modules Go) et #904 (actions) : en attente.

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

1. Recherche PostgreSQL : #914, la mesure Q11/Q12, #913 (section « En cours »).
2. Vision : campagne matériels, voir son `ETAT.md`.
3. #846 (LNA WebView2) : attend un poste Windows.
4. Travail de fond : voir `tasks/BACKLOG.md`.

## Pièges

- Conflits `.po` de `mode_headless` : prendre l'union des msgid, puis `doc-po-update`.
- e2e : port libre, `--workers=1`, `--timeout=40000`.
