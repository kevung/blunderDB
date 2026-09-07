# Diriger un tournoi — UX et budgets de gestes

Ce document mesure ; [fonctionnel.md](fonctionnel.md) définit. Deux contraintes de premier
rang, posées par l'utilisateur pendant le cadrage (D17, D18) : **la souris est première, le
clavier accélère** ; **la charge mentale et le coût d'entrée sont bas** — un TD qui dirige un
tournoi par an ne doit rien apprendre, un TD qui en dirige un par mois ne doit rien attendre.
Les budgets sont tenus par persona ([decisions.md](decisions.md), D16) : Marc (club, seul,
interrompu), Sophie (grand tournoi, rigueur), Yanis (première fois), Léa (dirige et joue).

## 1. La règle de mesure

Opérateurs KLM (Card, Moran & Newell), identiques à [../transcription/ux.md](../transcription/ux.md) :

| Opérateur | Valeur | Ce que c'est |
|---|---|---|
| K | 0,28 s | une touche |
| P | 1,10 s | pointer une cible (0,46 s pour une cible de 40 px à 300 px, loi de Fitts) |
| B | 0,10 s | presser ou relâcher (un clic = 0,20 s) |
| H | 0,40 s | main clavier ↔ souris |
| M | 1,35 s | préparation mentale, **comptée une fois par décision**, hors comparaison |
| R | mesuré | rejeu de la Direction + `Propose` : à mesurer sur 64 joueurs × 300 événements ; budget < 50 ms, hors chemin sinon |

Un **clic** ci-dessous vaut P + B + B ≈ 1,3 s sur une cible ordinaire, 0,66 s sur une grosse
cible proche (bouton de 40 px dans la fiche ouverte). Les budgets sont exprimés en clics et
en secondes ; les specs Playwright du lot 3 comptent les clics.

## 2. Le placement

La vue tournoi remplace le plateau dans `.scrollable-content` quand l'onglet Tournoi est
actif et qu'une Direction est ouverte (D15). Le panneau ancré (bas ou côté) reste le panneau
Tournoi existant. Largeur de référence : 1024 px, comme le reste de l'application.

```
┌ Open de Lyon · en cours · Suisse 2 vies → Tableau ── [Direction] [Joueurs] [Arbres] [Classement] [Historique] [Réglages] ⓘ ┐
│ 21:42 · 3 h 12 · joués 28 · en cours 6 · restants ~14 · 8,4 min/pt (8) · fin est. 23:10 · ⚠ 2 · pause 12:30          │
├──────── propositions (3)  [Tout lancer] ────────────┬─────────────── tables (12) ────────────────────────────┤
│ ▶ Alice – Bob      7 pts · table 5   [Lancer] ⋯     │ ┌ 1 Émile–Fanny ┐ ┌ 2 libre ┐ ┌ 3 ⚠ 1h32 Gilles–Hugo ┐ │
│ ▶ Chloé – Dan      7 pts · table 8   [Lancer] ⋯     │ │ 7 pts · 0h41  │ │         │ │ 7 pts · lent          │ │
│ ▶ bye : Fanny (ronde 4)              [Donner]       │ └───────────────┘ └─────────┘ └───────────────────────┘ │
│ [+ Apparier à la main]                              │ … une case par table ; clic = fiche de résultat         │
├──────── en attente (4) : Gilles · Hugo · Inès · Jean ─── prochaine micro-ronde dans 03:12 ────────────────────┤
│                                                                                                               │
├─ panneau ancré : Tournoi ─────────────────────────────────────────────────────────────────────────────────────┤
│ Open de Lyon ▸ (dirigé, en cours)    Matchs rattachés (3 / 28 slots)   [Ouvrir la direction] [Exporter…]     │
└───────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

En dock latéral, la page Direction empile propositions, tables, attente ; la grille des
tables passe à trois colonnes. Sous 900 px de large pour la zone, les cases de table
deviennent des lignes.

### 2.1 Les vues secondaires

Chaque vue est un onglet de la barre de titre, un clic, sans état à retenir ; la page
Direction est toujours à un clic (ou `Échap` depuis n'importe quelle vue).

| Vue | Contenu | Gestes propres |
|---|---|---|
| Joueurs | table des Participants : nom, club, cote, vies/défaites, victoires, adversaires, état (en jeu, libre, retiré, bye) ; filtre texte ; [+ Inscrire] [Annuaire ▾] | inscrire, retirer (menu ⋯ sur la ligne : retirer maintenant / après son match, corriger nom/club/cote) |
| Arbres | une section par bloc (SVG de `render/`), poules en grille, blocs GSL, consolante à côté du principal ; zoom par molette ; clic sur un match = sa fiche | — |
| Classement | par section, rang, joueur, note, prix ; [CSV] | — |
| Historique | événements horodatés, filtre par joueur ou match ; menu ⋯ = corriger / annuler quand l'événement s'y prête | corriger, annuler |
| Réglages | phases, tables, pauses, dotation, tirage, sorties (dossier de la page HTML), suppression de la Direction | en préparation : édition libre ; en cours : les champs figés sont grisés avec la raison |

### 2.2 La fiche de résultat

Ouverte par un clic sur une case de table (ou sur un match en cours ailleurs), en surimpression
au-dessus de la case, jamais une fenêtre modale plein écran :

```
┌ Table 3 · Gilles – Hugo · 7 pts · 1h32 ──────────── ⋯ ┐
│                                                        │
│   ┌──────────────────┐     ┌──────────────────┐        │
│   │      Gilles      │     │       Hugo       │        │   ← deux gros boutons : le vainqueur
│   └──────────────────┘     └──────────────────┘        │
│   score  [ 7 ] – [ 4 ]   (facultatif)                  │
│                                              [Valider] │
└────────────────────────────────────────────────────────┘
      ⋯ → forfait de Gilles / forfait de Hugo / remarque… / changer de table / annuler ce match
