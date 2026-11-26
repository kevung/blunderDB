.. _cli_document:

Importer les positions d'un document
====================================

À quoi cela sert
----------------

Un ouvrage que l'on possède — livre de problèmes, recueil, article — montre des
positions en diagrammes, dans un ordre qui est celui de l'étude. La commande
``document`` les range dans une collection de la base, dans cet ordre, pour les
réétudier avec le reste de la bibliothèque. L'interface a le même chemin dans un
assistant (:ref:`assistant_import_document`) ; la ligne de commande est décrite ici.

Le travail passe par un fichier intermédiaire, le **relevé** : un fichier JSON
qui liste les positions lues dans le document. On le relit, on le corrige si
besoin, puis on l'importe. Lire un document n'écrit rien en base ; seul
l'import d'un relevé écrit.

Ce qui est lu, ce qui est refusé
--------------------------------

``document analyze`` lit deux sortes de documents :

* un PDF dont les diagrammes sont dessinés avec la police de plateau
  d'eXtreme Gammon (les PDF imprimés par XG) : chaque diagramme y est du texte,
  que la commande décode exactement, glyphe par glyphe ;
* un livre scanné : un PDF dont les pages sont faites d'images, avec ou sans
  couche texte, ou ses pages en images (un dossier de fichiers PNG ou JPEG, une
  page par fichier, lus dans l'ordre de leurs noms, ou une image seule). Les
  diagrammes y sont lus par leur structure (voir :ref:`cli_document_images`).

Un PDF dont aucune page n'est faite d'images ni dessinée avec cette police (un
livre dessiné en vecteurs) est refusé ; ses pages se donnent alors en images :

.. code-block:: text

   Error: document: no diagram drawn with eXtreme Gammon's board font, and no page made of images (a page drawn with vectors is not rendered); give its pages as images: a folder of PNG or JPEG files, one a page; structure: no page of the document could be read: page 1: pdf: the page draws no image (a page drawn with vectors is not rendered)

Un fichier qui n'est ni un PDF ni une image, un PDF chiffré ou plus gros que
512 Mio, une image de plus de 64 Mio ou de plus de 100 millions de pixels est
refusé aussi, avec un message qui le dit.

Le parcours en trois commandes
------------------------------

.. code-block:: bash

   ./blunderdb document analyze livre.pdf -o releve.json
   ./blunderdb document check releve.json
   ./blunderdb document import --db base.db releve.json

1. ``analyze`` lit le PDF et écrit le relevé (sans ``-o``, il l'écrit sur la
   sortie standard). Il résume ce qu'il a trouvé :

   .. code-block:: text

      Document:   Mon livre
      Entries:    6 (sure 3, to verify 2, uncertain 1)
      Context:    read 5, default 0, to verify 1
      Positions:  5 distinct, 1 repeated
      Relevé:     releve.json

   Le titre et l'auteur viennent des métadonnées du PDF, à défaut le titre est
   le nom du fichier ; ``--title`` et ``--author`` les remplacent. Le chapitre
   de chaque entrée vient des signets du PDF.

2. ``check`` valide le relevé et compte ses entrées, sans lire ni écrire de
   base. Il liste aussi, après le décompte, les entrées à vérifier (rang, page,
   état et champs à comparer au diagramme) ; ``--format json`` donne les mêmes
   comptes et la même liste (``review``) en JSON :

   .. code-block:: text

      Document:   Mon livre
      Entries:    6 (sure 3, to verify 2, uncertain 1; verified 0)
      Context:    read 5, default 0, to verify 1
      Positions:  5 distinct, 1 repeated
      Unverified: rank 4, 5, 6 (import refused without --force)
      Review:
        rank 4, p. 12, Problem 3, to-verify: context not read, to check against the diagram: cube (drawn)

3. ``import`` écrit les positions dans la collection :

   .. code-block:: text

      Created collection "Mon livre" (id 1)
      Entries: 6, distinct positions: 5 (5 new in the library), added to the collection: 5, repeated: 1

   ``--collection`` choisit un autre nom que le titre du document. Si le
   fichier de base n'existe pas, il est créé.

Le relevé
---------

Le relevé est un fichier JSON lisible, que l'on peut ouvrir dans un éditeur de
texte. L'en-tête (``document``) donne le titre, l'auteur, l'édition ou l'adresse
du document, l'empreinte SHA-256 du fichier lu, le profil de ses diagrammes et,
pour un livre scanné, son contexte par défaut (``defaults``, voir
:ref:`cli_document_defaults`). Puis chaque diagramme est une entrée, dans
l'ordre du document :

* ``rank`` — son rang, de 1 sans trou ;
* ``page``, ``chapter`` et ``label`` — sa page, le chapitre des signets et le
  nom que le document lui donne (« Problem 191 ») quand il est imprimé au-dessus ;
* ``position`` — la position, en XGID : pions, videau, trait, dés, score et
  longueur du match en une seule chaîne ;
* ``checkers`` — la certitude des pions : ``sure``, ``to-verify`` ou
  ``uncertain`` ;
* ``complete`` — ``true`` quand les pions ont été lus à quinze par camp sans
  aucun doute, sans qu'un recoupement les confirme (voir
  :ref:`cli_document_complete`) ;
