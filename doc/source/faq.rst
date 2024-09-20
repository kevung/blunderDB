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

Une autre utilisation commode de blunderDB est la constitution de
catalogues de positions de référence. Avec la possibilité de créer des
bibliothèques, l'utilisateur peut disposer à l'aide d'un unique fichier
l'ensemble de ses positions de référence de manière structurée. Je
souhaite que blunderDB facilite le partage de positions entre joueurs.


Qu'est ce qui a motivé la création de blunderDB?
------------------------------------------------

J'avais l'habitude de stocker dans différents dossiers des positions
intéressantes ou des blunders. Toutefois, je rencontrais des difficultés
à retrouver des positions selon des critères n'ayant pas été prévus
initialement par mon choix de catégories de thématiques. Par exemple, si
les positions ont été triés selon le type de jeu (course, holding game,
blitz, backgame, ...), comment récupérer toutes les positions à un
certain score? ou à un niveau de cube donné? Enfin, certaines vieilles
positions avaient tendance à tomber dans l'oubli. Je voulais un outil
qui aggrège toutes mes positions et qui ne présuppose pas *a priori* de
catégories thématiques, et ensuite pouvoir poser des questions la base
de données. Ce type de logiciel est tout à fait courant aux échecs,
comme ChessBase.

Puis-je modifier, copier, partager blunderDB?
---------------------------------------------

Oui, tout à fait. blunderDB est sous licence MIT.

Comment sauvegarder la base de données courante?
------------------------------------------------

La base de données est modifiée immédiatement la validation de la
requête. Aucune opération de sauvegarde explicite est nécessaire.

Quel format de données utilise blunderDB?
-----------------------------------------

La base de données est un simple fichier Sqlite. En l'absence de
blunderDB, elle peut ainsi s'ouvrir avec tout éditeur de fichier sqlite.

Quelles ont été les principes de conception de blunderDB?
---------------------------------------------------------

Le fonctionnement modal de blunderDB (NORMAL, EDIT, COMMAND) s'inspire
du très puissant éditeur de texte Vim. Je souhaitais blunderDB léger et
autonome d'où mon choix du langage C et de la bibliothèque GUI IUP. Pour
la sérialisation de la base de données, le format de fichiers doit être
multi-plateforme et adapté pour contenir une base de données. Le format
de fichier sqlite semblait tout indiqué.

blunderDB fonctionne-t'il sous Linux?
-------------------------------------

Tout à fait. Il est exécutable avec Wine.

