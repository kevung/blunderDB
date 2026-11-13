.. _manuel_partage:

Import, origine et partage
==========================

.. _import_regles:

Import : ce qui est écrit, ce qui ne l'est jamais
-------------------------------------------------

Importer un match, une position ou une autre base ajoute ce qui manque ; cela
ne remplace pas ce qui est déjà là.

* **Une position n'est jamais dupliquée.** C'est son identité — pions, videau,
  dés, score — qui la reconnaît, jamais le fichier d'où elle vient : la même
  position rencontrée dans deux matchs reste une seule ligne.

* **Importer une base relit tout ce qu'un export écrit.** Les positions, avec
  leurs analyses et commentaires, mais aussi les matchs (parties et coups
  compris), les tournois avec leur Direction et leur Rencontre, les brouillons
  de transcription, les alias, les collections, les leçons et les paquets
  Anki : voir :ref:`guide_fusionner_base`.
  Rien n'est écrasé : un match déjà présent n'est pas recopié, un tournoi du même
  nom reçoit les matchs sans changer la date de l'événement (sa date de dernière
  modification avance, puisqu'il reçoit de nouveaux matchs), un tournoi qui a
  déjà sa Direction la garde et reste hors de la Rencontre importée, un match
  déjà rangé dans un autre tournoi ou une autre place du tableau y reste, et ce
  que la base décidait déjà reste décidé par elle. Les tournois, Directions, Rencontres, brouillons de transcription et
  alias créés par l'import gardent leurs dates d'origine ; les collections,
  leçons et paquets Anki créés n'en gardent pas.

* **Une analyse par moteur.** eXtreme Gammon, GNUbg, BGBlitz et l'évaluateur
  embarqué cohabitent sur une même position, et le panneau Analyse indique
  l'origine de chacune. Importer l'une n'efface pas l'autre.

* **Une analyse importée n'est jamais recalculée.** blunderDB la range telle
  quelle, avec son étiquette de niveau (« 3-ply », « XG Roller++ », « Book »),
  ses équités, ses erreurs, ses probabilités et la chance du lancer. La règle
  est « une évaluation ne comble qu'un trou » : l'analyse automatique après
  import ne visite que les positions sans **aucune** analyse, et
  la réanalyse des analyses gammonNet périmées laisse intacte toute position portant
  une analyse importée (voir :ref:`configuration`).

* **Réimporter un match déjà présent n'apporte que du plus profond.** Le match
  est reconnu à son jeu (joueurs, longueur, dés, coups, videau), pas à son
  analyse : aucun match, partie ni coup n'est réécrit. Les marques posées dans
  le logiciel d'origine sont ajoutées, et une analyse plus profonde que celle
  rangée la remplace, position par position — une version Roller++ du même
  match remplace la version 3-ply, quel que soit l'ordre d'import ; à
  profondeur égale ou moindre, l'analyse rangée reste. Réimporter le même
  fichier ne réécrit donc rien. Le rapport d'import distingue les doublons qui
  n'apportaient rien de ceux qui ont approfondi des analyses. En ligne de
  commande, ``--skip-duplicates`` ignore un doublon sans rien en reprendre
  d'autre que les marques. Un match tronqué puis complété (plus de parties)
  n'est pas le même match : il est importé comme un second match.

* **Une analyse illisible n'empêche pas l'import du match.** Une décision dont
  l'analyse contient une valeur qui n'est pas un nombre fini (NaN ou infini,
  que portent certains fichiers XG) est importée sans cette analyse ; le reste
  du match entre normalement, et le rapport d'import compte les décisions
  concernées.

* **Un match déjà présent sous d'autres noms est signalé, jamais fusionné.**
  Les deux empreintes de match portent les noms des joueurs : « Martin A. » et
  « Alice Martin » font deux matchs. L'import compare aussi les dés (longueur,
  score initial, dés de chaque partie) et signale, sous la ligne du fichier et
  dans le rapport, le match déjà en base sous d'autres noms dont il a les dés.
  ``blunderdb repair --duplicates`` liste ces paires dans une base existante,
  ainsi que les matchs tronqués et leur version plus longue.

* **Un nom connu sous une autre graphie est enregistré sous son nom canonique.**
  Un *alias* dit que « Martin A. » est une autre graphie d'« Alice Martin » (ou
  qu'un nom d'événement en est un autre). À l'import, les noms des joueurs et de
  l'événement sont remplacés par leur nom canonique, et le match rangé dans le
  tournoi de l'événement canonique. Les empreintes du match gardent les noms du
  fichier : un fichier importé avant que son alias existe se reconnaît donc
  toujours. Un match dont les dés sont ceux d'un match en base, et dont les noms
  ne diffèrent que par des alias connus, n'est pas un second match : ses
  analyses enrichissent celui qui est rangé.

