.. _cli_analyse_chapitre:

Analyse et moteur
=================

analyze — Rattrapage gammonNet
---------------------------------

Écrit une analyse gammonNet pour chaque position qui n'en a aucune — le
rattrapage d'une bibliothèque constituée avant que cette fonctionnalité
existe (`ADR-0013 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md>`__,
`ADR-0015 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0015-blunderdb-serve-operates-on-a-library-it-does-not-expose-an-evaluator.md>`__).
C'est la même opération que le déclenchement
automatique après import et la ligne « Positions sans analyse » de l'onglet
*Maintenance* de l'interface graphique, et que le point d'accès ``/v1/gammonnet.analyzeMissing`` du démon
``serve`` pour un tenant — trois formes différentes de la même opération, pas
trois logiques distinctes (voir :ref:`headless`).

.. code-block:: bash

   ./blunderdb analyze --db <path> [options]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--ply`` — Profondeur de recherche (défaut: 2, le paramètre canonique).
* ``--prune-k`` — Largeur d'élagage (défaut: 12, le paramètre canonique).
* ``--candidates`` — Nombre de coups candidats conservés par décision de
  déplacement (défaut: 10).
* ``--jobs`` — Nombre de positions analysées en parallèle (défaut: le nombre
  de cœurs de la machine). Quand il reste moins de positions que de cœurs (un
  match court, la fin du lot), chaque recherche s'étale sur les cœurs libres ;
  le résultat ne change pas d'un bit.
* ``--match`` — Restreint le lot aux positions d'un seul match (0, le défaut,
  signifie toute la bibliothèque).
* ``--compare`` — **N'écrit rien** : compare gammonNet aux analyses importées
  au lieu de combler des trous (voir plus bas).
* ``--stale`` — Réanalyse les positions dont l'analyse gammonNet est périmée —
  écrite par une version antérieure du moteur, ou à une autre profondeur que
  ``--ply`` — au lieu de combler des trous. Une position qui porte aussi une
  analyse XG, GNUbg ou BGBlitz n'est jamais touchée.
* ``--limit`` — Avec ``--compare``, s'arrête après ce nombre de positions
  (0 = toutes).
* ``--format`` — Format de sortie: ``text`` (défaut, avec la progression) ou
  ``json`` (un seul document récapitulatif, imprimé à la fin).
* ``--rollout`` — Fait un rollout des positions que choisit ``--query`` au lieu
  de combler des trous (voir plus bas).
* ``--query`` — Avec ``--rollout``, les positions à jouer, dans le langage de
  requête de la recherche (``search --query-help``) ; vide, toutes.

**Un match importé sans analyse obtient ainsi un PR.** C'est le cas d'un
match joué en ligne, ou d'un fichier Jellyfish ``.mat``, que personne n'a fait
passer par XG. blunderDB en connaissait les positions et les coups joués, mais
rien ne disait ce qu'ils valaient ; une fois le lot passé, le coup
effectivement joué est comparé au classement de gammonNet et l'écart alimente
le PR et tous les autres indicateurs. Le coup joué vient de la table des coups
du match, écrite à l'import, que le fichier ait porté une analyse ou non — il
n'est jamais deviné.

Une base analysée avec une version antérieure à celle-ci n'a pas besoin d'être
réévaluée : :ref:`repair <cli_repair>` recalcule les colonnes à partir de ce
qui est déjà en base et rend leur PR à ces matchs.

**Un seul match** (``--match``). Avec l'identifiant qu'affiche
``list --type matches``, le lot ne parcourt que les positions de ce match :
même règle du trou, mêmes garanties, portée plus étroite. Un match qui vient
d'être importé reçoit ses analyses sans que le reste de la bibliothèque soit
balayé, et un match corrigé puis analysé une seconde fois ne coûte que les
positions créées par la correction, puisque toutes les autres portent déjà une
analyse. L'option ne se combine ni avec ``--stale`` ni avec ``--compare``, qui
regardent tous deux des positions ayant déjà une analyse : demander les deux
est une erreur, plutôt qu'une portée silencieusement ignorée.

