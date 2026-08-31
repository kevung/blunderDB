.. _manuel:

Manuel
======

Introduction
------------

blunderDB est un logiciel pour constituer des bases de données de
positions de backgammon. Sa force principale est de fournir un lieu unique
pour agréger les positions qu'un joueur a rencontrées (en ligne, en tournoi)
et de pouvoir les réétudier en les filtrant selon divers filtres arbitrairement
combinables. blunderDB peut également être utilisé pour créer des catalogues
de positions de référence.

Les positions sont stockées dans une base de données représentée par un fichier
*.db*.

Interactions principales
------------------------

Les principales interactions possibles avec blunderDB sont:

* ajouter une nouvelle position,

* modifier une position existante,

* copier l'image du board dans le presse-papier (PNG) via **Ctrl+X**, ou avec l'analyse complète via **Ctrl+X Ctrl+X**,

* supprimer une position existante,

* rechercher une ou plusieurs positions,

* importer des matchs depuis différentes sources (XG, GNUbg, BGBlitz, Jellyfish), y compris les commentaires depuis les fichiers XG,

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

* calculer l'EPC (Effective Pip Count) d'une position de bearoff (panneau Eval),

* étudier les positions par répétition espacée (panneau Anki),

* afficher les métadonnées de la base de données (panneau métadonnées).

Des fenêtres modales peuvent s'afficher pour:

* afficher l'aide de blunderDB,

* afficher le catalogue des visites guidées (voir :ref:`visites_guidees`),

* paramétrer l'export de la base de données,

* configurer blunderDB, notamment la langue de l'interface (voir
  :ref:`configuration`).

La zone d'affichage principale met à disposition à l'utilisateur:

* un board afin d'afficher ou d'éditer une position de backgammon,

* le niveau et le propriétaire du cube,

* le compte de course de chaque joueur,

* le score de chaque joueur,

* les dés à jouer. Si aucune valeur n'est affichée sur les dés, la
  position des dés indique quel joueur a le trait et que la position est
  une décision de cube. Lorsque la décision de cube est une réponse à un
  doublement (prise/passe), le videau proposé est affiché au centre du
  plateau, à la valeur offerte.

La barre d'état est structurée de gauche à droite par les informations
suivantes:

* la ligne de commande, accessible en appuyant sur la touche *ESPACE*,

* un message d'information lié à une opération réalisée par l'utilisateur,

* l'index de la position courante, suivi du nombre de positions dans la
  bibliothèque courante (ou les informations de coup/partie lors de la
  navigation dans un match).

.. note:: Dans le cas de positions issues d'une recherche par l'utilisateur, le
   nombre de positions indiqué dans la barre d'état correspond au nombre de
   positions filtrées.

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

* **Créer une vue** : cliquer sur le bouton *+* de la barre d'onglets ou
  appuyer sur *CTRL-T*. La nouvelle vue démarre comme une copie de la vue
  courante.

* **Fermer une vue** : cliquer sur la croix de l'onglet ou appuyer sur
  *CTRL-W*. La dernière vue ne peut pas être fermée.

* **Changer de vue** : cliquer sur un onglet, appuyer sur *CTRL-PageUp* /
  *CTRL-PageDown* (ou *MAJ-J* / *MAJ-K*) pour passer à la vue précédente /
  suivante, ou *CTRL-1* à *CTRL-9* pour atteindre directement la n-ième vue.

* **Renommer une vue** : double-cliquer sur l'onglet, saisir le nouveau nom
  et valider avec *ENTREE*.

Les vues sont enregistrées avec l'état de session de la base de données et
restaurées à sa réouverture.

.. _configuration:

Configuration
-------------

Le bouton de configuration (icône en forme de rouage) situé dans la barre
d'outils, à gauche du bouton d'aide, ouvre la fenêtre de configuration de
blunderDB. Elle est organisée en cinq onglets :

* **Interface** — langue, échelle d'affichage, position du panneau ;
* **Couleurs** — les couleurs du plateau ;
* **Bearoff** — la base de sortie two-sided étendue utilisée par le panneau
  Bearoff ;
* **gammonNet** — les réglages de l'évaluateur embarqué, décrits ci-dessous ;
* **Identité d'émetteur** — la clé qui signe vos filigranes, décrite à la
  section :ref:`diffusion_controlee`.

L'onglet *Interface* permet de choisir la langue parmi l'anglais, le
français, l'allemand, l'italien, l'espagnol, le finnois, le japonais, le grec
et le russe. L'ensemble de l'interface (barre d'outils, panneaux, messages,
aide) est traduit dans la langue sélectionnée. Le choix de la langue est
enregistré et conservé d'une session à l'autre.

Le même onglet propose aussi le bouton **Compacter la base**, qui récupère
l'espace disque laissé par les suppressions (matchs, tournois, purges) : la
base de données ne rétrécit jamais toute seule quand on supprime des données,
il faut demander explicitement ce compactage. L'opération peut prendre du
temps sur une grosse base et nécessite, temporairement, environ deux fois sa
taille en espace disque libre (blunderDB refuse de démarrer plutôt que de
risquer un compactage interrompu) ; une confirmation est donc demandée avant
de lancer l'opération. Le résultat — l'espace gagné, en mégaoctets — s'affiche
ensuite dans la barre d'état. La même opération est disponible en ligne de
commande via ``blunderdb vacuum`` (voir :ref:`cli`).

L'onglet *Couleurs* permet de personnaliser les couleurs du
plateau. Chaque élément dispose de son propre sélecteur de couleur : le fond,
la bordure, les flèches claires et foncées, les pions du joueur 1 et du joueur
2, les dés, les points des dés et le videau. Le bouton *Réinitialiser* rétablit
l'ensemble des couleurs par défaut. Comme la langue, les couleurs choisies sont
conservées d'une session à l'autre.

L'onglet *Bearoff* gère la base two-sided qui étend le domaine exact du
panneau Eval (voir :ref:`panneau_epc`) au-delà de la base TS-06-06
embarquée dans l'exécutable. Il affiche le domaine actuellement actif
(``TS-06-06`` ou ``TS-06-11`` une fois téléchargée) et son origine, propose de
lancer le téléchargement de la base étendue TS-06-11 avec une barre de
progression, et **reprend automatiquement un téléchargement interrompu** au
lieu de repartir de zéro (requêtes HTTP par plage). Une fois téléchargée, la
base peut être supprimée — une confirmation est demandée avant toute
suppression, la taille du fichier étant rappelée dans le message. L'onglet
permet aussi de pointer vers un fichier ``.bd`` two-sided externe (par exemple
une base générée soi-même) plutôt que d'utiliser le téléchargement intégré.

L'onglet **gammonNet** règle l'évaluateur embarqué (voir ADR-0011). Deux
profondeurs de recherche y sont réglables, nommées et conservées
séparément — abaisser l'une ne modifie jamais l'autre :

* **Profondeur d'affichage** — le confort interactif pendant l'édition du
  plateau ; jamais écrite en base.
* **Profondeur d'analyse** — ce que le lot d'analyse après import écrit dans
  l'Analyse d'une position.

Les deux valent par défaut **2-ply**, la configuration canonique. L'onglet
propose aussi l'**élagage** (par défaut ``k=12``) et le **nombre de coups
candidats affichés** (par défaut 10), ainsi qu'une case **analyser
automatiquement après import** qui, une fois activée, vérifie après chaque
import s'il reste des positions **sans aucune analyse** (ni gammonNet, ni XG,
ni GNUbg, ni BGBlitz — la règle est « une évaluation ne comble qu'un trou »,
jamais un remplacement) et, le cas échéant, lance en tâche de fond une analyse
gammonNet à la profondeur d'analyse configurée. Le travail est **borné,
visible et annulable, jamais un démon silencieux** : sa progression
(``positions analysées / total``) et un bouton d'annulation apparaissent dans
la barre de statut pendant toute sa durée, et disparaissent une fois terminé.
Fermer l'application pendant l'analyse ne perd rien : chaque position
analysée est écrite au fil de l'eau, et un prochain import concerné reprend
exactement là où l'analyse s'était arrêtée, sans aucun journal à tenir.

