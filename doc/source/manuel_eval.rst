.. _manuel_eval:

Évaluation
==========

.. _panneau_epc:

Panneau Eval
------------

Le panneau **Eval** (*CTRL-E*) évalue en direct la position posée sur le
plateau, quelle qu'elle soit ; sur une position de bearoff il se spécialise et
calcule en plus l'EPC (Effective Pip Count). Il est activé en appuyant sur
*CTRL-E*, en cliquant sur l'onglet Eval dans le panneau inférieur, ou en
exécutant la commande ``eval`` ; ``epc``, son ancien nom, l'ouvre aussi. Le
panneau s'est appelé *EPC*, puis *Bearoff*, avant de devenir *Eval* — c'est
donc ici qu'il faut chercher ce qu'une version antérieure appelait le panneau
Bearoff, le nom ne désignant plus que l'onglet de configuration des tables de
sortie.

Le panneau montre toujours la **seule décision** que la position posée sur
le plateau appelle — jamais deux à la fois — et les faits qui vont avec.
Chaque quantité se lit dans l'axe qui lui convient plutôt que dans un axe
unique imposé : la probabilité de gain, de gammon, de backgammon et
l'équité cubeless de chaque joueur, calculées *avant le jet*, se lisent
**par joueur** (bas, haut, puis Δ), à gauche de la décision de videau,
quand aucun dé n'est posé. Les faits et la décision restent côte à côte :
la décision de videau ne passe jamais sous les chiffres qui la justifient,
quelles que soient la langue de l'interface et la position sur le plateau. Dès que des dés sont posés, ces mêmes valeurs
*avant le jet* changent d'axe : elles se lisent **au trait**, en tête de la
liste des coups candidats, sous forme d'une ligne italique *avant le jet* —
pas un coup candidat de plus, un repère contre lequel lire chaque coup.
L'écart entre cette ligne et un coup contient la chance du jet, jamais le
mérite du coup, et elle ne porte donc aucune colonne d'erreur. Sur une
position de bearoff pur, un second tableau, toujours **par joueur** et
toujours présent, dés posés ou non, porte l'EPC, le pip count, le wastage,
le nombre moyen de lancers et l'écart type ; ces cinq colonnes ne migrent
jamais. Les deux tableaux sont empilés et partagent la même grille de
colonnes : mêmes bords, mêmes repères de colonne, une seule colonne de
pastilles — ils se lisent comme un seul objet à deux étages. Au-dessus des
tableaux, la bande d'en-tête porte le titre, le badge de régime et
l'attribution du moteur (la profondeur de la dernière évaluation y figure
aussi), puis, à droite, la case *Défi* et le bouton *Ajouter à la base*.
Tant que gammonNet n'a rien rendu, le panneau le dit en une ligne :
« Évaluation… », ou « Posez une position » sur un plateau vide.

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
mélangées dans un même tableau. L'en-tête de la colonne le dit explicitement
plutôt que de laisser deviner l'échelle : « Équité (money) » en money game,
« Équité (match) » à un score de match. Elle tient compte du **videau
vivant** : la
recherche valorise chaque position finale par le modèle de videau (Janowski,
efficacité mesurée) dans l'état du videau de la position, comme le font XG
et GNU Backgammon en évaluation *cubeful*. C'est ce qui rend visibles au
score les effets gammon-go et gammon-save — à 4-away/2-away, le joueur mené
joue 8/2 6/2 sur un 6-4 d'ouverture parce que son double précoce donnera
au gammon la valeur du match, ce qu'une évaluation sans videau ne peut pas
voir. La ligne *avant le jet*, elle, reste une équité **cubeless** : c'est
un fait de la position, pas une décision. L'évaluation n'est jamais
enregistrée : c'est un calcul, pas une analyse. Cliquer un coup
candidat l'affiche sur le plateau sous forme de flèches, exactement comme
dans le panneau Analyse. Le bouton **?** discret, dans la bande de
badges, mène au dépôt du moteur
`gammonNet <https://github.com/kevung/gammonNet>`_ ; l'attribution complète
(réseau Strehl, configuration gammonNet) figure dans les Remerciements de
l'aide.

