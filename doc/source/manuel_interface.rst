.. _manuel_interface:

Prise en main et interface
==========================

Introduction
------------

blunderDB est un logiciel pour constituer des bases de données de
positions de backgammon. Sa force principale est de fournir un lieu unique
pour agréger les positions qu'un joueur a rencontrées (en ligne, en tournoi)
et de pouvoir les réétudier en les filtrant selon divers filtres arbitrairement
combinables. blunderDB peut également être utilisé pour créer des catalogues
de positions de référence.

Les positions sont stockées dans une base de données représentée par un fichier
*.db*. L'application de bureau ouvre ce fichier directement, jamais une adresse
réseau : le :ref:`mode serveur <headless>` est un autre mode du même binaire, et
l'on passe de l'un à l'autre en exportant ou en migrant la base, pas en pointant
l'application vers une URL.

Interactions principales
------------------------

Les principales interactions possibles avec blunderDB sont:

* ajouter une nouvelle position,

* modifier une position existante,

* copier l'image du plateau dans le presse-papier (PNG) via **CTRL-X**, ou avec l'analyse complète via **CTRL-X CTRL-X**,

* supprimer une position existante,

* rechercher une ou plusieurs positions,

* importer des matchs depuis différentes sources (XG, GNUbg, BGBlitz, Jellyfish, HedgeHog), y compris les commentaires depuis les fichiers XG,

* naviguer dans les coups d'un match importé,

* organiser les positions en collections,

* organiser les matchs en tournois.

L'utilisateur peut étiqueter librement les positions à l'aide de tags et les
annoter via des commentaires.

Description de l'interface
--------------------------

L'interface de blunderDB est constituée de haut en bas par:

* [en haut] la barre d'outils, qui rassemble l'ensemble des principales
  opérations réalisables sur la base de données,

* [au milieu] la zone d'affichage principale, qui permet d'afficher ou d'éditer des
  positions de backgammon,

* [en bas] la barre d'état, qui présente différentes informations sur la
  base de données ou la position courante, et intègre la ligne de commande.

Des panneaux peuvent être affichés pour:

* afficher les données d'analyse associées à la position courante issues
  d'eXtreme Gammon (XG), GNUbg, ou BGBlitz,

* afficher, ajouter ou modifier des commentaires,

* rechercher et filtrer des positions selon des critères combinables,

* afficher et gérer les collections de positions (panneau collections),

* afficher la liste des matchs importés et naviguer dans les coups d'un match (panneau matchs),

* afficher et gérer les tournois (panneau tournois),

* afficher les statistiques de performance (panneau Stats),

* évaluer n'importe quelle position avec gammonNet (panneau Eval) et, en bearoff, calculer son EPC (Effective Pip Count),

* s'entraîner sur des exercices de calcul (panneau Entraînement),

* étudier les positions par répétition espacée (panneau Anki),

* jouer un match entier contre le Bot (panneau Duel, voir :ref:`panneau_duel`),

* transcrire un match à la main (panneau Transcription),

* consulter les tables du score — MET, points de prise, valeurs du gammon, nombres de Neil — (panneau Données de référence, voir :ref:`panneau_reference`),

* comparer la position avec les autres bases ouvertes et échanger avec elles
  (panneau Autres bases, voir :ref:`vue_ailleurs`),

* afficher les métadonnées de la base de données (panneau métadonnées).

La hauteur du panneau se règle en tirant sa poignée ; elle reste la même d'un onglet à l'autre.

Un clic droit sur un onglet du panneau propose *Masquer cet onglet*. Dès qu'un onglet est masqué, un bouton *∨* (*Onglets masqués*) apparaît à droite de la rangée : il liste les onglets masqués, et un clic sur *Afficher : …* le rend. Les onglets masqués sont conservés d'une session à l'autre.

Des fenêtres modales peuvent s'afficher pour:

* afficher l'aide de blunderDB, dont le champ de recherche (*/* y place le
  curseur) sélectionne les occurrences du texte saisi, sans égard aux
  majuscules ni aux accents (*ENTREE* passe à la suivante, *MAJ-ENTREE* à la
  précédente),

* afficher le catalogue des visites guidées (voir :ref:`visites_guidees`),

* paramétrer l'export de la base de données,

* configurer blunderDB, notamment la langue de l'interface (voir
  :ref:`configuration`).

La zone d'affichage principale met à disposition à l'utilisateur:

* un plateau afin d'afficher ou d'éditer une position de backgammon,

* le niveau et le propriétaire du cube,

* le compte de course de chaque joueur,

* le score de chaque joueur,

* les dés à jouer. Si aucune valeur n'est affichée sur les dés, la
  position des dés indique quel joueur a le trait et que la position est
  une décision de cube. Lorsque la décision de cube est une réponse à un
  doublement (prise/passe), le videau proposé est affiché au centre du
  plateau, à la valeur offerte.

.. _board_menu_contextuel:

Un clic droit sur le plateau ouvre un menu contextuel. Les deux gestes les
plus courants, *Évaluer cette position* (dans le panneau Eval) et *Copier
l'image* (l'équivalent de *CTRL-X*), sont au premier niveau ; les autres sont
rangés en sous-menus marqués ▸ — *Copier / partager* (dont **enregistrer
l'image dans un fichier** en SVG ou en PNG), *Annoter*, *Naviguer* (positions
voisines, nouvelle vue), *Analyser* (miroir, rollouts), *S'entraîner* (paquet
Anki, défi) et *Gérer* (enregistrer, corbeille). Le menu est construit à
partir d'un catalogue d'actions partagé avec les listes de matchs, de
tournois et de collections, et il n'offre que les actions qui s'appliquent à
la position affichée : voir :ref:`menu_contextuel_actions` pour le détail des
sous-menus et la navigation au clavier.
Pendant un coup joué au plateau (quiz, Transcription), un clic droit sur le
damier reprend tout le coup sans toucher la position ; le menu ne s'y ouvre
qu'avant le premier pas. En édition
et en Eval, où le clic droit sur le plateau pose des pions, le menu s'ouvre
d'un clic droit hors du plateau — hors des dés, du videau et des scores — et
commence par *Effacer la position* (comme *RETOUR ARRIERE*) et *Position de
départ*, qui pose les pions d'une partie neuve, suivis des actions qui valent
pour un brouillon.

Le presse-papier est le geste courant ; enregistrer est l'autre besoin —
l'illustration d'un article, d'un message de forum, d'une leçon. Le **SVG** y
est proposé parce que le plateau en est un : c'est la forme qui survit à un
agrandissement, celle qu'on met dans un document sans la flouter. Le PNG en
dérive, comme la copie dans le presse-papier : un seul rendu, trois
destinations, donc aucune ne peut diverger des autres. Dans le panneau
Recherche et dans Eval, le bouton droit sert déjà à poser les pions de l'autre
couleur : le menu y reste hors du plateau, comme ci-dessus. Voir
:ref:`eval_amener_position` pour amener une position dans le panneau Eval.

La barre d'état est structurée de gauche à droite par les informations
suivantes:

* la ligne de commande, accessible en appuyant sur la touche *ESPACE*,

* un message d'information lié à une opération réalisée par l'utilisateur,

* l'index de la position courante, suivi du nombre de positions dans la
  bibliothèque courante (ou les informations de coup/partie lors de la
  navigation dans un match),