* ``context`` — comment le contexte (videau, dés, trait, score) est connu :
  ``read`` (lu et recoupé), ``default`` (le contexte par défaut du document),
  ``to-verify`` (ni lu ni sous le défaut : un videau ou des dés dessinés et non
  lus, un score introuvable) ou ``uncertain`` (contredit) ;
* ``doubts`` — les doutes nommés de la lecture ;
* ``rect`` — où le diagramme se trouve sur la page, en fraction de sa taille,
  pour le retrouver dans le PDF ;
* ``verified`` — ``true`` quand une personne a comparé la lecture au diagramme.

On relit un relevé en ouvrant le PDF à la page de l'entrée. On y corrige la
position (le XGID), on marque ``"verified": true`` une entrée contrôlée, on
retire une entrée en renumérotant les rangs. Un relevé se vérifie en plusieurs
séances : ``verified`` reste dans le fichier. ``check`` refuse un relevé dont un
champ est inconnu, dont les rangs ont un trou, dont une page recule, dont un
XGID n'est pas une position légale, ou d'un format plus récent que le programme.
Des pions ``sure`` ne portent de doute que sur le contexte ; un contexte ``read``
n'en porte aucun ; une entrée ``default`` porte le contexte par défaut dans son
XGID.

Le relevé est au format de version 2. Un relevé de version 1, qui donnait un
seul état (``state``) par entrée, se lit encore : une entrée ``sure`` y a ses
pions sûrs et son contexte lu, une entrée d'un livre scanné passe sous le
contexte par défaut, non confirmé, et toute autre a son contexte à vérifier.
``analyze`` écrit toujours la version 2.

Les trois états
---------------

L'état d'une entrée découle de ses deux certitudes ; il n'est pas écrit dans le
relevé.

``sure``
   Les pions sont sûrs **et** le contexte est certifié : lu, ou sous le contexte
   par défaut du document une fois celui-ci confirmé. Les pions sont sûrs quand
   les recoupements du diagramme tiennent : chaque camp compte quinze pions
   (sur les points, sur la barre et dans le plateau de sortie), et les pips
   calculés d'après la position égalent les pips imprimés des deux joueurs. Le
   contexte d'un PDF d'XG est lu : le score et la longueur du match sont ceux
   qu'il imprime à côté. Celui d'un livre scanné ne l'est jamais : voir
   :ref:`cli_document_images`.

``to-verify``
   La lecture tient, mais rien ne la confirme : un pip imprimé diffère du pip
   calculé, ou le diagramme montre ce que les impressions de référence n'ont
   jamais montré — pile de plus de 9 pions, pions numérotés sur la barre, plus
   de deux pions blancs ou de trois noirs sur la barre, videau du côté du joueur
   du haut ou au-delà de 8, joueur du haut au trait, pas de marque de trait,
   joueur noir en bas, jeu de Crawford, joueur à un point de la victoire (la
   partie de Crawford et les suivantes ne se distinguent pas), glyphes de la
   police que la table ne lit pas. Ces cas sont lus mais jamais déclarés sûrs.

``uncertain``
   Les pions ou le contexte sont contredits, ou la lecture est incomplète : un glyphe inconnu, un camp qui ne
   compte pas quinze pions et dont les pips ne correspondent pas, un cadre
   abîmé. Un diagramme dont on ne tire aucune position légale garde sa place
   dans le relevé, sans position (``"position": ""``).

