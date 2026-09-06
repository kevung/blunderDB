.. _faq:

Foire aux questions
===================


Quelle est l'utilité de blunderDB?
----------------------------------

blunderDB permet de constituer une base de données personnalisée de
positions. Sa force est de ne présupposer aucune classification *a
priori*. L'utilisateur a ainsi la liberté d'interroger les
positions avec une grande flexibilité en combinant à sa guise
différents critères (course, structure, cube, score, pions arriérés,
pions dans la zone, chances de gain/gammon/backgammon, ...).

Une autre utilisation commode de blunderDB est la constitution de catalogues de
positions de référence. Avec la possibilité d'étiqueter des positions,
l'utilisateur peut rassembler l'ensemble de ses positions de référence de
manière structurée à l'aide d'un unique fichier. Je souhaite que blunderDB
facilite le partage de positions entre joueurs.


Qu'est-ce qui a motivé la création de blunderDB?
------------------------------------------------

J'avais l'habitude de stocker dans différents dossiers des positions
intéressantes ou des blunders. Toutefois, je rencontrais des difficultés à
retrouver des positions selon des critères n'ayant pas été prévus initialement
par mon choix de catégories de thématiques. Par exemple, si les positions ont
été triées selon le type de jeu (course, holding game, blitz, backgame, ...),
comment récupérer toutes les positions à un certain score? ou à un niveau de
cube donné? Enfin, certaines vieilles positions avaient tendance à tomber dans
l'oubli. Je voulais un outil qui agrège toutes mes positions et qui ne
présuppose pas *a priori* de catégories thématiques, et ensuite pouvoir poser
des questions à la base de données. Avec cette approche souple, de nouveaux
filtres peuvent être ajoutés sans casser l'organisation des positions. Ce type
de logiciel est tout à fait courant aux échecs, comme ChessBase.

Comment sauvegarder la base de données courante?
------------------------------------------------

La base de données est modifiée immédiatement après exécution des requêtes.
Aucune opération de sauvegarde explicite est nécessaire.

Dois-je créer différentes bases de données pour différentes catégories de positions?
------------------------------------------------------------------------------------

Sauf pour des raisons bien identifiées, il est essentiel de ne pas
répartir les positions dans des bases de données séparées au risque
de ne pas pouvoir les mettre en relation dans des recherches futures.
La philosophie de blunderDB est de ne pas présupposer de catégories de
positions *a priori* et de permettre à l'utilisateur de les interroger
de manière flexible. Lorsque les positions ont été rencontrées dans des conditions
particulières ou pour des raisons spécifiques, il peut être judicieux de les
stocker dans des bases de données distinctes.
On peut par exemple constituer des bases de données de positions distinctes
pour :

* les positions de référence,
* les blunders en tournoi réel,
* les blunders en ligne.

Comment fusionner plusieurs bases de données?
---------------------------------------------

Si vous avez plusieurs bases de données blunderDB que vous souhaitez regrouper,
utilisez le bouton **Fusionner une base de données** (*CTRL-MAJ-I*) :

#. Ouvrez la base principale (celle qui recevra les positions)
#. Cliquez sur **Fusionner une base de données** dans la barre d'outils
#. Sélectionnez la base à fusionner
#. blunderDB fusionne les positions

Lors de la fusion, blunderDB évite les doublons et fusionne intelligemment
les analyses et commentaires. Les positions identiques ne seront pas dupliquées,
mais leurs analyses et commentaires seront combinés.

.. note::
   Il est recommandé de faire une copie de sauvegarde de votre base de données
   principale avant d'importer une autre base de données.


Quels formats de fichiers de match sont supportés?
--------------------------------------------------

blunderDB supporte les formats de match suivants:

* **eXtreme Gammon (XG)**: fichiers *.xg*, avec l'analyse complète des coups,
  décisions de cube, coups joués et support multi-moteurs. Fichiers *.xgp*
  pour l'import de positions individuelles avec analyse.

* **GNUbg**: fichiers *.sgf* (Smart Game Format), avec l'analyse.

* **Jellyfish**: fichiers *.mat* et *.txt*.

* **BGBlitz**: fichiers *.bgf* et positions texte.

