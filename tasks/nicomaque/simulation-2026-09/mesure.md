# La règle de mesure

Décision Q5 ([README.md](README.md) § 2). Ce fichier dit **ce qu'on compte, sur quoi, et
comment on le rapporte**. Il prolonge [ux.md](../ux.md) § 1 (opérateurs KLM) et § 4
(budgets par flux) sans les remplacer : les budgets restent la référence ; la mesure les
confronte à des tournois de taille réelle.

## 1. Les compteurs

On compte **par geste**, pas par seconde. Le temps KLM est calculé à côté, avec les
opérateurs de `ux.md` § 1 (K 0,28 s, clic 1,3 s, gros bouton proche 0,66 s, M une fois par
flux), uniquement pour comparer aux budgets existants.

| Compteur | Ce que c'est | Comment on le compte |
|---|---|---|
| **clic** | un clic de souris sur une cible | `countGestures` de `frontend/tests/e2e/helpers/gestureCount.js` (déjà utilisé par `direction-budgets.spec.js`) |
| **touche** | une touche, y compris `Entrée`, `Tab`, `Échap` ; un nom tapé compte ses lettres | idem |
| **vue** | un changement d'onglet de la Direction (Direction, Joueurs, Arbres, Emplacements, Classement, Historique, Réglages) ou d'onglet du dock | clics sur `[data-testid="direction-tab-*"]` et sur les onglets du dock, comptés à part des autres clics |
| **menu** | un menu déplié (⋯ d'une fiche ou d'une ligne, liste déroulante, panneau replié qu'on ouvre) | clic sur une cible qui révèle d'autres cibles sans agir |
| **modale** | une fenêtre ou une confirmation à fermer avant de continuer (liste « ce qui va changer », `window.confirm`, dialogue système) | apparition d'un `[role="dialog"]`, d'un `.modal`, ou d'un `page.on('dialog')` |
| **défilement** | un cran de molette ou un glissement ; on note aussi les **pixels** parcourus et **quel** conteneur défile | `scrollTop`/`scrollY` avant et après, sur `.scrollable-content`, la grille des tables, la liste des joueurs, la file ; un `wheel` synthétique = 100 px = un cran |
| **lecture** | ce que le persona doit lire ailleurs que sur l'écran où il agit (une infobulle, une autre vue, la doc) pour savoir quoi faire | compté à la main par l'exécutant, justifié d'une phrase |

Une opération a un coût = la somme des six premiers compteurs, et un coût **hors saisie**
= la même somme sans les touches des noms et des scores (c'est ainsi que `ux.md` § 4.2
compte « ≤ 8 clics hors saisie des noms »).

## 2. Les deux viewports

| Viewport | Pourquoi | Poids |
|---|---|---|
| **1024×768** | référence de l'application (`ux.md` § 2, ADR-0021 : mesurer à 1024) | un défilement ici est un constat plein |
| **1366×768** | le portable de salle le plus courant | un défilement qui n'existe qu'à 1024 est noté mais pèse moins |

Chromium système (`/usr/bin/chromium`, voir [outillage.md](outillage.md) § 2), locale
forcée en français ; **une** mesure de contrôle en allemand ou en grec sur l'opération la
plus large (la grille des tables), parce que ce sont les langues qui débordent (ADR-0021).

## 3. Les opérations mesurées

La liste vient de la demande de l'utilisateur, complétée par les incidents de
[scenarios.md](scenarios.md) § 2 et par les flux de [fonctionnel.md](../fonctionnel.md)
§ 11 qui ont un budget. Chaque opération est mesurée **dans l'état réel du scénario** (une
grille de 14 tables occupées, une liste de 50 joueurs, une file de 20 propositions), pas
sur une page vide.