* le **compteur de bibliothèque** — « 412 positions · 38 blunders · 5 matchs »
  — où chaque nombre **ouvre ce qu'il compte** : les positions, la recherche
  ``E>`` au seuil de la bibliothèque (lancée aussitôt, comme si on l'avait
  saisie, et rangée dans l'historique des recherches), ou la liste des matchs.
  Quand la liste à l'écran n'est pas la bibliothèque entière (recherche,
  collection, match), le nombre de blunders s'écrit « 12 / 340 blunders » : les
  blunders de cette liste, puis ceux de la bibliothèque, au même seuil ; une
  liste de plus de 20 000 positions n'est pas comptée et garde le seul total.
  Un chiffre qu'on ne peut pas suivre est une décoration. Le
  seuil des blunders est celui de la bibliothèque, réglé dans l'onglet
  *Bibliothèque* de la configuration et partagé avec les statistiques : deux
  seuils feraient dire deux choses au même mot. Le compteur promet exactement
  ce que le lien ouvre, y compris pour une position jouée de plusieurs façons,
  qui vaut son coût le plus élevé. Sur une très grande bibliothèque (plus de
  200 000 lignes), le compteur ne balaie pas les tables : un nombre précédé de
  « ≈ » est une estimation (une borne haute, les lignes supprimées laissant des
  trous). Si les positions sont estimées, le nombre de blunders, qui n'a pas
  d'estimation honnête, est le dernier décompte exact, gardé tant qu'aucune
  écriture ne peut le changer. Après un import ou une modification, il
  s'affiche « ? » le temps qu'un nouveau décompte se fasse en tâche de fond,
  puis revient de lui-même ; ouverte en lecture seule, la bibliothèque ne
  recompte pas et garde « ? » tant que l'instance qui l'écrit n'a pas refait le
  décompte. Le lien lance la recherche, qui donne le compte exact.
  Les deux nombres de blunders sont des liens distincts : le premier lance la sous-recherche ``ss E>`` dans la liste à l'écran, le second la recherche dans toute la bibliothèque ; quand seul le total s'affiche, il n'y a qu'un lien.

.. note:: Dans le cas de positions issues d'une recherche par l'utilisateur, le
   nombre de positions indiqué dans la barre d'état correspond au nombre de
   positions filtrées.

L'onglet **Anki** porte un **badge** quand des cartes sont à réviser, tous
paquets confondus. Ce chiffre est la raison d'ouvrir l'onglet ; il n'a donc
rien à faire derrière lui. Zéro n'affiche rien : un badge qui dit « 0 » est du
bruit.

La commande ``log`` ouvre le **journal d'activité** : les deux cents dernières
lignes du fichier de journal, un bouton pour les copier — de quoi joindre un
rapport à un signalement — et un autre pour ouvrir le dossier qui les
contient. Le journal n'est ni filtré ni reformaté : un journal qu'on embellit
est un journal qu'on ne peut plus citer.

La commande ``grid`` ouvre la **planche-contact** : la liste parcourue —
résultats d'une recherche, bibliothèque, collection — en grille de
mini-plateaux, dessinés comme le plateau, par pages de vingt-quatre. Elle
s'ouvre sur la page de la position courante, dont la vignette est encadrée ;
un clic ou *ENTREE* sur une vignette ouvre sa position sur le plateau et
referme la planche, et elle se parcourt entièrement au clavier (voir
:ref:`raccourcis_contact_sheet`). Elle ne s'ouvre ni en mode édition ni dans
un match, qui se parcourt par ses coups.

Dans l'**historique de recherche** du panneau Recherche, chaque jeton d'une
commande enregistrée s'affiche en pastille nommée — *Sans contact*, *Erreur de
coup* — plutôt qu'en jeton nu. La commande exacte reste en infobulle, car
c'est elle qu'on relance ; et un jeton que blunderDB ne reconnaît pas
s'affiche **tel quel** plutôt que traduit au plus proche.

.. _onglets_vues:

Onglets de vues
---------------

Sous la barre d'outils, une barre d'onglets permet de travailler avec
plusieurs **vues** en parallèle. Chaque vue est un espace de travail
indépendant qui conserve sa propre liste de positions, l'index de la position
courante, la position affichée, l'analyse et le coup sélectionné, le panneau
actif, le commentaire en cours ainsi que le contexte de navigation dans un
match. Il est ainsi possible, par exemple, de garder une recherche ouverte
dans une vue tout en parcourant un match dans une autre.

* **Créer une vue** : cliquer sur le bouton *+* de la barre d'onglets (nommé
  *+ Nouvelle vue* tant qu'il n'y a qu'une vue) ou appuyer sur *CTRL-T*. La nouvelle vue démarre comme une copie de la vue
  courante.

* **Fermer une vue** : cliquer sur la croix de l'onglet ou appuyer sur
  *CTRL-W*. La dernière vue ne peut pas être fermée ; la croix n'est dessinée que sur les onglets du groupe actif, quand il compte plus d'une vue. Fermer la vue active affiche le dernier onglet restant.

* **Changer de vue** : cliquer sur un onglet, appuyer sur *CTRL-Page préc.* /
  *CTRL-Page suiv.* (ou *MAJ-J* / *MAJ-K*) pour passer à la vue précédente /
  suivante, ou *CTRL-1* à *CTRL-9* pour atteindre directement la n-ième vue
  de la bibliothèque active.

* **Renommer une vue** : double-cliquer sur l'onglet du groupe actif, saisir le
  nouveau nom et valider avec *ENTREE* ; quitter le champ valide aussi,
  *ECHAP* annule. Un nom vide laisse l'ancien.

Une vue laissée dans le panneau Eval y revient telle qu'elle a été laissée :
son plateau brouillon, et en sortant du panneau, la position qu'elle
étudiait. Le dernier plateau brouillon du panneau Eval est lui aussi propre à
chaque vue.

Les vues sont enregistrées avec l'état de session de la base de données et
restaurées à sa réouverture.

.. _groupes_bibliotheques:

Plusieurs bibliothèques ouvertes
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Plusieurs bases de données peuvent rester ouvertes en même temps. Chacune
forme un **groupe** d'onglets : dans la barre, un en-tête de couleur porte le
nom du fichier, suivi des vues de cette bibliothèque, dont le bord supérieur
reprend la même couleur. Une seule bibliothèque est active à la fois : c'est
elle que lisent et écrivent les panneaux, la recherche, les imports et
l'assistant.

Avec une seule bibliothèque ouverte, la barre ne montre aucun en-tête : seul
le titre de la fenêtre nomme la base. Tant qu'il n'y a qu'une vue en tout, la barre ne dessine aucun onglet non plus : seul le bouton *+ Nouvelle vue* reste. L'infobulle d'un en-tête de groupe donne le chemin complet du fichier.

* **Ouvrir une bibliothèque dans un nouveau groupe** : bouton *⧉ Autre
  bibliothèque* à droite de la barre d'onglets (*⧉* seul dès que deux
  groupes sont ouverts), *CTRL-MAJ-O*, ou la commande ``open_group``
  (``og``). La bibliothèque active jusque-là reste ouverte dans son groupe.
  *CTRL-O* remplace, lui, la bibliothèque du groupe actif ; l'infobulle du
  bouton *Ouvrir* de la barre d'outils rappelle les deux raccourcis.

* **Changer de bibliothèque** : cliquer un onglet d'un autre groupe active
  sa bibliothèque et cette vue d'un seul geste ; cliquer l'en-tête du groupe
  la ramène sur la vue où elle a été laissée. *CTRL-MAJ-Page préc.* /
  *CTRL-MAJ-Page suiv.* passent à la bibliothèque précédente / suivante ;
  *CTRL-Page préc.* / *CTRL-Page suiv.* (ou *MAJ-J* / *MAJ-K*) parcourent les
  vues de tous les groupes à la suite.

* **Fermer un groupe** : la croix de son en-tête ferme la bibliothèque et ses
  vues. Fermer le groupe actif affiche son voisin ; la barre d'état nomme la
  bibliothèque fermée et celle affichée.

* **Réordonner** : glisser un onglet le déplace avant ou après un autre onglet
  du même groupe (un trait de la couleur d'accent marque l'emplacement) ;
  glisser un en-tête de groupe déplace le groupe entier, avec toutes ses
  vues, avant ou après un autre groupe. Une vue appartient à sa bibliothèque :
  la déposer sur un autre groupe est refusé et ne change rien. L'ordre des
  vues est celui des raccourcis *CTRL-1* … *CTRL-9* et il est conservé avec la
  session ; l'ordre des groupes vaut pour la session en cours.

Le bouton *+* qui crée une vue est rattaché à droite du dernier onglet du
groupe actif : la nouvelle vue rejoint ce groupe et apparaît là où l'on a
cliqué. La barre se partage en deux zones : à gauche les groupes et leurs
onglets, à droite, alignés au bord, les outils (autre bibliothèque, écran
partagé).

Passer d'un groupe à l'autre ne déplace jamais le plateau : dès que deux
bibliothèques sont ouvertes, la place de la barre d'information de la position
(sous les onglets) reste réservée, vide quand le groupe affiché n'a rien à y
dire.

