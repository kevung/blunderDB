# Reprise — 2026-11-26

État de main au 26 novembre, pour reprendre depuis GitHub sur un autre poste.

## État

- Lot du 2026-11-26 **clos** : toutes ses issues sont fusionnées (#987-#997 et #1006). Release
  **0.60.0** préparée sur `release/0.60.0` (détail : `git log --oneline 0.59.0..main`) :
  - **Cartes des jets et des scores** : trois paliers 0 → 1 → 2 plis, cache, précalcul en fond de
    la carte un jet plus bas, priorité aux calculs affichés (ADR-0120), panneau Calculs.
  - **ArkID**, **archive papier** en codes QR, **feuille de match** et **listes de positions**
    imprimables avec leur code de réimport (#987).
  - **Import de document** : relevé v2, lecteur de structure, PDF scannés, assistant graphique avec
    revue, défauts appris par document, JPEG 2000 et JBIG2 en Go pur (#993, #997).
- Issues ouvertes : #998-#1005 (défauts de lecture de l'import de document trouvés par la
  mesure), #846 (poste Windows), #858 et #870 (vision).

## CI

- Cycle d'import de transcript cassé (paquet `matrender`) ; #973 et #976 fermées.
- `document/structure` tourne dans son propre lot, sans `-race`.
- Proposé, non fait : `TestImportStoredRowsAreStable` dans le job `test-postgres` de `build.yml`
  (≈ 35 s). Autres suivis : `tasks/BACKLOG.md`.

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

1. **Lot suivant proposé : #998-#1005**, les défauts de lecture de l'import de document trouvés
   par la mesure : pions dessinés en lettres X/O (#998), videau dessiné (#999), plateau de sortie
   (#1000), dés comptés dans l'image (#1001), contexte en légende sans couche texte (#1002),
   police Snowie (#1003), Masterclass (#1004), score imprimé non lu en entier (#1005).
2. Essai réel imprimer-plier-scanner de l'archive papier.
3. Vision (#858, #870) et #846 : en pause, inchangés.

## Pièges

- Conflits `.po` de `mode_headless` : prendre l'union des msgid, puis `doc-po-update`.
- e2e : port libre, `--workers=1`, `--timeout=40000`.
- `npm run check:types` local compte environ 4 erreurs de plus que la CI ; `fake-indexeddb`
  manquant dans `node_modules` fait échouer deux tests vision : `npm install`.