Le panneau **roule** la position comme le panneau Analyse (voir
:ref:`rollouts`) : *Ctrl+clic* et *Maj+clic* sélectionnent des coups, le
clic droit ouvre le même menu — rollout, copie de la position et de
l'évaluation, ou de la position et des coups sélectionnés —, *Maj+R* lance ou
arrête le rollout et *Échap* l'annule ; une barre fine suit les parties
jouées, et chaque coup roulé porte son résultat dans la colonne **Rollout**
(sans dés, sous la décision de videau). Le plateau étant un brouillon, le
rollout n'est jamais enregistré : il se joue en mémoire, sans base ouverte
comme sur une base en lecture seule, et son résultat ne s'affiche que tant
que le plateau reste le même. Un rollout lancé avant une modification du
plateau ne s'affiche pas sur la nouvelle position.

Le bouton **Ajouter à la base**, en tête de la bande de badges, enregistre
dans la base la position posée sur le plateau ; *CTRL-S*, la commande ``w``
et le bouton *Enregistrer la position* de la barre d'outils font de même.
L'évaluation gammonNet affichée pour ce plateau (les coups quand les dés
sont posés, le videau sinon, avec sa profondeur) est enregistrée avec la
position, telle que l'analyse par lot l'aurait écrite : la recherche
``s i ad:gammonnet`` retrouve ces positions (en ligne de commande,
``--individual`` avec ``--analysis gammonnet``). Si la position figurait déjà dans la base avec
une analyse (importée, gammonNet à une autre profondeur, rollout), cette
analyse est conservée telle quelle et la barre d'état dit que l'évaluation
affichée n'a pas été enregistrée ; sans évaluation affichée (moteur en cours
ou position refusée), seule la position est écrite, de même si la table d'équité de
match a changé depuis le calcul, ce que la barre d'état signale. Si l'analyse
automatique gammonNet est activée, son lot démarre ensuite, comme après un
import. La barre d'état annonce le numéro de la position, y compris quand
elle figurait déjà dans la base : elle y est alors marquée comme importée
seule, et le filtre *Importée seule* (``s i``) la retrouve. Le panneau reste ouvert sur le même plateau, qui reste un plateau
brouillon : on peut déplacer un pion et ajouter la variante, et quitter le
panneau ramène à ce qu'on étudiait. Le bouton est désactivé tant qu'aucune
base n'est ouverte, ou tant que la position ne peut pas être enregistrée
(par exemple la position de départ du panneau, dont le joueur du haut a
sorti tous ses pions) ; son infobulle en donne la raison.

L'utilisateur édite la position des pions sur l'ensemble du plateau,
exactement comme en mode édition : clic gauche place un pion du joueur du
bas, clic droit un pion du joueur du haut. Le second tableau, celui de la
course, n'apparaît que lorsque la position obtenue est un bearoff pur (tous
les pions des deux joueurs dans leur jan) ; sur toute autre position, seul
le tableau des quatre colonnes communes (gain, gammon, backgammon,
cubeless) répond, et la décision porte sur les pions ou sur
un videau générique selon que des dés sont posés.