L'en-tête du groupe actif porte le nom en blanc sur sa couleur ; ceux des
autres groupes, le nom sur le fond de la barre, souligné en haut de leur
couleur. Six couleurs se succèdent : un septième groupe reprend la première.
Les vues sont numérotées dans leur bibliothèque : une base ouverte dans un
nouveau groupe commence à « #1 », et *CTRL-1* … *CTRL-9* visent les vues du
groupe actif. Quand les onglets ne tiennent plus dans la largeur de la
fenêtre, les onglets rétrécissent d'abord (le nom se termine par une
ellipse, le nom complet est dans l'infobulle) jusqu'à une largeur minimale ;
au-delà la barre défile (molette, ou le changement de vue qui amène l'onglet
affiché à l'écran). Des flèches *‹* et *›* apparaissent alors aux deux bords de
la rangée : un clic défile d'un cran, un appui maintenu défile en continu, et
la flèche est grisée en bout de liste. Un dégradé marque le bord où des onglets
sont cachés. Le bouton *∨* des outils, à droite, visible dès deux vues, ouvre le menu *Toutes les
vues* : les vues de chaque bibliothèque, la vue courante cochée ; un clic
active la vue et son groupe.

La bascule est immédiate : chaque groupe garde en mémoire ses vues, sa liste
de positions, la position courante, les critères de recherche (même en cours
de saisie, sans avoir lancé la recherche), la liste des matchs, les compteurs
et le défilement de chaque liste du panneau, et les retrouve sans relire la
base. Une bibliothèque ouverte dans un nouveau groupe part d'un formulaire de
recherche vierge : les matchs et tournois choisis dans un groupe ne
s'appliquent jamais dans un autre. Le travail en cours le suit aussi : un
brouillon de transcription, une leçon, une révision Anki, la file d'étude et
la Direction ouverte se retrouvent tels qu'ils ont été laissés. Ce qui a pu
changer pendant l'absence est relu juste après l'affichage : les compteurs
toujours, la liste des matchs si leur nombre a changé ou si une tâche de fond
a écrit dans la bibliothèque pendant l'absence. Les listes
propres à chaque base (collections, tournois, paquets Anki, brouillons,
Duels, historique de recherche, filtres enregistrés) sont relues dans la
bibliothèque affichée.

Un même fichier ne s'ouvre pas dans deux groupes : le rouvrir affiche le
groupe qui le tient déjà.

Ce qu'un autre programme écrit dans une bibliothèque ouverte — ``blunderdb``
en ligne de commande, un serveur ``blunderdb serve`` sur le même fichier —
est relu aussi : quand la fenêtre revient au premier plan, ou quelques
secondes après la fin de l'écriture, et au plus toutes les dix secondes
pendant un long import. Une bibliothèque d'un autre groupe est relue à son
retour.

Les tâches de fond restent attachées à leur bibliothèque : un import de
fichiers de matchs, le dossier surveillé et l'analyse GammonNet par lots
continuent dans le groupe où ils ont commencé quand on en affiche un autre, et
n'écrivent jamais dans la bibliothèque affichée. Leur progression et leur
compte rendu restent dans leur groupe : tant qu'il n'est pas affiché, son
en-tête porte une pastille, qui clignote pendant que la tâche travaille et
devient fixe quand elle est terminée. Au retour sur le groupe, la pastille
s'efface, la barre d'état dit ce que la tâche a fait et la liste des matchs
est relue. L'analyse automatique qui suit un import se lance dans la
bibliothèque de l'import. Le dossier surveillé appartient au groupe où il a
été démarré : les fichiers de matchs y sont importés en fond, tandis qu'une
base à fusionner ou un texte de position attend que ce groupe soit de nouveau
affiché. Fermer un groupe arrête ses tâches et sa surveillance.

**Copier d'une bibliothèque à l'autre.** Une ligne glissée sur l'en-tête d'un
autre groupe y est copiée, dans un sens comme dans l'autre : un match de la
liste des matchs, une collection de la liste des collections, une position
d'une collection ouverte, la position affichée (on la saisit par son compteur
« n / N » dans la barre d'état) ou la leçon en cours de lecture (par son nom
dans la barre de la leçon). L'en-tête visé s'entoure d'un pointillé. La copie
est celle de la commande ``transfer`` : la sélection d'un export, fusionnée
par l'import natif, avec les analyses, les commentaires et les coups joués ;
une collection emporte ses positions, un match sa transcription. Ce que la
bibliothèque d'arrivée tient déjà n'est pas recopié : copier de A vers B puis
de B vers A ne crée aucun doublon, et rien n'y dit d'où vient la copie. La
copie tourne en fond, attachée au groupe d'arrivée : son en-tête porte la
pastille, et la barre d'état dit, au retour, combien de positions et de matchs
sont nouveaux et combien étaient déjà là.

Le **lot** rassemble des lignes de plusieurs bases pour les copier en une
fois ; il vit dans le panneau Autres bases (voir :ref:`vue_ailleurs`).

Le **cadenas** d'un en-tête, *Protéger des copies*, protège sa base : rien n'y
est plus copié, ni par glisser, ni depuis le panneau Autres bases, ni par le
lot ou le menu contextuel, et la barre d'état le dit. La protection ne touche
ni les imports ni les modifications faites dans la base ; un second clic la
lève. Elle est mémorisée par fichier d'une session à l'autre, dans les
préférences de blunderDB et non dans la base : la rouvrir la protège de
nouveau.

Le changement de groupe reste refusé, au clic comme au clavier, avec un
message dans la barre d'état qui nomme la cause et dit où l'arrêter, quand ce
qui tourne suit l'écran : un rollout, dont le résultat s'affiche sur la
position du plateau (``rollout stop``) ; l'affichage d'un tournoi, qui sert la
bibliothèque affichée sur le réseau du club (on l'arrête dans la Direction) ;
un Duel en cours, qui tient le plateau et ses vues ; une séance
d'entraînement, dont le chronomètre et le journal ne suivent aucun groupe ;
enfin, le temps qu'il dure, l'import des fichiers que seule l'interface sait
lire (textes de position, importés un par un) ou la fusion d'une base.
À la fermeture de blunderDB, seule la session de la bibliothèque active est
enregistrée, et la liste des groupes ne l'est pas : au démarrage suivant,
blunderDB rouvre la dernière base active.

.. _vue_ailleurs:

Comparer et échanger : le panneau Autres bases
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Le panneau **Autres bases** (*CTRL-MAJ-A*) est le poste d'échange entre les
bases ouvertes en groupes : il dit, pour la position affichée, ce qu'en savent
les autres bases, et permet de la copier de l'une à l'autre. La position y est
retrouvée par son identité de plateau (son hachage), non par son numéro : un
même numéro désigne deux positions différentes dans deux bases. Avec une seule
base ouverte, le panneau l'explique et propose *Ouvrir une base pour échanger*
(*CTRL-MAJ-O*) ; sans position affichée, il le dit et ne lit rien.

Chaque autre base a son bloc, bordé de la couleur de son groupe. Son en-tête
dit si elle contient la position : *présente*, *absente*, ou *présente avec
une autre analyse* quand son meilleur coup n'est pas celui de la base active.
Deux boutons échangent la position : *Copier vers …* la copie de la base
active dans celle-ci, *Récupérer depuis …* la copie de celle-ci dans la base
active, avec son analyse, ses commentaires et ses coups joués. Une copie
n'écrase jamais l'analyse que la base d'arrivée tient déjà : le bloc l'écrit,
*Analyse conservée là-bas*, avant comme après la copie. Une position encore en
brouillon doit d'abord être enregistrée. Viennent ensuite des blocs titrés,
repliables :

* **son analyse** : le meilleur coup, ou la meilleure action de videau, et le
  moteur qui l'a trouvé. Une analyse valorisée avec une autre table d'équité
  de match que celle de la bibliothèque active porte la mention *autre MET* ;
* **ses commentaires**, chacun avec son auteur, ou son signataire et l'état de
  la signature comme dans le :ref:`panneau des commentaires <panneau_commentaires>` ;
* **les matchs où la position a été jouée** : joueurs, épreuve, date, numéro
  du coup, coup ou décision joués et leur coût. Le verdict (*bon*, *erreur*,
  *gaffe*) est rendu avec les seuils de la bibliothèque **active**, pas avec
  ceux de l'autre ; un coût calculé avec une autre MET n'est pas comparé et
  reste marqué *autre MET*. Au-delà de 50 matchs, les 50 derniers importés
  sont listés et le total est indiqué ;
* **ses positions proches**, les cinq plus proches, en vignettes avec leur
  distance ; ce bloc est replié quand la base contient la position, déplié
  sinon.

*Ouvrir là-bas*, un clic sur un match ou sur une vignette active le groupe de
cette base et y affiche la position. Lire n'écrit rien dans aucune base. Le
pied du panneau rappelle les seuils du jugement, et *⟳ Relire les bases* relit
les autres bases.

**La sélection.** Quand un match, un tournoi, une collection ou une leçon est
sélectionné dans son panneau, un bloc en tête du panneau le nomme et dit, pour
chaque autre base, si elle le contient déjà, avec ce qu'il contient là-bas
(parties, matchs, positions ou étapes), et propose les mêmes *Copier vers …* et
*Récupérer depuis …*. Le match est reconnu à son empreinte, même importé d'un
autre format ; le tournoi, la collection et la leçon à leur nom, comme une
copie les fusionne.

**Annuler.** Chaque copie qui a ajouté quelque chose à la base d'arrivée,
position, match, tournoi, collection ou lot, par le panneau, le menu
contextuel, un glisser ou l'écran partagé, entre dans l'historique des copies
de la session : la barre d'état propose *Annuler la copie (n)*, qui reprend la
plus récente, puis la précédente, jusqu'à la fermeture de blunderDB. Annuler
ne retire que ce que la copie a créé : une position, un match ou un tournoi
que la base tenait déjà reste, et une collection du même nom ne perd que les
positions que la copie lui a ajoutées. Une annulation se refuse, et le dit, si
ce que la copie a créé a changé depuis (un commentaire, une collection, un
match rangé dans le tournoi) ou si le groupe a ouvert une autre base. Une copie
qui a créé une leçon ne s'annule pas : retirer ses positions couperait ses
étapes.

**Le lot**, en bas du panneau, rassemble des lignes de plusieurs bases pour
les copier en une fois. *MAJ-B* y met la position affichée, ou l'en retire,
comme *b* pour la Pile ; le groupe *Envoyer* du menu contextuel y ajoute une
position, un match ou un tournoi, et une ligne glissée sur le lot y entre
aussi. Chaque ligne se retire d'un clic sur ×. On choisit la *Base d'arrivée*,
puis *Copier le lot vers …* ; le titre du lot, glissé sur l'en-tête d'un
groupe, l'y copie aussi. Un tournoi part avec tous ses matchs ; un match copié
seul n'emporte que la fiche de son tournoi. Le lot vit en mémoire, jamais dans
une base : fermer un groupe en retire ses lignes, et il disparaît à la
fermeture de blunderDB.

Le serveur n'a pas de *Copier vers* : un tenant n'écrit pas chez un autre ;
la route ``across.transferPull`` y fait la copie de *Récupérer depuis*, et
``blunderdb transfer`` la fait en ligne de commande.

**L'échange sur place.** La touche *x* montre, à la place du plateau, au même
endroit et à la même taille, la position de l'autre bibliothèque, sous un
bandeau à sa couleur qui rappelle son nom, son meilleur coup et le nombre de
ses commentaires. Maintenue, la touche ramène le plateau quand on la relâche ;
pressée brièvement, l'échange reste affiché jusqu'à la pression suivante.
Changer de position ou de fenêtre ramène aussi le plateau. Le groupe actif,
la vue et la liste ne changent pas. La position montrée est, par défaut, la
même position dans la première autre bibliothèque qui la contient ; le
bouton *Montrer avec x* sous une vignette du panneau Autres bases choisit
plutôt cette voisine, jusqu'à *Ne plus montrer avec x* ou un changement de
groupe.

**L'écran partagé.** Sur un grand écran, le bouton ◫ de la barre des onglets
(*CTRL-MAJ-X*) coupe en deux la seule zone du plateau. Chaque volet porte une
bibliothèque ouverte et garde son côté : le groupe actif à gauche à
l'ouverture, la première autre bibliothèque à droite (la même, s'il n'y en a
qu'une ; l'en-tête dit alors *même base*). Le volet actif est vivant, cerné de
la couleur de son groupe. L'autre est figé et atténué : le plateau de sa
bibliothèque là où elle a été laissée, l'analyse de la position et son
commentaire ; rien de ce qu'il montre n'est écrit nulle part. Chaque volet
affiche son compteur *n / N*.