**Le parallélisme** (``--jobs``). Les positions d'un lot sont indépendantes —
aucune recherche n'informe la suivante — donc elles sont réparties sur
``--jobs`` fils d'exécution, chacun avec son propre évaluateur. Les analyses
écrites sont **identiques quelle que soit la valeur de** ``--jobs`` ; seul le
temps de calcul change. ``--jobs 1`` laisse la machine libre pour autre
chose. L'annulation n'est pas affectée : Ctrl-C arrête le lot avant toute
nouvelle position, et tout ce qui était déjà calculé est écrit.

**La règle du trou** (`ADR-0013 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md>`__). Une position portant déjà une analyse —
XG, GNUbg, BGBlitz, ou un précédent passage de gammonNet — n'est jamais
touchée, quel que soit le moteur manquant. Seule une position **sans aucune**
analyse est écrite. La commande peut donc être relancée à tout moment sans
risque, et interrompue proprement : Ctrl-C annule sans rien perdre de ce qui
est déjà écrit, et la prochaine exécution reprend exactement là où la
précédente s'est arrêtée — aucun journal n'est nécessaire, puisque « les
positions sans analyse » est recalculé à chaque lancement.

**Exemple:**

.. code-block:: bash

   ./blunderdb analyze --db base.db

   # Analyzing 1204 position(s) with gammonNet (2-ply, k=12, 16 job(s))...
   #   1/1204 (0%)
   #   61/1204 (5%)
   #   ...
   #   1204/1204 (100%)
   # Done.

   ./blunderdb analyze --db base.db --jobs 1

   # Un seul match, celui qui vient d'être importé
   ./blunderdb analyze --db base.db --match 12

**Rollouts en lot** (``--rollout``). Chaque position que choisit ``--query``
est jouée par un :ref:`rollout <cli_rollout>`, l'une après l'autre sur tous
les cœurs, et le rollout est enregistré à côté de son analyse, jamais à sa
place. La valeur est un préréglage — ``fast`` (216 parties tronquées à 7) ou
``standard`` (1296 parties tronquées à 11) — ou des réglages libres : un
préréglage facultatif, puis ``games=``, ``min-games=``, ``truncation=``,
``jsd=``, ``ply=``, ``candidates=``, ``seed=``, séparés par des virgules. Une
position qui porte déjà un rollout aux mêmes réglages est sautée : une
exécution interrompue par Ctrl-C reprend là où elle s'est arrêtée, la position
en cours étant abandonnée entière. Une position que seul un rollout analyse
est trouvée par la recherche à travers lui ; une position déjà analysée garde
les colonnes de son analyse.

.. code-block:: bash

   ./blunderdb analyze --db base.db --rollout fast --query 'E>80'
   ./blunderdb analyze --db base.db --rollout 'standard,ply=1' --query 'c'

.. _analyze_compare:

``--compare`` : **que vaut gammonNet sur votre bibliothèque ?**

La précision du moteur est mesurée ailleurs contre des corpus de référence et
contre la table de bearoff exacte. Aucune de ces mesures ne répond à la
question qu'un utilisateur se pose vraiment, qui porte sur **ses** positions :
sur les matchs importés d'XG, où le moteur embarqué est-il en désaccord avec
l'analyse venue du fichier, et que coûterait ce désaccord ?

``--compare`` répond, et **n'écrit rien**. Ce n'est pas une précaution mais
l'intérêt de la commande : l'ADR-0013 protège inconditionnellement une analyse
importée, et la comparaison peut donc se lancer sur une bibliothèque qu'on ne
veut surtout pas voir réécrite.

Le compte rendu donne :

* le **taux d'accord** sur la meilleure réponse, séparé entre coups de pions
  et décisions de videau — les deux n'ont rien à voir et un taux unique
  cacherait lequel des deux décroche ;
* le **coût du désaccord**, tarifé sur l'échelle de l'analyse importée : ce
  que le coup préféré par gammonNet vaut *selon le moteur importé*, moins ce
  que vaut son propre meilleur coup. Ce sens est le seul que les deux moteurs
  puissent chiffrer ensemble ; tarifer un désaccord deux fois inviterait à
  lire le plus petit des deux nombres ;
* la ventilation **par phase de partie**, qui est ce qui dit *où* les
  désaccords se concentrent ;
