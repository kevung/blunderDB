# Mesures comparées aux budgets `ux.md`

Une ligne par opération, les cinq scénarios en colonnes, le coût **hors saisie** mesuré (à 1024
puis 1366 quand ils diffèrent), le budget de `ux.md` § 4 et le constat — la colonne que
[../lots.md](../lots.md) transforme en issues. « — » : l'opération n'a pas lieu dans ce
scénario (« sans objet »). Le détail de chaque ligne est dans `S<n>.md`.

| # | Opération | S1 | S2 | S3 | S4 | S5 | Budget | Constat |
|---|---|---|---|---|---|---|---|---|
| O1 | créer + diriger | 4 | — | — | — | — | 2 + nom, puis 2 | tenu |
| O2 | déclarer un par un | 8 (4 noms) | — | — | — | — | ≈ 7 K / nom | tenu |
| O3 | reprendre un tournoi précédent | 3 | — | — | — | — | ≤ 4 | tenu |
| O4 | coller un CSV de 50 lignes | 6 | — | — | — | — | — | tenu en gestes ; **aucune erreur détectée** (ligne sans séparateur, doublon) |
| O5 | tout lancer | 4 / 6 | 2 | 4 | 3 | 2 | 2 | tenu à 24 propositions quand « Confirmer » est visible ; **dépassé** dès que la liste de contrôle le pousse sous la ligne de flottaison (S1 : 1 à 3 crans) ; **en rondes, la moitié de la ronde disparaît** (S4, G5) |
| O6 | feuille d'appariements, page murale | — | 6 | 20 (3 épreuves) | 1 | — | 2 ; 0 après réglage | la feuille : 5 crans pour atteindre le bouton en S2 ; **aucune feuille d'une ronde non lancée** (S4) ; un dossier par épreuve (S3) |
| O7 | résultat sans score | 2 | 2 / 3 | — | — | — | 2 | tenu si la table est visible ; **table du bas de la grille : 3 à 4 crans** (S2) |
| O7 | résultat avec score | 4 (7,1 s) | 8 / 7 (8,3 s) | — | 4 (7,1 s) | — | ≤ 4,5 s | **jamais tenu** en secondes ; Tab entre les deux scores quitte la Direction (E3) |
| O8 | forfait | 3 | — | — | — | — | ≤ 4 | tenu |
| O9 | corriger la dernière | 2 | 2 / 3 | — | — | — | 2 | tenu |
| O9 | corriger un résultat ancien | sans chemin | (I3 moteur seul) | — | sans chemin | — | 4-5 | **sans chemin** (E2) |
| O10 | classement en cours | — | 2 | — | 2 | — | 3 | tenu ; presse-papier seulement (H4) |
| O11 | retirer (filtre) | 2 | 7 | — | 2 | 3 | 4 + 3 K | tenu à 20, **dépassé à 51** (4 crans même après filtre en S2) |
| O11 | retirer (sans filtre, liste de 51) | — | 6 | — | — | — | 4 + 3 K | dépassé de 2 (4 crans) |
| O12 | inscrire un retardataire | 2 | 2 | — | — | — | 2 + 6 K | tenu ; au tableau tiré, n'entre nulle part (H11) |
| O13 | absence prévue | — | sans chemin | sans chemin | 2 + 2 (retrait, puis Corriger) | sans chemin | — | **aucun geste dédié** ; le retour passe par un effet de bord (G4) |
| O14 | table cassée : déplacer le match | 4 | 10 | — | — | — | 3 + 2 K ; ≤ 4 | tenu à 8 tables, **dépassé à 14** (5 crans) ; **aucun champ « table indisponible »** |
| O15 | reprendre après une absence / le matin | — | 1 | — | — | 0 | ≤ 10 s | rien ne dit « depuis votre dernier geste » ; aucune fin estimée (H13) |
| O16 | ajouter une consolante en cours | — | sans chemin | — | — | — | ≤ 4 | **aucun champ** (H5) ; acceptée par la config même après le tirage, sans effet (G3) |
| O16 | passer à la phase suivante | — | 2 / 1 | — | — | — | 1 | dépassé d'une vue (l'onglet) |
| O17 | qui est libre pour une autre épreuve | — | — | sans chemin | — | — | — | **aucune vue** ne croise les épreuves |
| O18 | changer d'épreuve | — | — | 3 | — | — | — | ferme l'autre sans confirmation (H1) ; 1 à 6 fois par heure |
| O19 | clore | 2 | — | — | — | — | ≤ 4 | tenu — **sans confirmation, matchs en cours** (H7) |
| O19 | sections et prix | — | 1 | — | — | — | ≤ 4 | une seule section : la consolante n'a pas la sienne (C6) |
| O19 | rouvrir | — | 2 | — | — | — | ≤ 4 | tenu ; `window.confirm` natif (H7) |
| O20 | rattacher 28 matchs | — | 29 | — | — | — | 1 + n | tenu |
| O21 | historique d'un joueur | 2 | — | — | — | 2 | ≤ 2 | tenu |
| O22 | crédit | 1 (démo) | — | — | — | — | 1 | tenu |

## Ce que la colonne « constat » dit en une phrase

Les gestes unitaires tiennent leurs budgets **quand leur cible est visible**. Ce qui casse à
l'échelle, c'est la **hauteur** : à 768 px, dock ouvert, la Direction a 370 px utiles, la file
de 24 propositions pousse la grille entière hors de l'écran, et **la molette ne défile pas** (E1).
Ce qui n'a **pas de chemin du tout** : corriger un résultat ancien (E2), déclarer une table hors
service, ajouter une consolante, absenter un joueur une ronde, savoir qui est libre pour une
autre épreuve, annoncer une ronde avant de la lancer.

## Temps du moteur

`Open` (rejeu complet) : 1,2 à 7,0 ms selon le journal (scenarios-go § 2) — **budget 50 ms tenu
d'un facteur 7** au pire. Ce n'est pas là que le directeur attend.