Un clic dans le volet figé, ou *CTRL-ALT-←* / *CTRL-ALT-→*, active sa
bibliothèque ; le contour passe de l'autre côté et rien d'autre ne bouge. Avec
une seule bibliothèque, les deux vues échangent leurs positions, sauf en mode
match : un match rejoué n'a pas de liste où retrouver la position, la barre
d'état le dit. Barre d'outils, onglets, panneaux et barre d'état restent en un exemplaire et
parlent toujours de la bibliothèque active : sa pastille et son nom précèdent
les onglets des panneaux et le message de la barre d'état, et le titre de la
fenêtre la nomme. Le clavier, *CTRL-S*, les imports et les commandes vont au
volet actif. Pendant un Duel, une séance d'entraînement ou un import, le
changement est refusé : le volet cliqué tremble et la barre d'état dit
pourquoi. Tant qu'un changement de bibliothèque n'est pas achevé, un second est
ignoré, et *CTRL-S*, la mise à jour et la suppression d'une position sont
refusées.

Glisser le compteur d'un volet, ou le plateau du volet figé, sur l'autre volet
copie la position dans la bibliothèque de ce volet, avec son analyse et son commentaire, sauf si
sa base est protégée des copies ; la copie rejoint l'historique que *Annuler*
reprend. À partir
de 1600 pixels de large, le panneau reste à côté des plateaux ; entre 1280 et
1600, il passe dessous ; sous 1280, l'écran partagé se refuse et propose
l'échange sur place (*x*), et il se ferme si la fenêtre devient trop étroite.
L'écran partagé et la bibliothèque de chaque côté sont repris à la session
suivante.

.. _configuration:

Configuration
-------------

Le bouton de configuration (icône en forme de rouage) situé dans la barre
d'outils, à gauche du bouton d'aide, ouvre la fenêtre de configuration de
blunderDB. Elle est organisée en dix onglets :

* **Interface** — thème, langue, échelle d'affichage, position du panneau, pas de
  PageUp / PageDown (10, 50, 100, 500 ou 1 000 positions, ou 10 % de la liste),
  journaux, vérification des mises à jour, votre nom et positions voisines ;
* **Couleurs** — les couleurs du plateau ;
* **Bibliothèque** — ce qui appartient à la base ouverte : les seuils
  d'erreur et de blunder, décrits ci-dessous ;
* **Maintenance** — l'état de santé de la base, une ligne par opération,
  décrit ci-dessous ;