Dans chaque tableau de faits, une ligne par joueur — repérée par sa pastille
de couleur, le joueur noir étant toujours en bas. Le premier porte, tant
qu'aucun dé n'est posé, le gain, le gammon, le backgammon (probabilités,
sans le signe %) et l'équité cubeless du joueur ; le second, sur une
position de bearoff et dés posés ou non, l'EPC, le pip count, le wastage
(différence entre l'EPC et le pip count), le nombre moyen de lancers et
l'écart type.
Lorsque les deux joueurs ont des valeurs à comparer, une ligne **Δ** donne
les différences *signées* (bas − haut : négatif quand le joueur noir est en
avance). Hors position de course, poser des dés fait donc disparaître les
tableaux de faits eux-mêmes : les quatre colonnes qu'ils portaient viennent
de changer d'axe, au trait, en tête de la liste des coups.

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
  score hors de l'horizon de la table d'équité de match, c'est-à-dire un camp
  à plus de 64 points à faire ;
* *videau adverse* et *videau mort (Crawford)* — le videau ne peut pas être
  retourné. Les équités restent affichées, à titre indicatif, mais aucune
  option ne porte d'écart : une erreur, c'est ce que coûte un choix, et il
  n'y a pas de choix.

En money game, les règles **Jacoby** et **Beaver** actives sur la position
apparaissent sous le tableau de videau, en petits badges à côté du verdict
qu'elles changent : le verdict *pas de double* d'une position sous la règle
Jacoby n'est pas le même calcul que sans elle, et rien d'autre à l'écran ne
le disait.

Sous la règle **Beaver**, la ligne *double, prend* vaut la meilleure réponse
de l'adversaire autre que passer : prendre, ou *beaver* — redoubler aussitôt en
gardant le videau, la partie se jouant à quatre fois l'enjeu —, le doubleur
répondant alors par un *raccoon* (huit fois l'enjeu, videau chez lui) quand il
y gagne. Le verdict et les écarts se lisent sur cette ligne, comme sans la
règle ; au score, la règle ne s'applique pas.

Un troisième badge, **Videau max**, apparaît lorsque l'identifiant d'origine
plafonne le videau — au score comme en money game. Celui-là ne décrit pas le
calcul affiché au-dessus : l'évaluateur intégré ne modélise pas de plafond,
et le verdict est donc celui d'un videau libre. C'est justement pourquoi le
badge est là : un videau plafonné est la seule raison visible pour laquelle
blunderDB et eXtreme Gammon peuvent annoncer deux verdicts différents sur la
même position.

Le **joueur au trait** et la **position du videau** s'éditent
directement sur le plateau, comme en mode édition : cliquer le rectangle
bearoff/score d'un joueur lui donne le trait ; cliquer le videau fait
tourner centré → possédé bas → possédé haut (clic droit en sens inverse).
La valeur du videau reste épinglée — en money game les équités sont
exprimées en unités du videau courant, seul son propriétaire compte.
L'analyse est recalculée aussitôt. En régime estimé, le badge lui-même est
cliquable et ouvre directement l'onglet *Bearoff* de la configuration ; son
infobulle explique pourquoi (verdict de videau non estimable,
`ADR-0009 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md>`__) et
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

Les **dés** s'éditent enfin de la même façon, et ce sont eux qui décident de
la question posée : des dés posés font une décision de pions (la liste des
coups candidats), pas de dés une décision de videau. Clic gauche sur un dé
fait monter sa valeur (6 revient à 1), clic droit la fait descendre (1 revient
à 6) ; cliquer un dé sur un plateau qui n'en a pas en pose deux d'un coup —
un seul dé ne serait ni une décision de pions ni une décision de videau.
Cliquer le rectangle d'un joueur retire les dés pour poser une question de
videau, et le clic suivant sur un dé les remet tels qu'ils étaient.

*RETOUR ARRIERE*, ou *Effacer la position* dans le menu d'un clic droit en
dehors du plateau, efface la position : plateau vide, score money (-1, -1),
pas de dés posés — des valeurs propres au panneau Eval, différentes de
celles utilisées en mode édition (7 partout, dés 3-1), pour rester
cohérentes avec ce que le panneau affiche par défaut. *Position de départ*,
dans le même menu, pose les pions d'une partie neuve sur ces mêmes valeurs.

.. _eval_matrice_videau:

Matrice du videau
~~~~~~~~~~~~~~~~~

Une décision de videau n'est pas une propriété du damier. Les mêmes pions,
le même compte de pips, se doublent à 2-away/4-away et ne se doublent pas à
4-away/2-away ; un joueur qui a appris la réponse money n'a appris qu'une
case d'une grille. Le panneau Eval montre la case que la position porte ; la
**matrice du videau** montre la grille entière.

La commande ``cm`` l'ouvre sur la position affichée. Chaque case donne le
verdict à un score : la ligne est le nombre de points qu'il reste à faire au
joueur au trait, la colonne celui qu'il reste à faire à son adversaire. Les
quatre verdicts s'écrivent *PD* (pas de double), *DP* (double, prend), *DR*
(double, refuse) et *TB* (trop bon) ; une case que le moteur refuse porte un
point d'interrogation et dit pourquoi au survol, qui donne aussi les trois
équités de la case. Trois longueurs de match sont proposées : 5, 7 et 9
points.

La case du score que la position porte réellement est encadrée, et ses deux
en-têtes de ligne et de colonne soulignés : la lecture part de là, « ma case,
et autour d'elle ». Elle l'est dès que les deux scores *away* de la position
tiennent dans la grille affichée ; changer de longueur la déplace ou la
retire. Une position money, la partie Crawford, ou un *away* au-delà de la
grille n'en désignent aucune : il n'y a pas de case à montrer, et en montrer
une approchante serait faux.

Le score de la position est remplacé par celui de chaque case ; son
**videau**, lui, est conservé. La grille répond à « à quel score
retournerais-je *ce* videau », pas à ce que ferait une position centrée. Elle
est post-Crawford d'un bout à l'autre : pendant la partie Crawford le videau
n'est pas en jeu, et une colonne de « vous ne pouvez pas doubler » ne dirait
rien de la position.

Chaque case est une recherche à part entière. Le moteur tient compte du score
— il ne joue pas la même partie à 2-away qu'à 7-away —, donc une seule
recherche relue à travers des équités de match différentes serait fausse
exactement là où le score compte. La grille arrive d'abord en 0-ply, puis se
recalcule à la profondeur d'affichage configurée une fois la fenêtre au
repos : la même escalade que le reste du panneau, pour une grille de 9 points
qui coûte environ une seconde et demie.

La même grille se calcule hors de l'interface, avec la commande
:ref:`cubematrix <cli_cubematrix>` de la ligne de commande.

.. _panneau_reference:

Panneau Données de référence
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Le panneau **Données de référence** réunit les tables du score. Il s'ouvre par
*CTRL-MAJ-R* ou la commande ``ref``, sans base ouverte. Les commandes ``met``,
``tp2``, ``tp4`` et leurs variantes ``_live`` / ``_last``, ``gv1``, ``gv2`` et
``gv4`` l'ouvrent sur l'onglet de la table demandée. Les lignes et les colonnes
sont les points restants pour gagner le match : la ligne est le joueur dont on
parle, la colonne son adversaire. Cinq onglets :

* **MET** — la table d'équité de match Kazaross-XG2, de 1 à 15 points restants,
  en pour cent de chances de gagner le match ;
* **Points de prise** — au videau 2 (de 2 à 9 points restants) et au videau 4
  (de 3 à 9), chacun en deux grilles côte à côte : course longue (videau
  vivant) et dernier lancer (videau mort) ;
* **Valeurs du gammon** — aux videaux 1, 2 et 4 ;
* **Agrégé** — une grille de 2 à 9 points restants dont chaque case porte trois
  lignes : le point de prise au videau 2 (vivant / mort), au videau 4 (vivant /
  mort), puis les valeurs du gammon aux videaux 1, 2 et 4, en couleur atténuée.
  Un tiret marque une valeur que la table ne définit pas à ce score ;
* **Nombres de Neil** — un raccourci de tête de la MET, dû à Neil Kazaross.

**Les nombres de Neil.** L'équité du meneur s'estime par
E ≃ 50 % + (écart de score) × Neil(*n*), où *n* est le nombre de points
restants du **suiveur**. Les nombres valent 10 à 3 points restants, 9 à 4,
8 à 5, 7 à 6, 6 à 8, 5 à 11, 4 à 15, 3½ à 19 et 3 à 25 (mémo : « 8 is 6, 11 is
5, 15 is 4 ») ; entre deux, ils s'interpolent linéairement (6½ à 7, 5⅔ à 9,
5⅓ à 10, 4¾ à 12). Par exemple, à 3 points restants contre 5, l'estimation est
50 + 2 × 8 = 66 %, et la MET donne 64,8 %. L'onglet montre la ligne des
nombres, puis une grille meneur × suiveur, de 2 à 15 points restants : chaque
case donne l'estimation et, en petit, son écart avec la MET Kazaross-XG2, pour
voir où le raccourci dévie. Le raccourci ne tient pas quand le meneur est à 1
ou 2 points : la ligne du meneur à 2 points montre la valeur de la MET, en
couleur atténuée et sans écart. Source : Kit Woolsey, `Match Equity Table
<https://bkgm.com/articles/GOL/demo/equity.htm>`__, GammOnLine, 1999.

.. _eval_amener_position:

Amener une position dans le panneau Eval
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Le panneau s'ouvre par défaut sur une position de bearoff, mais l'étude part
le plus souvent d'une position déjà en main. Quatre gestes l'y amènent :

* **Clic droit sur le plateau**, dans un panneau d'analyse ou pendant la
  navigation d'un match, puis *Évaluer cette position* : le panneau Eval
  s'ouvre directement sur cette position, telle qu'elle est affichée ;
  *Évaluer le miroir de cette position* l'y ouvre vue de l'autre camp. Le
  menu contextuel n'apparaît pas dans le panneau Eval ni dans le panneau
  Recherche, où le bouton droit sert déjà à poser les pions de l'autre
  couleur.

* **Clic droit sur le plateau, puis** *Évaluer dans un nouvel onglet* : une
  nouvelle vue, nommée *Variante de #n* (n étant le numéro de la vue
  d'origine), s'ouvre dans le panneau Eval sur la position affichée, où
  poser librement des variantes. La vue d'origine garde sa position, sa liste
  et son analyse. Ce choix n'est pas offert dans le panneau Transcription.

* **CTRL-C puis CTRL-V** : copier la position depuis le panneau d'analyse,
  puis la coller une fois dans le panneau Eval. Le collage accepte aussi un
  identifiant venu d'ailleurs — un XGID (eXtreme Gammon, GNU Backgammon, une
  autre instance de blunderDB) ou un OGID (OpenGammon) : il suffit qu'il soit
  dans le presse-papier.

