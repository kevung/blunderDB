.. _index:

blunderDB
=========

blunderDB est un logiciel pour constituer des bases de données de positions de
backgammon. Sa force principale est d'offrir un lieu unique où agréger
les positions qu'un joueur a pu rencontrer (en ligne, en tournoi) et de pouvoir
réétudier ces positions en les filtrant selon différents filtres combinables
arbitrairement. blunderDB peut également être utilisé pour constituer des
catalogues de positions de référence.

Nouveau venu ? Le **guide utilisateur** propose quatre tutoriels de bout en
bout (premier import, étude d'un match, session Anki, mode serveur) et une
page « comment progresser » — la lecture la plus rentable avant de se lancer.

La présente documentation est structurée en quatorze sections :

* **téléchargement et installation** explique comment se procurer et lancer
  blunderDB ;

* le **guide utilisateur** est une introduction pratique et des tutoriels de
  bout en bout pour utiliser rapidement blunderDB ;

* le **manuel** décrit exhaustivement le fonctionnement de blunderDB, panneau
  par panneau ;

* le **mode commande** documente la ligne de commande intégrée à
  l'application (filtres de recherche, actions) ;

* la liste des **raccourcis** clavier permet une utilisation efficace au
  clavier ;

* l'**interface en ligne de commande (CLI)** décrit les commandes du binaire
  disponibles hors interface graphique, pour l'import en masse,
  l'automatisation et le scripting ;

* le **glossaire** définit en une page les termes du domaine (position,
  référentiel, régime, verdict…) employés dans le reste de la
  documentation ;

* la **FAQ** répond aux interrogations les plus fréquentes ;

* la **feuille de route** recense les pistes d'évolution envisagées, sans
  engagement de date ;

* le **mode serveur (headless)** documente le démon ``serve``, ses routes
  HTTP et ses deux backends (SQLite, PostgreSQL) ;

* l'**annexe des filtres de recherche** détaille chaque critère du panneau
  de recherche et son jeton en ligne de commande ;

* les **annexes de sécurité** (Windows, macOS) expliquent les avertissements
  du système d'exploitation à l'installation d'un binaire non signé ;

* l'**annexe du schéma de base de données** décrit le format ``.db`` ;

* la page de **parité des statistiques** énonce les règles de comptage du
  PR, du Snowie Error Rate et du coût MWC face à eXtreme Gammon et GNUbg.

Historique des versions
=======================

Dernière version — 0.36.0 (2026-09-05)
---------------------------------------

- Le binaire **maigrit de 7,3 Mo** : les tables de sortie ne sont plus embarquées ni téléchargées, mais calculées à la première ouverture, en arrière-plan et en silence — identiques octet pour octet à celles de gnubg, empreinte SHA-256 vérifiée. La table étendue TS-06-11 se calcule dans l'onglet *Bearoff* au lieu d'un téléchargement de 1,2 Go.
- **Les tournois se remplissent à l'import** : un match entre dans le tournoi que son fichier nomme, créé au besoin ; un match déjà rangé n'est jamais déplacé.
- **Une seule grammaire de recherche**, désormais lisible depuis la ligne de commande (``blunderdb search --query``) autant que depuis l'application.
- La ligne de commande gagne ``--format json`` sur neuf commandes, la **complétion pour bash, zsh et fish**, la commande ``repair`` et les sous-commandes ``anki card`` et ``anki log``.
- L'**aide intégrée est engendrée depuis le manuel** : elle ne peut plus prendre de retard sur lui. Trois visites guidées de plus (Eval, Anki, Stats), et les pages Raccourcis et Ligne de commande deviennent des tableaux, sans encarts.
- Un **binaire Linux arm64**, livré en archive, ``.deb``, ``.rpm`` et paquet AUR.
- Le panneau **Eval** retrouve le plateau sur lequel on l'a quitté, distingue l'équité *money* de l'équité *match*, et se parcourt au clavier.
- Côté serveur : **contrat d'API engendré du code** (``openapi.yaml`` et son annexe, 135 routes), identifiant de corrélation par requête, métriques de travail en vol, compression des flux NDJSON (13,5 % de la taille), et les deux appels qui dépassent le tenant — vidange et purge — passent sous ``/ops/``.
- **Schéma 2.18.0** : Jacoby et beaver quittent l'identité de la position, si bien qu'une même position d'argent n'entre plus deux fois par des portes différentes.
- Corrections notables : le volet de détail du panneau Matchs n'occupe plus la moitié de l'écran quand rien n'est sélectionné, la première partie d'un transcript se replie comme les autres, SHIFT-J et SHIFT-K changent de vue depuis n'importe quel panneau, et huit raccourcis « afficher/cacher » cachent enfin.

Voir :ref:`cli`, :ref:`headless` et :ref:`manuel`.

Historique complet
-------------------

Chaque version ci-dessous liste ses évolutions puce par puce, pour qu'une
correction ultérieure (une coquille, un ajout tardif) n'invalide qu'une
puce et non la traduction de toute la version.

0.36.0 (2026-09-05)
~~~~~~~~~~~~~~~~~~~

- Le binaire **maigrit de 7,3 Mo** : les tables de sortie ne sont plus embarquées ni téléchargées, mais calculées à la première ouverture, en arrière-plan et en silence — identiques octet pour octet à celles de gnubg, empreinte SHA-256 vérifiée. La table étendue TS-06-11 se calcule dans l'onglet *Bearoff* au lieu d'un téléchargement de 1,2 Go.
- **Les tournois se remplissent à l'import** : un match entre dans le tournoi que son fichier nomme, créé au besoin ; un match déjà rangé n'est jamais déplacé.
- **Une seule grammaire de recherche**, désormais lisible depuis la ligne de commande (``blunderdb search --query``) autant que depuis l'application.
- La ligne de commande gagne ``--format json`` sur neuf commandes, la **complétion pour bash, zsh et fish**, la commande ``repair`` et les sous-commandes ``anki card`` et ``anki log``.
- L'**aide intégrée est engendrée depuis le manuel** : elle ne peut plus prendre de retard sur lui. Trois visites guidées de plus (Eval, Anki, Stats), et les pages Raccourcis et Ligne de commande deviennent des tableaux, sans encarts.
- Un **binaire Linux arm64**, livré en archive, ``.deb``, ``.rpm`` et paquet AUR.
- Le panneau **Eval** retrouve le plateau sur lequel on l'a quitté, distingue l'équité *money* de l'équité *match*, et se parcourt au clavier.
- Côté serveur : **contrat d'API engendré du code** (``openapi.yaml`` et son annexe, 135 routes), identifiant de corrélation par requête, métriques de travail en vol, compression des flux NDJSON (13,5 % de la taille), et les deux appels qui dépassent le tenant — vidange et purge — passent sous ``/ops/``.
- **Schéma 2.18.0** : Jacoby et beaver quittent l'identité de la position, si bien qu'une même position d'argent n'entre plus deux fois par des portes différentes.
- Corrections notables : le volet de détail du panneau Matchs n'occupe plus la moitié de l'écran quand rien n'est sélectionné, la première partie d'un transcript se replie comme les autres, SHIFT-J et SHIFT-K changent de vue depuis n'importe quel panneau, et huit raccourcis « afficher/cacher » cachent enfin.
- Voir :ref:`cli`, :ref:`headless` et :ref:`manuel`.

0.35.0 (2026-09-02)
~~~~~~~~~~~~~~~~~~~

- La ligne de commande gagne deux familles : ``blunderdb collection`` (lister, afficher, créer, renommer, supprimer, exporter une collection) et ``blunderdb anki`` (paquets, statistiques, prévision des révisions, resynchronisation d'un paquet).
- Le mode serveur est publié en **image Docker** (``ghcr.io/kevung/blunderdb-serve``, amd64 et arm64), gagne la route ``maintenance.vacuum`` et **filigrane ses exports** avec l'identité d'émetteur du serveur (``--identity-dir``).
- L'évaluateur gammonNet **value ses feuilles avec le videau** : au score, les verdicts de la recherche se rapprochent de ceux d'eXtreme Gammon.
- Chaque release livre désormais les **manifestes winget et Homebrew** et un bundle Flatpak, en plus des paquets Debian, RPM et AUR.
- Dans l'interface, la bibliothèque est **paginée** (une base de 50 000 positions ne charge plus tout en mémoire à chaque rafraîchissement), les fenêtres modales partagent un même socle (focus, Échap, accessibilité), le plateau ne redessine que ce qui change, et le binaire perd 5,5 Mo de police japonaise inutile.
- Corrections notables : une base importée par « Importer une base » gardait des positions invisibles aux filtres et dédoublées au second import (réparé à l'ouverture), le bouton Fusionner de la fenêtre de fusion des joueurs restait grisé, l'export du serveur était vide, et sous Windows la suppression de la base bearoff téléchargée et ``create --force`` échouaient.
- Voir :ref:`cli`, :ref:`headless` et :ref:`manuel`.

0.34.0 (2026-09-02)
~~~~~~~~~~~~~~~~~~~

- Un **évaluateur embarqué** : gammonNet, porté en Go et compilé dans blunderDB, évalue n'importe quelle position hors ligne, sans XG ni gnuBG — réseau de neurones, recherche 0 à 2-ply avec élagage, décision de videau selon Janowski et la table d'équité de match de blunderDB, et une recherche qui honore le score du match.
- Le panneau Bearoff devient le panneau **Eval** : il présente les faits de la position (chances de gain, gammon et backgammon des deux camps, équité) puis la seule décision que le plateau demande, sur une grille stable et sans défilement ; les équités de match sont **normalisées** et le verdict « trop bon pour doubler » redevient visible.
- L'évaluation est **progressive** : 0-ply immédiat au geste, puis 2-ply au repos, annulable par tout nouveau geste, l'étiquette de profondeur disant toujours celle qui a produit le chiffre.
- Le plateau entier, les dés et le score s'y éditent, au clic comme au clavier, et une position de la bibliothèque entre dans le panneau et en ressort.
- Sur une course au-delà de la base bi-face, un troisième régime **évalué** fournit le verdict de videau au score que le panneau ne donnait pas.
- Un onglet **gammonNet** de la configuration sépare la profondeur d'affichage de la profondeur d'analyse.
- Après un import, un **lot d'analyse** borné, visible, annulable et reprenable comble les positions sans aucune analyse, sans jamais écraser une analyse importée ; le **rattrapage d'une bibliothèque** existante se lance depuis l'interface, par la nouvelle commande ``blunderdb analyze`` ou en mode serveur.
- Corrections notables : la réponse à un videau importée d'eXtreme Gammon héritait de l'analyse du doubleur sans être retournée, la notation des coups des Blancs et des entrées de barre suit gnuBG et XG, la profondeur d'une analyse gnuBG est lue correctement, et la taille du panneau latéral est mémorisée.
- Voir :ref:`panneau_epc`, :ref:`manuel` et :ref:`cli`.

0.33.0 (2026-08-26)
~~~~~~~~~~~~~~~~~~~

- Le panneau Stats gagne un onglet **Joueurs** : les trois onglets précédents décrivent un joueur, celui-ci les compare tous, une ligne par joueur de la base — matchs, victoires/défaites, décisions comptées, PR global / pions / videau, Snowie Error Rate, blunders et chance. Le tableau se trie par n'importe quelle colonne, se restreint aux dates d'une compétition par les filtres habituels, et un clic sur une ligne bascule sur le détail du joueur. Un tiret y signale une valeur jamais mesurée, jamais un zéro. Le tableau est également disponible en ligne de commande (``blunderdb list --type players``, en texte, JSON ou CSV) et en mode serveur.
- La **chance** de chaque lancer est désormais conservée en base et affichée : chance moyenne par lancer, en millièmes de point, signée. Elle est lue à l'import depuis les fichiers eXtreme Gammon comme depuis gnuBG. Elle n'est pas rétroactive — les matchs importés auparavant portent un tiret, et il faut réimporter les fichiers source pour la récupérer ; les formats qui ne la transportent pas (BGF, Jellyfish ``.mat``) n'en fourniront jamais.
- L'onglet **Erreurs** indique désormais dans quel *sens* se trompent les décisions de videau, et non plus seulement combien elles coûtent : une ligne *Offrir* (doubles manqués, doubles prématurés) et une ligne *Répondre* (passes à tort, prises à tort), les deux séparées à dessein puisqu'un même joueur peut doubler tard et prendre large. Chaque case porte son nombre, l'infobulle l'équité perdue, et le clic charge les positions correspondantes.
- Nouvelle commande **Compacter la base**, qui récupère l'espace laissé par les suppressions : bouton dans les paramètres et sous-commande ``blunderdb vacuum``. Elle ne se déclenche jamais toute seule.
- Ergonomie : les actions destructives demandent confirmation, une recherche en cours est signalée, une commande inconnue reçoit un message au lieu du silence, et les onglets, panneaux dockés et petites fenêtres se pilotent entièrement au clavier. La recherche est par ailleurs sensiblement plus rapide.
- Corrections notables : les non-doubles étiquetés « Double No » par gnuBG étaient comptés comme des doubles, ce qui faussait le PR videau (une base déjà constituée se répare sans réimport, en mode serveur) ; le filtre joueur rétrécissait à tort le dénominateur du Snowie Error Rate ; les exports écrivaient un schéma périmé ; et ``blunderdb vacuum`` ouvrait l'interface graphique au lieu de compacter la base.
- Voir :ref:`stats` et :ref:`cli`.

0.32.0 (2026-08-09)
~~~~~~~~~~~~~~~~~~~

- Le panneau EPC devient le panneau **Bearoff** et gagne l'analyse de course complète.
- Sur une position de bearoff pur, il affiche — en plus de l'EPC, du pip count, du wastage et du nombre moyen de lancers de chaque joueur, désormais présentés en tableau avec une ligne Δ signée — les probabilités de gain des deux joueurs et les équités money (cubeless, sans double, double/prend, double/passe, avec l'écart de chaque décision à la meilleure) ainsi que la **meilleure décision de videau**.
- Deux régimes clairement distingués : *exact* (lecture dans une base two-sided ; base TS-06-06 de GNUbg intégrée, couvrant jusqu'à 6 pions par joueur) et *estimé* (convolution calibrée, marge d'erreur affichée — le verdict de videau n'est jamais estimé).
- La base étendue TS-06-11 (verdict exact jusqu'à 11 pions par joueur, 1,2 Go) se télécharge depuis le nouvel onglet *Bearoff* de la configuration, avec vérification d'intégrité et reprise des téléchargements interrompus ; un fichier ``.bd`` two-sided quelconque de gnubg peut aussi être désigné.
- **Mode défi** pour l'entraînement : les résultats se masquent à chaque modification et se révèlent zone par zone, d'un clic.
- Édition entièrement au plateau : pions des deux jans, joueur au trait (clic sur le rectangle sortie/score) et position du videau (clic sur le videau).
- Le clic sur le plateau est désormais exact à toute échelle d'interface (un clic sur le point 1 pouvait atterrir sur le point 2).
- Nouvelle commande ``blunderdb epc`` en ligne de commande et option ``--bearoff-ts`` du mode serveur.
- Une nouvelle section du manuel, « Méthodologie et hypothèses du panneau Bearoff », énonce exhaustivement les hypothèses derrière chaque valeur affichée.
- Voir :ref:`panneau_epc` et :ref:`epc_methodologie`.

0.31.0 (2026-08-04)
~~~~~~~~~~~~~~~~~~~

- Diffusion d'une base à des tiers : deux mécanismes indépendants, tous deux facultatifs et choisis au moment de l'export.
- Le **filigrane d'origine** inscrit dans le fichier ce qu'il est et d'où il vient, signé par une identité d'émetteur créée toute seule au premier usage ; la marque est infalsifiable, se lit en tête du panneau Métadonnées (ou par ``blunderdb info``, sans jamais écrire dans le fichier) et ne consigne rien du côté de celui qui reçoit la base — aucun journal, aucun suivi.
- La **protection par mot de passe** produit un fichier ``.dbx`` chiffré (AES-256-GCM, clé dérivée par Argon2id), vérifié à chaque ouverture, dont l'en-tête reste lisible en clair ; l'identité s'exporte et s'importe en un seul fichier depuis les préférences.
- Fenêtre d'export refondue : un seul écran au lieu de quatre, rappel de ce qui part réellement (les positions affichées), part couverte de chaque collection signalée en rouge lorsqu'elle est partielle, tournois exportables seulement avec leurs matchs — et un export trois fois plus rapide.
- Fenêtre de configuration réorganisée en trois onglets (Interface, Couleurs du plateau, Identité d'émetteur), de taille et de position stables.
- Panneau Métadonnées redisposé en une seule grille.
- Le panneau Journal, qui ne recevait que l'écho des commandes, est retiré.
- Corrections notables : un mot de passe erroné n'ouvre plus une base protégée, les champs des fenêtres modales ne perdent plus le focus au clic, le calcul des indicateurs PR/MWC ne plante plus sur une partie d'argent, et les fichiers temporaires (``-wal``, ``-shm``, verrou) sont nettoyés à la fermeture.
- Voir :ref:`diffusion_controlee` et :ref:`manuel`.

0.30.0 (2026-07-26)
~~~~~~~~~~~~~~~~~~~

- Recherche par commentaire : le filtre « Texte de recherche » devient le filtre « Commentaire » et propose trois modes exclusifs — *contient le texte* (comportement précédent, jeton ``t""mot1;mot2""``), *a un commentaire* (jeton ``co``) et *sans commentaire* (jeton ``xco``) — pour retrouver d'un geste les positions annotées, ou au contraire celles qu'il reste à commenter.
- Positions marquées dans eXtreme Gammon : les marques (*flags*) posées lors de l'analyse d'un match XG sont désormais lues à l'import et retrouvables par le filtre « Marquée » (jeton ``fl``), combinable avec tous les autres critères ; une décision de videau marquée donne deux positions marquées, le double et la prise/passe.
- Le marquage n'étant pas rétroactif, il suffit de réimporter le fichier ``.xg`` concerné pour que ses marques rejoignent la base, sans toucher aux commentaires ni aux analyses existants.
- Corrections notables : le PR affiché sur le badge d'un tournoi se rapporte au joueur de référence et non plus au cumul des deux joueurs, les paquets Anki fondés sur une recherche respectent de nouveau les filtres de cette recherche au lieu de reprendre toute la base, et un blocage de la recherche combinant certains filtres (texte, date, erreur du coup joué) est corrigé.
- Voir :ref:`cmd_mode` et :ref:`manuel`.

0.29.0 (2026-07-18)
~~~~~~~~~~~~~~~~~~~

- Export d'un match au format Jellyfish/gnubg (``.mat``) depuis l'interface (panneau Matchs) comme en ligne de commande, avec la route serveur correspondante — pratique pour rejouer un match dans un autre logiciel de backgammon.
- Recherche : nouveau discriminateur « dés exclus » (jeton ``xD65``, plusieurs autorisés) qui écarte les positions dont le lancer correspond à un jet donné.
- Ouverture concurrente sécurisée : une base déjà ouverte en écriture dans une autre fenêtre de blunderDB s'ouvre désormais en lecture seule (navigation, recherche et analyse possibles, modifications désactivées, mention « [lecture seule] » dans la barre de titre) au lieu de risquer une corruption.
- Provenance des positions : hors mode match, la barre d'informations au-dessus du plateau indique le match — et son tournoi — d'où provient la position étudiée, avec un badge « +N » lorsqu'elle apparaît dans plusieurs matchs.
- Robustesse accrue face à l'environnement : repli automatique lorsqu'une police, le presse-papier ou la dernière base ouverte ne sont pas disponibles.
- Un rechargement explicite de la bibliothèque (``CTRL-R``, bouton de la barre d'outils ou commande ``e``) rouvre le panneau d'analyse de la position affichée.
- Corrections notables : suppression d'un blocage à l'export de grosses bases, bouton « Aucun » de l'export qui n'exportait pas réellement aucun match, filtre par commentaire multi-mots, dédoublonnage de l'événement et du tournoi dans la barre d'informations, et divers ajustements d'affichage.
- Voir :ref:`cmd_mode`, :ref:`panneau_matchs` et :ref:`manuel`.

0.28.0 (2026-07-15)
~~~~~~~~~~~~~~~~~~~

- Panneau de recherche : filtre unifié « Matchs & Tournois » remplaçant les champs de saisie d'identifiants par une fenêtre de sélection (listes à cocher filtrables par texte, boutons *Tout*/*Aucun*, cocher un tournoi coche ses matchs membres) — plus rapide sur les grosses bases, l'onglet Matchs et la vue Tournois ne recalculent plus les indicateurs PR/MWC que pour les matchs affichés ; ``cli list --type tournaments`` comble l'écart de parité correspondant.
- Nouveau filtre « Importée individuellement » (case à cocher du groupe Autre, ou jeton ``i`` en ligne de commande, ``s i`` ; disponible aussi en CLI via ``--individual``) pour retrouver une position enregistrée ou importée seule plutôt que noyée dans un import de match.
- Correction d'une perte de données : la suppression d'un match ne supprime plus les positions individuellement importées, mises en collection ou intégrées à un paquet Anki que ce match contenait aussi.
- Mode serveur (headless) : six nouvelles méthodes pour le planificateur Anki à répétition espacée — journal des révisions (``anki.reviewLog``), prévision des cartes dues (``anki.forecast``), suspension / mise en veille / suppression de carte (``anki.suspendCard``, ``anki.buryCard``, ``anki.removeCard``) et ajustement automatique du taux de rétention visé d'un paquet vers son taux de réussite observé (``anki.optimizeParams``) ; nouvelle méthode ``tenant.purge`` pour supprimer définitivement les données d'un tenant sur un déploiement PostgreSQL multi-utilisateurs ; nouvelles méthodes ``matches.list`` (filtrage/tri/pagination) et ``stats.matchBadges`` (indicateurs PR/MWC bornés à une liste de matchs).
- Distribution Linux native : paquets ``.deb``/``.rpm``, qui conservent le bit d'exécution perdu lors du téléchargement du binaire brut.
- Import : après l'import d'un match, l'onglet Matchs s'affiche et les positions importées sont immédiatement visibles ; après l'import d'une position isolée (fichier XGP, position BGBlitz, fichier de position texte, position collée ou lot de positions), l'onglet Analyse s'ouvre directement sur la position importée, au lieu de l'onglet Matchs ; correction du collage d'une analyse eXtreme Gammon en français ou en allemand depuis macOS, qui restait vide (accents décomposés par le presse-papier du système).
- Voir :ref:`headless`, :ref:`cmd_mode` et :ref:`telecharge_install`.

0.27.0 (2026-06-13)
~~~~~~~~~~~~~~~~~~~

- Panneau Anki : ajout d'un mode « entraînement libre » (*cram*), accessible par le bouton *Cram* à côté de *Study*, qui présente des positions aléatoires du paquet sans modifier le planning de révision espacée — idéal pour s'échauffer avant un tournoi ou réviser intensément un paquet sans en perturber l'ordonnancement.
- Recherche : nouvelle commande ``blunders`` (alias ``bl``) qui charge directement les pires erreurs (équité/MWC) dans la vue d'analyse, avec un nombre optionnel pour en choisir la quantité (``bl 50``) ; nouveau filtre par joueur ``pl'nom'`` qui retrouve toutes les positions issues d'un match impliquant un joueur donné, sur l'un ou l'autre camp ; et des info-bulles, au survol de chaque filtre du panneau de recherche, indiquant le jeton de commande correspondant.
- Mode serveur (headless) : nouvelles méthodes ``matches.movesByMatch`` (tous les coups d'un match en un seul appel) et ``positions.epc`` (Effective Pip Count des deux joueurs).
- Voir :ref:`panneau_anki` et :ref:`cmd_mode`.

0.26.0 (2026-06-09)
~~~~~~~~~~~~~~~~~~~

- Réglages d'affichage de l'interface dans la fenêtre de configuration : un curseur d'échelle pour agrandir ou réduire l'ensemble de l'interface, et un choix de position des panneaux (en bas, sur le côté ou automatique) afin de mieux exploiter les écrans larges.
- Ajout d'une barre d'informations au-dessus du plateau rappelant les joueurs et le contexte du match (événement, lieu, ronde, date, longueur).
- À l'ouverture d'une base, le panneau des matchs est affiché d'emblée et la revue débute sur la première position.
- Les raccourcis clavier sont désormais indépendants de la disposition du clavier (AZERTY, QWERTY, QWERTZ…).
- Correction du décodage XGID (indicateurs Jacoby/Beaver et valeur du videau).
- Ajout des onglets de vues multiples : plusieurs espaces de travail indépendants (liste de positions, position courante, recherche) avec les raccourcis *CTRL-T* (nouvelle vue), *CTRL-W* (fermer), *CTRL-PageUp*/*CTRL-PageDown* (changer de vue) et renommage par double-clic sur l'onglet.
- Voir :ref:`configuration` et :ref:`onglets_vues`.

0.25.0 (2026-06-07)
~~~~~~~~~~~~~~~~~~~

- Mode serveur (headless) : deux nouvelles méthodes pour reconstruire une position sans l'enregistrer — ``positions.fromXGID`` décode une chaîne XGID en position, et ``positions.fromXGP`` lit un fichier de position unique ``.xgp``.
- Voir :ref:`headless`.

0.24.0 (2026-06-06)
~~~~~~~~~~~~~~~~~~~

- Ajout d'une image conteneur (Docker) pour le mode headless : un point d'entrée dédié ``serve`` et un ``Dockerfile.serve`` produisant un binaire statique sans interface graphique, afin de déployer le moteur de blunderDB comme service derrière un reverse-proxy.
- Voir :ref:`headless`.

0.23.0 (2026-06-05)
~~~~~~~~~~~~~~~~~~~

- Ajout de visites guidées de l'interface (tour général, recherche, matchs, tournois), rejouables depuis la barre d'outils ou la commande ``tour``, et d'une base d'exemple chargeable par la commande ``demo`` pour découvrir l'outil sans importer ses propres parties.
- Personnalisation des couleurs du plateau (fond, bordure, flèches, pions, dés, videau) depuis la fenêtre de configuration.
- Autocomplétion de la ligne de commande (touche *TAB*).
- Panneau de recherche : contrôle explicite du type de décision (Pions / Videau), avec un sous-type Double / Pas de double ou Prise / Passe pour les décisions de cube, synchronisé avec le plateau ; le videau proposé est affiché au centre du plateau pour les décisions de prise/passe.
- Recherche d'une position par son identifiant (filtre ``id``).
- Correction de l'attribution de l'erreur de videau au joueur 1.
- Voir :ref:`visites_guidees`.

0.22.0 (2026-06-02)
~~~~~~~~~~~~~~~~~~~

- Ajout d'un mode headless (serveur), avancé et facultatif, qui complète l'application de bureau : un démon ``serve`` exposant le moteur en HTTP + JSON, un backend PostgreSQL multi-utilisateur avec cloisonnement par tenant (et Row-Level Security en option), une commande ``migrate`` pour transférer une base SQLite vers PostgreSQL, et un dispatcher générique ``call`` donnant accès en ligne de commande à toutes les opérations de stockage.
- Import de positions uniques depuis de nouveaux formats (eXtreme Gammon ``.xgp``, BGBlitz texte, bibliothèque native ``.db``) avec enrichissement des doublons entre formats.
- Corrections : les panneaux ne provoquent plus d'erreur lorsqu'aucune base n'est ouverte et le panneau Commentaires ne boucle plus à l'ouverture.
- Voir :ref:`headless`.

0.21.0 (2026-06-01)
~~~~~~~~~~~~~~~~~~~

- Internationalisation de l'interface : l'intégralité de blunderDB (barre d'outils, panneaux, messages, aide) peut désormais être affichée au choix en anglais, français, allemand, italien, espagnol, finnois, japonais, grec ou russe.
- Ajout d'une fenêtre de configuration, accessible par un bouton en forme de rouage dans la barre d'outils, permettant de sélectionner la langue.
- Le choix de la langue est conservé d'une session à l'autre.
- Voir :ref:`configuration`.

0.20.0 (2026-05-31)
~~~~~~~~~~~~~~~~~~~

- Ajout de la structure d'exclusion *Except* au panneau de recherche : exclut les positions contenant l'un des pions dessinés, avec marqueur « doit être vide » (double-clic) et nombre de pions par point non limité (commande x).
- Ajout de l'option « premier dé uniquement » au filtre de lancer de dés (variante D1, option CLI --dice).
- Panneau Commentaires : focus automatique du champ de saisie à l'ouverture et boutons éditer / supprimer toujours visibles.
- Correction du filtre « Search Text » qui ne trouvait pas tous les tags de commentaires.

0.19.0 (2026-05-07)
~~~~~~~~~~~~~~~~~~~

- Ajout du panneau Stats : indicateurs PR (Performance Rate), Snowie Error Rate et MWC cost (Match Winning Chance cost), barre de filtre (joueur, tournoi, dates, type de décision, longueur de match), onglet Dashboard avec cartes de niveau / PR glissant / top blunders, onglet Progression avec courbe par tournoi et scatter plot par match, onglet Erreurs avec répartition par action de videau et histogramme des magnitudes.
- Drill-down interactif vers les positions / matchs / tournois depuis tous les indicateurs.
- Toggle PR / MWC instantané.
- Commande CLI list --type stats.
- Alignement des indicateurs PR / Snowie ER / MWC sur eXtremeGammon et gnuBG (seuil 0.16 d'équité pour les cubes proches).
- Correction du calcul de cube_error pour les décisions Double/Pass.
- Documentation du modèle de statistiques (:ref:`stats_parity`).
- Voir :ref:`stats`.

0.18.0 (2026-04-20)
~~~~~~~~~~~~~~~~~~~

- Refactoring majeur du code : découpage de db.go (10k lignes) en 19 fichiers spécialisés, extraction de 7 modules de service depuis App.svelte (4888→469 lignes), consolidation des stores modaux/panneaux.
- Migration complète vers Svelte 5 runes.
- Remplacement de 9 modales de tableau par un composant générique DataTableModal.
- Ajout d'ESLint + Prettier + vitest (125 tests frontend) avec CI.
- Conformité WCAG 2.1 AA (focus visible, rôles ARIA, navigation clavier).
- Passage du mutex Database en RWMutex pour un meilleur parallélisme en lecture.
- Documentation CLI complète (CLI_USAGE.md + Sphinx FR/EN).
- Réécriture du README.
- Correction de tous les avertissements ESLint (46→0) et Vite (6→0).

0.17.0 (2026-04-20)
~~~~~~~~~~~~~~~~~~~

- Optimisation du stockage : compression zlib des données d'analyse (~80% de réduction), encodage compact des positions (~90% de réduction de la taille).
- Ajout de 5 index manquants pour améliorer les performances de recherche.
- Correction de la recherche par erreur de cube.
- Correction du mode EDIT après une recherche sans résultats.
- Restauration de l'état du panneau de recherche lors du changement d'onglets.
- Suppression de 62 instructions de débogage des chemins critiques.

0.16.0 (2026-04-18)
~~~~~~~~~~~~~~~~~~~

- Schéma de base de données v2.0.0 : déduplication des positions via hash Zobrist, colonnes de filtrage dénormalisées, préfiltre de motifs bitboard, journalisation WAL.
- Import par lot >=3x plus rapide, recherche filtrée <=100 ms sur 10k+ positions.
- NOTE : les fichiers DB créés avec la v0.16.0 ne peuvent pas être ouverts par les versions plus anciennes ; les anciennes DB sont migrées automatiquement sur place (faire une sauvegarde d'abord).

0.15.0 (2026-03-31)
~~~~~~~~~~~~~~~~~~~

- Export de la position en image PNG dans le presse-papier (board seul via Ctrl+X, ou board avec analyse via Ctrl+X Ctrl+X).

0.14.0 (2026-03-30)
~~~~~~~~~~~~~~~~~~~

- Panneau Anki dédié pour l'étude par répétition espacée (algorithme FSRS).
- Import des commentaires depuis les fichiers XG.

0.13.0 (2026-03-28)
~~~~~~~~~~~~~~~~~~~

- Simplification de l'interface : la navigation dans les matchs et les collections se fait directement via les panneaux.
- Ligne de commande intégrée dans la barre d'état.
- Panneau Console renommé en panneau Log.
- Panneau Bearoff dédié dans le panneau inférieur.
- Copier/coller de position dans le panneau de recherche.
- Glisser-déposer pour réordonner les collections, les positions dans les collections, et les matchs dans les tournois.
- Colonne tournoi dans le panneau des matchs avec édition inline.
- Affichage automatique du panneau d'analyse après une recherche.

0.12.0 (2026-03-19)
~~~~~~~~~~~~~~~~~~~

- Import de fichiers de position eXtreme Gammon (XGP) avec analyse.

0.11.0 (2026-03-06)
~~~~~~~~~~~~~~~~~~~

- Filtre de recherche dans les positions courantes.
- Ajout de filtres par match et par tournoi.
- Effacement automatique du plateau lors de l'ouverture du panneau de recherche.

0.10.0 (2026-02-25)
~~~~~~~~~~~~~~~~~~~

- Import de matchs depuis eXtreme Gammon (XG/XGP), GNUbg (SGF), Jellyfish (MAT/TXT) et BGBlitz (BGF/TXT).
- Navigation dans les matchs : parcours des coups d'un match importé, avec mise en évidence du coup joué.
- Panneau des matchs : liste, tri, édition inline, permutation des joueurs, assignation de tournoi.
- Import par dossier récursif et import par glisser-déposer.
- Calculateur EPC (Effective Pip Count) avec base de données de bearoff GNUbg intégrée.
- Collections : regroupement personnalisé de positions.
- Tournois : regroupement de matchs par événement.
- Sauvegarde et restauration de l'état de session (dernière recherche, position courante).
- Migration automatique du schéma de base de données.
- Affichage multi-moteurs dans l'analyse.
- Filtre d'erreurs/blunders du joueur 1 dans les recherches.
- Export de la base de données avec sélection granulaire (matchs, collections, tournois, coups joués).
- Bouton de navigation dans les matchs.
- Compte de course (pipcount) dans la navigation des matchs.
- Interface en ligne de commande (CLI) complète.
- Réouverture automatique de la dernière base de données.
- Amélioration de la barre d'outils et des icônes.

0.9.0 (2025-11-02)
~~~~~~~~~~~~~~~~~~

- Correction de bug de la bibliothèque de filtres.
- Import/export de base de données.
- Affichage de flèches pour les coups sélectionnés.
- Raccourcis clavier pour l'import/export.

0.8.0 (2025-05-03)
~~~~~~~~~~~~~~~~~~

- Possibilité de cacher le compte de course.
- Chargement d'une position aléatoire.

0.7.0 (2025-02-16)
~~~~~~~~~~~~~~~~~~

- Prise en charge du japonais et de l'allemand dans les exports de XG.

0.6.0 (2025-02-13)
~~~~~~~~~~~~~~~~~~

- Ajout de la bibliothèque de filtres.
- Affichage de la version de la base de données dans les métadonnées.

0.5.0 (2025-02-04)
~~~~~~~~~~~~~~~~~~

- Ajout de nouveaux filtres (miroir, non contact, jan blot, outfield blot).

0.4.0 (2025-02-03)
~~~~~~~~~~~~~~~~~~

- Résolutions de bugs divers.
- Ajout d'une icône pour blunderDB.
- Corrections de filtres.
- Ajout du support de MacOS.

0.3.0 (2025-01-27)
~~~~~~~~~~~~~~~~~~

- Résolutions de bugs divers.
- Sauvegarde automatiquement le dimensionnement de la fenêtre.
- Importe les éventuels commentaires depuis XG.

0.2.0 (2025-01-06)
~~~~~~~~~~~~~~~~~~

- Résolutions de bugs divers.
- Ajout de tables de matchs/TP/GV.
- Ajout de filtres de recherche (coups, décision de videau, date).
- Ajout de métadonnées sur les positions.
- Fonction d'import/export entre instances de blunderDB.
- Ajout de fonction de métadonnées sur les bases de données.
- Introduction des numéros de versions (base de données et application).

0.1.0 (2024-12-31)
~~~~~~~~~~~~~~~~~~

- Création version bêta.

Sommaire
========

.. toctree::
   :maxdepth: 2
   :numbered:

   telecharge_install
   guide_utilisateur
   manuel
   cmd_mode
   raccourcis
   cli
   glossaire
   faq
   roadmap
   mode_headless
   api_reference
   annexe_filtres
   annexe_windows_securite
   annexe_mac_securite
   annexe_db_scheme
   stats_parity

.. _contacts:

Contacts
========

Auteur: Kévin Unger <blunderdb@proton.me>.
Vous pouvez aussi me trouver sur Heroes sous le pseudo postmanpat.

J'ai développé blunderDB initialement pour mon usage personnel afin de
pouvoir détecter des motifs dans mes erreurs. Mais il est très agréable
d'avoir un retour surtout quand on a dépensé un paquet d'heures de
conception, codage, débuggage... Aussi n'hésitez pas à m'écrire pour
faire part de votre retour d'expérience. Tous les retours (constructifs)
sont bienvenus.

Voici plusieurs manières de discuter:

* rejoindre le serveur Discord de blunderDB: https://discord.gg/DA5PpzM9En

* m'écrire un mail à blunderdb@proton.me,

* discuter avec moi, si on se retrouve dans un tournoi,

* sur Github,

  * ouvrir un ticket: https://github.com/kevung/blunderDB/issues

  * pour des corrections de bugs ou des propositions d'amélioration,
    créer une pull request.

Faire un don
============

Si vous appréciez blunderDB et que vous voulez soutenir les développements passés et futurs, vous pouvez

* me payer un verre si on a le plaisir de se rencontrer!

* faire un petit don par PayPal à l'adresse blunderdb@proton.me

Remerciements
=============

Je dédie ce petit logiciel à ma compagne Anne-Claire et notre tendre
fille Perrine. Je tiens à remercier tout particulièrement quelques amis:

* *Tristan Remille*, de m'avoir initié au backgammon avec joie et
  bienveillance; de montrer la Voie dans la compréhension de ce
  merveilleux jeu; de continuer à m'encourager malgré mes piètres
  tentatives de mieux jouer.

* *Nicolas Harmand*, joyeux camarade depuis maintenant plus d'une dizaine
  d'années dans de chouettes aventures, et un fantastique partenaire de jeu
  depuis qu'il a choppé le virus du backgammon.

Crédits
=======

blunderDB embarque du code, des données et des polices d'autres personnes.
L'inventaire complet, avec le texte des licences, est le fichier
``THIRD_PARTY.md`` livré avec chaque paquet (``/usr/share/doc/blunderdb/``)
et présent à la racine du `dépôt
<https://github.com/kevung/blunderDB/blob/main/THIRD_PARTY.md>`__.
L'essentiel :

* le réseau de neurones ``strehl-prob5-512-512-256-128`` est l'œuvre
  d'*Alexander Strehl* (`alexstrehl/backgammon-ai-engine
  <https://github.com/alexstrehl/backgammon-ai-engine>`__, licence MIT) ; la
  recherche, le modèle de videau et la table d'équité de match qui l'entourent
  forment la configuration propre de `gammonNet
  <https://github.com/kevung/gammonNet>`__ (MIT) ;

* la table d'équité de match Kazaross-XG2 est l'œuvre de *Neil Kazaross* ;

* les tables de take points et de valeurs de gammon sont tirées de *The Theory
  of Backgammon* de *Dirk Schiemann* ;

* les bases de bearoff unilatérale (6 points, 15 pions, pour l'EPC) et
  bilatérale (6 points, 6 pions, pour les verdicts de videau en course) ont été
  générées avec `GNU Backgammon <https://www.gnu.org/software/gnubg/>`__ ;
  GNUbg est un logiciel libre sous licence GPL, et ces tables sont des données
  qu'il a produites, créditées comme telles ;

* les fichiers de match sont lus par `xgparser
  <https://github.com/kevung/xgparser>`__, `gnubgparser
  <https://github.com/kevung/gnubgparser>`__ et `bgfparser
  <https://github.com/kevung/bgfparser>`__ (MIT) ;

* côté Go : `modernc.org/sqlite <https://gitlab.com/cznic/sqlite>`__
  (BSD-3-Clause), `pgx <https://github.com/jackc/pgx>`__, `Wails
  <https://wails.io>`__ et `go-fsrs
  <https://github.com/open-spaced-repetition/go-fsrs>`__ (MIT) ;

* côté interface : `Svelte <https://svelte.dev>`__, `two.js
  <https://two.js.org>`__, `Chart.js <https://www.chartjs.org>`__ et
  `driver.js <https://driverjs.com>`__ (MIT) ;

* les polices `Nunito <https://github.com/googlefonts/nunito>`__ et `Noto Sans
  JP <https://fonts.google.com/noto/specimen/Noto+Sans+JP>`__ (SIL Open Font
  License 1.1).