L'import peut se faire par fichier unique, par sélection multiple, par dossier
récursif, par collage depuis le presse-papier, ou par glisser-déposer.

blunderDB détecte automatiquement les doublons et empêche l'import d'un match
déjà présent dans la base de données.


Mes matchs joués en ligne sont-ils importables ?
-------------------------------------------------

Oui, par un détour : **Backgammon Studio (Heroes)**, **Backgammon Galaxy** et
**GammonSpace** produisent tous les trois des fichiers que eXtreme Gammon sait
lire — Studio livre même un paquet d'intégration qui se dépose dans le dossier
XG. Ces matchs arrivent donc dans blunderDB par le chemin ``.xg`` existant,
sans qu'un lecteur dédié soit nécessaire.

.. note::
   Ce que ce détour transporte exactement — analyses, chance du lancer, marques,
   commentaires — **n'a pas été mesuré** : il y faudrait un match exporté depuis
   chacune des trois plateformes. Si vous en avez un et que quelque chose se
   perd à l'import, ouvrez une issue avec le fichier : c'est la mesure qui
   décidera s'il faut un lecteur propre à ces plateformes, pas une supposition.

Les formats texte plus anciens que GNU Backgammon sait déjà lire — ``.sgg``
(GridGammon), ``.tmg``, ``.gam``, le ``.txt`` de Snowie — sont dans le même cas :
passer par GNU Backgammon les rend importables aujourd'hui.


Puis-je synchroniser ma base entre plusieurs machines ?
--------------------------------------------------------

Une base blunderDB est **un seul fichier**. Cela rend la sauvegarde et la copie
triviales, et cela décide de la réponse :

* **Dropbox, Syncthing, iCloud, OneDrive** : cela fonctionne, à une condition —
  ne pas ouvrir la base **des deux côtés en même temps**. Ces outils
  synchronisent des fichiers, pas des écritures concurrentes : deux instances
  qui écrivent en parallèle produisent un conflit que le service résout en
  gardant une version et en renommant l'autre. blunderDB pose un verrou
  d'écriture sur le fichier ouvert, ce qui protège une même machine, mais aucun
  verrou ne traverse un service de synchronisation.

* **Plusieurs postes en même temps** : c'est ce à quoi le mode ``serve`` répond
  (voir :doc:`mode_headless`). Un démon détient la base, les postes s'y
  connectent, et il n'y a plus qu'un seul écrivain.

* **Deux bases qui ont divergé** : elles se **fusionnent** plutôt qu'elles ne se
  synchronisent — voir « Comment fusionner plusieurs bases de données ? »
  ci-dessus. La déduplication par empreinte Zobrist fait le travail : les
  positions communes ne sont pas dupliquées et leurs analyses et commentaires
  sont combinés. C'est une fusion, pas une réconciliation : rien n'est supprimé
  d'un côté parce que l'autre l'a supprimé.


Ai-je besoin d'eXtreme Gammon pour utiliser blunderDB?
-------------------------------------------------------

Non. blunderDB lit également les fichiers de GNUbg, BGBlitz et Jellyfish, et
son évaluateur embarqué (gammonNet) analyse n'importe quelle position sans
dépendre d'un logiciel tiers — voir « Que vaut l'évaluateur intégré ? »
ci-dessous. Un import XG apporte cependant l'analyse la plus complète
(coups, décisions de videau, marques, chance du lancer) : c'est le format le
plus richement exploité par les statistiques.


Qu'est-ce qu'une collection?
-----------------------------

Une collection est un regroupement personnalisé de positions. Contrairement
à une recherche par filtres qui est dynamique, une collection est un ensemble
fixe de positions choisies manuellement par l'utilisateur. Les collections
permettent par exemple de regrouper des positions de référence pour une
thématique particulière.


Qu'est-ce que l'EPC?
---------------------

L'EPC (Effective Pip Count) est une mesure plus précise que le simple pip count
pour évaluer les positions de bearoff. Le panneau Eval de blunderDB utilise
la base de données de bearoff à 6 points de GNUbg et calcule en temps réel
l'EPC, le nombre moyen de lancers, l'écart type, le pip count et le wastage.

