.. _faq:

Foire aux questions
===================


Quel est l'utilité de blunderDB?
--------------------------------

blunderDB permet de constituer une base de données personalisée de
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


Qu'est ce qui a motivé la création de blunderDB?
------------------------------------------------

J'avais l'habitude de stocker dans différents dossiers des positions
intéressantes ou des blunders. Toutefois, je rencontrais des difficultés à
retrouver des positions selon des critères n'ayant pas été prévus initialement
par mon choix de catégories de thématiques. Par exemple, si les positions ont
été triées selon le type de jeu (course, holding game, blitz, backgame, ...),
comment récupérer toutes les positions à un certain score? ou à un niveau de
cube donné? Enfin, certaines vieilles positions avaient tendance à tomber dans
l'oubli. Je voulais un outil qui aggrège toutes mes positions et qui ne
présuppose pas *a priori* de catégories thématiques, et ensuite pouvoir poser
des questions à la base de données. Avec cette approche souple, de nouveaux
filtres peuvent être ajoutés sans casser l'organisation des positions. Ce type
de logiciel est tout à fait courant aux échecs, comme ChessBase.

Comment sauvegarder la base de données courante?
------------------------------------------------

La base de données est modifiée immédiatement après exécution des requêtes.
Aucune opération de sauvegarde explicite est nécessaire.


Puis-je modifier, copier, partager blunderDB?
---------------------------------------------

Oui, tout à fait (et c'est même encouragé!). blunderDB est sous licence MIT.


Quel format de données utilise blunderDB?
-----------------------------------------

La base de données est un simple fichier Sqlite. En l'absence de
blunderDB, elle peut ainsi s'ouvrir avec tout éditeur de fichier sqlite.


Quelles ont été les principes de conception de blunderDB?
---------------------------------------------------------

Le fonctionnement modal de blunderDB (NORMAL, EDIT, COMMAND) s'inspire du très
puissant éditeur de texte `Vim
<https://en.wikipedia.org/wiki/Vim_(text_editor)>`_. Je souhaitais blunderDB
léger, autonome, sans installation et disponible pour différentes plateformes,
d'où mon choix du langage Go et de la bibliothèque Svelte. Pour la
sérialisation de la base de données, le format de fichiers doit être
multi-plateforme et adapté pour contenir une base de données. Le format de
fichier sqlite semblait tout indiqué.


Quel est l'architecture logicielle de blunderDB?
------------------------------------------------

* Le backend est codé en `Go <https://go.dev/>`_. Il est en charge de
  l'ensemble des opérations sur la base de données Sqlite qui stocke les
  positions.

* Le frontend est codé en `Svelte <https://svelte.dev/>`_. Il est en charge du
  rendu de l'interface graphique et du board de Backgammon.

* L'application est encapsulée avec `Wails <https://wails.io/>`_, permettant la
  production d'applications Desktop natives, déclinables sous Windows et Linux.

* La base de données est gérée par `Sqlite <https://www.sqlite.org/>`_.

Pour plus d'informations, voir le `dépôt Github de blunderDB <https://github.com/kevung/blunderDB>`_.

Sur quelles plateformes blunderDB fonctionne-t'il?
--------------------------------------------------

blunderDB fonctionne sur Windows, Linux et Mac.


