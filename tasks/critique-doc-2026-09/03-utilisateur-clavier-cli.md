# Persona 3 — Utilisateur clavier et CLI qui scripte ses imports

## Qui je suis (3 lignes)

Théo, 29 ans, développeur, je joue sur Backgammon Galaxy et Heroes. Je vis dans vim et tmux, je touche
la souris le moins possible, et un outil ne m'intéresse vraiment que le jour où je peux le mettre dans
un cron et dans un pipe. Je veux trois choses : un import nocturne de mon dossier de matchs, un export
JSON de mes pires erreurs, et taper *la même* recherche dans l'application et dans le terminal.

## Parcours suivi (liste ordonnée des pages/sections lues)

1. Interface en ligne de commande (CLI) › Introduction, Syntaxe générale, Commandes disponibles
2. CLI › `import` (dont « Import par lot ») puis › Codes de retour
3. CLI › `search`, en particulier « Le langage de requête »
4. CLI › `list` (type `stats`, option `--top-blunders`), `export`, `verify`, `completion`
5. CLI › Exemples de flux de travail (« Import d'un répertoire de tournoi », « Analyse des erreurs »)
6. Liste des commandes › Édition et recherche, puis › Filtres de recherche
7. Raccourcis clavier › Position (CTRL-SHIFT-F)
8. Annexe: Utilisation avancée des filtres › Exemples
9. Guide utilisateur › « Importer un dossier de matchs », « Rechercher des positions selon des critères »
10. Manuel › La ligne de commande, Panneau Recherche, Panneau Stats › Top blunders
11. Mode headless (serveur) › famille `search` ; Contrat d'API › `search`, `exports`

## Ce que j'ai trouvé en cinq minutes

Beaucoup, et de bonne qualité. La CLI est réelle, pas un alibi : `import --type batch --dir`,
`--format json` sur presque tout, `completion bash|zsh|fish`, `healthcheck` pensé pour un `HEALTHCHECK`
Docker et une unité systemd, `analyze` interruptible par Ctrl-C sans rien perdre. La page dit d'emblée
que « La CLI partage exactement le même format de base de données que l'interface graphique », et la
section `--query` promet mieux encore : « --query donne accès au langage de requête de l'application ».
J'ai trouvé mes trois scénarios *nommés* dans « Exemples de flux de travail ». Le squelette est là.

Ce qui manque, c'est tout ce qui sépare une commande d'un script : ce que la commande imprime, ce
qu'elle renvoie, et comment traduire un jeton de l'application en drapeau de la CLI.

## Mes trois scénarios (commande composée ou blocage, pour chacun)

### 1. Import nocturne d'un dossier — commande composée, garde impossible

```sh
blunderdb import --db ~/bg/galaxy.db --type batch --dir ~/bg/inbox/ --format json --fail-on-error
```

La commande, je la compose sans hésiter : elle est dans la doc, mot pour mot. Le blocage est sur le
code de retour, c'est-à-dire sur la seule chose qui compte dans un cron. La page dit d'abord
« N'importer aucun élément est toujours une erreur, que --fail-on-error soit passé ou non », puis, deux
paragraphes plus loin : « relancer un import par lot sur un répertoire déjà importé, sans nouveau
fichier, reste un succès ». Or c'est exactement la nuit ordinaire de mon cron : rien de neuf, que des
doublons. Selon la phrase que je crois, je reçois un mail d'alerte toutes les nuits ou jamais. Je ne
peux pas trancher depuis la doc, et rien ne décrit non plus le JSON de `--format json` qui me
permettrait de trancher moi-même. Second point : aucune option ne restreint le lot aux fichiers
nouveaux (pas de `--since`, pas de déplacement des fichiers traités), donc je relis tout mon dossier
chaque nuit sans savoir ce que ça coûte.

### 2. Export JSON des pires erreurs — deux chemins, aucun documenté jusqu'au bout

```sh
blunderdb search --db ~/bg/galaxy.db --error-min 0.1 --format json > pires.json
blunderdb list   --db ~/bg/galaxy.db --type stats --top-blunders 50 --format json > top.json
```

Trois blocages. (a) `search --format json` : aucun champ n'est nommé, aucun exemple de sortie. Je ne
sais pas si j'obtiens un XGID, un identifiant, l'équité perdue. Idem pour `--format xgid`, dont la
page ne dit que le nom. (b) L'unité : la doc écrit `--error-min 0.1` (« Rechercher les positions avec
erreur >= 0.1 ») et à côté `--move-error-min / --move-error-max — Erreur du coup joué (millipoints) ».
0,1 ou 100 ? Un facteur 1000 sépare le script juste du script vide. (c) `search` ne trie pas : « pires »
suppose un ordre, et seul `list --type stats --top-blunders` en produit un — sous une autre commande,
avec une autre sortie, jamais rapprochée de la première.

### 3. La même recherche en CLI et dans l'app — impasse assumée par la doc elle-même

Dans l'application je tape `s cube p>30 E>50`. La page CLI me propose exactement cette chaîne :
`./blunderdb search --db base.db --query 's cube p>30 E>50'`. Puis, huit lignes plus bas, elle me
retire l'exemple qu'elle vient de donner : « cube, score et D se comparent au plateau de recherche,
vide ici — utilisez --dice, --cube et --score1/--score2 ». Je tente donc la traduction :

```sh
blunderdb search --db base.db --decision cube --query 'p>30 E>50'   # refusé
```

Refusé aussi, par la règle du paragraphe précédent : « --query remplace les drapeaux de filtre au lieu
de s'y ajouter : les combiner est refusé ». Il me reste la traduction complète en drapeaux :

```sh
blunderdb search --db base.db --decision cube --pip-min 30 --move-error-min 50 --format json
```

… que j'ai devinée, sans certitude sur `p` → `--pip-min` (retard à la course ?) ni sur `E` →
`--move-error-min`. Aucune table de correspondance jeton ↔ drapeau n'existe. La parité annoncée
(« Une seule grammaire de recherche », dit l'historique) s'arrête à la première recherche réelle.

## Où je me suis égaré

Sur la question « où est la grammaire de recherche ? ». La page CLI renvoie à *Liste des commandes*. La
*Liste des commandes* donne une table de jetons sans un seul exemple complet. L'*Annexe: Utilisation
avancée des filtres* donne les exemples, mais pas la table. Le *Manuel* décrit les mêmes filtres sous
leurs noms d'interface (Marquée, Commentaire, Matchs & Tournois). Quatre pages, aucune ne se présente
comme la référence ; j'ai fait trois allers-retours pour reconstituer `s tn2 E>40`.

Égaré aussi dans le tableau « Commandes disponibles » : j'y ai cherché `repair` et `bearoff`, absents.
Ils ont pourtant chacun leur section complète juste en dessous, et une entrée dans le sommaire latéral.
J'ai cru qu'elles n'existaient pas.

## Ce qui a entamé ma confiance

Le copier-coller cassé. Dans la table des filtres, les jetons textuels sont rendus avec des guillemets
typographiques dépareillés : `t’mot1;mot2;…”`, `m’motif1,motif2,…'`, `pl’nom”`. Collés dans la ligne de
commande, ils ne peuvent pas fonctionner — et la page CLI, elle, écrit `t"blunder"`, `m"13/11"`,
`pl"Alice"` avec des guillemets droits. La page de référence des filtres est la seule à ne pas être
copiable.