La fenêtre de configuration regroupe également des réglages d'affichage de
l'interface. Un curseur d'**échelle de l'interface** permet d'agrandir ou de
réduire l'ensemble des éléments, ce qui est utile sur les écrans à haute
densité ou pour améliorer la lisibilité. Un menu **position des panneaux**
détermine l'emplacement des panneaux (recherche, matchs, analyse) par rapport
au plateau : *en bas*, *sur le côté* ou *automatique* (le côté est alors choisi
sur les écrans larges afin de mieux exploiter l'espace disponible). Comme les
autres réglages, ces choix sont conservés d'une session à l'autre.

.. _visites_guidees:

Visites guidées et base d'exemple
---------------------------------

Pour faciliter la prise en main, blunderDB propose des **visites guidées** de
l'interface. Le catalogue des visites s'ouvre depuis la barre d'outils ou avec
la commande ``tour`` (alias ``tutorial``). Quatre visites sont disponibles : un
tour général de l'interface, et des visites dédiées à la recherche de positions,
à la revue des matchs et à la revue des tournois. Chaque visite met en évidence
les éléments concernés de l'interface, étape par étape, et peut être rejouée à
tout moment. Au premier démarrage, le tour général est proposé automatiquement.

La commande ``demo`` charge une **base d'exemple** (matchs, tournoi et analyses)
permettant de découvrir les fonctionnalités de l'outil sans importer ses propres
parties. Les visites guidées s'appuient sur cette base lorsqu'aucune base n'est
ouverte.

.. _navigation_positions:

Navigation dans les positions
-----------------------------

Par défaut, blunderDB permet de:

* faire défiler les différentes positions de la bibliothèque courante,

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

.. tip:: Se référer à la :numref:`cmd_mode` pour la liste de commandes
   disponible en ligne de commande.

.. _panneau_analyse:

Panneau Analyse
---------------

Le panneau **Analyse** (*CTRL-L*) affiche les données d'analyse de la position
courante importées depuis eXtreme Gammon (XG), GNUbg ou BGBlitz. Il présente
les meilleures alternatives (coups de pions ou décisions de videau) avec leurs
valeurs d'équité et les erreurs correspondantes. La touche *d* bascule entre
l'analyse des coups de pions et l'analyse du cube. Lors de la navigation dans
un match, le coup effectivement joué est mis en évidence dans la liste des
alternatives. Appuyer sur *CTRL-L* ou exécuter la commande ``list`` pour
afficher ou masquer le panneau.

.. _panneau_commentaires:

Panneau Commentaires
--------------------

Le panneau **Commentaires** (*CTRL-P*) affiche, ajoute et modifie les
commentaires associés à la position courante. Les commentaires importés depuis
les fichiers XG sont automatiquement associés aux positions correspondantes.
Appuyer sur *CTRL-P* ou exécuter la commande ``comment`` pour afficher ou
masquer le panneau.

.. _panneau_recherche:

Panneau Recherche
-----------------

Le panneau **Recherche** (*CTRL-F* ou *TAB*) permet de filtrer les positions
selon des critères combinables librement : structure de pions, type de décision
de videau, magnitude d'erreur, dates, tags, etc. La touche *TAB* ouvre
simultanément le panneau de recherche et l'éditeur de position, permettant de
définir une structure de pions à rechercher sur le plateau.

Pour affiner une recherche parmi les positions actuellement filtrées, utiliser
la commande ``ss`` suivie de filtres (ex: ``ss nc``, ``ss E>40``). Le panneau
de recherche propose également une case à cocher *Search in current results*
pour la même fonctionnalité.

Le panneau propose un contrôle explicite du **type de décision** recherché :
*Indifférent* (aucun filtre), *Pions* (décisions de coup) ou *Videau*
(décisions de cube). Lorsque *Videau* est sélectionné, une seconde liste précise
le sous-type : *Tous*, *Double / Pas de double* (le joueur au trait doit décider
de doubler) ou *Prise / Passe* (réponse à un doublement adverse). Le contrôle est
synchronisé avec le plateau : modifier les dés ou le videau sur le plateau met à
jour le type de décision, et inversement. En mode *Prise / Passe*, le videau est
affiché au centre du plateau à la valeur offerte ; cette valeur reste éditable.

Le filtre **Marquée** retient les positions que vous avez marquées (*flag*) dans
le logiciel d'origine du match. Seul eXtreme Gammon produit cette information,
enregistrée coup par coup dans le fichier ``.xg`` ; blunderDB la lit à l'import
et la conserve. Une décision de videau marquée donne deux positions marquées, le
double et la prise/passe, blunderDB scindant en deux ce que le fichier source
enregistre comme une seule décision.

.. note:: Le marquage n'est pas rétroactif : les matchs déjà présents dans la
   base ne portent pas cette information, puisqu'elle n'existe que dans les
   fichiers source. Il suffit de réimporter le fichier ``.xg`` concerné —
   l'import détecte le doublon et n'ajoute rien d'autre que les marques, sans
   toucher aux commentaires ni aux analyses existants. Le marquage ne peut ni
   être posé ni être retiré depuis blunderDB : pour une liste de travail
   temporaire, utilisez plutôt une collection.

Le filtre **Commentaire** interroge les commentaires attachés aux positions
selon trois modes exclusifs. *contient le texte* recherche un ou plusieurs mots
dans le texte des commentaires (champ de saisie, mots séparés par ``;``, au
moins un doit correspondre) ; *a un commentaire* retient toute position portant
un commentaire, quel qu'en soit le contenu ; *sans commentaire* retient au
contraire les positions non annotées — utile, combiné à un filtre d'erreur ou de
date, pour dresser la liste de ce qu'il reste à commenter.

.. note:: Les commentaires importés depuis un fichier de match (XG, GNUbg)
   comptent comme des commentaires : blunderDB ne conserve pas leur origine et ne
   peut donc pas distinguer une note que vous avez saisie d'une note reprise du
   fichier source. Par ailleurs, les commentaires attachés à un *match* ou à un
   *tournoi* ne sont pas concernés : ils annotent le match ou le tournoi, non ses
   positions.

Le filtre **Matchs & Tournois** s'appuie sur un sélecteur commun (fenêtre modale)
plutôt que sur la saisie d'identifiants numériques : deux listes à cocher, une
pour les matchs et une pour les tournois, chacune filtrable par texte (joueur,
date, événement pour les matchs ; nom, date, lieu pour les tournois), avec des
boutons *Tout* / *Aucun* qui n'agissent que sur le sous-ensemble actuellement
filtré. Cocher un tournoi coche automatiquement (et grise) ses matchs membres
dans la liste des matchs, rendant visible le fait qu'un tournoi équivaut à
l'ensemble de ses matchs.

Le panneau de recherche comporte trois onglets sur son bord gauche :
*Recherche* (les filtres), *Historique* et *Enregistrés*. L'onglet
**Historique** liste les recherches passées avec leur date et leur commande :
un clic sélectionne une recherche et affiche la position associée sur le
plateau, un double-clic la ré-exécute. Chaque entrée peut être enregistrée
dans la bibliothèque de filtres (icône signet, en donnant un nom au filtre) ou
supprimée. L'onglet **Enregistrés** contient la **bibliothèque de filtres** :
double-cliquer sur un filtre enregistré pour relancer la recherche
correspondante (voir :ref:`annexe_filtres`). La commande ``history`` (alias
``hi``) ouvre le panneau de recherche.

.. tip:: Se référer à la :numref:`cmd_mode` pour la liste des filtres
   disponibles.

.. _panneau_collections:

Panneau Collections
-------------------

Le panneau **Collections** (*CTRL-B*) permet de gérer des collections de
positions. Les collections peuvent être créées, renommées et supprimées. Des
positions peuvent y être ajoutées ou retirées (touche *Suppr*, confirmation
demandée). Double-cliquer sur une collection pour parcourir ses positions
avec les touches *GAUCHE* et *DROITE*. L'ordre des collections et des
positions au sein des collections peut être modifié par glisser-déposer.
Appuyer sur *CTRL-B* ou exécuter la commande ``collection`` pour afficher ou
masquer le panneau.

.. _panneau_matchs:

Panneau Matchs
--------------

Le panneau **Matchs** (*CTRL-Tab*) liste les matchs importés. Double-cliquer
sur un match (ou appuyer sur *ENTREE*) pour naviguer dans ses coups. La
commande ``m`` reprend la navigation dans le dernier match visité.

L'utilisateur peut:

* parcourir les coups d'un match en utilisant les touches *GAUCHE* et *DROITE*,

* passer d'une partie à l'autre à l'aide des touches *PageUp* et *PageDown*,

* afficher l'analyse des coups (pions et cube) en appuyant sur *CTRL-L*,

* basculer entre l'analyse des coups de pions et du cube avec la touche *d*,

* voir le coup effectivement joué mis en évidence dans l'analyse.

La dernière position visitée dans chaque match est mémorisée et restaurée
automatiquement. Appuyer sur *CTRL-Tab* ou exécuter la commande ``match``
pour afficher ou masquer le panneau.

Chaque match peut être exporté en transcription Jellyfish ``.mat`` via le
bouton ⬇ de la liste des matchs ou le bouton *.mat* de la fiche du match.

Le bouton **Fusionner les joueurs** de la barre d'outils du panneau ouvre une
fenêtre listant tous les noms de joueurs de la base avec leur nombre de
matchs : sélectionner les variantes d'orthographe d'un même joueur, choisir le
nom canonique à conserver, puis fusionner. Utile pour unifier les statistiques
par joueur lorsqu'un même joueur apparaît sous plusieurs noms.

Lorsqu'un match est ouvert, une **barre d'informations** apparaît au-dessus du
plateau : elle rappelle les joueurs en présence (*joueur 1* contre *joueur 2*)
ainsi que le contexte du match (événement, lieu, ronde, date et longueur du
match, lorsque ces informations sont disponibles). Cette barre s'affiche aussi
en dehors du mode match : lorsqu'une position étudiée (issue d'une recherche,
d'une collection ou d'un accès direct) provient d'un ou de plusieurs matchs,
elle en indique la **provenance** — le premier match concerné et, le cas
échéant, un badge « +N » listant les autres au survol. Une position importée
seule, qu'aucun match ne référence, n'affiche rien.