```

Cliquer un nom **valide** si aucun score n'est saisi (un clic = un résultat). Saisir un score
avant de cliquer le nom valide au clic du nom. `Entrée` = Valider, `Échap` = fermer. La
remarque et le forfait sont dans le menu ⋯ : visibles pour qui les cherche, absents pour les
autres (D17).

### 2.3 Une proposition

Une ligne : les deux noms en grand, longueur, table, [Lancer], et ⋯ (changer la table,
changer la longueur, ignorer pour l'instant — ce qui la retire de l'affichage jusqu'au
prochain `Propose`). [Tout lancer] lance la file entière après une liste de contrôle sur
laquelle on clique [Confirmer]. Une proposition « en attente de table » porte un champ table
vide et [Lancer] demande le numéro.

### 2.4 La bande d'horloge

Une ligne de texte, pas de tableau de bord. Les nombres ne clignotent pas ; les
avertissements sont un compteur cliquable qui fait défiler la vue jusqu'à l'objet concerné.

## 3. Le clavier (D18)

Ce que le clavier fait, il le fait aussi à la souris. Les touches sont dans les infobulles.

| Contexte | Touche | Effet |
|---|---|---|
| page Direction | `j`/`k`, `Entrée` | proposition suivante / précédente ; lancer la proposition sélectionnée |
| page Direction | `1`–`9`, puis `0`–`9` | ouvre la fiche de la table N (deux chiffres si > 9 tables, délai 400 ms) |
| fiche ouverte | `←`/`→`, `1`/`2` | vainqueur gauche / droite ; chiffres dans le champ score ; `Entrée` valide ; `Échap` ferme |
| partout | `Échap` | ferme la fiche, ou revient à la page Direction |
| partout | `Ctrl+Maj+D` | vue tournoi ↔ plateau |
| ligne de commande | `direct [nom]`, `result <table> <vainqueur> [a-b]`, `start <a> <b> [pts] [t<n>]`, `withdraw <nom> [after]`, `table <n> <m>`, `propose` | tout geste a sa commande ; autocomplétion des noms |

Pas de touche-mode, pas de lettre à retenir hors `j`/`k` qui sont l'idiome de l'application.

## 4. Les budgets

M compté une fois par flux, hors comparaison. Un clic ordinaire = 1,3 s ; un gros bouton
proche = 0,66 s.

### 4.1 Les gestes courants (Marc, Léa)

| Flux | Gestes souris | Clics | Temps | Budget |
|---|---|---|---|---|
| F6 confirmer une proposition | clic [Lancer] | 1 | 1,3 s | 1 clic |
| F6 tout lancer (n propositions) | [Tout lancer], [Confirmer] | 2 | 2,6 s | 2 clics quel que soit n |
| F7 résultat, sans score | clic case table, clic nom | 2 | 1,3 + 0,66 = 2,0 s | 2 clics |
| F7 résultat, avec score | clic case, clic champ, `7`, `Tab`, `4`, clic nom | 3 + 3 K | 4,2 s | ≤ 4,5 s |
| F7 résultat au clavier | `3`, `→`, `Entrée` | 3 K | 0,84 s | référence |
| F7 résultat par commande | `:` `result 3 hugo` `Entrée` | ~14 K | 3,9 s | plus lent que la souris : la commande sert aux scripts et à l'habitude, pas au budget |
| F7 forfait | clic case, ⋯, « forfait de Hugo » | 3 | 3,9 s | ≤ 4 clics |
| F10 apparier à la main | [+ Apparier], nom A (autocomplété, 3 K + clic), nom B, [Lancer] | 4 + 6 K | 6,9 s | ≤ 7 s |
| F11 changer de table | clic case, ⋯, « changer de table », `9`, `Entrée` | 3 + 2 K | 4,5 s | ≤ 5 s |
| F12 retirer un joueur | Joueurs, ligne (filtre 3 K), ⋯, « retirer maintenant », [Confirmer] | 4 + 3 K | 6,0 s | ≤ 6 s |
| F13 retardataire | Joueurs, [+ Inscrire], nom (5 K), `Entrée` | 2 + 6 K | 4,3 s | ≤ 5 s ; la vue dit où il entre |

**Corriger une erreur de saisie** (F28, exigence utilisateur) : l'erreur se voit tout de
suite et se défait sur place.

| Cas | Gestes | Clics | Budget |
|---|---|---|---|
| mauvais vainqueur, vu aussitôt | ligne « dernier résultat : Gilles bat Hugo · [Corriger] » sous la file → fiche rouverte, clic sur l'autre nom | 2 | 2 clics |
| idem au clavier | `Ctrl+Z` → fiche rouverte sur le dernier événement, `←`, `Entrée` | 3 K | < 1 s |
| score faux, vu aussitôt | [Corriger], champ, chiffres, clic nom | 3 + K | ≤ 4 s |
| erreur vue plus tard | Historique ou Arbres, match, ⋯, « corriger » (F8) | 4–5 | ≤ 5 |
| nom mal tapé à l'inscription | Joueurs, ligne, ⋯, « corriger », champ, `Entrée` | 3 + K | ≤ 4 ; le slug ne change pas, les Slots suivent |
| joueur inscrit par erreur | Joueurs, ligne, ⋯, « retirer » | 3 | en préparation : disparaît ; en cours : retrait tracé |
| match lancé par erreur | case, ⋯, « annuler ce match », [Confirmer] (F9) | 4 | ≤ 4 |

`Ctrl+Z` ne supprime jamais rien : il ouvre la correction (ou l'annulation) du dernier
événement, qui devient un événement de plus (D6). La case d'un match fini reste visible,
grisée avec le résultat, jusqu'à ce qu'un nouveau match prenne la table : on la rouvre d'un
clic pour corriger.

**Léa, retour après son match** (F18) : la page Direction est ouverte telle quelle ; la bande
d'horloge dit « depuis 21:12 : 4 résultats, 3 propositions » ; elle confirme les trois en 2
clics et saisit ce qu'on lui apporte. Budget : **de « je reviens » à « la salle tourne » ≤ 10 s
et zéro lecture ailleurs que la page Direction.**

### 4.2 Le coût d'entrée (Yanis, Marc)

| Étape | Gestes | Clics |
|---|---|---|
| F1 créer un tournoi dirigé | panneau Tournoi [+ Nouveau], nom (K), [Diriger ce tournoi] | 2 + nom |
| F1 choisir le format | Réglages : une carte par configuration nommée (« Suisse 2 vies puis tableau — recommandé pour 16 à 32 joueurs »), clic ; tables : un champ | 2 |
| F2 inscrire 20 joueurs | Joueurs : nom, `Entrée`, nom, `Entrée`… (autocomplétion sur les Players) | 20 × (≈ 6 K + 1 K) ≈ 40 s |
| F3 reprendre les inscrits de la dernière fois | [Annuaire ▾] → « reprendre ceux de <tournoi> » → cocher / [Tous] → [Inscrire] | 4 |
| F5 lancer | Direction : la première proposition est là ; [Tout lancer], [Confirmer] | 2 |

**Budget : de « Nouveau » à « première ronde lancée » avec 20 joueurs retapés ≤ 90 s, avec
l'annuaire ≤ 30 s ; ≤ 8 clics hors saisie des noms ; aucune fenêtre modale en série (pas
d'assistant).** Chaque champ de Réglages a une valeur par défaut qui tient pour un tournoi de
club, et une infobulle d'une phrase. Une Direction d'exemple dans `demo.db` (tournoi fictif,
32 joueurs, en cours, phase suisse au bord de la bascule) montre à Yanis une salle qui tourne
avant la sienne.

### 4.3 La rigueur (Sophie)

| Flux | Gestes | Clics | Budget |
|---|---|---|---|
| F8 corriger un résultat | Historique ou Arbres : clic sur le match, ⋯, « corriger », fiche (comme F7), [Valider] | 4–5 | ≤ 5 ; la réparation proposée apparaît dans la file, comme une proposition |
| F9 annuler un match lancé par erreur | clic case, ⋯, « annuler ce match », [Confirmer] | 4 | ≤ 4 |
| F15 changer la bascule en cours | Réglages, champ, valeur, [Appliquer], [Confirmer] (liste de ce qui change) | 3 + K | ≤ 4 |
| F19 transcrire depuis un Slot | Arbres ou Joueurs : match, ⋯, « transcrire » → l'onglet Transcription s'ouvre, en-tête rempli | 3 | ≤ 3 ; retour par `Ctrl+Maj+D` ou l'onglet |
| F20 rattacher n matchs importés | Réglages › Sorties… non : Direction › bandeau « 12 matchs importés non rattachés » → liste, un [Rattacher] par ligne | 1 + n | 1 + n ; jamais 0 (D3) |
| F21 clore | proposition « clore », [Lancer], Classement, [CSV] | 3 | ≤ 4 |
| F24 imprimer les appariements | ⋯ de la file, « feuille de la ronde », dialogue d'impression du système | 2 + système | ≤ 3 |
| F25 historique d'un joueur | Historique, filtre (K) | 1 + K | ≤ 2 |

### 4.4 Le mur (Marc)

F23 : Réglages › Sorties : dossier (dialogue système, une fois) ; la page s'écrit à chaque
événement ; « ouvrir dans le navigateur » = 1 clic. Après le premier réglage : **0 geste**.

## 5. Le retour visuel

- Un événement confirmé se voit **là où il agit** : la proposition quitte la file, la case de
  table se remplit, la file d'attente se raccourcit — en moins de 100 ms, sans toast.
- Un avertissement est une pastille sur l'objet (case, ligne, proposition) et un compteur
  dans la bande ; jamais une fenêtre.
- Un match lent : la case passe en couleur d'alerte et son temps s'affiche en gras. Pas de
  son, pas de clignotement.
- Une micro-ronde tombée pendant que le TD est ailleurs : badge sur l'onglet Tournoi,
  ligne dans la barre d'état ; rien ne se lance seul.
- La fiche de résultat s'ouvre *sur* la case, pas au centre : le regard ne quitte pas la
  grille.
- Les couleurs sont celles du thème (`style.css`), un seul style de bouton principal par
  page, la taille de police de l'application (ADR-0008) — les noms de joueurs dans la
  fiche sont l'exception nommée : 1,5 × la taille de base, parce qu'on clique dessus en se
  penchant.

## 6. La vérification

- **Specs Playwright** (lot 3) : F6, F7 (avec et sans score), F28, F10, F12, F13 comptent les
  clics et échouent au-dessus du budget ; F1 + F3 + F5 mesurent le coût d'entrée.
- **Prototype jetable avant le lot 1** (`/prototype`) : la page Direction avec la file, la
  grille et la fiche de résultat, sur des données simulées de Nicomaque (`sim.Run`), pour
  valider le dessin de la case de table et de la fiche à la main. Ce qui en sort est le
  dessin, pas le code.
- **Le premier tournoi réel** est la recette finale : le TD note chaque fois qu'il a voulu
  faire autre chose que ce que la page proposait (RESTE_A_FAIRE.md de Nicomaque, §1).

## 7. Hypothèses prises seul, à confirmer

1. **Cliquer un nom valide** quand aucun score n'est saisi (F7 en 2 clics) ; avec un score,
   le clic du nom valide aussi. Aucun bouton [Valider] n'est nécessaire à la souris, il est là
   pour le clavier.
2. **« Ignorer pour l'instant »** retire une proposition de l'affichage jusqu'au prochain
   `Propose` ; ce n'est pas un événement.
3. **[Tout lancer] demande une confirmation** avec la liste ; [Lancer] unitaire n'en demande
   pas.
4. **La fiche s'ouvre en surimpression sur la case**, pas en fenêtre modale.
5. **Le bandeau des matchs importés non rattachés** apparaît sur la page Direction après un
   import qui a touché ce Tournament, et se ferme d'un clic.
6. **Une Direction d'exemple** entre dans `demo.db` (32 joueurs fictifs, suisse au bord de la
   bascule).
7. **Les noms dans la fiche de résultat** sont l'exception typographique nommée à l'ADR-0008.
8. **Deux chiffres pour une table > 9** avec un délai de 400 ms, comme une numérotation de
   canal ; les chiffres suivent `event.code` (convention du clavier).