* les dix désaccords les plus coûteux, position par position.

Deux moteurs écrivent le même coup différemment — XG note « 13/7 » là où
gammonNet écrit « 13/8 8/7 », les frappes sont marquées d'un côté et pas de
l'autre, la répétition est parfois condensée en « (2) ». Ces différences sont
du dialecte et non du désaccord : la comparaison ramène les deux notations à
une forme canonique avant de les comparer. Sans cela, un corpus de test
affichait 78,8 % d'accord au lieu de 93,2 % — quinze points de faux
désaccords.

Un coup que le moteur importé n'a **pas listé** ne peut pas être tarifé sur
son échelle : il compte comme un désaccord de coût nul plutôt que d'un coût
inventé.

.. code-block:: bash

   # Comparer sur un échantillon de 500 positions
   ./blunderdb analyze --db base.db --compare --limit 500

   # compared: 118 decision(s)  (refused 2, failed 0)
   # same best answer: 93.2% (110/118)
   #   checker play:   93.7% (59/63)
   #   cube decision:  92.7% (51/55)
   # ...

.. _cli_rollout:

rollout — Rollout d'une position
---------------------------------

Joue une position un grand nombre de fois avec gammonNet, pour trancher ce
qu'une recherche ne tranche pas : deux coups à quelques millièmes, ou une
décision de videau où le modèle hésite. Avec des dés, ce sont ses coups qui
sont joués (les meilleurs à la profondeur du rollout, au moins 2 ply, ou ceux nommés par
``--move``) ; sans dés, sa décision de videau (Pas de double et Double/Prend ;
Double/Passe vaut +1 exactement). La position vient d'un XGID ou d'un OGID,
ou d'une base (``--db`` et ``--id``) ; sans ``--store``, rien n'est enregistré.

.. code-block:: bash

   ./blunderdb rollout [options] '<XGID|OGID>'
   ./blunderdb rollout --db <path> --id <position> [--store] [options]

**Options:**

* ``--preset`` — Réglage de départ : ``fast`` (défaut : 216 parties tronquées
  à 7 demi-coups, arrêt à JSD 3 après 108) ou ``standard`` (1296 parties
  tronquées à 11, arrêt à JSD 3 après 324). Les deux jouent à 0 ply — le
  réseau seul, pour les coups, le videau et les feuilles ; ``--ply 1`` ou plus
  joue plus profond, pour un temps plusieurs fois plus long. Les options
  suivantes remplacent le préréglage une à une.
* ``--games``, ``--min-games``, ``--truncation``, ``--jsd``, ``--ply``,
  ``--candidates`` — Les paramètres du rollout (``--truncation 0`` joue chaque
  partie jusqu'au bout, ``--jsd 0`` n'arrête jamais avant la fin).
* ``--move`` — Un coup à jouer, en notation blunderDB (répétable).
* ``--seed`` — Graine des dés, fixe par défaut : la même commande donne les
  mêmes nombres.
* ``--jobs`` — Parties jouées en parallèle (défaut : une par cœur) ; seul le
  temps change.
* ``--format`` — Format de sortie : ``text`` ou ``json`` (défaut : ``text``).
* ``--db``, ``--id`` — La base et l'identifiant de la position à jouer, à la
  place d'un XGID.
* ``--store`` — Enregistre le rollout terminé sur la position, comme une
  seconde analyse portant ses propres réglages, à côté de l'analyse importée
  ou évaluée, qu'il ne remplace jamais. Un rollout interrompu n'est pas
  enregistré ; de deux rollouts aux mêmes réglages, la série la plus longue
  est gardée, un rollout à d'autres réglages s'ajoute à côté.
* ``--list`` — Affiche les rollouts enregistrés sur la position, du plus récent
  au plus ancien, au lieu d'en jouer un.

Tous les candidats jouent les mêmes dés, la chance de chaque lancer est
retirée du résultat de chaque partie (réduction de variance), les deux premiers
lancers sont stratifiés et une partie s'arrête là où la base de sortie
two-sided la couvre. Chaque ligne donne l'équité, son intervalle à 95 %, le
nombre de parties jouées et la JSD, l'écart au meilleur en écarts-types de la
différence. Le videau est joué pendant les parties : le classement est plus
sûr que l'équité absolue. Ctrl-C affiche ce que les parties terminées ont
établi.