À l'ouverture d'une base contenant des matchs, le panneau **Matchs** est affiché
d'emblée et la revue débute directement sur la première position, afin de
commencer immédiatement la navigation.

.. note:: Une base de données ne peut être ouverte en écriture que par une seule
   fenêtre à la fois. Si vous ouvrez une base déjà ouverte dans une autre
   fenêtre de blunderDB, elle s'ouvre en **lecture seule** : la navigation, la
   recherche et l'analyse restent possibles, mais toute modification est
   désactivée et la barre de titre affiche « [lecture seule] ».

.. tip:: Se référer à :ref:`raccourcis` pour les raccourcis disponibles.

.. _panneau_tournois:

Panneau Tournois
----------------

Le panneau **Tournois** (*CTRL-Y*) permet de regrouper des matchs en tournois
pour un suivi organisé et une analyse statistique par événement. Les tournois
peuvent être créés, renommés et supprimés ; les matchs peuvent leur être
assignés. Les statistiques du panneau Stats peuvent être filtrées par tournoi.
Appuyer sur *CTRL-Y* pour afficher ou masquer le panneau.

La colonne **PR** de chaque tournoi affiche le PR du **joueur de référence** —
c'est-à-dire le joueur présent dans le plus grand nombre de matchs du tournoi
(en cas d'égalité, celui ayant pris le plus de décisions). Le PR ne mélange donc
pas votre jeu avec celui de vos adversaires : pour vos propres tournois, il
reflète votre performance seule. Le nom du joueur de référence apparaît en
infobulle au survol de la valeur.

.. _stats:

Panneau Stats
-------------

Introduction
~~~~~~~~~~~~~

Le panneau **Stats** permet d'analyser son niveau de jeu et de suivre sa
progression dans le temps à partir des positions importées dans la base de
données. Il calcule et affiche les indicateurs **PR** (Performance Rate) et
**MWC cost** (Match Winning Chance cost) pour l'ensemble des positions ou un
sous-ensemble filtré.

Le panneau Stats est particulièrement utile pour :

* **situer son niveau** par rapport aux seuils de référence (world-class,
  expert, avancé…) grâce au PR global ;

* **suivre sa progression** tournoi après tournoi ou match après match grâce
  aux graphiques de l'onglet Progression ;

* **identifier ses points faibles** : onglet Erreurs pour voir la répartition
  entre coups joués et décisions de videau, et la distribution des magnitudes
  d'erreur ;

* **comparer les joueurs de la base** entre eux, une ligne par joueur, grâce à
  l'onglet Joueurs — utile pour suivre une compétition entière ;

* **accéder directement aux positions concernées** en cliquant sur n'importe
  quel indicateur (drill-down).

Ouverture du panneau
~~~~~~~~~~~~~~~~~~~~~

Pour ouvrir le panneau Stats :

* Appuyer sur *CTRL-D*.
* Saisir la commande ``:stats`` ou ``:st`` dans la ligne de commande.

.. note::
   Le panneau se rafraîchit automatiquement à chaque modification du filtre.
   Il ne recalcule pas les statistiques lors d'un simple basculement PR ↔ MWC :
   les deux métriques sont calculées simultanément par le backend.

Barre de filtre
~~~~~~~~~~~~~~~

La barre de filtre, en haut du panneau, permet de restreindre le calcul à un
sous-ensemble de positions.

Perspective joueur
^^^^^^^^^^^^^^^^^^

La liste déroulante **Joueur** permet de filtrer les statistiques selon le
joueur analysé. blunderDB sélectionne automatiquement le joueur dont le nom
apparaît le plus souvent dans la base de données — modifiable à tout moment.

.. tip::
   Changer de joueur ne provoque pas de perte de données ; il suffit de
   re-sélectionner le joueur précédent dans la liste.

Filtres disponibles
^^^^^^^^^^^^^^^^^^^

* **Tournoi(s)** — restriction à un ou plusieurs tournois. Plusieurs tournois
  peuvent être sélectionnés simultanément.

* **Dates** — plage temporelle (*De* … *À*). Si seule la date de début est
  renseignée, les positions plus récentes sont incluses.

* **Type de décision** — Tous / Coups joués / Décisions de videau.

* **Longueur de match** — restriction à des longueurs de match précises (1, 3,
  5, 7, 9, 11, 13, 15, 21 points). Plusieurs longueurs peuvent être combinées.

Un bouton **Reset** remet tous les filtres à zéro (sauf le joueur
auto-détecté).

.. note::
   Les filtres sont persistés dans la configuration de blunderDB
   (``config.yaml``) et sont restaurés à la prochaine ouverture.

Toggle PR / MWC
~~~~~~~~~~~~~~~

Le bouton **PR / MWC** en haut du panneau bascule la métrique affichée dans
tous les onglets.

**PR (Performance Rate)**

  Mesure la qualité de jeu *money-game* : somme des erreurs en millièmes de
  point de backgammon, divisée par le nombre de décisions. Indépendant du
  score de match.

  Seuils de référence approximatifs :

  .. csv-table::
     :header: "Niveau", "PR"
     :widths: 20, 10
     :align: center

     "World-class", "< 3"
     "Expert", "3 – 5"
     "Avancé", "5 – 8"
     "Intermédiaire", "8 – 12"
     "Débutant", "> 12"

**MWC cost (Match Winning Chance cost)**

  Probabilité cumulée de victoire de match perdue à cause des erreurs, sur
  l'ensemble du jeu de données filtré. Calculé à partir de la MET
  Kazaross-XG2 embarquée dans blunderDB.

  .. caution::
     Le MWC cost **n'est pas applicable** aux positions *money-game* (sans
     enjeu de match). Ces positions sont exclues du calcul MWC.
     Les valeurs MWC dépendent de la MET utilisée ; elles ne sont pas
     directement comparables entre logiciels utilisant des METs différentes.