Une entrée est **importable** quand elle est ``sure`` ou vérifiée, ou, sur
demande, quand elle est complète (:ref:`cli_document_complete`). Les doutes
qu'elle porte disent quoi contrôler, par exemple :

.. code-block:: text

   pips read 167/167, printed 160/167 (bottom/top)

Le dé n'est pas lu : l'analyse écrit toujours « pas de dés » (``00``).

.. _cli_document_images:

Livres scannés
--------------

Une page en image est lue sans style supposé : la commande y cherche deux
rangées de douze pointes coupées par une barre, et des pions, disques de deux
classes empilés depuis la base d'une pointe. Elle apprend le style du document
sur le document lui-même : la forme du pion sur les premiers diagrammes, puis,
sur tous, les deux classes de pions, la numérotation imprimée et le sens de jeu.
Rien n'est gardé d'un document à l'autre, et aucune image n'est conservée.

Elle lit les diagrammes au trait et les captures de logiciel imprimées, dans
l'ordre de lecture : page de gauche puis de droite d'une double page, colonne,
puis haut en bas. Une page légèrement de biais est redressée, l'ombre de reliure
effacée. Résolution utile : 150 à 300 dpi, celle que le fichier indique ; à
défaut, le grand côté de la page est pris pour onze pouces.

Un PDF scanné est lu directement, sans outil externe : chaque page est composée
de ses images, placées comme le PDF les place, à la résolution de la plus fine
d'entre elles bornée entre 150 et 300 dpi, puis lue et oubliée. Sont décodées
les images JPEG (``DCTDecode``), fax (``CCITTFaxDecode``, groupes 3 en une
dimension et 4) et brutes (``FlateDecode``, gris, couleur ou indexées) ; les
masques de pochoir et les masques doux sont appliqués, si bien qu'une page faite
d'un fond JPEG et de masques fax se compose lisiblement. Ne sont pas gérés : un
masque par couleur clé (l'image est peinte entière), les images JBIG2 et
JPEG 2000 (la page est passée), une page dessinée en vecteurs (passée aussi).
Les pages passées sont nommées sur la sortie d'erreur, avec leur motif ; les
autres sont lues. La couche texte d'une page, visible ou non (celle qu'une OCR
laisse sous l'image), donne ses mots à leur place sur l'image ; un texte que sa
police ne traduit pas en caractères ne donne aucun mot.

Sur un livre scanné, les pions et le contexte sont certifiés à part. Les pions
sont ``sure`` quand les pips imprimés à côté du diagramme égalent les pips
calculés des deux camps, qu'aucun camp n'a plus de quinze pions et qu'aucun
doute ne reste sur les pions. Le contexte est lu champ par champ, et un champ
n'est lu que recoupé :

* le videau, par sa valeur lue dans son carré (chiffres du document) ou dans la
  couche texte — deux sources présentes doivent concorder — et cohérente avec sa
  place : 64 au milieu est centré, 2 à 64 en haut ou en bas appartient au joueur
  de ce côté ;
* les dés, comptés sur l'image et confirmés par le texte (« to play 41 ») ; le
  trait, par la couleur des dés et leur moitié du plateau (la droite pour le
  joueur du bas), qui doivent désigner le même joueur ;
* le score, par le bandeau de la couche texte (``score:``, ``pip:``,
  ``7 point match``), chaque score attribué au joueur dont il porte les pips.

Une entrée dont tout le contexte est lu est ``"context": "read"``. Un videau
introuvable n'est jamais pris pour centré : ce qui n'est pas lu prend le
contexte par défaut du document (``"context": "default"``, avec le doute
``context not read: cube, dice, player on roll, score``). Un videau, des dés ou
un score dessinés ou imprimés et non certifiés, un contexte lu en partie hors du
défaut sur un champ que le document n'imprime pas (voir
:ref:`cli_document_learned`), un videau possédé par le joueur du haut sans dés
dessinés, un videau introuvable dans un document qui le dessine ou un score
introuvable dans un document qui l'imprime (trouvés à côté d'au moins un tiers
de ses diagrammes) laissent le contexte à vérifier
(``"context": "to-verify"``, avec un doute nommé) ; le XGID porte alors ce qui a
été lu. Une entrée d'un livre scanné dont le contexte n'est pas lu n'est donc
**jamais** ``sure`` tant que le contexte par défaut n'est pas confirmé, et celle
dont le contexte est à vérifier ne l'est jamais.
``analyze`` et ``check`` comptent les entrées par contexte, et à part celles dont
seuls les pions sont certifiés :

