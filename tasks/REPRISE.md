# Reprise — 2026-11-26

État de main au 26 novembre, pour reprendre depuis GitHub sur un autre poste.

## État

- Dernière release : **0.59.0** (non publiée). Depuis, sur main (détail des commits :
  `git log --oneline 0.59.0..main`) :
  - **Cartes des jets et des scores** en trois paliers 0 → 1 → 2 plis : le palier 1 s'achève en
    50-150 ms, la première case à 2 plis en 0,2-0,8 s au lieu de 0,3-3,6 s, valeurs finales
    identiques.
  - **Priorité aux calculs affichés** (ADR-0120) : un calcul visible réquisitionne les cœurs des
    lots de fond ; 44 ms en médiane au lieu de 0,8-4,6 s.
  - **ArkID** (#984), notation textuelle d'une position ; **archive papier** en codes QR (#985)
    avec ses entrées d'interface, y compris l'import depuis la liste vide et `import_archive` ;
    **feuille de match imprimable** (#986, fermée) ; **listes de positions imprimables** (#987,
    ouverte : reste le code de réimport des listes, choix de conception consigné en commentaire de
    l'issue).
  - **Import de document** : Relevé v2 (deux certitudes par entrée, pions et contexte ; défauts du
    document et sens de jeu confirmés une fois ; `--accept-complete`) ; lecture du contexte par le
    lecteur de structure (videau, dés, trait, score). Aucun contexte n'est entièrement « lu » sur
    751 diagrammes ; mesure faite sans le banc `~/bkgm/bancs/import-document/`, absent du poste.
  - Décision du 2026-11-26 : notation du videau proposé `X?` / `X2?` confirmée ; Relevé v2 et état
    « 15 + 15 sans doute » acceptés.
- Issues ouvertes : #987, #988 (garde `splitDepth`), #846 (poste Windows), #858 et #870 (vision).

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

1. **Cartes** : cache des cartes (clé position + profondeur + PruneK + MET), puis séquences de
   plusieurs jets successifs : précalcul en fond des cartes suivantes à 0 puis 1 pli pour que
   descendre d'un jet soit instantané, 2 plis sur la seule branche regardée. Coûts mesurés : deux
   jets 0,2 s à 0 pli, 1,3 s à 1 pli, 100-150 s à 2 plis ; trois jets 5 s, 30 s, 35-50 min.
   Exigence : analyse quasi instantanée et mise à jour progressive.
2. **#987** : code de réimport des listes, après décision de conception.
3. **Import de document** : défaut de contexte appris par document (un score de match lu rend
   aujourd'hui tout Deep Cube et Olsen « à vérifier »), repli assisté, assistant graphique
   (tranche 5), JPEG 2000, mesure finale sur la réserve.
4. **#988** : garde `splitDepth`, puis retrait du contournement de `runMapPhase`.
5. Essai réel imprimer-plier-scanner de l'archive papier.
6. Vision et #846 inchangés.

## Pièges

- Conflits `.po` de `mode_headless` : prendre l'union des msgid, puis `doc-po-update`.
- e2e : port libre, `--workers=1`, `--timeout=40000`.
- `npm run check:types` local compte environ 4 erreurs de plus que la CI ; `fake-indexeddb`
  manquant dans `node_modules` fait échouer deux tests vision : `npm install`.