Sur les positions de bearoff pur, le panneau affiche aussi la probabilité de
gain du joueur au trait et, lorsque la position est couverte par une base
two-sided (base intégrée jusqu'à 6 pions par joueur, base téléchargeable
jusqu'à 11), le verdict de videau money exact. Hors de ce domaine, la
probabilité est estimée avec sa marge d'erreur et le verdict n'est
volontairement pas affiché. Voir la section « Méthodologie et hypothèses du
panneau Eval » du manuel pour le détail des hypothèses.


Que vaut l'évaluateur intégré (gammonNet) ?
---------------------------------------------

gammonNet est un réseau de neurones entraîné par un tiers (voir Crédits),
porté en Go et compilé dans blunderDB : aucun logiciel externe, aucune
connexion réseau. Il joue le rôle d'XG ou de GNUbg pour les positions non
importées — recherche à 0 ou 2 lancers d'avance, décision de videau selon
Janowski et la table d'équité de match de blunderDB, honorant le score du
match. Ce n'est ni le seul ni forcément le meilleur moteur du marché : c'est
celui qui fonctionne hors ligne, sans compte ni abonnement, sur la position
que vous regardez. Rien n'empêche par ailleurs d'importer les analyses d'XG
ou de GNUbg quand elles existent — les deux sources cohabitent, une colonne
indique l'origine de chaque analyse. Voir la section « Panneau Eval » et
« Méthodologie et hypothèses du panneau Eval » du manuel.

**Ce que cela vaut, en chiffres.** Sur les positions de bearoff que la table
exacte intégrée couvre — le seul endroit où un oracle existe — gammonNet à
2 plis rend le **même verdict de videau dans 93,4 % des cas**, et le désaccord
se concentre exactement au point de prise (61 % d'accord à moins de 1 % du
point de prise, 94 % au-delà de 10 %) : c'est là que deux méthodes légitimes
divergent le plus sur une décision serrée, pas une erreur diffuse. Le détail
et la méthode sont dans « Méthodologie et hypothèses du panneau Eval ».

Pour la même question posée de **votre** bibliothèque plutôt que d'un corpus
de référence, ``blunderdb analyze --compare`` compare le moteur embarqué aux
analyses venues de vos fichiers XG ou GNUbg — accord sur la meilleure réponse,
coût des désaccords, ventilation par phase de partie — **sans rien écrire**.


Quelle est la différence entre le PR et le Snowie Error Rate?
----------------------------------------------------------------

Le PR (Performance Rating) est la moyenne des erreurs d'équité (normalisée,
en millièmes de point) sur l'ensemble des décisions comptées ; plus il est
bas, mieux on joue. Le Snowie Error Rate rapporte cette même moyenne au
nombre de *coups* plutôt qu'au nombre de *décisions* — un match plus long
n'aggrave donc pas mécaniquement le SER. blunderDB affiche les deux dans le
panneau Stats, alignés sur les conventions d'eXtreme Gammon et de GNUbg (voir
:ref:`stats_parity` pour le détail des règles de comptage).


blunderDB dispose-t-il d'une interface en ligne de commande?
------------------------------------------------------------

Oui, blunderDB dispose d'une interface en ligne de commande (CLI) permettant
d'effectuer sans interface graphique des opérations telles que la création de
bases de données, l'import de matchs, l'export, la recherche de positions,
l'affichage de statistiques, etc. Consulter la documentation CLI pour plus
de détails.


blunderDB propose-t-il un mode serveur?
-----------------------------------------