.. code-block:: text

   Context:    read 0, default 140, to verify 12
   Defaults:   centred cube, no dice, the bottom player on roll, money play (0:0:1:00:0:0:0:0:10), not confirmed: set "confirmed": true once the document's diagrams follow them
   Checkers:   112 confirmed, their context alone to verify (cube, dice, player on roll, score)

Pour une entrée dont le contexte reste à vérifier, on ouvre le livre à la page de
l'entrée, on compare au diagramme le videau (valeur et propriétaire : un videau
dessiné au milieu peut être possédé), les dés, le joueur au trait et le score, on
corrige ces champs du XGID s'il le faut, puis on marque ``"verified": true``.

.. _cli_document_defaults:

Le contexte par défaut
~~~~~~~~~~~~~~~~~~~~~~

Le contexte par défaut d'un livre scanné est celui qu'écrit la lecture : videau
centré, pas de dés, joueur du bas au trait, partie libre. Il figure dans
l'en-tête du relevé :

.. code-block:: json

   "defaults": {
     "context": "0:0:1:00:0:0:0:0:10",
     "description": "centred cube, no dice, the bottom player on roll, money play",
     "confirmed": false
   }

``context`` est la fin du XGID après le plateau ; toute entrée ``default`` la
porte. Quand les diagrammes du document suivent ce contexte — un livre de
problèmes en partie libre, joueur du bas au trait, par exemple — on le
**confirme une fois** en écrivant ``"confirmed": true`` : toutes les entrées
``default`` ont alors leur contexte certifié, et celles dont les pions sont
sûrs deviennent ``sure``. Seules restent à vérifier une à une les exceptions que
la lecture a vues (videau possédé, dés dessinés, score de match) et les entrées
dont les pions ne sont pas sûrs. On ne confirme pas un contexte que le document
ne suit pas : un videau que la lecture ne trouve pas, un score imprimé dans
l'image sans couche texte ou un trait donné par la prose ne sont pas vus.

.. _cli_document_learned:

Le défaut appris sur le document
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

La lecture apprend sur le document quels champs du contexte il imprime à côté
de ses diagrammes : le videau, les dés (avec le joueur au trait) ou le score, lus
à côté d'au moins un tiers des diagrammes. Elle les nomme dans ``printed`` :

.. code-block:: json

   "defaults": {
     "context": "0:0:1:00:0:0:0:0:10",
     "description": "for what a diagram does not print: centred cube, no dice, the bottom player on roll, money play; the document prints its score, read beside each diagram",
     "printed": ["score"],
     "confirmed": false
   }

Un champ imprimé est la lecture de chaque entrée, que la lecture a certifiée ;
le contexte par défaut ne couvre que les autres. Dans un livre de problèmes en
match, chaque diagramme porte son score : ses entrées restent ``default``, chacune
avec son score dans le XGID, et la confirmation unique du contexte par défaut
certifie le videau centré, l'absence de dés et le joueur au trait de toutes. Un
diagramme dont le score n'est pas lu dans un tel document n'est pas pris en
partie libre : son contexte reste à vérifier (``the document prints its score
beside its diagrams, none is read here: it is not taken as money play``).
``analyze`` et ``check`` affichent ce qui a été appris :

.. code-block:: text

   Printed:    score, read beside each diagram (learned from the document)

Rien n'est gardé d'un document à l'autre. Un relevé sans ``printed`` garde la
règle stricte : tout champ lu hors du défaut laisse le contexte à vérifier.

.. _cli_document_complete:

Quinze et quinze sans doute
~~~~~~~~~~~~~~~~~~~~~~~~~~~