**Exemples:**

.. code-block:: bash

   ./blunderdb rollout 'XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:0:10'
   ./blunderdb rollout --move '8/5 6/5' --move '24/23 13/10' '<XGID>'
   ./blunderdb rollout --db base.db --id 42 --preset standard --store
   ./blunderdb rollout --db base.db --id 42 --list

epc — Calculatrice EPC
------------------------

Calcule l'Effective Pip Count, la probabilité de gain et le verdict de videau
money d'une position de sortie donnée par XGID ou par OGID (OpenGammon). Calcul
pur : aucun fichier de base de données n'est impliqué.

.. code-block:: bash

   ./blunderdb epc [options] '<XGID|OGID>'

**Options:**

* ``--format`` — Format de sortie: ``text`` ou ``json`` (défaut: ``text``).
* ``--bearoff-ts`` — Base bearoff two-sided optionnelle (``.bd``) élargissant
  la base intégrée TS-06-06 (également lue depuis la variable d'environnement
  ``BLUNDERDB_TS_PATH``). La base valide la plus large l'emporte ; un fichier
  invalide est ignoré avec un avertissement.

**Régimes.** Dans le domaine couvert par la base two-sided, la probabilité de
gain et l'analyse money du videau (cubeless, ND, D/T, D/P, verdict) sont
**exactes**. En dehors, la probabilité de gain est **estimée** (convolution
des distributions de lancers one-sided plus une correction calibrée) et
affichée avec sa marge d'erreur mesurée ; le verdict de videau n'est
volontairement jamais estimé (voir `ADR-0009 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md>`__).

**Exemples:**

.. code-block:: bash

   # Régime exact : six pions ou moins de chaque côté
   ./blunderdb epc 'XGID=-BBB------------------bbb-:0:0:1:00:0:0:0:0:10'

   # Avec la table TS-06-11 calculée : exact jusqu'à onze pions par joueur
   ./blunderdb epc --bearoff-ts ~/.local/share/blunderdb/gnubg_ts6x11.bd 'XGID=…'

.. _cli_cubematrix:

cubematrix — Matrice du videau
-------------------------------

Donne le verdict de videau d'une position à **tous les scores** d'un match :
pour chaque case *away × away*, si la position se double et si elle se prend.
Calcul pur : aucune base de données n'est ouverte, la position arrive par son
XGID ou son OGID (OpenGammon).

.. code-block:: bash

   ./blunderdb cubematrix [options] '<XGID|OGID>'

**Options:**

* ``--format`` — Format de sortie : ``text`` ou ``json`` (défaut : ``text``).
* ``--match-length`` — Longueur du match que la grille couvre, de 1 à 25
  (défaut : 7).
* ``--ply`` — Profondeur de recherche de chaque case, ``0`` ou ``2``
  (défaut : 2).
* ``--prune-k`` — Nombre de coups candidats retenus par le réseau d'élagage
  (défaut : 12).
* ``--jobs`` — Recherches menées en parallèle (défaut : une par cœur). La
  grille est identique quelle que soit la valeur ; seul le temps change.

Le score propre à la position est ignoré — la grille le remplace — mais son
**videau** est conservé : la question posée est « à quel score retournerais-je
*ce* videau ». La grille est post-Crawford d'un bout à l'autre.

Chaque case est une recherche à part entière, parce que le moteur tient compte
du score : une seule recherche relue à travers des équités de match
différentes serait fausse exactement là où le score compte.

**Exemples:**

.. code-block:: bash

   # Grille d'un match en 5 points
   ./blunderdb cubematrix --match-length 5 'XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:7:10'

   # Les équités de chaque case, pour un script
   ./blunderdb cubematrix --format json '<XGID>'

Sortie ``text`` : une grille dont les lignes sont les points restant à faire
au joueur au trait, les colonnes ceux de l'adversaire, puis la légende des
sigles ``ND`` / ``DT`` / ``DP`` / ``TG`` et la raison de chaque case refusée.

capture — Position depuis une capture d'écran
----------------------------------------------

