# Reprise — 2026-11-06

État de main au 6 novembre, pour reprendre depuis GitHub sur un autre poste.

## État

- Release **0.51.0** coupée le 2026-11-06. Depuis la reprise du 30 oct. : stockage compact,
  recherche rapide, synchronisation du bureau avec un serveur distant, changeset (paquets Anki,
  transcriptions, tournois dans les deux sens), corrections du panneau de transcription.
- Livre d'ouverture (#664) : le calcul `standard` de smith (terminé le 23 oct.) est embarqué pour
  les 20 scores de la grille ; les 12 autres scores (money, DMP, Crawford, match en 5) restent au
  préréglage `fast`. Chaque ligne porte sa signature.
- Issues ouvertes : #845 (vérif. CORS à la main), #846 (poste Windows), #858 et #870 (vision).

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
- **Bloquant** : les prototypes (`~/src/vision-proto/`) et les notes de banc (`~/bkgm/bancs/`)
  sont sur un poste auquel Kevin n'a pas accès pour l'instant. Reprendre de là, ou les recopier.

## Prochaines étapes

1. #845 : vérification à la main sous `make dev` (procédure dans
   `tasks/transcription-vision/bancs/03-cors.md`).
2. Vision : reprendre #870 sur le poste qui a le prototype (voir ci-dessus).
3. #846 (LNA WebView2) : attend un poste Windows.
4. Travail de fond : voir `tasks/BACKLOG.md`.

## Pièges

- Conflits `.po` de `mode_headless` : prendre l'union des msgid, puis `doc-po-update`.
- e2e : port libre, `--workers=1`, `--timeout=40000`.