* **Corpus** — les doublons à l'import, les alias de joueurs et d'événements
  et la recherche des doublons probables (voir l'import de matchs) ;
* **Bearoff** — les tables de sortie utilisées par le panneau Eval ;
* **gammonNet** — les réglages de l'évaluateur embarqué, décrits ci-dessous ;
* **Dossier surveillé** — l'import automatique des matchs qui arrivent dans un
  dossier, décrit ci-dessous ;
* **Assistant et MCP** — le serveur MCP local et l'assistant interne, décrits
  ci-dessous ;
* **Identité d'émetteur** — la clé qui signe vos filigranes, décrite à la
  section :ref:`diffusion_controlee`.

L'onglet *Interface* propose d'abord un **thème** : *suivre le système*,
*clair*, *sombre*, *contraste élevé* ou *imprimable*. Le thème règle les
couleurs de l'interface et **propose une palette de plateau** — une interface
sombre autour d'un plateau clair n'est pas un thème sombre, c'est la moitié
d'un, puisque le plateau occupe l'essentiel de la fenêtre.

Vous gardez le dernier mot, et le mécanisme le garantit plutôt que de le
promettre : l'onglet *Couleurs* continue de régler le plateau directement, et
une couleur choisie après le thème est la vôtre. Au démarrage, seuls les
jetons de l'interface sont appliqués, jamais la palette du plateau — celle que
vous avez réglée est déjà chargée, et la réécrire à chaque lancement
effacerait votre travail une session à la fois. Voir
`ADR-0038 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0038-a-named-theme-carries-the-board-palette-and-the-user-still-has-the-last-word.md>`__.

*Suivre le système* est le réglage par défaut : il obéit à la préférence
clair/sombre du bureau, y compris lorsqu'elle change en cours de session. Un
outil n'impose pas son clair ou son sombre à un bureau qui a déjà tranché.

L'onglet *Interface* permet aussi de choisir la langue parmi l'anglais, le
français, l'allemand, l'italien, l'espagnol, le finnois, le japonais, le grec
et le russe. L'ensemble de l'interface (barre d'outils, panneaux, messages,
aide) est traduit dans la langue sélectionnée. Le choix de la langue est
enregistré et conservé d'une session à l'autre.

L'onglet *Bibliothèque* réunit ce qui appartient au fichier ouvert et non à la
machine. Il est vide tant qu'aucune base n'est ouverte, et il le dit.

Il porte d'abord les deux **seuils** qui décident du vocabulaire de toute
l'application : une décision est une **erreur** dès que son coût atteint le
seuil d'erreur, et cette erreur est un **blunder** dès qu'il atteint le seuil
de blunder. Tout blunder est une erreur, donc le premier seuil ne peut pas
dépasser le second, et blunderDB refuse la paire inversée. Les valeurs sont
saisies en équité — « 0,080 » — l'unité de toutes les tables ; la ligne de
commande, elle, parle en millipoints, si bien que le seuil 0,080 s'écrit
``E>80`` dans une recherche.

Ces seuils suivent le fichier, pas l'ordinateur : la même base compte les
mêmes blunders partout où on l'ouvre, ``blunderdb info`` les affiche, et
``blunderdb edit --error-threshold`` / ``--blunder-threshold`` les règle. Ils
ne voyagent pas dans un export : un seuil est une habitude de lecture, pas un
fait des positions.

Trois **préréglages** sont proposés d'un clic, chacun sous le nom du programme
qui a tracé cette ligne : blunderDB (0,050 / 0,100), XG (0,020 / 0,080) et
gnubg (0,040 / 0,080). Par défaut, une bibliothèque lit 0,050 et 0,100.

Ce qu'ils changent, à l'écran : le nombre de blunders du compteur de la barre
d'état et la recherche que son lien prépare, les colonnes « Erreurs » et
« Blunders » des statistiques et du tableau des joueurs, les marques des
coups dans la fiche d'un match (:ref:`panneau_matchs`), et la liste des positions que
blunderDB propose de revoir après un import.

La ligne **Espace récupérable** de l'onglet *Maintenance* porte le bouton
**Compacter**, qui récupère
l'espace disque laissé par les suppressions (matchs, tournois, purges) : la
base de données ne rétrécit jamais toute seule quand on supprime des données,
il faut demander explicitement ce compactage. L'opération peut prendre du
temps sur une grosse base. La copie compactée est écrite à côté du fichier,
puis le remplace : il faut, temporairement, environ la taille de la base en
espace disque libre sur son volume, et non en mémoire vive. Si un autre
programme a la base ouverte (une seconde instance en lecture seule, par
exemple), le fichier n'est pas remplacé : la base est compactée sur place, ce
qui demande environ deux fois sa taille en espace libre, et un fichier
temporaire va dans le dossier temporaire du système, ou dans celui que
désigne la variable d'environnement ``SQLITE_TMPDIR`` : sur un petit
``/tmp``, pointez-la vers le volume de la base. blunderDB refuse de démarrer,
avec un message qui dit ce qui manque (espace disque, ou moins de 512 Mo de
mémoire disponible), plutôt que de risquer un compactage interrompu.
Une confirmation est demandée avant de lancer l'opération. Le résultat — l'espace gagné, en mégaoctets — s'affiche
ensuite dans la barre d'état. La même opération est disponible en ligne de
commande via ``blunderdb vacuum`` (voir :ref:`cli`).

Après un compactage sur place, blunderDB vérifie que le fichier a bien
rétréci. Sous Windows, le système refuse de raccourcir un fichier qu'un autre
programme garde ouvert de certaines façons : le compactage se termine alors
par un message qui le dit — fermez l'autre programme et relancez-le — au lieu
d'annoncer un gain qui n'a pas eu lieu.

Le bouton **Ouvrir le dossier des journaux**, dans l'onglet *Interface*, ouvre le
dossier contenant le journal de l'application — utile pour joindre des
détails à un signalement de problème, en particulier quand blunderDB est
lancé depuis un raccourci ou un double-clic, sans terminal attaché pour
afficher quoi que ce soit.

La case **Vérifier les mises à jour au démarrage**, du même onglet, désactivée par défaut,
interroge une fois la page des dernières versions du dépôt GitHub à chaque
lancement et affiche, dans la barre d'état, un message si une version plus
récente est disponible — jamais une fenêtre qui bloque l'utilisation.
Cette vérification reste désactivée automatiquement sur une installation
passée par un gestionnaire de paquets (Flatpak, Homebrew, un paquet de
distribution…) : c'est ce canal-là qui gère alors les mises à jour, pas
blunderDB lui-même.

Deux réglages de l'onglet *Interface* gouvernent le classement des **positions voisines** (jeton
``like``, voir :ref:`panneau_recherche`) : le nombre de voisines rendues, et la
distance maximale au-delà de laquelle une position cesse d'en être une. Cette
distance vaut zéro par défaut, c'est-à-dire aucun plafond : l'échelle dépend de
la phase de la partie, et une valeur choisie ici se lirait comme une mesure. Le
jeton ``like<12`` impose la sienne pour une recherche, sans toucher au réglage.

L'onglet *Couleurs* permet de personnaliser les couleurs du
plateau. Chaque élément dispose de son propre sélecteur de couleur : le fond,
la bordure, les flèches claires et foncées, les pions du joueur 1 et du joueur
2, les dés, les points des dés et le videau. Le bouton *Réinitialiser* rétablit
l'ensemble des couleurs par défaut. Comme la langue, les couleurs choisies sont
conservées d'une session à l'autre.

L'onglet *Bearoff* gère les tables de sortie du panneau Eval (voir
:ref:`panneau_epc`). Elles ne sont **ni embarquées dans l'exécutable, ni
téléchargées** : blunderDB les calcule sur la machine qui s'en sert, et le
résultat est identique octet pour octet à ce que produit gnubg — l'empreinte
SHA-256 est vérifiée avant que la table ne soit acceptée.