| # | Opération | Flux `ux.md` | Budget existant | Où l'état pèse |
|---|---|---|---|---|
| O1 | créer le tournoi dirigé et choisir le format | F1 | 2 + nom, puis 2 | — |
| O2 | déclarer les joueurs : un par un | F2 | 20 × (≈ 6 K + 1 K) ≈ 40 s | autocomplétion sur une base de 300 Players |
| O3 | déclarer les joueurs : reprendre un tournoi précédent | F3 | 4 clics | annuaire de 200 noms |
| O4 | déclarer les joueurs : coller un CSV de 50 lignes | F4 | — | erreurs par ligne, aperçu |
| O5 | lancer le tournoi / une ronde / tout lancer | F5, F6 | 2 clics | file de 25 propositions : la liste de contrôle tient-elle sans défiler ? |
| O6 | annoncer une ronde : feuille d'appariements, page murale | F24, F23 | 2 + système ; 0 après réglage | S3 : trois épreuves ; S4 : ronde datée à l'avance |
| O7 | saisir un résultat, sans puis avec score | F7 | 2 clics ; ≤ 4,5 s | trouver la table 12 dans une grille de 14 |
| O8 | saisir un forfait | F7 | ≤ 4 clics | — |
| O9 | corriger le dernier résultat ; corriger un résultat ancien | F28, F8 | 2 ; 4-5 | historique de 300 événements : le filtre |
| O10 | publier le classement en cours | F21 | 3 | CSV presse-papier seulement ; page murale |
| O11 | retirer un joueur (immédiat, différé) | F12 | 4 + 3 K | liste de 50 : filtre, défilement |
| O12 | inscrire un retardataire | F13 | 2 + 6 K | places libres proposées |
| O13 | absence prévue à une ronde (I6) | — | — | S4 et S5 : le chemin trouvé, s'il existe |
| O14 | table cassée ; déplacer un match | F11, F15 | 3 + 2 K ; ≤ 4 | en cours : Enregistrer + confirmation |
| O15 | reprendre après une coupure ; après une absence | F17, F18 | 0 ; ≤ 10 s | S5 chaque matin |
| O16 | passer à la phase suivante, ajouter une consolante en cours | F14, F15 | — ; ≤ 4 | S2 samedi soir |
| O17 | savoir qui est libre pour une autre épreuve | — | — | S3 : deux Directions |
| O18 | changer d'épreuve (fermer une Direction, ouvrir l'autre) | — | — | S3 : combien de fois par heure |
| O19 | clore, classer par section, prix, rouvrir pour corriger | F21, F22 | ≤ 4 | — |
| O20 | transcrire depuis un Slot ; rattacher un Match importé | F19, F20 | ≤ 3 ; 1 + n | S2 : 28 matchs à rattacher |
| O21 | consulter l'historique d'un joueur | F25 | ≤ 2 | 300 événements |
| O22 | lire le crédit | F27 | 1 | — |

Une opération que le scénario ne rencontre pas est marquée « sans objet ». Une opération
qu'on n'a pas pu mesurer est marquée « non mesuré » avec la raison — **jamais estimée**
(décision Q9, cas limite : un bug bloquant devient une issue, le scénario continue au
suivant).

## 4. La grille du rapport

Un tableau par scénario, une ligne par opération réellement faite, dans l'ordre du
déroulé ; le même ordre de colonnes partout, pour que `mesures.md` puisse les agréger :

```
| heure | persona | opération | clics | touches | vues | menus | modales | défil. (crans / px / conteneur) | lecture | KLM (s) | budget ux.md | écart | capture |
```

- **écart** : « tenu », « dépassé de n », « sans budget » ; à 1024 et à 1366 quand ils
  diffèrent (deux lignes).
- **capture** : le nom du fichier, seulement quand un défilement, une modale ou une
  confirmation apparaît.
- Pour un incident, une ligne de plus, en prose sous le tableau : **voulu** (ce que le
  persona cherchait à faire), **trouvé** (le chemin que l'interface offre), **fait à la
  place** (le contournement, s'il y en a un), **gestes**, **ce qui manque**.

`mesures.md` agrège : une ligne par opération, les cinq scénarios en colonnes, le budget
`ux.md`, et une colonne « constat » qui ne dit qu'une chose : tenu partout, tenu à 20 mais
pas à 50, jamais tenu, sans chemin. C'est cette colonne que [lots.md](lots.md) transforme
en issues.

## 5. Ce qui n'est pas une mesure

- Le temps de calcul du moteur (`R` de `ux.md` § 1) : mesuré une fois par scénario Go
  (`BenchmarkOpen` sur le journal final) et rapporté à part, pas par opération.
- L'esthétique, la couleur, la police : hors sujet ici (ADR-0008, ADR-0031).
- Ce que le persona « aurait trouvé plus joli » : seul compte ce qu'il n'a pas pu faire ou
  ce qui lui a coûté plus que le budget.