Lit la position que montre une capture d'écran d'un plateau eXtreme Gammon
(thème par défaut) : pions, barre, pions sortis (ce qui manque à quinze), dés,
camp au trait et videau. L'image (PNG, JPEG ou GIF) peut contenir autre chose
que le plateau ; plus grande qu'un écran 4K (3840×2160), elle est réduite avant
la lecture. Le score n'est jamais lu. Ce que
la lecture ne peut trancher est listé comme un doute, jamais deviné : flèche
masquée, pile plus haute que les cinq pions que XG dessine côte à côte, dés
illisibles.

.. code-block:: bash

   ./blunderdb capture [options] <image>

**Options:**

* ``--format`` — Format de sortie : ``text``, ``json`` ou ``xgid`` (défaut : ``text``).
* ``--db`` et ``--save`` — Enregistre la position dans la base comme position
  individuelle.
* ``--score`` — Le score, que la capture ne montre pas : ``money``, ou les
  scores Away des deux joueurs, celui du joueur 1 (en bas) d'abord, par exemple
  ``2a/5a``.
* ``--turn`` — Le camp au trait, ``1`` (en bas) ou ``2`` (en haut), quand aucun
  dé ne le montre.
* ``--force`` — Enregistre malgré les doutes de la lecture, une fois le plateau
  vérifié. Il ne remplace ni ``--score`` ni ``--turn`` : ``--save`` refuse tant
  que le score, ou le trait sans dés, n'est pas fourni, et nomme l'option qui
  manque.

**Exemples:**

.. code-block:: bash

   ./blunderdb capture capture.png
   ./blunderdb capture --format xgid capture.png
   ./blunderdb capture --db base.db --save --score 2a/5a capture.png
   ./blunderdb capture --db base.db --save --score money --turn 2 videau.png

Le daemon offre la même lecture : ``POST /v1/positions.fromImage`` avec
``{"data": "<image en base64>"}``. Il refuse une image plus grande qu'un écran 4K
au lieu de la réduire, et ne lit que deux images à la fois.

bearoff — Bases de sortie
--------------------------

Fabrique et gère les bases de bearoff. Rien n'est téléchargé et rien n'est
embarqué : une table est calculée ici et vérifiée contre l'empreinte que gnubg
produit pour son domaine. Aucune sous-commande ne parle à une base de données —
une table de bearoff est de l'arithmétique sur le jeu, pas sur les positions de
quelqu'un — donc aucune ne prend ``--db``.

.. code-block:: bash

   ./blunderdb bearoff generate --ts <domain> [options]
   ./blunderdb bearoff list [options]
   ./blunderdb bearoff verify <file.bd> [options]
   ./blunderdb bearoff delete --ts <domain> [options]

Le domaine s'écrit comme sous ``makebearoff`` : ``6x9`` pour la table
two-sided à neuf pions par joueur, ``os8`` pour la table une face à huit
points (``os`` seul vaut ``os6``).

Les deux familles ne répondent pas à la même question. Une table **deux
faces** élargit le domaine où la probabilité de gain et le verdict de videau
sont exacts ; une table **une face** élargit la distance à laquelle un pion
peut se trouver sans que l'EPC se taise (jusqu'à dix points).

**generate.** Annonce la taille, la mémoire et le temps estimé avant de
commencer, puis affiche le pourcentage et le temps restant mesuré.

* ``--ts`` — Domaine deux faces à calculer, par exemple ``6x9``.
* ``--os`` — Domaine une face à calculer, en nombre de points : 6 à 12. Un et
  un seul des deux est requis.