Le basculement PR ↔ MWC est instantané : aucun recalcul backend n'est
effectué.

Onglet Dashboard
~~~~~~~~~~~~~~~~

L'onglet **Dashboard** donne une vue synthétique des indicateurs clés.

Cartes de niveau
^^^^^^^^^^^^^^^^

Trois cartes affichent le PR (ou MWC) pour :

* **All** — toutes les décisions (coups + videau) ;
* **Checker** — coups joués seulement ;
* **Cube** — décisions de videau seulement.

Cliquer sur une carte charge dans le panneau d'analyse les positions du
sous-ensemble correspondant (drill-down).

.. note::
   Le nombre total de décisions est affiché en bas de chaque carte au survol.

PR glissant sur N dernières décisions
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Une ligne de valeurs PR (ou MWC) calculées sur les *N* dernières décisions
(N = 5, 10, 50, 100, 250, 500, 1000) permet de mesurer la tendance récente.
Les valeurs grisées correspondent à un N supérieur au nombre de décisions
disponibles.

Cliquer sur une valeur charge les *N* dernières positions correspondantes.

Top blunders
^^^^^^^^^^^^

La liste des 10 pires erreurs (ou MWC cost), triées par magnitude décroissante.
Cliquer sur une ligne charge la position concernée dans le panneau d'analyse.

Onglet Progression
~~~~~~~~~~~~~~~~~~

L'onglet **Progression** présente l'évolution du niveau dans le temps.

Courbe par tournoi
^^^^^^^^^^^^^^^^^^

Un graphique en ligne affiche le PR (ou MWC) pour chaque tournoi (axe X :
ordre des tournois, axe Y : valeur de la métrique). Des bandes de couleur
matérialisent les seuils de niveau.

Cliquer sur un point du graphique ouvre un menu contextuel avec deux options :

* **Open tournament** — ouvre le tournoi dans le panneau Tournois.
* **Open positions** — charge toutes les positions du tournoi dans le panneau
  d'analyse.

Scatter plot par match
^^^^^^^^^^^^^^^^^^^^^^

Un nuage de points représente chaque match (axe X : date, axe Y : PR ou MWC).
La taille du point est proportionnelle au nombre de décisions dans le match.

Cliquer sur un point ouvre un menu contextuel :

* **Open match** — ouvre le match dans le panneau des matchs.
* **Open positions** — charge toutes les positions du match dans le panneau
  d'analyse.

Onglet Erreurs
~~~~~~~~~~~~~~

L'onglet **Erreurs** décompose les sources d'erreurs.

Répartition par action de videau
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Un diagramme en barres affiche le PR (ou MWC) pour chaque type de décision
de videau : *NoDouble*, *DoubleTake*, *DoublePass*, *TooGood*. Chaque barre
indique également le nombre de décisions et le taux de blunders en infobulle.

Cliquer sur une barre charge les positions correspondant à cette action de
videau, **uniquement celles avec une erreur** (drill-down).

Direction des erreurs de videau
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

La répartition ci-dessus indique *combien* coûtent les décisions de videau ;
ce tableau indique dans *quel sens* elles se trompent.

Une position de videau porte deux décisions prises par deux joueurs
différents, présentées ici en deux lignes :

* **Offrir** — le joueur qui tient le videau double ou ne double pas. Ses
  erreurs sont les **doubles manqués** (il fallait doubler) et les **doubles
  prématurés** (il ne fallait pas).

* **Répondre** — le joueur à qui le videau est offert prend ou passe. Ses
  erreurs sont les **passes à tort** (une prise correcte a été passée) et les
  **prises à tort** (une passe correcte a été prise).

Les deux lignes restent séparées à dessein : un joueur peut parfaitement
doubler tard *et* prendre large, et un indicateur unique appellerait cela
« équilibré » en perdant les deux moitiés de l'information.

Chaque case affiche le nombre de décisions ; l'infobulle donne l'équité perdue
cumulée. Cliquer sur une case charge les positions correspondantes. Une case à
zéro n'est pas cliquable.

.. note::
   Ce tableau compte des décisions, il ne porte pas de jugement. À partir de
   quel écart une tendance mérite d'être nommée dépend de l'effectif et d'un
   point de référence, qui ne sont pas des données du moteur.

Répartition Checker / Cube
^^^^^^^^^^^^^^^^^^^^^^^^^^^

Un diagramme comparatif place côte à côte le PR des coups joués et des
décisions de videau. Cliquer sur une barre charge les positions du
sous-ensemble avec erreur.

Histogramme des magnitudes d'erreur
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Un histogramme distribue les erreurs selon leur magnitude en millièmes de
point (tranches : 0–5, 5–10, 10–25, 25–50, 50–100, ≥ 100). Cliquer sur
une barre charge les positions de la tranche.

Onglet Joueurs
~~~~~~~~~~~~~~

Les trois onglets précédents décrivent **un** joueur ; l'onglet **Joueurs** les
compare tous. Il affiche une ligne par joueur de la base, ce qui répond au
besoin d'un organisateur suivant une compétition entière plutôt qu'un joueur en
particulier.

Colonnes, dans l'ordre :

.. csv-table::
   :header: "Colonne", "Signification"
   :widths: 22, 78

   "Joueur", "Le nom **tel qu'il figure dans les matchs**. Un joueur enregistré sous deux orthographes apparaît donc sur deux lignes ; utilisez la fusion de joueurs pour les réunir."
   "Matchs", "Nombre de matchs disputés dans la période retenue."
   "V–D", "Victoires et défaites. Un match inachevé (journal tronqué, abandon) ne compte ni l'une ni l'autre : V + D peut donc être inférieur au nombre de matchs."
   "Décisions", "Nombre de décisions comptées — le dénominateur du PR. C'est la colonne qui dit ce que valent les taux voisins : un PR calculé sur douze décisions ne signifie rien."
   "PR", "Performance Rate global."
   "PR pions, PR videau", "Le PR ventilé par type de décision."
   "Snowie", "Snowie Error Rate (voir :ref:`stats_parity`)."
   "Blunders", "Nombre d'erreurs graves (au moins 0,100 EMG)."
   "Chance", "Chance moyenne par lancer, en millièmes de point, signée : positive si les dés ont été favorables."

Utilisation :

* **Trier** — cliquez sur un en-tête de colonne. Le tableau s'ouvre trié par PR
  croissant, meilleur joueur en tête. Les joueurs dont rien n'a été mesuré
  restent en bas quel que soit le sens du tri : un zéro faute de données n'est
  pas une performance parfaite.
* **Ouvrir le détail d'un joueur** — cliquez sur une ligne. Le joueur est
  sélectionné dans la barre de filtres et l'affichage bascule sur l'onglet
  Dashboard.
* **Restreindre la période** — les filtres de dates, de tournois et de longueur
  de match s'appliquent normalement, ce qui permet de borner le tableau aux
  dates d'une compétition.