Les alias se gèrent dans l'onglet **Corpus** des paramètres : choisir
**Joueurs** ou **Événements**, saisir l'alias et le nom canonique, ou retirer un
alias de la liste. **Proposer** liste les noms qui ne diffèrent que par la
casse, les accents, la ponctuation ou l'ordre des mots ; rien n'est appliqué
sans un clic sur **Appliquer**. Le même onglet porte la case **Ignorer les
doublons à l'import** (l'équivalent de ``--skip-duplicates``, pour la session)
et la recherche des **doublons probables** d'une base existante, celle de
``blunderdb repair --duplicates`` : chaque paire est donnée par ses numéros de
match, rien n'est fusionné. Sous une paire aux mêmes dés, **Créer ces alias**
enregistre en un clic les alias de joueur qui feraient nommer aux deux matchs
les mêmes joueurs, la graphie du match le plus récent devenant l'alias : une
seule proposition quand un nom est commun aux deux, sinon deux — siège pour
siège, puis croisée — dont on choisit celle qui dit qui est qui. En ligne de commande : ``blunderdb players alias``
et ``blunderdb events alias``.

* **Un dossier s'importe en parallèle.** Les fichiers sont lus sur plusieurs
  cœurs à la fois et écrits par groupes, toujours dans l'ordre du dossier : les
  numéros de match ne dépendent pas de la machine. Un fichier identique, octet
  pour octet, à un fichier déjà lu du même dossier n'est pas relu : il est
  compté comme doublon. Annuler arrête l'import au groupe en cours ; ce qui
  était déjà écrit reste.

* **La progression se lit en positions par seconde.** La fenêtre d'import
  donne le pourcentage lu, le débit, le temps restant estimé et les comptes en
  cours (importés, doublons, en erreur). **Réduire** la range dans la barre de
  statut, d'où une pastille la rouvre : l'import continue pendant que vous
  travaillez, et la fenêtre revient d'elle-même avec le rapport à la fin. Les
  cent premières erreurs sont listées ; les suivantes sont comptées, et le
  journal de l'application les nomme toutes. En ligne de commande,
  ``blunderdb import --type batch`` affiche la même progression sur la sortie
  d'erreur, et ``--format json`` la rend dans l'objet final (``progress``).

* **Chaque fichier d'un import est journalisé, et un import interrompu se
  reprend.** Pour chaque fichier, le journal du lot garde le chemin, la
  taille, la date de modification, l'empreinte SHA-256 et le résultat : match
  nouveau (avec son numéro), doublon (avec le match qui le couvre), match
  enrichi, ou erreur (avec le message). Un import annulé ou coupé se continue
  sans reprendre ce qui est déjà décidé : un fichier de même chemin, même
  taille et même date est sauté sans être lu ; un fichier de même contenu est
  lu mais pas analysé ; un fichier en erreur est retenté. Dans l'application, la fenêtre de fin d'import liste le journal (un
  fichier par ligne, avec son résultat et le message d'une erreur ; un
  fichier retenté montre sa dernière issue), et une ligne qui a donné un match
  l'ouvre. Un import **annulé** laisse la fenêtre ouverte avec le bouton
  **Reprendre**. En ligne de commande, ``blunderdb import --type batch --dir
  <dossier> --resume <lot>`` reprend le lot dont le numéro a été affiché au
  départ de l'import, et ``--format json`` rend le journal dans l'objet final
  (``journal``). Le serveur accepte ``resume`` dans la requête de
  ``imports.batch`` et rend le journal par ``imports.files``. Le journal est
  une donnée de l'import : ouvrir ou lire une base n'y écrit rien.

* **Un import coupé par l'arrêt de l'application se reprend depuis la
  fenêtre.** La commande ``:resume`` liste les imports que rien n'a terminé ;
  on choisit celui à continuer, puis on désigne à nouveau le dossier ou les
  fichiers, comme ``--dir`` en ligne de commande. Le journal du lot décide
  de ce qui n'est pas relu.

* **Un gros dossier s'importe en mode masse.** À partir de 200 fichiers,
  blunderDB écrit avec un cache plus grand et moins de points de contrôle. Si
  la base ne contient encore aucune position, il va plus loin : les index de
  recherche ne sont reconstruits qu'à la fin, et les écritures ne sont plus
  synchronisées sur le disque. Une coupure de courant pendant un tel import
  peut alors abîmer la base : il faut la recréer et relancer l'import. Un
  arrêt brutal du programme, lui, laisse seulement des index absents, que
  l'ouverture suivante reconstruit (le journal le signale).