Un diagramme sans pips imprimés n'a pas de recoupement : ses pions restent
``to-verify``. Lus à quinze pions par camp sans aucun doute, ils sont marqués
``"complete": true`` : sur toutes les mesures faites, ces lectures étaient
exactes, mais rien ne les confirme diagramme par diagramme. C'est un état
intermédiaire, distinct de ``sure``. Une entrée complète ne porte, outre le
contexte, que deux doutes : l'absence de pips imprimés, et le sens de jeu
supposé quand aucun pip du document ne l'a confirmé. Des pips imprimés qui
diffèrent des pips calculés sont un doute sur les pions : l'entrée n'est alors
jamais complète. Une position dont des pions sont sortis ne l'est jamais non
plus.

Le sens de jeu supposé est lui aussi un défaut du document, avec sa propre
confirmation : un sens faux échangerait les joueurs de toutes les positions
lues sous lui. Il figure dans le contexte par défaut, sous ``direction`` :

.. code-block:: json

   "direction": {
     "home": "right",
     "description": "direction of play assumed, no printed pips confirm it: checkers move toward the side with the fewer pips, the bottom player's home on the right",
     "confirmed": false
   }

Confirmer le contexte ne le confirme pas. On le contrôle sur quelques
diagrammes, puis on écrit ``"confirmed": true`` dans ``direction``.

``import --accept-complete`` accepte en bloc les entrées complètes dont le
contexte est certifié (lu, ou sous le contexte par défaut confirmé) et dont le
sens de jeu est certain (voté par les pips, ou supposé puis confirmé), sans les
marquer vérifiées ; ``check`` les compte, et compte à part celles qu'un sens
non confirmé retient :

.. code-block:: text

   Direction:  direction of play assumed, no printed pips confirm it: checkers move toward the side with the fewer pips, the bottom player's home on the right, not confirmed: check it on a few diagrams, then set "direction": {"confirmed": true}
   Complete:   84 more held back until the direction of play is confirmed

.. code-block:: text

   Complete:   84 with 15 + 15 checkers read with no doubt and a certified context (imported with --accept-complete)
   Unverified: rank 3, 7, 12 (import refused without --force; rank 12 with --accept-complete)

L'import dit combien il en a acceptées (``Complete: 84 entries of 15 + 15
checkers read with no doubt``). Une entrée complète dont le contexte n'est pas
certifié, ou dont le sens de jeu supposé n'est pas confirmé, reste refusée ; le
message d'erreur dit de le confirmer.

Les pips imprimés se lisent à deux sources : la couche texte d'un PDF, et
l'image elle-même. Dans l'image, la commande lit les nombres isolés autour du
diagramme, chiffre par chiffre, par comparaison avec la numérotation des pointes
du document, avec huit polices courantes embarquées, puis avec les chiffres déjà
lus dans le document ; un nombre n'est rendu que si chacun de ses chiffres est
reconnu nettement, sinon elle s'abstient. L'emplacement des pips est appris sur
le document : un nombre lu qui égale les pips d'un diagramme compté à quinze
pions par camp vote pour sa place, et deux places (haut et bas) doivent réunir
assez de votes. Aucun chiffre n'est choisi d'après les pips du diagramme qu'il
accompagne, et rien n'est gardé d'un document à l'autre. Quand le texte et
l'image donnent chacun des pips et qu'ils diffèrent, les pions ne sont pas
confirmés et le doute le dit (``the two disagree``). Une entrée sans pips lus reste
``to-verify`` (``no printed pip count was found next to the diagram``).

La lecture des chiffres suppose des chiffres arabes sombres sur fond clair,
écrits à l'horizontale, d'au moins 15 pixels de haut à la résolution de lecture,
des pips imprimés à la même place dans tout le document, et une trentaine de
diagrammes lus à quinze pions par camp pour apprendre cette place.

Les doutes nomment ce qui reste à contrôler : une pile qui porte un chiffre, un
disque d'aucune des deux classes (pion pâli ou taché), une colonne pleine de haut
en bas, une pile sombre comptée par sa longueur, un demi-plateau (une rangée sans
pointe), le sens de jeu supposé (les pions avancent vers le côté qui a le moins
de pips), les pions sortis comptés par différence, un videau dessiné hors du
centre, des dés dessinés.

Ne sont jamais lus : la valeur et le propriétaire du videau (le contexte par
défaut le dit centré), les dés (aucun), le score (partie libre), le joueur au
trait (celui du bas), le plateau de sortie (un camp dessiné avec moins de quinze pions a les
autres sortis).