Les deux tables ordinaires (TS-06-06 pour le verdict de videau, OS-06 pour
l'EPC) sont calculées au premier lancement, en arrière-plan et sans rien
demander : environ six secondes sur un cœur, pendant lesquelles l'application
s'utilise normalement. Le panneau Eval ne le signale que si l'on y pose une
position qui a besoin d'une table pas encore prête.

L'onglet affiche le domaine actif et son origine, l'état de la table une face
que lit l'EPC, le dossier où tout cela vit, et la liste des tables présentes
avec leur taille et leur verdict. Chaque ligne se supprime individuellement,
après confirmation.

**Vérifiée ou non vérifiée.** Une table *vérifiée* a exactement les octets que
gnubg produit pour son domaine : son empreinte SHA-256 figure dans blunderDB et
a été retrouvée. Les empreintes enregistrées pour les tables une face (OS-06 à
OS-10) sont celles que produit l'outil ``makebearoff`` de GNUbg 1.08. Une table *non vérifiée* est bien formée mais son domaine n'a
pas d'empreinte enregistrée — rien ne lui est reproché, simplement personne ne
l'a comparée à la référence. Une table *corrompue* se contredit elle-même et
n'est jamais lue ; elle est recalculée.

**Calculer une table plus large.** Le domaine se choisit dans une liste à deux
familles, avec le nombre de cœurs à y consacrer (par défaut tous sauf un, pour
que la machine reste utilisable) :

* **videau exact (deux faces)**, de TS-06-06 à TS-06-15 : élargit le domaine où
  la probabilité de gain et le verdict de videau sont lus plutôt qu'estimés ;

* **EPC hors du jan (une face)**, de OS-06 à OS-10 : élargit la distance à
  laquelle un pion peut se trouver sans que le bloc EPC se taise. Ce balayage
  ne lit que des positions plus petites que celle qu'il calcule, donc il est
  séquentiel par construction et le nombre de cœurs ne lui sert à rien — le
  sélecteur le dit en se grisant.

Avant de lancer quoi que ce soit, l'onglet annonce trois chiffres pour le
domaine choisi : la taille sur le
disque, la mémoire nécessaire pendant le calcul, et le temps que cela devrait
prendre *sur cette machine*. Ce dernier commence par une estimation, puis
devient une mesure : chaque calcul assez large relève sa propre vitesse et la
conserve. Un domaine que la mémoire disponible ne permet pas est proposé grisé,
avec la raison — « il faudrait 24 Go, il en reste 12 » est une réponse, une
ligne absente n'en serait pas une.

À titre d'ordre de grandeur, sur une machine à seize fils : TS-06-09 pèse
191 Mo et demande une dizaine de secondes, TS-06-11 pèse 1,2 Go et quelques
minutes, TS-06-13 dépasse ce que la plupart des machines peuvent tenir en
mémoire. Du côté une face, sur un cœur : OS-07 pèse 4,9 Mo et prend 17 s,
OS-08 15 Mo et 1 min 20, OS-10 117 Mo et une demi-heure.

**Pause et reprise.** Pendant le calcul, la progression affiche le temps
restant *mesuré*, et deux boutons distincts : *Pause* et *Annuler*. La pause
écrit l'état du calcul à côté de la table ; le relancer reprend là où il s'est
arrêté au lieu de tout recommencer. Annuler ne garde rien. Fermer la fenêtre de
configuration n'interrompt rien — le calcul continue en arrière-plan.

Un calcul mis en pause se retrouve au lancement suivant, nommé et chiffré
(« TS-06-09 interrompue à 43 % »), avec *Reprendre* et *Supprimer*. Rien ne
redémarre tout seul : c'est l'utilisateur qui a demandé l'arrêt.

L'onglet permet enfin de pointer vers un fichier ``.bd`` two-sided externe, par
exemple une base produite par gnubg lui-même : la table au domaine le plus
large l'emporte.

.. _onglet_maintenance:

L'onglet **Maintenance** montre l'état de santé de la base : une ligne par
opération, avec son compte, sa dernière exécution (date et issue : terminée,
annulée ou en échec) et son action. Seul le calcul des matchs sans
statistiques se lance tout seul, à l'ouverture.

* **Positions sans analyse** — les positions qui ne portent aucune analyse ;
  l'action est l'analyse gammonNet décrite plus bas. Les positions que
  gammonNet refuse d'évaluer (une danse, un score hors de la table de match)
  restent dans ce compte, qui n'atteint donc pas forcément zéro.
* **Analyses gammonNet périmées** — décrites plus bas.
* **Matchs sans statistiques** — les matchs dont les statistiques ne sont pas
  encore calculées ; l'action les calcule tous. Une base ouverte en écriture
  qui en compte les calcule d'elle-même en arrière-plan, par lots : la barre
  d'état montre la progression de cette ligne et son bouton d'annulation, les
  badges des tournois et l'onglet Joueurs du panneau Statistiques attendent la
  fin du calcul, et le reste de l'application reste utilisable. Ouvrir une
  autre base l'arrête. En lecture seule, ou en ligne de commande, la première
  lecture des statistiques les calcule, ce qui peut être long sur une grosse
  base : le panneau Statistiques affiche alors ce nombre, qui ouvre cet
  onglet.
* **Colonnes d'analyse** — les colonnes que la recherche et les statistiques
  interrogent sont une projection des analyses stockées, lesquelles restent
  intactes : l'action les recalcule sans rien réimporter.
* **Erreurs des coups joués** — note les coups joués qui n'ont pas encore
  d'erreur.
* **Analyses à l'ancien format** — réécrit au format binaire les analyses
  qu'une version antérieure stockait en JSON.
* **Corbeille** — le nombre d'entrées de la corbeille, qui l'ouvre.
* **Espace récupérable** — l'espace libre du fichier, que le compactage
  décrit plus haut rend au disque.

Un compte coûteux (il lit une table entière) attend le bouton **Compter**.
Un compte de positions ouvre ces positions dans la liste ; celui des matchs
sans statistiques ouvre ces matchs dans le panneau des matchs. Une opération
tourne en arrière-plan : sa progression et un bouton d'annulation s'affichent
dans la barre d'état, et une seule opération qui écrit dans la base tourne à
la fois — une analyse gammonNet, un rollout ou une autre opération de
maintenance en cours fait refuser la suivante, avec le nom de celle qui
occupe la base. Sur une base ouverte en lecture seule, les actions sont
désactivées. Les mêmes opérations existent en ligne de commande
(``blunderdb maintenance``).

L'onglet **gammonNet** règle l'évaluateur embarqué (voir `ADR-0011 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0011-gammonnet-is-ported-to-go-and-the-representation-boundary-sits-at-the-evaluator-s-edge.md>`__). Deux
profondeurs de recherche y sont réglables, nommées et conservées
séparément — abaisser l'une ne modifie jamais l'autre :

* **Profondeur d'affichage** — le confort interactif pendant l'édition du
  plateau ; jamais écrite en base.
* **Profondeur d'analyse** — ce que le lot d'analyse après import écrit dans
  l'Analyse d'une position.

Les deux valent par défaut **2-ply**, la configuration canonique. Le menu
marque d'un ⚠ les profondeurs 3-ply et 4-ply, dont le coût par position
dépasse de loin celui de 2-ply. L'onglet
propose aussi l'**élagage** (par défaut ``k=12``) et le **nombre de coups
candidats affichés** (par défaut 10), ainsi qu'une case **analyser
automatiquement après import** qui, une fois activée, vérifie après chaque
import s'il reste des positions **sans aucune analyse** (ni gammonNet, ni XG,
ni GNUbg, ni BGBlitz — la règle est « une évaluation ne comble qu'un trou »,
jamais un remplacement) et, le cas échéant, lance en tâche de fond une analyse
gammonNet à la profondeur d'analyse configurée. La ligne **Positions sans
analyse** de l'onglet *Maintenance* relance manuellement le même rattrapage,
utile pour une bibliothèque constituée avant l'existence de cette
fonctionnalité.

