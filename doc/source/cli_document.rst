.. _cli_document:

Importer les positions d'un document
====================================

À quoi cela sert
----------------

Un ouvrage que l'on possède — livre de problèmes, recueil, article — montre des
positions en diagrammes, dans un ordre qui est celui de l'étude. La commande
``document`` les range dans une collection de la base, dans cet ordre, pour les
réétudier avec le reste de la bibliothèque. Elle ne fonctionne qu'en ligne de
commande ; il n'y a pas d'écran correspondant.

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
* les pages d'un livre scanné, en images : un dossier de fichiers PNG ou JPEG,
  une page par fichier, lus dans l'ordre de leurs noms, ou une image seule. Les
  diagrammes y sont lus par leur structure (voir :ref:`cli_document_images`).

Un autre PDF est refusé ; ses pages se donnent alors en images :

.. code-block:: text

   Error: document: no diagram readable by its glyphs (only PDF diagrams drawn with eXtreme Gammon's board font are read); for a scanned book, give its pages as images: a folder of PNG or JPEG files, one a page

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
      Positions:  5 distinct, 1 repeated
      Relevé:     releve.json

   Le titre et l'auteur viennent des métadonnées du PDF, à défaut le titre est
   le nom du fichier ; ``--title`` et ``--author`` les remplacent. Le chapitre
   de chaque entrée vient des signets du PDF.

2. ``check`` valide le relevé et compte ses entrées, sans lire ni écrire de
   base. ``--format json`` donne les mêmes comptes en JSON :

   .. code-block:: text

      Document:   Mon livre
      Entries:    6 (sure 3, to verify 2, uncertain 1; verified 0)
      Positions:  5 distinct, 1 repeated
      Unverified: rank 4, 5, 6 (import refused without --force)

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
du document, l'empreinte SHA-256 du fichier lu et le profil de ses diagrammes.
Puis chaque diagramme est une entrée, dans l'ordre du document :

* ``rank`` — son rang, de 1 sans trou ;
* ``page``, ``chapter`` et ``label`` — sa page, le chapitre des signets et le
  nom que le document lui donne (« Problem 191 ») quand il est imprimé au-dessus ;
* ``position`` — la position, en XGID : pions, videau, trait, dés, score et
  longueur du match en une seule chaîne ;
* ``state`` — son état, avec ``doubts``, les doutes nommés de la lecture ;
* ``rect`` — où le diagramme se trouve sur la page, en fraction de sa taille,
  pour le retrouver dans le PDF ;
* ``verified`` — ``true`` quand une personne a comparé la lecture au diagramme.

On relit un relevé en ouvrant le PDF à la page de l'entrée. On y corrige la
position (le XGID), on marque ``"verified": true`` une entrée contrôlée, on
retire une entrée en renumérotant les rangs. Un relevé se vérifie en plusieurs
séances : ``verified`` reste dans le fichier. ``check`` refuse un relevé dont un
champ est inconnu, dont les rangs ont un trou, dont une page recule, dont un
XGID n'est pas une position légale, ou d'un format plus récent que le programme.
Une entrée ``sure`` n'a aucun doute.

Les trois états
---------------

``sure``
   Les recoupements du diagramme tiennent : chaque camp compte quinze pions
   (sur les points, sur la barre et dans le plateau de sortie), et les pips
   calculés d'après la position égalent les pips imprimés des deux joueurs. Le
   score et la longueur du match sont ceux que le PDF imprime à côté. Pour une
   page en image, voir :ref:`cli_document_images`.

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
   La lecture est contredite ou incomplète : un glyphe inconnu, un camp qui ne
   compte pas quinze pions et dont les pips ne correspondent pas, un cadre
   abîmé. Un diagramme dont on ne tire aucune position légale garde sa place
   dans le relevé, sans position (``"position": ""``).

Une entrée est **importable** quand elle est ``sure`` ou vérifiée. Les doutes
qu'elle porte disent quoi contrôler, par exemple :

.. code-block:: text

   pips read 167/167, printed 160/167 (bottom/top)

Le dé n'est pas lu : l'analyse écrit toujours « pas de dés » (``00``).

.. _cli_document_images:

Pages en images
---------------

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

Une telle lecture n'est ``sure`` que si les pips imprimés à côté du diagramme
égalent les pips calculés des deux camps, qu'aucun camp n'a plus de quinze pions
et qu'aucun doute ne reste. Les pips imprimés ne se lisent que dans la couche
texte d'un document, qu'une image n'a pas : une entrée lue d'une image est donc
``to-verify`` et le dit (``no printed pip count was found next to the
diagram``).

Les doutes nomment ce qui reste à contrôler : une pile qui porte un chiffre, un
disque d'aucune des deux classes (pion pâli ou taché), une colonne pleine de haut
en bas, une pile sombre comptée par sa longueur, un demi-plateau (une rangée sans
pointe), le sens de jeu supposé (les pions avancent vers le côté qui a le moins
de pips), les pions sortis comptés par différence, un videau dessiné hors du
centre, des dés dessinés.

Ne sont jamais lus : la valeur et le propriétaire du videau (la position le dit
centré), les dés (aucun), le score (partie libre), le joueur au trait (celui du
bas), le plateau de sortie (un camp dessiné avec moins de quinze pions a les
autres sortis).

Pourquoi l'import refuse
------------------------

Une entrée ni ``sure`` ni vérifiée n'est pas fiable, et ce qui entre dans une
collection se partage à l'export. L'import refuse donc **tout** le relevé, avant
d'écrire quoi que ce soit, et nomme les rangs :

.. code-block:: text

   Error: document: entries neither sure nor verified: storage: invalid argument: rank 4, 5, 6; verify them, or force the import

On vérifie ces entrées, ou on passe ``--force`` pour importer le relevé tel quel ;
l'import dit alors combien d'entrées il a forcées (``Forced: 3 entries neither
sure nor verified``). Même avec ``--force``, une entrée sans position est
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