.. note::
   Dans cet onglet, la liste **Joueur** et le choix du **type de décision** sont
   désactivés : le tableau montre tous les joueurs, et il ventile déjà les
   décisions de pions et de videau en colonnes distinctes.

.. important::
   Un tiret (« — ») signale une valeur **jamais mesurée**, à ne pas confondre
   avec zéro. C'est notamment le cas de la colonne Chance pour tout match
   importé avant la version 2.15.0 du schéma : la chance n'était alors pas
   conservée, et rien ne permet de la reconstituer après coup — il faut
   réimporter les fichiers source. Les formats qui ne la transportent pas (BGF,
   Jellyfish ``.mat``) n'en fourniront jamais.

Règle d'agrégation
~~~~~~~~~~~~~~~~~~

.. important::
   Le PR d'un tournoi (ou d'un sous-ensemble quelconque) est calculé par
   la règle **somme/somme** — jamais comme moyenne des PR individuels des
   matchs.

   Formule :

   .. math::

      PR_{tournoi} = \frac{\sum_{i} \text{erreur}_i}{\text{nombre total de décisions}}

   **Exemple :** un joueur dispute deux matchs dans un tournoi —

   * Match A : 10 décisions, erreur totale 50 mp → PR = 5,0
   * Match B : 90 décisions, erreur totale 270 mp → PR = 3,0

   Moyenne naïve des PR : (5,0 + 3,0) / 2 = **4,0** *(incorrect)*

   Règle somme/somme : (50 + 270) / (10 + 90) = 320 / 100 = **3,2** *(correct)*

   La règle somme/somme est la seule qui résiste à la variation de longueur
   des matchs (un match en 21 points pèse plus qu'un match en 1 point).

MWC : limitations
~~~~~~~~~~~~~~~~~

* Le MWC cost est calculé à partir de la **MET Kazaross-XG2**, table de
  référence de facto dans le backgammon compétitif. Les résultats ne sont
  pas directement comparables avec des logiciels utilisant d'autres METs.

* Les positions *money-game* (sans score de match) sont **exclues** du
  calcul MWC. Si votre base de données contient beaucoup de positions
  money-game, le MWC cost peut être sous-estimé ou indisponible.

* Le MWC cost est cumulatif sur l'ensemble du jeu de données filtré — pas
  un indicateur par décision. Il mesure l'impact total de vos erreurs sur
  vos chances de victoire.

.. _panneau_epc:

Panneau Eval
------------

Le panneau **Eval** (*CTRL-E*) calcule l'EPC (Effective Pip Count) d'une
position de bearoff et l'évaluation en direct de la position sur le plateau.
Il est activé en appuyant sur *CTRL-E*, en cliquant sur l'onglet Eval dans le
panneau inférieur, ou en exécutant la commande ``epc``.

Le panneau montre toujours la **seule décision** que la position posée sur
le plateau appelle — jamais deux à la fois — et les faits qui vont avec.
Chaque quantité se lit dans l'axe qui lui convient plutôt que dans un axe
unique imposé : la probabilité de gain, de gammon, de backgammon et
l'équité cubeless de chaque joueur, calculées *avant le jet*, se lisent
**par joueur** (bas, haut, puis Δ), à gauche de la décision de videau,
quand aucun dé n'est posé. Dès que des dés sont posés, ces mêmes valeurs
*avant le jet* changent d'axe : elles se lisent **au trait**, en tête de la
liste des coups candidats, sous forme d'une ligne italique *avant le jet* —
pas un coup candidat de plus, un repère contre lequel lire chaque coup.
L'écart entre cette ligne et un coup contient la chance du jet, jamais le
mérite du coup, et elle ne porte donc aucune colonne d'erreur. Sur une
position de bearoff pur, un second tableau, toujours **par joueur** et
toujours présent, dés posés ou non, porte l'EPC, le pip count, le wastage,
le nombre moyen de lancers et l'écart type ; ces cinq colonnes ne migrent
jamais. Le badge de régime, l'attribution du moteur (la profondeur de la
dernière évaluation y figure aussi) et la case *Défi* forment une bande à
part, alignée à droite au-dessus des tableaux.

Seule la liste des coups candidats défile — la ligne *avant le jet*, elle
aussi, reste épinglée au-dessus d'elle ; le reste du panneau (faits, badge,
décision de videau) reste toujours visible, sans réglage particulier de la
taille du panneau.

Le tableau de faits et la décision sont calculés par gammonNet, embarqué,
sans XG ni gnubg. Le calcul suit la position sans jamais figer l'interface :
une profondeur 0-ply s'affiche immédiatement à chaque geste, puis, après une
demi-seconde d'immobilité, une évaluation plus profonde (2 plis par défaut,
réglable dans l'onglet *gammonNet* de la configuration) la remplace en
arrière-plan — tout nouveau geste annule ce calcul de fond. La profondeur
affichée dans la bande de badges, ou au sein du badge de régime sur une
position de course, est toujours celle qui a effectivement produit le
chiffre montré, jamais celle demandée ; elle ne se répète pas sur chaque
ligne, puisqu'une évaluation en direct partage la même profondeur pour tous
les coups.
L'équité des coups candidats et de la décision de videau suit le score de
la position : en money game elle est exprimée en points, à un score de
match en **équité normalisée** — la même échelle que XG et GNU Backgammon,
où gagner la valeur du videau courant vaut +1 et la perdre −1 — jamais
mélangées dans un même tableau. Ce panneau ne modifie jamais la
base : c'est un calcul, pas une analyse enregistrée. Cliquer un coup
candidat l'affiche sur le plateau sous forme de flèches, exactement comme
dans le panneau Analyse. Le bouton **?** discret, dans la bande de
badges, mène au dépôt du moteur
`gammonNet <https://github.com/kevung/gammonNet>`_ ; l'attribution complète
(réseau Strehl, configuration gammonNet) figure dans les Remerciements de
l'aide.

L'utilisateur édite la position des pions sur l'ensemble du plateau,
exactement comme en mode édition : clic gauche place un pion du joueur du
bas, clic droit un pion du joueur du haut. Les cinq colonnes de course
n'apparaissent dans le tableau de faits que lorsque la position obtenue est
un bearoff pur (tous les pions des deux joueurs dans leur jan) ; sur toute
autre position, seules les quatre colonnes communes (gain, gammon,
backgammon, cubeless) répondent, et la décision porte sur les pions ou sur
un videau générique selon que des dés sont posés.

Dans le tableau de faits, chaque ligne — repérée par sa pastille de couleur,
le joueur noir étant toujours en bas — porte, tant qu'aucun dé n'est posé,
le gain, le gammon, le backgammon (probabilités, sans le signe %) et
l'équité cubeless du joueur ; sur une position de bearoff, elle porte en
plus, dés posés ou non, l'EPC, le pip count, le wastage (différence entre
l'EPC et le pip count), le nombre moyen de lancers et l'écart type.
Lorsque les deux joueurs ont des valeurs à comparer, une ligne **Δ** donne
les différences *signées* (bas − haut : négatif quand le joueur noir est en
avance). Hors position de course, poser des dés fait donc disparaître le
tableau de faits lui-même : les quatre colonnes qu'il portait viennent de
changer d'axe, au trait, en tête de la liste des coups.

La décision de videau a toujours la même forme, quelle que soit l'origine des
chiffres — table exacte, régime évalué ou évaluation gammonNet ordinaire :
**une ligne par option**, dans l'ordre *pas de double*, *double/prend*,
*double/passe*, avec son équité dans le référentiel de la position et son
écart à la meilleure option. L'ordre ne change jamais, contrairement à la
liste des coups : les trois options portent un nom, c'est donc le nom qu'on
lit, pas le rang. La meilleure se reconnaît à sa mise en valeur et à sa
cellule d'écart laissée vide. Lorsque le videau a déjà été retourné, les
options se lisent *pas de redouble*, *redouble/prend*, *redouble/passe*.