* **La commande** ``import XGID=…`` (ou ``import OGID=…``) pour le cas où
  l'identifiant n'est pas dans le presse-papier mais dans un message, sur un
  forum lu dans un terminal, ou produit par un script. C'est le même verbe
  qu'``import`` tout court : sans argument il ouvre un sélecteur de fichiers,
  avec un argument il lit l'identifiant. Le chemin est ensuite identique à
  celui du collage — même lecture, même déduplication, même ouverture de la
  position importée.

Un OGID ne porte qu'une position : ni évaluation, ni commentaire. La position
arrive donc sans analyse, exactement comme un XGID nu, et l'évaluateur intégré
peut la combler ensuite.

Dans un OGID, la partie Crawford se lit au ``C`` qui suit la longueur du match
(``7C``) : sans lui, un joueur à un point du but est après la Crawford.

**Une capture d'écran du plateau** (eXtreme Gammon, thème par défaut) se lit de
trois façons : coller l'image (*CTRL-V*), la glisser sur la fenêtre, ou l'ouvrir
par *CTRL-I* (``.png``, ``.jpg``). Aucune base n'est nécessaire. La position lue
arrive sur le plateau d'édition, et elle n'est qu'une proposition : les points et
le videau dont la lecture n'est pas sûre sont cerclés d'orange ; un doute qui ne
se cercle pas (le sens du plateau quand les numéros des points sont hors de
l'image, des dés illisibles) et ce qu'une capture ne montre jamais (le score, le
camp au trait quand les dés sont cachés) sont nommés dans la barre d'état. Corrigez le plateau comme d'habitude, puis
enregistrez par le geste habituel : rien n'est enregistré avant. Une image où
aucun plateau connu n'est reconnu est refusée, avec le motif dans la barre
d'état. La même lecture existe en ligne de commande (``capture``).

Le plateau du panneau Eval est un brouillon : la position y arrive sans son
identifiant de base, de sorte qu'aucune modification faite ici ne peut
réécrire l'enregistrement dont elle provient. Toutes les éditions habituelles
du plateau y restent disponibles (pions, videau, dés, score), et l'évaluation
suit chaque modification.

Dans l'autre sens, *CTRL-C* copie le plateau du panneau Eval dans le
presse-papier, avec un XGID recalculé à partir des pions posés — donc collable
directement dans eXtreme Gammon ou dans une autre instance de blunderDB. Seule
la position voyage : l'évaluation affichée par le panneau n'est pas un
enregistrement de la base et n'accompagne pas la copie.

En quittant le panneau Eval, la position consultée auparavant est restaurée :
le brouillon n'est jamais enregistré tout seul.

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

**Élargir le domaine exact.** La table calculée au premier lancement couvre
6 pions par joueur. Deux moyens d'aller au-delà, dans l'onglet *Bearoff* de la
configuration :

* calculer une table deux faces plus large — jusqu'à TS-06-15 si la machine a
  la mémoire pour. L'onglet annonce la taille, la mémoire et le temps sur cette
  machine avant de commencer, et le calcul se met en pause et se reprend. Un
  calcul annulé laisse un fichier ``.part`` qui n'est jamais lu comme une
  table ;

* indiquer un fichier ``.bd`` two-sided de gnubg quelconque. La base au
  domaine le plus large l'emporte automatiquement.

**Le plateau du panneau est un brouillon, et il est retenu.** Quitter le
panneau Eval puis y revenir retrouve la position sur laquelle on l'a laissé,
et non le plateau de sortie par défaut : ce dernier n'est servi qu'à la
première ouverture de la session. Envoyer une position de la base vers le
panneau l'emporte sur ce souvenir, et *RETOUR ARRIERE* rend le plateau par
défaut à tout moment. Rien n'est enregistré dans la base au passage — le
brouillon n'a pas d'identité de position, et son évaluation est recalculée à
l'arrivée plutôt que transportée.

**Mode défi.** La case *Défi*, dans la bande d'en-tête, active un mode
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

.. _eval_sans_base:

**Sans base ouverte.** Le panneau Eval n'a pas besoin de base : une fois
l'écran d'accueil écarté, on y pose, colle (*CTRL-V*) et modifie une position,
on l'évalue, on la roule, on la copie en texte (*CTRL-C*) ou en image
(*CTRL-X*, et *CTRL-X CTRL-X* avec l'évaluation). Le clic droit dans le
panneau propose aussi *Enregistrer l'image (SVG)…* et *Enregistrer l'image
(PNG)…*. Le plateau de l'onglet Recherche est lui aussi un brouillon : il
s'édite et reçoit un collage sans base, et l'on passe de l'un à l'autre sans
rien perdre ; lancer la recherche, elle, demande une base. Seuls *Ajouter à la
base* et *CTRL-S* exigent une base dans le panneau, et la barre d'état le dit.
Aucune base n'est créée en coulisse. Dans le panneau Eval comme dans l'onglet
Recherche, la commande ``import XGID=…`` (ou ``import OGID=…``) pose la position
sur le plateau brouillon, sans base. L'onglet **Entraînement** s'ouvre lui
aussi sans base : *Scores*, *Pions*, *Bearoff* et *Évaluation* sont
disponibles, *Décision* est désactivé avec la raison affichée, et la session
terminée n'est pas enregistrée (le journal l'indique). La Direction de tournoi
exige une base ouverte.

Pour fermer le panneau Eval, appuyer sur *CTRL-E* ou basculer sur un autre onglet.

.. _epc_methodologie:

Méthodologie et hypothèses du panneau Eval
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Chaque valeur affichée par le panneau repose sur des hypothèses précises,
énoncées ici exhaustivement.

**Domaine.** La *zone course* — probabilité de gain et verdict de videau — ne
traite que le bearoff pur : tous les pions restants des deux joueurs dans leur
jan intérieur. La position est évaluée *avant le lancer* ; les dés
éventuellement posés sont ignorés.

Les **blocs EPC**, eux, vont plus loin : un camp obtient son
EPC dès que son pion le plus éloigné tient dans la table une face chargée. Avec
la table par défaut (six points) c'est l'ancienne règle du jan ; avec une table
à huit points, calculée depuis l'onglet *Bearoff*, un camp dont un pion est sur
la 8 est traité comme les autres. Rien n'est extrapolé : un pion un point trop
loin n'a simplement pas d'EPC, exactement comme un pion sur la 7 n'en avait pas
avant. Quand la table qui a répondu n'est pas celle à six points, son nom
apparaît dans le coin du bloc course (« OS-08 ») — sans lui, on lirait « six »
par défaut et on croirait le camp entièrement rentré.

**Blocs EPC (toujours exacts).** L'EPC, le nombre moyen de lancers et
l'écart type proviennent de la distribution exacte du nombre de lancers pour
sortir tous les pions, lue dans la base one-sided de GNUbg (6 à 10 points,
15 pions, calculée sur la machine). EPC = lancers moyens × 49/6 (49/6 ≈ 8,167
est la moyenne exacte de pips par lancer, doubles comptés quatre fois) ;
wastage = EPC − pip count. L'unique idéalisation est le *jeu one-sided
optimal* : chaque joueur minimise ses propres lancers en ignorant
l'adversaire — c'est la définition standard de l'EPC.

**Probabilité de gain, régime exact.** Lecture directe dans la base two-sided
disponible la plus large (TS-06-06 calculée au premier lancement, fichier
externe, ou TS-06-11 calculée depuis l'onglet *Bearoff*). Ces bases résultent d'une analyse rétrograde complète sous jeu
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
probabilité de gain provient de la sortie brute de gammonNet (recherche 0-ply,
puis à la profondeur configurée, jamais lue dans une table), et le verdict d'un
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

Cette mesure porte sur des décisions **money**, en course. Le verdict au score
de match — que ce régime est seul à savoir rendre — et les positions de contact
n'ont pas de mesure publiée : ce qui précède ne se transporte pas à ces cas.

**Pourquoi pas plus profond que 2 plis ?** Parce que la mesure dit que cela ne
rapporte rien. Une décision de pions à 3 plis évalue le réseau **14 fois plus**
qu'à 2 plis (159 000 évaluations contre 11 400, moyenne sur vingt décisions
réelles) et prend environ 14 fois plus de temps (3,9 s contre 0,27 s sur un
seul cœur de la même machine). Sur quarante décisions réelles rejouées aux
deux profondeurs, la recherche plus profonde a changé d'avis **deux fois**, et
les deux fois le gain qu'elle s'attribuait à elle-même valait au plus 0,0002
d'équité normalisée : deux ordres de grandeur sous 0,020, le seuil à partir
duquel eXtreme Gammon parle d'erreur. Par décision, tous cas confondus, le gain
est de 0,0000.

Les 2 plis restent donc le défaut ; 3 et 4 plis se choisissent dans l'onglet
*gammonNet* de la configuration. À 3 et 4 plis, comme dans GNUbg, seule la
racine compare plusieurs coups : à chaque nœud intérieur, la réplique retenue
est la meilleure à 0 pli, seule approfondie. Il ne s'agit pas de dire que 3 plis ne vaut
rien en général, mais que sur *ce* réseau, avec ce filtre, il ne paie
pas l'attente de quelqu'un devant un panneau. La mesure est reproductible
(``TestThreePlyMeasure``) et la conclusion se rejugera si le réseau change.

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

.. note:: Les bases de bearoff sont des tables mathématiques immuables.
   blunderDB les calcule lui-même, à l'identique de l'outil ``makebearoff`` de
   GNUbg — octet pour octet — dans l'onglet *Bearoff* de la configuration ou
   avec ``blunderdb bearoff generate``.