Ensuite, les affirmations qui ne survivent pas à la page suivante. « blunderDB détecte automatiquement
les doublons et empêche l'import d'un match déjà présent » (FAQ, guide) ; « l'import détecte le doublon
et n'ajoute rien d'autre que les marques » (manuel). Empêché, ou partiellement rejoué ? Pour un cron
qui réimporte chaque nuit, ce n'est pas un détail de style.

Enfin, `verify` : « le code de sortie reste 0 » y est écrit trois fois, pour les orphelins, l'écart de
schéma et les contraintes. Je comprends l'intention. Mais on me laisse sans le moyen d'en faire une
garde : le JSON qui porte le verdict est décrit par une parenthèse — « statistiques, orphelins, écart
de schéma » — et pas par un exemple.

## Ce qui manque

- La sortie attendue. Cinq commandes la montrent (`analyze`, `vacuum`, `repair`, `healthcheck`,
  `anki forecast`), une quinzaine ne la montrent pas — dont les trois que je scripte : `import`,
  `search`, `list`.
- Le contrat JSON. Le *Contrat d'API* liste 135 routes sans un seul schéma, et renvoie à
  « openapi.yaml à la racine du dépôt » : un fichier du dépôt, pas une page du site.
- Une table jeton ↔ drapeau (`p`/`--pip-min`, `E`/`--move-error-min`, `i`/`--individual`, `fl`/`--flagged`…).
- Une phrase sur l'accès concurrent : la page affirme que tout est « immédiatement visible dans
  l'interface graphique et inversement », sans dire ce que devient mon cron si l'application est ouverte.