* ``--cores`` — Cœurs à utiliser (défaut : tous sauf un).
* ``--data-dir`` — Où écrire (défaut : le dossier de données de l'application).
* ``--quiet`` — Pas de ligne de progression.

**CTRL-C met en pause.** Le signal est capté : l'état est écrit à côté de la
table et la même commande relancée reprend là où elle s'était arrêtée, au lieu
de tout recalculer. Une demi-heure d'arithmétique mérite d'être écrite.
``bearoff delete`` jette une reprise en attente. Seul le balayage deux faces
se met en pause ; le balayage une face est séquentiel et ``--cores`` ne lui
sert à rien.

**list.** Chiffre chaque domaine — taille, mémoire, temps sur cette machine —
et dit lesquels sont déjà présents, avec leur verdict, et lesquels ont un
calcul en pause. ``--format json`` pour un script, ``--cores`` pour changer
l'hypothèse de l'estimation.

**verify.** Répond ``verified`` (les mêmes octets que la référence),
``unverified`` (bien formée, mais aucune empreinte n'est enregistrée pour ce
domaine) ou ``corrupt`` (le fichier se contredit). Sort en erreur sur le
dernier cas : cette commande est faite pour être mise dans un script.

**delete.** Retire la table, la reprise en attente et les débris d'un calcul
mort. Un domaine par défaut est recalculé au prochain lancement de
l'application ; un domaine plus large ne l'est pas.

**Exemples:**

.. code-block:: bash

   # Ce que cette machine a, et ce que chaque domaine coûterait
   ./blunderdb bearoff list

   ./blunderdb bearoff generate --ts 6x9 --cores 4

   # OS-08 : l'EPC répond alors jusqu'à un pion sur la 8
   ./blunderdb bearoff generate --os 8

   # Sur un serveur, dans le volume que lit le démon
   ./blunderdb bearoff generate --ts 6x11 --data-dir /srv/bearoff

   ./blunderdb bearoff verify /srv/bearoff/gnubg_ts6x11.bd

.. _cli_openingbook:

openingbook — Quel coup à ce score ?
------------------------------------

L'exercice *Ouverture* du panneau Entraînement (voir :ref:`panneau_entrainement`)
en ligne de commande : le coup d'ouverture (1st roll) ou la réponse (2nd roll)
d'une partie à un score, tiré du livre d'ouverture embarqué et jugé contre lui. Tout coup à égalité avec le
meilleur est juste (voir :ref:`metrique_livre_ouverture`). Aucune base n'est
lue ni écrite.

**openingbook status** — Le nombre de lignes du livre et ses scores.

**openingbook question** — Tire une question : le jet d'ouverture et le score,
avec le XGID de la position d'ouverture ; avec ``--roll second``, le jet de
réponse dans la position que laisse un coup d'ouverture à égalité avec le
meilleur à ce score, avec ce coup et le XGID de la position de réponse.

.. code-block:: bash

   ./blunderdb openingbook question [options]

**Options:**

* ``--roll first|second`` — Le jet posé : ``first`` (l'ouverture, par défaut) ou ``second`` (la réponse).
* ``--score <liste>`` — Scores à tirer, séparés par des virgules : ``4-2``
  (le répondeur à 4 points, l'ouvreur à 2), ``1-2`` (partie Crawford), ``1-1`` (le DMP, un point partout), ``money``.
* ``--opening <liste>``, ``--reply <liste>`` — Jets d'ouverture ou de réponse à tirer (``31,65``) ; ``--reply`` avec ``--roll second`` seulement.
* ``--seed <n>`` — Tirage reproductible (0 = au hasard).
* ``--book <fichier>`` — Un livre ``.jsonl`` ou ``.jsonl.gz`` à la place du livre embarqué.
* ``--format text|json`` — Format de sortie.

**openingbook answer** — Juge un coup et affiche chaque candidat contre le
meilleur : écart, IC 95 % et « tied » (à égalité), en équité normalisée au
score, avec le nombre de parties et le réglage du rollout. Sans ``--after``,
le coup est l'ouverture ; avec ``--reply`` et ``--after``, le coup est la
réponse jouée après ce coup d'ouverture, celui que nomme la question
``--roll second``.

.. code-block:: bash

   ./blunderdb openingbook answer --opening <jet> --score <score> [--reply <jet> --after <coup>] [options] '<coup>'

**Exemples:**

.. code-block:: bash

   ./blunderdb openingbook question --score 4-2,2-4
   ./blunderdb openingbook answer --opening 31 --score 4-2 '8/5 6/5'
   ./blunderdb openingbook question --roll second --score 4-2
   ./blunderdb openingbook answer --opening 31 --reply 21 --score 4-2 --after '8/5 6/5' '13/11 6/5'