Une dernière ligne donne le **verdict**. Il prend quatre valeurs : *pas de
double*, *double, prend*, *double, passe* et *trop bon pour doubler*, cette
dernière lorsque jouer la position rapporte davantage que d'encaisser le
point : doubler serait alors une erreur pour la raison inverse de celle du
simple *pas de double*. C'est aussi le seul endroit où le panneau dit qu'il
n'y a **pas** de verdict, plutôt que de laisser croire à un calcul en cours :

* *pas de décision* — le régime n'y a pas droit ; le verdict de videau n'est
  jamais estimé (voir le badge *estimé*) ;
* *non évaluable à ce score* — le moteur refuse la position, typiquement un
  score hors de l'horizon de la table d'équité de match ;
* *videau adverse* et *videau mort (Crawford)* — le videau ne peut pas être
  retourné. Les équités restent affichées, à titre indicatif, mais aucune
  option ne porte d'écart : une erreur, c'est ce que coûte un choix, et il
  n'y a pas de choix.

Le badge de régime, la profondeur d'évaluation, le lien vers le moteur et la
case *Défi* forment une bande à part, alignée à droite au-dessus des
tableaux.

Le **joueur au trait** et la **position du videau** s'éditent
directement sur le plateau, comme en mode édition : cliquer le rectangle
bearoff/score d'un joueur lui donne le trait ; cliquer le videau fait
tourner centré → possédé bas → possédé haut (clic droit en sens inverse).
La valeur du videau reste épinglée — en money game les équités sont
exprimées en unités du videau courant, seul son propriétaire compte.
L'analyse est recalculée aussitôt. En régime estimé, le badge lui-même est
cliquable et ouvre directement l'onglet *Bearoff* de la configuration ; son
infobulle explique pourquoi (verdict de videau non estimable, ADR-0009) et
comment étendre le domaine exact.

Le **score** s'édite lui aussi directement sur le plateau, comme en mode
édition : clic gauche sur le rectangle score d'un joueur décrémente son
nombre de points à faire, clic droit l'incrémente. Sortir du score *money*
(-1, -1) en éditant un seul camp aligne automatiquement l'autre camp sur la
même valeur plutôt que de laisser un score incohérent. Sur une position de
bearoff en régime *exact*, passer d'un score money à un score de match
laisse la probabilité de gain telle quelle (une lecture en base, valable
quel que soit le référentiel) mais bascule l'équité et le verdict de videau
affichés vers ceux du régime *évalué* — la table exacte étant money par
construction, elle ne sait pas répondre à la question posée au score. Le
badge devient alors composite (« exact (gain) · évalué (videau) ») pour le
dire explicitement.

*RETOUR ARRIERE*, ou un double-clic en dehors du plateau, efface la
position : plateau vide, score money (-1, -1), pas de dés posés — des
valeurs propres au panneau Eval, différentes de celles utilisées en mode
édition (7 partout, dés 3-1), pour rester cohérentes avec ce que le panneau
affiche par défaut.

Lorsque la position est un bearoff pur (tous les pions des deux joueurs dans
leur jan) et qu'aucun dé n'est posé, la décision de videau affiche, pour le
joueur au trait :

* en régime *exact* : les équités money (cubeless, sans double, double/prend,
  double/passe) et le **verdict de videau money** (pas de double, double/prend,
  double/passe ou trop bon pour doubler) — hors score de match, voir plus
  haut pour le cas du score,

* en régime *évalué* : les mêmes équités et le même verdict à quatre valeurs,
  mais **joués par gammonNet** (recherche + modèle de videau Janowski) plutôt
  que lus dans une table — disponibles **même au score de match**, ce que le
  régime estimé n'a jamais pu offrir ;

* en régime *estimé* : le verdict de videau n'est alors volontairement pas
  affiché — seule la probabilité de gain, dans le tableau de faits,
  accompagnée de sa marge d'erreur, reste disponible.

Dès que des dés sont posés sur une position de course, cette décision de
videau *avant le jet* disparaît — le plateau demande alors une décision de
pions, pas de videau — mais la probabilité de gain, elle, reste un fait de
la position, pas une décision : elle rejoint la ligne *avant le jet* en
tête de la liste des coups, à côté de l'EPC qui, lui, reste affiché juste à
gauche.

Un badge indique le régime : **exact** (valeur lue dans une base de données
two-sided), **évalué · <profondeur>** (joué par gammonNet — la profondeur
affichée est celle qui a effectivement produit le chiffre montré),
**estimé ± marge**, ou, au score de match dans le domaine exact,
**exact (gain) · évalué (videau)** — voir plus haut. Le régime exact l'emporte
partout où il est disponible ; sinon le régime évalué s'affiche dès qu'il a
fini de calculer, remplaçant en place le régime estimé montré pendant
l'attente. Voir :ref:`epc_methodologie` pour la définition précise des trois
régimes et de leurs hypothèses.

**Élargir le domaine exact.** La base intégrée couvre 6 pions par joueur.
Deux moyens d'aller au-delà, dans l'onglet *Bearoff* de la configuration :

* télécharger la base étendue TS-06-11 (1,2 Go, vérifiée par SHA-256,
  supprimable au même endroit après confirmation) : verdict exact jusqu'à
  11 pions par joueur. Un téléchargement interrompu ou annulé **reprend où
  il s'était arrêté** (le fichier partiel est conservé et complété par
  requête HTTP Range) ;

* indiquer un fichier ``.bd`` two-sided de gnubg quelconque. La base au
  domaine le plus large l'emporte automatiquement.

**Mode défi.** La case *Défi*, dans la bande de badges, active un mode
entraînement : à chaque modification de la position, les valeurs de trois
zones sont masquées (remplacées par « ··· ») ; un clic sur une zone révèle
cette zone seulement. Sans dés, ce sont la ligne du joueur du bas, la ligne
du joueur du haut et la décision de videau — la
ligne Δ n'apparaît qu'une fois les deux lignes joueurs révélées. Le bloc de
décision garde alors ses trois lignes : ce sont ses valeurs, son verdict et
la mise en valeur de la meilleure option qui disparaissent, faute de quoi
l'exercice se résoudrait en cherchant la ligne en gras. Dés posés
sur une position de course, la ligne EPC de chaque joueur se masque comme
avant, mais la troisième zone couvre alors la ligne *avant le jet* et la
liste des coups **ensemble** : la liste étant classée du meilleur coup au
pire, la révéler partiellement en donnerait déjà la réponse. Dés posés hors
position de course, cette même zone unique couvre à elle seule tout ce que
le panneau affiche. On peut ainsi s'entraîner à estimer l'EPC de chaque
camp, puis à se prononcer sur le videau ou sur le coup à jouer, avant de
vérifier. Le réglage est mémorisé.

Pour fermer le panneau Eval, appuyer sur *CTRL-E* ou basculer sur un autre onglet.

.. _epc_methodologie:

Méthodologie et hypothèses du panneau Eval
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Chaque valeur affichée par le panneau repose sur des hypothèses précises,
énoncées ici exhaustivement.

**Domaine.** Le panneau ne traite que le bearoff pur : tous les pions restants
des deux joueurs dans leur jan intérieur. La position est évaluée *avant le
lancer* ; les dés éventuellement posés sont ignorés. Les courses sans contact
dont des pions sont hors du jan ne sont pas traitées.

**Blocs EPC (toujours exacts).** L'EPC, le nombre moyen de lancers et
l'écart type proviennent de la distribution exacte du nombre de lancers pour
sortir tous les pions, lue dans la base one-sided de GNUbg (6 points,
15 pions, intégrée). EPC = lancers moyens × 49/6 (49/6 ≈ 8,167 est la moyenne
exacte de pips par lancer, doubles comptés quatre fois) ; wastage = EPC − pip
count. L'unique idéalisation est le *jeu one-sided optimal* : chaque joueur
minimise ses propres lancers en ignorant l'adversaire — c'est la définition
standard de l'EPC.