- Le format « une position JSON par ligne » attendu par `import --type position` : jamais décrit.
- `--offset` et `--query-help` : cités dans une phrase de prose, absents de la liste d'options de `search`.
- Une distinction claire entre ce que je n'ai pas trouvé (le schéma JSON, la table de correspondance,
  le comportement concurrent — peut-être écrits ailleurs) et ce qui n'existe visiblement pas (un tri
  dans `search`, un `--since` pour l'import par lot).

## Constats

| # | Constat | Page › section | Gravité | Proposition |
|---|---|---|---|---|
| 1 | Deux phrases opposées sur le même cas : « N'importer aucun élément est toujours une erreur, que --fail-on-error soit passé ou non » puis « relancer un import par lot sur un répertoire déjà importé, sans nouveau fichier, reste un succès » | CLI › import | bloquant | Énoncer une table à trois lignes : rien d'importé et rien de reconnu = 1 ; que des doublons = 0 ; échec partiel = 1 si `--fail-on-error` |
| 2 | L'exemple `--query 's cube p>30 E>50'` est invalidé huit lignes plus bas par « cube, score et D se comparent au plateau de recherche, vide ici » | CLI › search › Le langage de requête | bloquant | Remplacer l'exemple par une requête sans jeton de plateau, et montrer juste après l'équivalent en drapeaux |
| 3 | La traduction que la page recommande est interdite par la page : « utilisez --dice, --cube et --score1/--score2 » se heurte à « --query remplace les drapeaux de filtre […] les combiner est refusé » | CLI › search | bloquant | Dire explicitement qu'une recherche mêlant plateau et jetons doit s'écrire *entièrement* en drapeaux, et donner l'exemple complet |
| 4 | Aucun champ, aucun exemple pour `search --format json` ni pour `--format xgid` : « Format de sortie: table, json ou xgid (défaut: table) » est tout ce qui est dit | CLI › search | bloquant | Ajouter un bloc de sortie pour chacun des trois formats, comme le fait déjà `anki forecast` |
| 5 | Unité de l'erreur non écrite : `--error-min 0.1` (« erreur >= 0.1 ») contre `--move-error-min` en millipoints et `E>50` en millipoints | CLI › search ; Liste des commandes › Filtres | bloquant | Écrire l'unité sur chaque drapeau et rappeler la conversion (1 point = 1000 mpt) |
| 6 | Guillemets typographiques dépareillés dans la table des filtres : `t’mot1;mot2;…”`, `pl’nom”` — non copiables ; la page CLI utilise `t"…"` | Liste des commandes › Filtres de recherche | bloquant | Passer ces cellules en littéral de code, guillemets droits, alignés sur les exemples CLI |
| 7 | `repair` et `bearoff` ont une section complète mais manquent au tableau « Commandes disponibles » | CLI › Commandes disponibles | gênant | Compléter le tableau ; ajouter la ligne renvoyant vers `serve`, `migrate`, `call` traités ailleurs |
| 8 | `--offset` et `--query-help` n'apparaissent que dans une phrase de prose, jamais dans la liste d'options | CLI › search | gênant | Les ajouter à « Options principales » avec un exemple pour `--query-help` |
| 9 | La table des filtres ne dit pas comment écrire la valeur d'un lancer : `D` et `D1` n'ont pas de forme, on ne l'infère que de `xD65` | Liste des commandes › Filtres de recherche | gênant | Écrire `D65`, `D1 5` (ou la forme réelle) dans la cellule, comme pour `xD65` |
| 10 | Rien n'indique dans la table que `cube`, `score`, `d` et `x` lisent le plateau ; seules la page CLI et l'avertissement de l'annexe le disent | Liste des commandes › Filtres de recherche | gênant | Une note en tête de table : quatre jetons prennent leur valeur au plateau, pas dans le texte |
| 11 | `s t"Aachen2024"` est légendé « Position du tournoi Aachen2024 » alors que `t` interroge les commentaires et qu'un tournoi s'écrit `tn1` | Annexe filtres › Exemples | gênant | Corriger la légende (« commentaire contenant Aachen2024 ») ou l'exemple (`s tn1`) |
| 12 | La grammaire de recherche est éclatée sur quatre pages, aucune ne se déclare la référence, alors que l'historique annonce « Une seule grammaire de recherche » | Liste des commandes / Annexe filtres / CLI / Manuel | gênant | Nommer la *Liste des commandes* comme référence unique, y rapatrier deux exemples complets, et faire pointer les trois autres pages vers elle |
| 13 | Aucune table de correspondance jeton ↔ drapeau CLI | CLI › search | gênant | Ajouter une colonne « équivalent CLI » à la table des filtres, ou une table de correspondance dans la page CLI |
| 14 | La sortie attendue n'est montrée que pour cinq commandes ; `import`, `search`, `list`, `info`, `collection` n'en ont aucune | CLI (toute la page) | gênant | Un bloc de sortie commenté par commande, sur le modèle de `vacuum` et `analyze` |
| 15 | `verify` sort 0 en toute circonstance (« le code de sortie reste 0 », trois fois) et son JSON n'est décrit que par une parenthèse | CLI › verify | gênant | Montrer le JSON de `verify`, et indiquer quel champ un script doit tester pour faire une garde |
| 16 | Le doublon est décrit deux fois différemment : « empêche l'import d'un match déjà présent » vs « n'ajoute rien d'autre que les marques » | FAQ / Guide vs Manuel › filtre Marquée | gênant | Une seule formulation, reprise à l'identique aux trois endroits, et rappelée dans CLI › import |
| 17 | Rien sur l'accès concurrent, alors que la page promet une visibilité immédiate dans les deux sens | CLI › Introduction | gênant | Une note : ce qui se passe si le cron écrit pendant que l'application est ouverte |
| 18 | Le format attendu par `import --type position` est réduit à « une position JSON par ligne » | CLI › import › Import de positions | gênant | Montrer une ligne d'exemple, et dire si `export --type positions` la produit telle quelle |
| 19 | La forme négative des drapeaux booléens n'apparaît qu'incidemment (`--recursive=false`, `[--analysis=false]`) alors que huit options sont à « défaut: oui » | CLI › import, export | mineur | Écrire la forme `--option=false` une fois, en tête de la page |
| 20 | Le manuel écrit « la commande :stats ou :st » avec un deux-points, seul endroit du site où une commande en porte un | Manuel › Panneau Stats | mineur | Retirer le deux-points, aligner sur `stats, st` |
| 21 | `search --export base.db` (crée une base) et la commande `export` désignent deux gestes différents sous le même mot | CLI › search vs export | mineur | Renommer dans la prose (« dériver une base des résultats ») ou signaler explicitement la différence |
| 22 | La même case à cocher s'appelle « Search in current results » dans le manuel et « Rechercher dans les résultats actuels » dans l'annexe | Manuel › Panneau Recherche vs Annexe filtres | mineur | Un seul libellé, celui que l'application affiche |
| 23 | `bl` (charger les pires erreurs) n'a pas d'équivalent CLI annoncé ; le plus proche, `list --type stats --top-blunders`, n'est jamais rapproché | Liste des commandes › Positions et navigation ; CLI › list | mineur | Croiser les deux : une phrase dans chaque sens |
| 24 | La page CLI écrit `./blunderdb` partout sans dire d'où vient ce nom ; l'explication (lien minuscule, PATH) est sur la page d'installation | CLI › Syntaxe générale | mineur | Une phrase en tête de page renvoyant à la note d'installation |