* **Ce que blunderDB n'écrit jamais** : une chance recalculée — elle est lue
  dans le fichier source, ou reste inconnue — et un rollout qu'il n'a pas
  lancé lui-même : les données d'un rollout contenues dans un fichier ``.xg``
  ne sont pas ouvertes. Seuls les rollouts que blunderDB produit
  (:ref:`rollouts`) sont stockés, à côté de l'analyse.

.. _panneau_metadata:

Panneau Métadonnées
-------------------

Le panneau **Métadonnées** (*CTRL-M*) affiche les informations générales de
la base de données courante : *Utilisateur*, date de création (*Créé*),
*Version* du schéma et *Description*. L'utilisateur, la date et la description
se modifient sur place et s'enregistrent en quittant le champ ; la version est
en lecture seule. Accessible aussi via la commande ``meta``.

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

La fenêtre d'export tient en un seul écran : une bande d'en-tête — le nombre
de positions exportées, **Annuler** et **Exporter…** — puis une grille, une
ligne par famille avec son compte. Ces comptes sont exacts : ils sont lus
auprès du serveur à chaque changement de case, sur la même sélection que
l'export, et sont ce que le fichier contiendra (le nombre de positions de la
bande d'en-tête, les commentaires, les matchs, les transcriptions, les alias,
les tournois, les collections, les leçons, les paquets, les filtres). Le fichier
se choisit en dernier, après **Exporter…** ; fermer ce choix sans nom revient à
la grille. Une progression
se superpose ensuite le temps de l'écriture, la fenêtre se ferme d'elle-même et
le résultat s'affiche dans la barre d'état.

Neuf points méritent l'attention :

* **La portée** : depuis une liste — une recherche, une collection —, la ligne
  *Portée* choisit entre la *Liste courante* et *Toute la bibliothèque*. La
  liste courante exporte ses positions, et la ligne *Matchs* ne propose que
  ceux où ces positions ont été jouées ; chaque match coché part avec toutes
  ses positions. Depuis la bibliothèque, la portée est la bibliothèque.
* **Toute la bibliothèque** est l'export du serveur et de
  ``export --type database`` : toutes les familles sont cochées d'emblée, les
  paquets Anki avec leur historique de révision, et les transcriptions en
  cours partent avec le reste.
* **La synchronisation** : avec *Toute la bibliothèque*, la ligne
  *Synchronisation* propose *Seulement ce qui est nouveau depuis le dernier
  export vers ce dossier* et le bouton **Choisir le dossier…**. Le fichier
  n'est alors plus demandé : il est écrit dans ce dossier sous un nom neuf, et
  les comptes sont ceux de ce qui est nouveau (voir
  :ref:`synchroniser_deux_machines`).
* **Un match part avec son brouillon de transcription**, sous la case
  *Transcriptions* de la ligne *Matchs*, cochée d'emblée ; les matchs qu'un
  tournoi emporte aussi. Un brouillon qui n'a pas encore produit de match ne
  part qu'avec la bibliothèque entière. Un match ou une transcription liés à
  une vidéo locale n'emportent que le nom du fichier, sans son dossier, que le
  destinataire relocalise sur sa propre copie ; un lien web part tel quel. Le
  fichier d'où un match a été importé ne part aussi que par son nom.
* **Un match part avec les alias de ses joueurs et de son événement** : la
  case *Alias*, cochée d'emblée, fait suivre la fusion de joueurs, et le
  destinataire lit un joueur fusionné comme un seul. Un alias qui ne nomme
  personne des matchs exportés reste chez vous.
* **Une collection dont toutes les positions ne sont pas dans la liste arrive
  tronquée.** La liste des collections affiche, pour chacune, la part couverte
  (« 12/40 ») et la signale en rouge lorsqu'elle est partielle. La case
  *+ leurs positions* emporte les positions des collections cochées : celles
  d'une liste faite à la main, celles que la requête d'une collection vivante
  sélectionne. Une collection vivante dont la requête désigne des lignes que le
  fichier ne contient pas **part figée** : elle arrive comme la liste de ce que
  sa requête sélectionne aujourd'hui, et la liste la marque « figée » en rouge
  avec une note ; cocher *+ leurs positions* la laisse vivante.
* **Un tournoi part toujours avec ses matchs** : sans eux, le lien
  tournoi–match n'existe pas et le tournoi arriverait vide. Cocher un tournoi
  emporte ses matchs, même si la case des matchs est décochée ou restreinte à
  d'autres. Il part aussi avec sa Direction — journal, paires, tables et
  rattachement des matchs exportés — et sa Rencontre part dès qu'un de ses
  tournois part ; le dossier de sortie de la Direction reste chez vous. La
  Direction se retrouve en ouvrant le fichier, ou en l'important dans une
  autre base.