Une prise ou un refus est évalué par gammonNet sur la décision du doubleur,
avant le double. Un verdict gammonNet écrit sur une telle réponse par une
version antérieure de blunderDB jugeait le redouble du receveur : il est
supprimé à l'ouverture de la bibliothèque, ou à l'import d'une base qui le
porte. La prise ou le refus redevient une position sans analyse, absente des
statistiques jusqu'à ce que la ligne *Positions sans analyse* de l'onglet
*Maintenance* (ou l'analyse automatique après import) la réévalue.

La ligne **Analyses gammonNet périmées** de l'onglet *Maintenance* couvre le cas
inverse : une position déjà analysée par gammonNet, mais dont l'analyse
stockée a été écrite par une version de moteur plus ancienne que celle en
cours d'exécution, ou à une profondeur différente de la profondeur d'analyse
configurée ci-dessus, y est signalée comme périmée et réévaluée. Une position
portant en plus une analyse XG, GNUbg ou BGBlitz n'est jamais touchée par ce
lot, quel que soit son contenu gammonNet — la protection
d'`ADR-0013 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md>`__
reste inconditionnelle. Le compte des positions périmées lit chaque analyse
gammonNet de la base : il attend le bouton **Compter**. Le lot recalcule sa
propre liste au moment de démarrer.

Les deux lots sont **bornés, visibles et annulables, jamais un démon
silencieux** : leur profondeur, leur progression (``positions analysées / total``,
dès ``0`` au lancement) et un
bouton d'annulation apparaissent dans la barre de statut pendant toute leur
durée, et disparaissent une fois terminés au profit d'un message résumant le
résultat — combien de positions ont été **analysées**, combien ont été
**refusées** (une position que gammonNet décline d'évaluer, comme un score de
match hors de portée de sa table de match, ce qui n'est jamais une panne) et
combien ont **échoué** (retentées, inchangées, au prochain lancement).
Fermer l'application pendant l'un ou l'autre ne perd rien : chaque position
analysée est écrite au fil de l'eau, et un prochain lancement reprend
exactement là où l'analyse s'était arrêtée, sans aucun journal à tenir.
L'annulation comme la fermeture interrompent aussi la position en cours de
calcul, qui reste sans analyse.

Un seul travail écrit la bibliothèque à la fois : un lot gammonNet, un rollout
stocké ou une opération de maintenance. Un lot ou une opération lancés depuis
l'onglet *Maintenance* sont refusés tant qu'un autre travail est en cours, et la barre de
statut nomme ce travail ; le lot que lance un import, une transcription ou un
duel attend au contraire qu'il finisse, et la barre de statut l'annonce
« en attente » jusqu'à son départ.

.. _rollout_reglage:

Le même onglet porte le **réglage des rollouts** lancés depuis le panneau
Analyse (voir :ref:`rollouts`), conservé d'une session à l'autre : **Rapide**
(216 parties, tronquées à 7 demi-coups), **Standard** (1296 parties, tronquées
à 11 demi-coups) ou **Libre**, où tous les paramètres s'éditent — troncature,
parties minimum et maximum (multiples de 36), limite de JSD, profondeur (ply),
nombre de candidats, graine et nombre de processus.

**Table d'équité de match de la base.** Au bas de l'onglet, la liste
**Table d'équité de match (MET) de la base** choisit la table avec laquelle
gammonNet valorise les scores de match : Kazaross-XG2, intégrée, par défaut,
ou une table importée d'un fichier ``.xml`` de GNUbg par le bouton
**Importer une MET .xml…**. Le choix appartient à la base, pas à
l'application : ouvrir une autre base, c'est retrouver sa table. Seules les
tables explicites sont lues (pas les tables paramétriques ``zadeh`` ou
``mec``) ; au-delà de la longueur d'une table importée, la table intégrée
prend le relais. Une table s'identifie par ses valeurs, non par son nom :
importer le ``Kazaross-XG2.xml`` de GNUbg n'ajoute rien, c'est la table
intégrée. Chaque analyse calculée par gammonNet enregistre la table avec
laquelle elle l'a été ; les analyses importées (XG, GNUbg, BGBlitz) sont
réputées calculées avec Kazaross-XG2. Une analyse à un score de match
calculée avec une autre table que la table courante est marquée
**MET différente** dans le panneau d'analyse et sortie des statistiques
(moyennes d'erreur, PR, classements, face-à-face) ; une analyse en money game
ne l'est jamais. Changer de table ne réécrit aucune analyse : seul change ce
que les comparaisons retiennent.

L'évaluation en direct du panneau et la grille de videau sont valorisées avec
la table de la base, comme les analyses qu'elle enregistre. Une table voyage
avec les analyses qui la citent : l'export d'une base l'emporte, et l'import
d'une base l'ajoute à la base qui reçoit, sans doublon (une table déjà présente
est reconnue à ses valeurs) et sans la rendre courante. Une analyse gammonNet
remplacée par celle d'un autre moteur (XG, GNUbg) redevient réputée calculée
avec Kazaross-XG2.

**Un match importé sans analyse obtient ainsi un PR.** C'est le cas d'un match
joué en ligne, ou d'un fichier Jellyfish ``.mat``, que personne n'a fait
passer par XG : blunderDB en connaissait les positions et les coups joués,
mais aucune analyse ne disait ce qu'ils valaient. Une fois le lot passé, le
coup effectivement joué est comparé au classement de gammonNet et l'écart
alimente le PR, le taux d'erreur, les pires décisions et tous les autres
indicateurs, exactement comme un match analysé par XG. La comparaison ne
s'invente rien : le coup joué vient de la table des coups du match, écrite à
l'import, que le fichier ait porté une analyse ou non.

Une base analysée avec une version antérieure à celle-ci n'a pas besoin d'être
réévaluée : ``blunderdb repair`` recalcule les colonnes à partir des analyses
et des coups déjà en base et rend leur PR à ces matchs (voir
:ref:`repair <cli_repair>`).

Une réserve honnête : une position est identifiée par sa structure, donc une
position rencontrée deux fois — bien jouée une fois, mal l'autre — ne porte
qu'un seul écart, celui de sa première occurrence enregistrée. Ce n'est pas
propre à ce calcul : une bibliothèque XG a exactement la même forme.

.. _dossier_surveille:

Dossier surveillé
~~~~~~~~~~~~~~~~~

L'onglet **Dossier surveillé** demande à blunderDB de regarder un dossier
pendant qu'il tourne et d'importer chaque fichier de match qui y **apparaît**.
Jouer une session dans eXtreme Gammon, revenir à blunderDB, et trouver les
matchs déjà là.

Rien n'est deviné. Tant qu'aucun dossier n'est désigné, il n'y a pas de
surveillance : blunderDB ne se met pas à lire un répertoire parce qu'il a
supposé où vivent vos matchs. Le bouton **Proposer** cherche les emplacements
habituels sur cette machine et n'en propose un que s'il existe réellement ;
sinon il le dit, et c'est à vous de désigner le dossier.

Trois points méritent d'être connus avant d'activer la case :

* **Seuls les fichiers qui apparaissent sont importés.** Ce que le dossier
  contient déjà au moment où la surveillance démarre est enregistré comme
  connu et laissé tranquille : pointer une surveillance sur quatre ans de
  matchs ne doit pas les importer tous. Pour importer ce qui est là,
  utilisez l'import de dossier, qui existe pour cela — et les deux se
  composent très bien, l'import d'abord, la surveillance ensuite.
* **Un fichier n'est importé qu'une fois sa taille stable.** Un match qu'un
  autre programme est en train d'écrire grossit d'un coup d'œil à l'autre ;
  l'importer à moitié écrit donnerait une erreur d'analyse syntaxique sur
  laquelle personne ne peut agir. blunderDB attend donc de voir deux fois le
  même fichier inchangé.
* **L'import est silencieux.** Vous étiez en train d'étudier une position
  quand vos matchs sont arrivés : vous reprendre l'écran serait le pire
  moment. Le mode, la recherche active, l'onglet et la position affichée ne
  bougent pas ; la liste des positions n'est pas rechargée et montre les
  nouveaux matchs au prochain rechargement. L'import se fait sans fenêtre,
  et la barre d'état affiche un
  bandeau donnant le compte des matchs importés, ignorés (doublons) et en
  échec, avec un bouton qui ouvre le compte rendu complet si vous le
  souhaitez. Tout le reste est identique à un import manuel : mêmes doublons
  détectés, même lot d'import, même analyse automatique si elle est activée.

L'intervalle par défaut est de dix secondes ; le plancher est de deux. Le
dossier n'est pas parcouru récursivement : un dossier surveillé est l'endroit
où un outil dépose ses matchs, pas une arborescence à explorer. Un partage
réseau démonté n'arrête pas la surveillance et ne fait pas non plus passer son
contenu pour nouveau à son retour.

La même surveillance existe en ligne de commande, avec
``blunderdb import --type batch --dir <dossier> --watch`` (voir :ref:`cli`) :
c'est la forme qu'un serveur, une tâche planifiée ou un script peuvent
utiliser.

.. _synchroniser_deux_machines:

Synchroniser deux machines
~~~~~~~~~~~~~~~~~~~~~~~~~~

Copier sa base ``.db`` d'une machine à l'autre, ou la ranger dans un dossier
synchronisé (Syncthing, Dropbox…), finit par l'abîmer : SQLite écrit à côté de
la base des fichiers de journal qu'une synchronisation de fichiers recopie dans
le désordre. blunderDB synchronise plutôt deux bibliothèques par **exports
incrémentaux** déposés dans un dossier que l'autre machine **surveille**. Le
transport est à votre choix — Syncthing, clé USB, courriel : il n'y a ni
serveur ni compte.

* Sur la machine A, dans la fenêtre d'export, choisissez *Toute la
  bibliothèque*, cochez *Seulement ce qui est nouveau depuis le dernier export
  vers ce dossier* et désignez le dossier avec **Choisir le dossier…**. Chaque
  export y écrit un fichier ``sync-<machine>-<date>.db`` qui ne contient que ce
  qui est entré dans la bibliothèque depuis l'export précédent vers ce même
  dossier ; le premier contient tout.
* Sur la machine B, faites de ce dossier le :ref:`dossier surveillé
  <dossier_surveille>`. Chaque fichier ``.db`` qui y apparaît est fusionné sans
  aperçu, comme par l'import d'une base : une position déjà connue n'est pas
  dupliquée, un match déjà connu est reconnu, les collections et les leçons se
  fusionnent par leur nom.
* Faites de même dans l'autre sens, avec un second dossier.

Ce qu'il faut savoir :

* **Le repère « dernière synchronisation » est chez celui qui exporte**, un par
  dossier de destination, et n'avance que lorsque le fichier est complet. La
  machine qui reçoit n'enregistre rien. Un export qui échoue laisse le repère en
  place : le suivant emporte les mêmes lignes.
* **La synchronisation se fait par ajout seul.** Une suppression ne se propage
  pas : supprimer un match sur A ne le supprime pas sur B. Une modification en
  place non plus : un commentaire réécrit, un match renommé, une analyse refaite
  sur une position déjà envoyée. Partent les positions, matchs, tournois,
  collections, leçons et commentaires entrés depuis le dernier export ; une
  collection repart aussi quand une position y a été ajoutée, une position quand
  un commentaire y a été ajouté. Un commentaire réécrit ne repart pas, car la
  machine qui reçoit ne remplace jamais un texte qu'elle a déjà : pour faire
  passer une correction, ajoutez-la comme nouveau commentaire. Les filtres enregistrés et les
  paquets Anki n'en font pas partie : un export de *Toute la bibliothèque* sans
  la case les emporte.
* **Ce qu'une machine a reçu repart une fois vers l'autre**, puisque c'est
  nouveau dans sa bibliothèque ; l'autre le reconnaît sans rien ajouter, et
  l'échange suivant est vide.
* **Un fichier arrivé pendant que blunderDB était fermé n'est pas perdu** : au
  démarrage de la surveillance, chaque ``.db`` déjà présent dans le dossier est
  fusionné, contrairement aux fichiers de match. La machine qui reçoit ne note
  pas ce qu'elle a lu ; elle refusionne donc, à chaque démarrage, tous les
  fichiers que le dossier contient encore, sans rien ajouter, mais au prix du
  temps de les relire. Supprimez de temps en temps les fichiers ``sync-…``
  déjà reçus pour que ce démarrage reste rapide.
* **Un export incrémental n'écrase jamais un fichier.** Le fichier précédent
  porte peut-être des lignes que l'autre machine n'a pas encore reçues : un
  export vers un nom de fichier déjà pris est refusé. Désignez le dossier, et
  chaque export y prend un nom neuf.
* **Un fichier réécrit sous un nom déjà vu est fusionné de nouveau** : la
  surveillance reconnaît un fichier à son nom, sa taille et sa date de
  modification.
* **La bibliothèque ouverte n'est jamais fusionnée dans elle-même** : si elle
  se trouve dans le dossier surveillé, la surveillance la passe.
* **Un export qui a dû écarter des lignes** (signalées dans le journal) ne fait
  pas avancer le repère : le suivant les retente, avec tout ce qui les suit.
* **Un export n'est jamais vu à moitié écrit** : il est écrit sous un nom caché
  puis renommé d'un coup. Un fichier protégé par mot de passe (``.dbx``) n'est
  pas pris par la surveillance, qui ne peut pas demander le mot de passe.

En ligne de commande, ``blunderdb export --type database --incremental --file
<dossier>`` d'un côté et ``blunderdb import --type batch --dir <dossier>
--watch`` de l'autre (voir :ref:`cli`). Le serveur accepte ``"incremental":
true`` sur ``/v1/exports.sqlite`` (voir :ref:`headless`).

.. _assistant_mcp:

Assistant et MCP
~~~~~~~~~~~~~~~~

L'onglet **Assistant et MCP** règle deux choses, toutes deux désactivées par
défaut.

Le **serveur MCP local** offre les outils de blunderDB (recherche, lecture
d'une position et de son analyse, statistiques d'un joueur, quiz…) à un
assistant qui parle le Model Context Protocol, comme Claude Desktop ou Claude
Code, tant que la fenêtre est ouverte. Il n'écoute que sur cette machine, à
l'adresse ``http://127.0.0.1:<port>/mcp`` (port 8765 par défaut), et refuse une
requête venue d'une page web. Il ajoute deux outils d'affichage : ouvrir une
vue sur une recherche et montrer une position. Les outils ne font que lire,
sauf si **Autoriser l'écriture** est cochée : ils peuvent alors enregistrer une
position, la commenter, créer et remplir une collection, noter une carte Anki
et enregistrer un rollout à côté de l'analyse d'une position ; rien n'efface. Sans
fenêtre ouverte, ``blunderdb mcp`` sert les mêmes outils (voir :ref:`cli`).

L'**assistant interne** est un client de ces mêmes outils. Aucun modèle n'est
embarqué : il s'adresse à un fournisseur compatible OpenAI — Ollama sur cette
machine par défaut, ou Groq, OpenRouter, Gemini, Anthropic, ou une autre
adresse. Est distante toute adresse hors de cette machine, quel que soit le
fournisseur choisi : vos phrases et les résultats des outils la quittent alors,
l'onglet le dit et attend votre accord pour cette adresse précise ; une autre
adresse le redemande. La clé d'API va dans le trousseau du système, jamais dans
la base ni dans le fichier de réglages. Le modèle proposé par défaut pour
Ollama, ``qwen2.5:7b``, est un point de départ, pas une recommandation : aucune
recommandation de modèle n'est faite sans un score du banc de mesure livré avec
le code source.

Une fois activé, l'assistant est un sous-onglet **Assistant** du panneau
Recherche. Une phrase — « mes erreurs de plus de 50 millipoints dans la
course » — ouvre une nouvelle vue nommée d'après elle, y lance la recherche et
y bascule ; ses jetons restent dans l'historique des recherches. Ce que les
outils renvoient vient de la base ; ce que le modèle écrit est marqué **Texte
libre du modèle**. L'assistant ne propose de modification que si la case
**Laisser l'assistant proposer des modifications** est cochée — un réglage
distinct de l'écriture du serveur MCP local — et chaque modification qu'il
prépare s'affiche et n'est faite qu'après **Confirmer**.

La fenêtre de configuration regroupe également des réglages d'affichage de
l'interface. Un curseur d'**échelle de l'interface** permet d'agrandir ou de
réduire l'ensemble des éléments, ce qui est utile sur les écrans à haute
densité ou pour améliorer la lisibilité. Les raccourcis CTRL-+, CTRL-- et CTRL-0 règlent la même échelle
comme dans un navigateur. Un menu **position des panneaux**
détermine l'emplacement des panneaux (recherche, matchs, analyse) par rapport
au plateau : *en bas*, *sur le côté* ou *automatique* (le côté est alors choisi
sur les écrans larges afin de mieux exploiter l'espace disponible). Comme les
autres réglages, ces choix sont conservés d'une session à l'autre.

.. _visites_guidees:

Visites guidées et base d'exemple
---------------------------------

Pour faciliter la prise en main, blunderDB propose des **visites guidées** de
l'interface. Le catalogue des visites s'ouvre depuis la barre d'outils ou avec
la commande ``tutorial`` (alias ``tour``). Sept visites sont disponibles : un
tour général de l'interface, et des visites dédiées à la recherche de positions,
à la revue des matchs, à la revue des tournois, au panneau Eval, à la révision
Anki et aux statistiques. Chaque visite met en évidence les éléments concernés
de l'interface, étape par étape, ouvre au passage le panneau dont elle parle, et
peut être rejouée à tout moment. Au premier démarrage, le tour général est
proposé automatiquement.

La commande ``demo`` charge une **base d'exemple** permettant de découvrir les
fonctionnalités de l'outil sans importer ses propres parties : trois matchs
(dont deux regroupés dans un tournoi) analysés par eXtreme Gammon, BGBlitz et
gammonNet, trois collections thématiques, des commentaires étiquetés
(``#blunder``, ``#cube``) et un paquet Anki avec son journal de révisions. Les
joueurs, le tournoi et le lieu sont fictifs. Les visites guidées s'appuient sur
cette base lorsqu'aucune base n'est ouverte.

.. _navigation_positions:

Navigation dans les positions
-----------------------------

Par défaut, blunderDB permet de:

* faire défiler les différentes positions de la bibliothèque courante — qui
  n'est jamais chargée d'un bloc : blunderDB n'en tient que la liste des
  identifiants et charge les positions par fenêtres de cinquante autour de
  celle qui est affichée, si bien qu'une base de plusieurs dizaines de milliers
  de positions s'ouvre aussi vite qu'une petite,

* afficher les informations d'analyse associées à une position,

* afficher, ajouter et modifier les commentaires d'une position.

Le bouton **Aller à la position** de la barre d'outils ouvre une fenêtre où
saisir directement l'indice d'une position pour y sauter, sans avoir à
défiler. C'est l'équivalent graphique de la commande ``[number]`` en ligne de
commande (voir :ref:`cmd_positions`).

.. tip:: Se référer à :ref:`raccourcis` pour les raccourcis disponibles.

.. _edition_positions:

Édition de positions
--------------------

L'appui sur la touche *TAB* ouvre le panneau de recherche et permet
d'éditer une position sur le plateau pour l'ajouter à la base de données
ou pour définir une structure de position à rechercher.
La distribution des pions, du videau, du score, et du trait peuvent être
modifiés à l'aide de la souris (voir :ref:`guide_edit_position`).

.. tip:: Se référer à :ref:`raccourcis` pour les raccourcis disponibles.

.. _ligne_commande:

La ligne de commande
--------------------

La ligne de commande, intégrée dans la barre d'état, permet de réaliser
l'ensemble des fonctionalités de blunderDB disponibles à l'interface
graphique: opérations générales sur la base de données, navigation de
position, affichage de l'analyse et/ou des commentaires, recherche de
positions selon des filtres... Après une première prise en main de
l'interface, il est recommandé de progressivement utiliser la ligne de
commande qui permet une utilisation puissante et fluide de blunderDB,
notamment pour les fonctionnalités de recherche de positions.

Pour ouvrir la ligne de commande, appuyer sur
la touche *ESPACE*. Pour envoyer une requête et fermer la ligne de
commande, appuyer sur la touche *ENTREE*.

blunderDB exécute les requêtes envoyées par l'utilisateur sous réserve
qu'elles soient valides et modifie immédiatement l'état de la base de données
le cas échéant. Il n'y a pas d'actions de sauvegarde explicite de la part
de l'utilisateur.

.. tip:: Se référer à la :ref:`liste des commandes <cmd_mode>` pour la liste de commandes
   disponible en ligne de commande.

.. _palette_commandes:

La palette de commandes
-----------------------

La palette de commandes (*CTRL-MAJ-P*) retrouve par un nom approché ce que
l'on ne sait plus où chercher : une commande de la ligne de commande, un
onglet, un filtre de la bibliothèque ou un match. Les lettres tapées doivent
apparaître dans l'ordre, pas forcément côte à côte, et sans égard aux
majuscules ni aux accents : « mtrc » trouve la matrice du videau, « lyon » les
matchs d'un tournoi de Lyon.

Les flèches choisissent, *ENTREE* lance, *ECHAP* referme. Une commande se lance
comme si elle avait été tapée ; ``s`` et ``ss`` ouvrent la ligne de commande
pour y écrire les filtres ; un filtre se lance comme d'un double-clic dans la
bibliothèque ; un match s'ouvre comme d'un double-clic dans le panneau Matchs.

Quand une Direction est ouverte, la palette y ajoute le tournoi : joueurs, tables,
matchs en cours et épreuves (voir :ref:`la recherche rapide <direction_recherche>`).