Oui, un mode « headless » facultatif : le même binaire, lancé avec
``serve``, expose le moteur de blunderDB en HTTP + JSON derrière un
reverse-proxy authentifiant (blunderDB lui-même ne fait aucune
authentification). Il peut s'appuyer sur SQLite ou sur PostgreSQL en
multi-tenant, et sert par exemple à consulter ou importer des matchs depuis
un navigateur, à intégrer blunderDB à un autre outil, ou à mutualiser une
base entre plusieurs joueurs. L'usage normal reste l'application de bureau ;
voir :ref:`headless` pour le détail (y compris l'image Docker prête à
l'emploi) et le tutoriel « Déployer le mode serveur derrière un proxy » du
guide utilisateur.


Puis-je modifier, copier, partager blunderDB?
---------------------------------------------

Oui, tout à fait (et c'est même encouragé!). blunderDB est sous licence MIT.


Où sont stockées mes données?
-------------------------------

Sur votre disque, dans le fichier ``.db`` que vous avez choisi en créant la
base : aucun compte, aucun serveur, aucune synchronisation par défaut.
L'application de bureau ouvre ce fichier directement ; seul le mode serveur
facultatif (voir ci-dessous) fait tourner blunderDB à distance, et c'est
alors vous qui hébergez ce serveur.


Puis-je partager une base avec un autre joueur?
-------------------------------------------------

Oui : une base blunderDB est un simple fichier, il suffit de le copier ou de
l'envoyer. Pour diffuser une base à un tiers, deux mécanismes facultatifs,
choisis au moment de l'export, en font une diffusion *contrôlée* plutôt
qu'une simple copie : un **filigrane d'origine** signe le fichier avec votre
identité d'émetteur (infalsifiable, lisible dans le panneau Métadonnées ou
via ``blunderdb info``, sans jamais rien consigner côté destinataire), et une
**protection par mot de passe** produit un fichier ``.dbx`` chiffré. Voir
:ref:`diffusion_controlee` dans le manuel.


Quel format de données utilise blunderDB?
-----------------------------------------

La base de données est un simple fichier Sqlite. En l'absence de
blunderDB, elle peut ainsi s'ouvrir avec tout éditeur de fichier sqlite.


Quels ont été les principes de conception de blunderDB?
---------------------------------------------------------

Je voulais une interface accessible, mais pensée pour un usage avancé et
soutenu, où l'on enchaîne les positions et les recherches. La ligne de
commande, ouverte d'un appui sur la barre d'*ESPACE*, et les raccourcis clavier
servent cet usage, sans être un passage obligé.

Je souhaitais par ailleurs blunderDB léger, autonome, sans installation et
disponible pour différentes plateformes,
d'où mon choix du langage Go et de la bibliothèque Svelte. Pour la
sérialisation de la base de données, le format de fichiers doit être
multi-plateforme et adapté pour contenir une base de données. Le format de
fichier sqlite semblait tout indiqué.

Je tenais aussi à ce qu'une base reste un simple fichier, que l'on peut copier,
sauvegarder ou envoyer à un autre joueur.

Enfin, blunderDB ne se limite plus à l'application de bureau : le même binaire
offre une interface en ligne de commande et un mode serveur facultatif, qui
peut s'appuyer sur PostgreSQL pour les déploiements multi-utilisateurs. L'usage
normal reste toutefois l'application de bureau. Voir :ref:`cli` et
:ref:`headless`.


Quelle est l'architecture logicielle de blunderDB?
--------------------------------------------------

* Le backend est codé en `Go <https://go.dev/>`_. Il est en charge de
  l'ensemble des opérations sur la base de données SQLite qui stocke les
  positions.

* Le frontend est codé en `Svelte <https://svelte.dev/>`_. Il est en charge du
  rendu de l'interface graphique et du board de Backgammon.

* L'application est encapsulée avec `Wails <https://wails.io/>`_, permettant la
  production d'applications Desktop natives, déclinables sous Windows, Linux et
  macOS.

* La base de données est gérée par `Sqlite <https://www.sqlite.org/>`_.

* Le mode serveur facultatif peut s'appuyer sur `PostgreSQL
  <https://www.postgresql.org/>`_ à la place de Sqlite pour les déploiements
  multi-utilisateurs.

* L'évaluateur embarqué (`gammonNet <https://github.com/kevung/gammonNet>`_,
  MIT) est un réseau de neurones porté en Go et compilé dans blunderDB : il
  évalue n'importe quelle position hors ligne, sans XG ni GNUbg. Voir « Que
  vaut l'évaluateur intégré ? » ci-dessous.

Pour plus d'informations, voir le `dépôt Github de blunderDB <https://github.com/kevung/blunderDB>`_.

Sur quelles plateformes blunderDB fonctionne-t-il?
---------------------------------------------------

blunderDB fonctionne sur Windows, Linux et Mac.

D'où vient l'icône de blunderDB?
--------------------------------

L'icône de blunderDB est l'émoticône "goggling" de la série `SMirC <https://commons.wikimedia.org/wiki/SMirC>`_.