Pourquoi l'import refuse
------------------------

Une entrée ni ``sure`` ni vérifiée n'est pas fiable, et ce qui entre dans une
collection se partage à l'export. L'import refuse donc **tout** le relevé, avant
d'écrire quoi que ce soit, nomme les rangs et dit quoi faire :

.. code-block:: text

   Error: document: entries neither sure nor verified: storage: invalid argument: rank 4, 5, 6; check the cube, dice, player on roll and score of an entry whose context is not certified against its diagram; verify the others ("verified": true), or force the import

Quand des entrées sont sous un contexte par défaut non confirmé, ou complètes
sans ``--accept-complete``, le message le dit aussi :

.. code-block:: text

   Error: document: entries neither sure nor verified: storage: invalid argument: rank 1, 2; confirm the document's defaults once if its diagrams follow them ("defaults": {"confirmed": true}); accept the entries of fifteen checkers a side read with no doubt as a whole (accept complete); check the cube, dice, player on roll and score of an entry whose context is not certified against its diagram; verify the others ("verified": true), or force the import

On vérifie ces entrées, ou on passe ``--force`` pour importer le relevé tel quel ;
l'import dit alors combien d'entrées il a forcées (``Forced: 3 entries neither
sure nor verified``).

.. _cli_document_fallback:

Le repli
~~~~~~~~

Une entrée dont les pions sont certifiés (``sure``, ou complète avec
``--accept-complete`` et un sens de jeu certain) et dont seul le contexte reste à
vérifier n'est pas forcément rejetée : une fois le contexte par défaut confirmé,
l'import propose de la **replier** sur le contexte que porte son XGID, ce qui a
été lu et le défaut pour le reste. ``check`` nomme ces rangs, et le message
d'erreur propose le repli :

.. code-block:: text

   Fallback:   rank 4 imported with --fallback under the context its position carries, the fields to check named in its reference

``import --fallback`` importe ces entrées sans les marquer vérifiées ; la
référence de chacune, dans son commentaire, nomme les champs à contrôler sur le
diagramme avec le doute que la lecture y a laissé, contradiction comprise
(``p. 12, Problem 31 (context not read, to check against the diagram: cube
(cube drawn, value not read))``), et l'import les compte (``Fallback: 1 entries under the context their
position carries``). Le repli ne couvre ni une entrée dont les pions sont à
vérifier, ni un contexte contradictoire, ni un contexte par défaut non
confirmé. Même avec ``--force``, une entrée sans position est
refusée : il faut la renseigner ou la retirer.

Ce qui entre en base
--------------------

Seules les positions entrent, dans l'ordre du document, dans une collection au
nom du document (ou de ``--collection``). Chaque position entre comme une position
apportée seule, et la bibliothèque la reconnaît par son identité : une position
qu'elle a déjà n'est pas dupliquée (``new in the library`` compte celles qui lui
manquaient).

* La **description** de la collection porte le titre, l'auteur, l'édition et
  l'adresse du document, un par ligne. Une collection existante du même nom
  reçoit les positions qui lui manquent et garde sa description si elle en a une ;
  une collection dynamique (définie par une requête) est refusée.
* Chaque position porte un **commentaire** qui donne sa référence, après le
  titre du document : ``Mon livre — p. 146, Problem 191``.

Rien d'autre n'entre, jamais : ni image, ni rectangle, ni texte de l'auteur. Un
import interrompu ou refusé n'écrit rien : c'est une seule transaction.

Une position répétée
--------------------

Quand le document montre deux fois la même position, elle n'entre qu'une fois
dans la collection, à sa première apparition. Son commentaire cite chaque
référence : ``Mon livre — p. 2, Problem 1; p. 4``. ``repeated`` compte ces
répétitions.

Réimporter
----------

Réimporter le même relevé n'ajoute rien : aucune position nouvelle, rien dans la
collection (``added to the collection: 0``), et le commentaire de référence est
réécrit seulement si ses références ont changé, jamais doublé. On peut donc
reprendre un import après avoir corrigé le relevé.

Le démon
--------

Le démon ``serve`` importe un relevé sous ``/v1/documents.importReleve`` avec
les mêmes règles ; il n'analyse aucun document (voir :ref:`headless`).