**Probabilité de gain, régime exact.** Lecture directe dans la base two-sided
disponible la plus large (intégrée TS-06-06, fichier externe, ou TS-06-11
téléchargée). Ces bases résultent d'une analyse rétrograde complète sous jeu
two-sided optimal des deux camps : aucune hypothèse supplémentaire, erreur
limitée à la quantification (< 0,002 %).

**Probabilité de gain, régime estimé.** Hors du domaine de la base : la
probabilité est obtenue en convoluant les deux distributions one-sided (le
joueur au trait gagne si son nombre de lancers est inférieur ou égal à celui
de l'adversaire), puis en appliquant une correction polynomiale figée,
calibrée hors ligne contre la base TS-06-11. Trois hypothèses :

* **indépendance** des deux processus de sortie — structurelle en course,
  sans contact il n'y a aucune interaction ;

* **jeu one-sided optimal des deux camps** — c'est *l'approximation* : en
  réalité le joueur mené dévie pour jouer la variance et le meneur pour la
  sécurité. L'effet mesuré est un biais antisymétrique (la convolution
  exagère l'avance du meneur) que la correction absorbe statistiquement ;

* la **correction** a été calibrée et validée sur le domaine de l'oracle
  (jusqu'à 11 pions par joueur). Erreur résiduelle mesurée : écart type
  0,05 %, 99e centile 0,17 %, maximum observé 0,9 % (en points de
  probabilité de gain). **Au-delà de 11 pions par joueur, cette borne est
  extrapolée** — la tendance est monotone mais aucun oracle ne la certifie.

**Équités et verdict de videau (régime exact seulement).** Les équités
affichées sont celles du **money game, sans Jacoby**, dans le référentiel de
la littérature du bearoff. Dans le domaine ≤ 11 pions par joueur, les
gammons sont impossibles (chaque camp a déjà sorti au moins 4 pions) : ce
n'est pas une approximation. Le verdict (pas de double / double, prend /
double, passe) est reconstruit exactement des équités stockées, selon la
règle de GNUbg, validée trait pour trait contre son analyse.

.. note:: Les équités cubeful supposent un **jeu de videau optimal des deux
   camps jusqu'au bout** : les recubes futurs sont intégralement valorisés
   (analyse rétrograde complète). Dans les courses très volatiles de fin de
   partie, la cascade de recubes mange presque tout l'avantage du camp au
   trait — les équités « sans double » et « double/prend » peuvent alors
   être proches de zéro là où un moteur comme XG, dont le modèle de videau
   ne valorise pas cette cascade, affiche des valeurs proches du dead cube
   (par exemple 2 pions sur le point 3 contre 2 pions sur le point 2 :
   62 % de gain, D/T exact +0,006 contre +0,475 chez XG). La **décision**
   affichée, elle, coïncide avec celle des moteurs.

**Probabilité de gain et verdict, régime évalué.** Hors du domaine exact, la
probabilité de gain provient de la sortie brute de gammonNet (recherche 0- ou
2-plis selon le geste, jamais lue dans une table), et le verdict d'un
« Decide » Janowski appliqué à cette sortie — la recherche *joue* la
trajectoire au lieu d'en résumer un instantané, ce qui est précisément ce que
le régime estimé ne pouvait pas faire (voir plus bas) et permet, seul des
trois régimes avec l'exact, un verdict **au score de match**.

Ce régime a été mesuré, pas seulement supposé, contre la table two-sided
intégrée (``TestEvalMeasure``, 4000 décisions money échantillonnées,
paramètres canoniques 2-plis k=12) : accord de verdict money **93,4 %**
(3735/4000), ventilé par distance au point de prise de gammonNet — 61,1 % à
moins de 1 % du point de prise (la zone la plus sensible à un pile ou face),
88,3 % entre 1 et 5 %, 91,5 % entre 5 et 10 %, 94,0 % entre 10 et 20 %,
94,4 % au-delà. Écart de probabilité de gain : moyenne 0,85 %, médiane
0,44 %, 95e centile 3,21 %, maximum 8,30 %. Écart d'équité cubeful : moyenne
0,039, médiane 0,018, 95e centile 0,151, maximum 0,406. La forme est celle
attendue : l'essentiel du désaccord se concentre exactement au point de
prise, où deux méthodes légitimement différentes divergent le plus sur une
décision serrée — pas une erreur diffuse qui coûterait de l'équité partout.

**Pourquoi le verdict estimé n'existe-t-il pas ?** Ce qui suit vise
spécifiquement la méthode par *convolution* (régime estimé), pas le régime
évalué ci-dessus : l'équité cubeful est un problème de *trajectoire* (quand
doubler), qu'aucun résumé statistique de la position ne capture — le meilleur
modèle statique mesuré laisse une erreur résiduelle (écart type 0,016
d'équité, maximum 0,20) qui suffit à inverser toutes les décisions serrées.
De même, la conversion du verdict au score de match via une table d'équités
de match a été mesurée insuffisante (12 % de désaccords avec l'analyse
2-ply de GNUbg, avec de vraies bourdes). Un verdict faux affiché avec aplomb
étant pire que pas de verdict, la convolution n'a jamais eu le droit
d'afficher de verdict — c'est une recherche qui joue la trajectoire, pas un
résumé statistique, qui comble ce trou.

.. note:: Les bases de bearoff sont des tables mathématiques immuables,
   régénérables avec l'outil ``makebearoff`` de GNUbg.

.. _panneau_anki:

Panneau Anki
------------

Le panneau **Anki** (*CTRL-K*) permet d'étudier des positions par répétition
espacée en utilisant l'algorithme FSRS. L'utilisateur peut créer des paquets
à partir de collections ou de résultats de recherche.

**Création de paquets :** Cliquez sur *New Deck* pour créer un paquet à partir
d'une collection ou des résultats de recherche courants. Les paquets basés sur
une recherche se synchronisent automatiquement à l'activation de l'onglet Anki.

**Révision :** Sélectionnez un paquet puis cliquez sur *Study* (ou double-cliquez
sur un paquet) pour commencer la révision des cartes dues. Chaque carte affiche
la position correspondante sur le plateau. Évaluez votre rappel avec les touches
*1* (À revoir), *2* (Difficile), *3* (Bien), ou *4* (Facile). Appuyez sur *Esc*
pour arrêter et revenir à la liste des paquets.

**Entraînement libre (cram) :** Le bouton *Cram*, à côté de *Study*, lance une
session d'entraînement libre : des positions aléatoires du paquet vous sont
présentées sans tenir compte de l'échéancier FSRS. Ce mode **ne modifie jamais
le planning de révision espacée** — idéal pour s'échauffer avant un tournoi ou
réviser intensément un paquet thématique sans perturber son ordonnancement. Une
pastille *Cram* remplace l'état de la carte et un bouton *Suivant* (touches *1*
à *4*) fait défiler les positions. *Esc* revient à la liste sans enregistrer de
session interrompue.

**Arrêt/Reprise :** Vous pouvez interrompre une session de révision à tout moment
avec *Esc*. Le bouton change en *Resume* et affiche votre progression.
Cliquez dessus pour reprendre là où vous vous êtes arrêté.

**Gestion des paquets :** Utilisez les boutons d'action pour renommer,
synchroniser, réinitialiser ou supprimer des paquets (confirmation demandée
pour ces deux dernières actions). Les paramètres FSRS (rétention cible,
intervalle maximum, aléa) peuvent être configurés par paquet dans les
Paramètres (icône engrenage).

.. _panneau_metadata:

Panneau Métadonnées
-------------------

