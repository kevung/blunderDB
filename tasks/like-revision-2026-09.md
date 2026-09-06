# Révision de `like` et des commandes récentes — fiche de travail (2026-09-07)

Issue de l'entretien de conception du 2026-09-07 (skill `grilling`). Les décisions
sont dans l'[ADR-0043](../docs/adr/0043-a-neighbour-is-the-same-problem-nearby-and-like-is-a-ranking-token-of-the-search-grammar.md)
et, pour le quiz, dans la règle 6 de l'ADR-0040 ; le glossaire porte
*Neighbouring Position*. Cette fiche ne redit pas le pourquoi : elle liste ce
qu'il y a à faire, dans l'ordre où chaque étape rend la suivante vérifiable.

`ask` (#283) est volontairement hors de cette fiche : traité ailleurs.

## Ce que la mesure a montré

Sur la base de démo (757 positions, 3 matchs), les dix voisins de toute position
sont : le coup de pions jumeau de la décision de videau (distance 0), puis les
plis d'avant et d'après du même match (8 à 16 pions-pas). Aucune autre partie ne
descend en dessous. Le prototype « un joueur juge » demandé par la fiche J.3 n'a
pas eu lieu.

## Étapes

### 1. La classe d'équivalence, côté stockage

- `sqlshared.Similar` prend, en plus de la cible et de la limite : le type de
  décision exigé (ou aucun avec `*`), le régime exigé pour une décision de videau
  (money / match), l'identifiant du match à exclure (0 = aucun), le plafond de
  distance (0 = aucun).
- Le type de décision d'une ligne est déjà une colonne ; le régime se lit sur le
  score ; le match se lit par `move.game_id → game.match_id` — vérifier qu'une
  jointure ne fait pas exploser le balayage (un `IN (SELECT position_id …)`
  suffit).
- Contrat `storagetest` : (a) le jumeau à distance 0 n'apparaît plus ; (b) les
  plis du même match n'apparaissent plus ; (c) un plafond que rien ne passe rend
  une liste vide ; (d) `*` rend les deux types ; (e) une cible dessinée (ID 0,
  sans match) n'exclut rien.

### 2. Le jeton `like[id][<n][*]`

- `searchquery` : le jeton, et un drapeau « requête classée » sur le résultat de
  `Parse` ; `like` seul prend la position du plateau comme cible (enregistrée ou
  dessinée).
- Les deux backends trient par distance croissante quand la requête est classée,
  et joignent les autres jetons comme filtres ordinaires.
- `commandVocabulary.js` + `commandVocabulary.sync.test.js` ; `--query-help`.
- Supprimer : la commande `like`, `similarService.js`, `--like`, le corps
  particulier de `/v1/positions.similar` (la route prend une requête ordinaire),
  la ligne de `parity_test.go`.
- `ss like` et les collections vivantes doivent marcher sans une ligne de plus :
  un test pour chacun.

### 3. Les préférences

- `Config` : `like_limit` (défaut à fixer entre 20 et 30) et `like_max_distance`
  (0 = aucun plafond). Onglet *Interface* du panneau de configuration, deux champs.

### 4. La distance visible, et les entrées

- Ligne d'explication (#298) : « à {d} pions-pas de la position n° {id} » sur
  chaque voisine d'une liste classée ; le message de la barre d'état reste.
- Menu contextuel du plateau : « Positions proches » ; un raccourci dans
  `raccourcis.rst` (choisir une lettre libre par `letter()`, convention
  `event.key`).

### 5. La vérification que J.3 devait à l'origine

- Une trentaine de cibles prises dans une vraie bibliothèque, les cinq premiers
  voisins de chacune, un jugement « même problème : oui / non » par l'auteur de
  l'entretien. Le résultat est versé dans `docs/recherche/` à côté de P7. Si
  moins des deux tiers des premiers voisins sont « oui », les règles de classe
  sont à revoir avant la doc.

### 6. Documentation

- `cmd_mode.rst` (retirer la ligne `like`, ajouter le jeton), `manuel.rst`
  (le paragraphe « La commande like » devient « Le jeton like »),
  `raccourcis.rst`, `cli.rst` (`--like` disparaît), huit `.po` par fichier,
  `make help`, `scripts/doc-i18n-check.sh`.

## Hors `like`

- **Quiz (ADR-0040, règle 6)** : `training_item` porte l'identifiant de la
  position pour l'exercice *Decision* ; « revoir les ratées » remplace la liste
  parcourue. À faire dans le chantier de l'onglet Entraînement, pas avant.
- **`cm`** : mettre en évidence la case du score courant quand la position est
  un match de 5, 7 ou 9 points. Une classe CSS et une ligne de manuel.
- **Glossaire** : le doublon « Tag » (définition par sous-chaîne) est retiré ;
  reste la définition délimitée, celle que #295 applique.