* **Un paquet Anki part avec ses positions**, en copie d'étude neuve. La case
  *avec mon historique* emporte en plus l'état de chaque carte et le journal
  de ses révisions : une sauvegarde, ou vos paquets passés sur une autre
  machine. Elle est décochée pour une liste, cochée pour la bibliothèque.
* **Les filtres enregistrés** n'exportent pas du contenu mais vos propres
  recherches, sans utilité dans la base de quelqu'un d'autre ; la case est
  décochée pour une liste. Un filtre qui désigne une position, un match ou un
  tournoi par son numéro est renuméroté pour le fichier ; s'il en désigne un
  qui ne part pas, il reste chez vous.

Les collections, leçons, transcriptions, paquets Anki, alias et tournois partent
avec les dates de création et d'ajout qu'ils portent chez vous, non avec la date
de l'export.

Les champs *Utilisateur*, *Description* et *Date de création* décrivent le
**fichier produit** ; ils sont préremplis depuis la base source.

Cocher **Filigrane** fait apparaître deux champs :

* **Origine** — ce qu'est ce fichier et d'où il vient, dans vos mots :
  « Cours de Jean Dupont — 12 mars 2026 ». Ce champ est **obligatoire** : tant
  qu'il est vide, il est bordé de rouge et le bouton d'export reste inactif.
* **Note**, facultative — conditions d'utilisation, adresse de contact, une
  demande de ne pas rediffuser.

La marque est signée avec votre identité d'émetteur. Elle est donc
**inaltérable et infalsifiable** : nul ne peut la modifier, ni en fabriquer une
à votre nom. Elle n'est en revanche **pas ineffaçable** — le fichier distribué
est une base SQLite ordinaire, et blunderDB est un logiciel libre. Elle
n'empêche rien : elle dit d'où vient le fichier.

Les commentaires que vous avez écrits sont signés avec la même identité : le
destinataire voit qu'ils viennent de vous, et s'ils ont été altérés en chemin
(voir :ref:`panneau_commentaires`). Un commentaire qui porte déjà la signature
d'un autre émetteur la garde. La barre d'état dit, à la fin de l'export, combien
de vos commentaires ont été signés ; un export sans marque n'en signe aucun.

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

Cocher **Mot de passe** produit un fichier d'extension ``.dbx`` — y compris si
le nom choisi dans la fenêtre d'enregistrement finit en ``.db``. Pour
l'ouvrir, utilisez l'ouverture de base habituelle : la fenêtre de sélection
accepte aussi bien les ``.db`` que les ``.dbx``. blunderDB demande
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

* « ✓ marquée par vous » : le fichier porte votre marque,
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

.. _partager_une_base:

Publier une base pour d'autres
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Une base marquée se distribue comme n'importe quel fichier — courriel, site
personnel, clé USB. blunderDB **ne fournit aucun service** : ni dépôt, ni
catalogue hébergé, ni compte. C'est une conséquence directe de sa conception :
rien n'est jamais enregistré du côté de celui qui reçoit un fichier, et il n'y
aurait donc rien à faire remonter à un service, même s'il en existait un.

Ce qui rend une base publiée utilisable par quelqu'un d'autre tient à quatre
champs, tous déjà là :

* **Utilisateur** — qui l'a constituée, sous le nom que vous voulez voir cité.
* **Description** — ce que la base contient, en une phrase qui tienne dans une
  liste : « 240 décisions de videau au score, commentées, niveau intermédiaire ».
* **Origine** (du filigrane) — ce qu'est ce fichier et pour qui il a été
  produit. C'est ce que le destinataire lit en premier dans le panneau
  *Métadonnées*.
* **Empreinte d'émetteur** — publiez-la à côté du fichier, pas dedans : c'est
  en la comparant que le destinataire vérifie que le fichier vient de vous et
  non de quelqu'un qui a repris votre nom.

Une base publiée sans filigrane reste parfaitement utilisable ; elle est
simplement anonyme, et le panneau *Métadonnées* n'affiche alors aucune section
*Origine*.

Pour faire connaître une base, la catégorie *Show and tell* des `discussions du
dépôt <https://github.com/kevung/blunderDB/discussions>`_ sert d'annuaire :
c'est une liste tenue par ceux qui publient, pas un service rendu par
blunderDB. Y annoncer une base demande le lien, les quatre champs ci-dessus et
l'empreinte.