Le panneau **Métadonnées** affiche les informations générales de la base de
données courante : nom, description, nombre de positions, nombre de matchs et
de parties, version du schéma. Accessible via la commande ``meta``.

Il affiche également, **lorsqu'elle existe**, l'origine de la base — voir
:ref:`diffusion_controlee`. Une base ordinaire n'affiche pas cette section.

.. _diffusion_controlee:

Diffuser une base : origine et mot de passe
-------------------------------------------

Un enseignant qui distribue une base de positions dispose de deux mécanismes,
indépendants l'un de l'autre, tous deux facultatifs et choisis **au moment de
l'export** : marquer le fichier de son origine, et le protéger par un mot de
passe.

.. note::

   Aucun des deux ne suit ce que devient le fichier. blunderDB **n'enregistre
   rien du côté de celui qui reçoit la base** : ouvrir une base marquée est
   exactement comme ouvrir n'importe quelle autre, et rien nulle part ne
   consigne qui l'a ouverte, quand, ni d'où vient son contenu.

Marquer une base de son origine
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

La fenêtre d'export tient en un seul écran : le formulaire, puis une
progression qui se superpose à lui le temps de l'écriture. Elle se ferme
d'elle-même une fois terminée, et le résultat s'affiche dans la barre d'état.

Trois points méritent l'attention :

* **L'export porte sur les positions actuellement affichées**, pas sur la base
  entière. Après une recherche, seuls les résultats partent — la fenêtre le
  rappelle en tête.
* **Une collection dont toutes les positions ne sont pas dans la sélection
  arrive tronquée.** La liste affiche donc, pour chaque collection, la part
  couverte (« 12/40 ») et la signale en rouge lorsqu'elle est partielle.
* **Les tournois ne peuvent être exportés qu'avec les matchs** : sans eux, le
  lien tournoi–match n'existe pas et le tournoi arriverait vide. La case est
  désactivée tant que « inclure les matchs » ne l'est pas.

Les champs *Utilisateur*, *Description* et *Date* décrivent le **fichier
produit** ; ils sont préremplis depuis la base source. La case *Mes filtres
enregistrés* est à part des autres : elle n'exporte pas du contenu mais vos
propres recherches enregistrées, sans utilité dans la base de quelqu'un
d'autre.

Cocher **Marquer ce fichier de son origine** fait apparaître deux champs :

* **Origine** — ce qu'est ce fichier et d'où il vient, dans vos mots :
  « Cours de Jean Dupont — 12 mars 2026 ». Ce champ est **obligatoire** : tant
  qu'il est vide, le bouton d'export reste inactif.
* **Note**, facultative — conditions d'utilisation, adresse de contact, une
  demande de ne pas rediffuser.

La marque est signée avec votre identité d'émetteur. Elle est donc
**inaltérable et infalsifiable** : nul ne peut la modifier, ni en fabriquer une
à votre nom. Elle n'est en revanche **pas ineffaçable** — le fichier distribué
est une base SQLite ordinaire, et blunderDB est un logiciel libre. Elle
n'empêche rien : elle dit d'où vient le fichier.

Identité d'émetteur
~~~~~~~~~~~~~~~~~~~

Les marques sont signées avec votre **identité d'émetteur**, créée toute seule
la première fois que vous marquez un fichier ; il n'y a rien à configurer. Elle
appartient à une personne et non à une base : tous vos fichiers portent la même
empreinte publique, de la forme ``A3F1-9C24-7B05-E1D8``.

Vous pouvez communiquer cette empreinte à vos destinataires pour qu'ils
vérifient qu'un fichier vient bien de vous. L'identité se transporte d'un poste
à l'autre en un seul fichier (extension ``.bdbid``), éventuellement protégé par
une phrase secrète. **Ce fichier permet de signer en votre nom : ne le partagez
pas.**

Dans les préférences (icône engrenage de la barre d'outils), l'onglet *Identité
d'émetteur* affiche votre nom et votre empreinte, et propose *Enregistrer
l'identité…*, *Charger une identité…* et *Régénérer…*.

.. warning::

   **Régénérer ne révoque rien.** Un filigrane embarque la clé publique qui l'a
   signé : il se vérifie donc pour toujours, tout seul. Si votre fichier
   d'identité a fuité, celui qui le détient pourra continuer à signer sous votre
   ancienne empreinte, et ces marques resteront valides.

   Ce qui vous protège après une fuite n'est pas logiciel : c'est de publier
   votre nouvelle empreinte et de désavouer l'ancienne auprès de vos
   destinataires.

   La régénération écrase la clé actuelle ; blunderDB propose de l'enregistrer
   avant de la remplacer.

Protéger une base par un mot de passe
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Le mot de passe se saisit masqué, ici comme à l'ouverture d'un fichier protégé ;
l'icône en forme d'œil l'affiche **tant qu'on la maintient enfoncée**, et le
masque de nouveau dès qu'on relâche.

Cocher **Protéger ce fichier par un mot de passe** produit un fichier
d'extension ``.dbx`` — y compris si vous aviez choisi un nom en ``.db`` dans la
fenêtre d'enregistrement, celle-ci s'ouvrant avant que le mot de passe ne soit
demandé. Pour l'ouvrir, utilisez l'ouverture de base habituelle : la fenêtre de
sélection accepte aussi bien les ``.db`` que les ``.dbx``. blunderDB demande
alors le mot de passe et installe une base ordinaire à côté ; ensuite plus rien
n'est demandé.

La fenêtre propose de **supprimer le fichier protégé une fois ouvert** : sans
cela vous conservez le même contenu sous deux noms. La case n'est pas cochée par
défaut — le fichier protégé reste le vôtre si vous comptez le transmettre — et
la suppression n'a lieu qu'après une ouverture réussie.

.. warning::

   Le mot de passe protège le **transport** du fichier, pas la base. Il empêche
   un tiers d'ouvrir un fichier qui traîne dans un dossier de téléchargement ou
   une pièce jointe transférée par erreur. Il ne protège pas de celui à qui vous
   avez donné le mot de passe.

Le mot de passe est vérifié à **chaque** ouverture, y compris lorsque le fichier
a déjà été ouvert auparavant sur ce poste.

Techniquement, la base est chiffrée par **AES-256 en mode GCM**, avec une clé
dérivée du mot de passe par **Argon2id** (64 Mio de mémoire, 3 passes, 4 fils),
et un sel tiré au hasard propre à chaque fichier. Le mode GCM authentifie
l'ensemble : un mot de passe erroné est détecté comme tel, et toute altération du
fichier chiffré l'est également — on n'obtient jamais une base corrompue en
silence.

L'en-tête du fichier protégé reste **en clair** : son origine demeure lisible
sans le mot de passe.

Lire l'origine d'un fichier
~~~~~~~~~~~~~~~~~~~~~~~~~~~

Dans l'application, ouvrez le fichier et affichez le panneau **Métadonnées**
(commande ``meta``). Une section **Origine** apparaît en tête du panneau, en
lecture seule, indiquant ce qui a été inscrit, par qui, quand, et l'état de la
signature :

* « ✓ signature vérifiée — marquée par vous » : le fichier porte votre marque,
  intacte ;
* « ✓ signature vérifiée » : la marque est intacte et vient d'une autre clé —
  comparez son empreinte à celle que le producteur vous a communiquée ;
* « ⚠ signature invalide » : le document a été modifié ou contrefait.

Cette section n'apparaît pas sur une base ordinaire.

En ligne de commande, ``blunderdb info --db fichier.db`` affiche l'origine et
l'état de la signature, **sans jamais écrire dans le fichier**. La commande
fonctionne aussi sur un fichier protégé, sans le mot de passe. Voir
``CLI_USAGE.md`` pour les options ``--watermark`` et ``--password`` de
``export``, ainsi que pour ``identity`` et ``open``.

