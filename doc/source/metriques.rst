.. _metriques:

Métriques et outils d'étude
===========================

Chaque chiffre que blunderDB affiche sur votre jeu répond à une question.
Cette page les réunit : pour chacun, la question à laquelle il répond et ce
qu'il apporte à l'étude, sa définition, sa formule exacte, ce qui compte comme
décision, son unité, sa lecture, son incertitude, ses limites et ce qui le
distingue des chiffres d'eXtreme Gammon (XG) et de GNU Backgammon (gnuBG). Les
écrans qui les montrent sont décrits dans le :ref:`manuel` ; les liens y
mènent.

Les seuils et les paramètres de cette page ont été fixés avant tout examen de
résultats réels : les ajuster après coup reviendrait à faire dire aux chiffres
ce qu'on attend d'eux.

.. contents::
   :local:
   :depth: 1


.. _metrique_erreur:

L'erreur et la perte d'une décision
-----------------------------------

Toutes les mesures partent de deux grandeurs par décision.

**L'erreur e** est ce que la décision jouée a coûté en équité, comparée à la
meilleure décision de l'analyse importée (XG ou gnuBG ; blunderDB ne
réanalyse pas les matchs pour ses statistiques). Elle s'exprime en **équité
normalisée** : ±1 vaut le videau courant, à l'argent comme au score de match
(EMG au score). Elle est arrondie au millième (le millipoint, mpt).

* Coup de pions : équité du meilleur candidat moins équité du coup joué.
* Pas de double : l'erreur de « pas de double » donnée par l'analyse.
* Double : la plus petite des erreurs de « double, prise » et « double,
  refus » — le doubleur est jugé sur la meilleure réponse adverse.
* Prise ou refus : l'écart entre la réponse jouée et la meilleure des deux.

L'erreur est celle **du coup joué dans ce match**. Une position atteinte dans
deux matchs, et jouée différemment, porte deux erreurs ; aucune n'écrase
l'autre. Un coup joué que l'analyse ne nomme parmi aucun de ses candidats n'a
pas d'erreur : il n'est pas noté, et il sort de tous les comptes.

**La perte de MWC ℓ** est la même erreur exprimée en chances de gagner le
match (*Match Winning Chances*). Au score et au videau v de la décision :

.. math::

   \ell = e \times \frac{\mathrm{MWC}_{+v} - \mathrm{MWC}_{-v}}{2}

où :math:`\mathrm{MWC}_{+v}` et :math:`\mathrm{MWC}_{-v}` sont les chances de
gagner le match si le joueur gagne, ou perd, v points à ce score. La table
d'équité de match (MET) est la Kazaross-XG2, prolongée au-delà de 25 points
par la formule de Zadeh ; un score post-Crawford est lu comme tel. Une prise
ou un refus est converti au videau **d'avant** le double, comme le doubleur et
comme XG : convertie au videau doublé, une erreur de réponse coûterait deux
fois trop. Une partie d'argent n'a pas de MWC : ℓ n'y existe pas.

Ces deux grandeurs sont additives. Toutes les métriques ci-dessous en sont des
sommes, rapportées à des dénominateurs différents, ou comparées à une
référence.


.. _metrique_pr:

PR (*Performance Rating*)
-------------------------

**La question.** Quel est mon niveau, sur l'échelle que tout le monde
connaît ? Le PR est le chiffre des classements, des forums et des clubs ; il
permet de se situer et de comparer deux matchs analysés par XG.

**La formule.** Sur les décisions comptées du joueur :

.. math::

   \mathrm{PR} = 500 \times \frac{\sum_i e_i}{N_\text{compté}}

Le facteur 500 est la convention d'XG : un PR de 5,0 vaut une erreur moyenne de
0,010, soit 10 mpt par décision comptée. Le PR est une moyenne, sans unité ;
plus il est bas, mieux on a joué.

**Ce qui compte comme décision.** Les règles sont celles d'XG, vérifiées
match par match contre ses chiffres (:ref:`stats_parity`) :

+-----------------------------+-----------------------------------------------+
| Décision                    | Comptée si                                    |
+=============================+===============================================+
| Coup de pions               | il n'est pas forcé et le coup joué est noté   |
+-----------------------------+-----------------------------------------------+
| Double proposé              | toujours                                      |
+-----------------------------+-----------------------------------------------+
| Prise, refus                | toujours                                      |
+-----------------------------+-----------------------------------------------+
| Pas de double               | ND > 0 et ND − min(D/T, D/P) < 0,200          |
+-----------------------------+-----------------------------------------------+

* Un coup de pions est **forcé** quand le lancer n'offre qu'un coup légal, ou
  quand l'analyse couvre tous les coups légaux (comptés par le générateur de
  coups de blunderDB) et que tous valent la même équité au millième près :
  un choix qui ne coûte rien, quel que soit le coup, n'est pas une décision.
  C'est la règle d'XG pour les coups « non forcés mais sans enjeu » (fin de
  course décidée, sortie où tout se vaut). Quand les coups légaux ne peuvent
  pas être comptés, seul un candidat unique rend le coup forcé.
* Un **pas de double** est compté quand le joueur était près de doubler : son
  équité sans doubler (ND) est positive et dépasse de moins de 0,200 la
  meilleure issue du double pour l'adversaire, min(D/T, D/P). Mesurer contre
  ce minimum plutôt que contre D/T seul garde comme décision une position
  « trop bonne » tant qu'elle reste à moins de 0,200 de l'encaissement.
* La condition **ND > 0** écarte les positions où le joueur au trait est
  derrière : il n'a aucun double à envisager. XG les écarte aussi, ce que son
  manuel ne dit pas ; les matchs de référence le montrent.
* Une décision de videau sans analyse de videau n'est jamais un pas de double
  compté.

**Lecture.** Les bandes de niveau du panneau Stats (:ref:`stats`) sont un
repère propre à blunderDB, pas une norme publiée. Un PR se compare d'abord à
vos propres PR : même analyseur, même type d'adversaires, mêmes longueurs.

**Incertitude.** Le PR d'un match est très bruité : l'erreur-type d'un match
en 7 points est de l'ordre de 1,5 PR. blunderDB donne un intervalle à 95 %
(:ref:`metrique_intervalles`) en rééchantillonnant les parties d'un match, et
les matchs d'un agrégat.

**Limites.** Le numérateur ne dépend de rien, mais le dénominateur dépend de
ce que l'on compte : les seuils de 0,001 et 0,200 sont arbitraires. On peut
faire baisser son PR sans mieux jouer, en prenant un refus serré pour ajouter
des décisions à une course, ou en choisissant au bear-off un coup qui crée une
décision de plus. Le PR pèse une erreur au videau 8 comme une erreur au videau
1 (l'équité est normalisée au videau), et une erreur au DMP comme une erreur
à 0-0. Il hérite des imperfections du moteur, et l'adversaire qui crée des
positions difficiles fait monter le PR de l'autre. La :ref:`perte de MWC
<metrique_l7>` et la :ref:`difficulté <metrique_difficulte>` répondent à ces
trois défauts.

**XG et gnuBG.** Sur un fichier XG, le PR de blunderDB est celui d'XG, à la
précision d'affichage près, hormis les écarts expliqués de
:ref:`stats_parity`. gnuBG divise la même somme par un autre nombre : chaque
coup ayant plus d'un coup légal, et les décisions de videau qu'il juge proches
selon sa propre fenêtre. Son taux par décision ne se compare donc pas au PR
d'XG, même sur la même analyse. Le *Snowie Error Rate* (:ref:`stats_parity`)
divise par tous les coups des deux joueurs : il ne dépend d'aucune règle de
comptage, mais vaut environ la moitié du PR.


.. _metrique_perte_mwc:

Perte de MWC par décision et graphique du match
-----------------------------------------------

**La question.** Quelles décisions m'ont coûté ce match ? Le PR met toutes
les erreurs sur la même échelle ; ℓ les pèse au score où elles ont été
commises. Une erreur de 0,040 à 4-4 dans un match en 5 points ne coûte pas ce
qu'elle coûte à 0-0 dans un match en 17. Pour l'étude, la colonne et le
graphique désignent les quelques décisions qui ont décidé du match, et la
courbe cumulée montre où il a basculé.

**Définition.** ℓ (:ref:`metrique_erreur`) pour chaque décision notée ; la
perte d'un joueur sur le match est Σℓ sur ses décisions comptées au PR. Les
non-décisions coûtent zéro : elles ne changent pas la somme.

**Unité.** Un pourcentage de chances de gagner le match : 2,5 veut dire
2,5 % de MWC perdus.

**Lecture.** La colonne **MWC** de la transcription, la barre par décision et
la courbe cumulée de chaque joueur sont décrites au :ref:`panneau_matchs`. Une
barre isolée haute est une décision à revoir ; une courbe qui monte
régulièrement dit un match d'erreurs moyennes, pas d'accident.

**Incertitude.** ℓ est exact au regard de l'analyse : son incertitude est
celle de l'analyse elle-même (profondeur, rollout ou non) et de la MET.

**Limites.** ℓ dépend de la MET : deux logiciels qui n'utilisent pas la même
table ne donnent pas la même perte. Une partie d'argent n'en a pas, et une
analyse calculée avec une autre MET que celle de la base reste hors des
statistiques. Une décision non notée porte un tiret, jamais zéro.

**XG et gnuBG.** La somme est la perte de MWC qu'XG affiche pour le match :
pions, doubles et réponses concordent sur les matchs de référence à la
précision d'affichage d'XG (écart ≤ 0,006 point). gnuBG fait la même
conversion, avec ses propres équités : sur un match qu'il a analysé, ses
chiffres diffèrent de ceux d'XG par l'analyse, pas par la formule.


.. _metrique_l7:

Perte MWC (éq. 7 pts) L₇ et Elo face au moteur
----------------------------------------------

**La question.** Combien de match ai-je perdu par mes erreurs, quelle que
soit la façon de compter les décisions, et comment comparer un match en
5 points à un match en 11 ? L₇ ne divise par aucun nombre de décisions : il
ne se manipule pas en ajoutant des décisions faciles, et il pèse chaque
erreur ce qu'elle a coûté.

**Définition.** Sous jeu parfait, les chances de gagner le match évoluent
sans biais : contre un adversaire parfait, un joueur qui perd L de MWC sur le
match garde q = 0,5 − L de chances. Le modèle de la formule FIBS dit qu'à force
égale la perte croît comme la racine de la longueur N. L est donc ramené à un
match en 7 points, la longueur de tournoi la plus courante :

.. math::

   L_7 = L \times \sqrt{7 / N}
   \qquad
   L_7^\text{agrégat} = \sqrt{7} \times \frac{\sum L}{\sum \sqrt{N}}

L'agrégat (plusieurs matchs, un joueur, un tournoi) additionne les pertes et
les racines des longueurs avant le rapport : un match en 7 points pèse plus
qu'un match en 1 point, et un seul match rend sa propre valeur. Sans filtre
joueur, les deux sièges d'un match sont deux unités.

**L'Elo face au moteur** inverse la formule FIBS à 7 points :

.. math::

   D = \frac{2000}{\sqrt{7}} \log_{10} \frac{q}{1 - q}, \qquad q = 0{,}5 - L_7

Linéarisé, D ≈ −3474 × L / √N. Au-delà de L₇ ≈ 49 %, q n'a plus d'inverse
fini : q est plancher à 1 % (D ≥ −1509) et l'Elo est affiché « ≤ ». L₇, lui,
n'a pas de plancher. Les deux classent les joueurs dans le même ordre.

**Unité.** L₇ : un pourcentage de MWC. D : des points Elo, négatifs, l'écart
attendu entre le joueur et le moteur.

**Lecture.** 12,3 % : au lieu de 50 % de chances contre le moteur, le joueur
n'en avait plus que 37,7 % sur un match en 7 points. Le badge, le détail, les
statistiques, le tournoi et la progression l'affichent
(:ref:`Perte MWC (éq. 7 pts) <perte_mwc_7pts>`).

**Incertitude.** Intervalle à 95 % par les parties pour un match, par les
matchs pour un agrégat (:ref:`metrique_intervalles`), puis même
transformation que la valeur. Une ligne par match des statistiques n'en a
pas : c'est une seule unité ; le bilan du match porte l'intervalle par
parties.

**Limites.** Le modèle en √N est approximatif sous 7 points et autour du
Crawford. L₇ dépend de la MET. Une partie d'argent n'a pas de longueur, donc
pas de L₇ : l'écran le dit au lieu d'afficher un nombre. Des joueurs réels
perdent plus d'un demi-match, car les erreurs commises après un retournement
de chance s'additionnent : l'Elo atteint alors son plancher.

**XG et gnuBG.** Pour un match en 7 points, L₇ est exactement la perte de MWC
qu'affiche XG. Aucun des deux logiciels ne ramène la perte à une longueur de
référence ni ne la convertit en Elo. Cet Elo mesure l'écart au moteur, pas un
classement FIBS ou BMAB entre joueurs humains.


.. _metrique_difficulte:

Difficulté, erreurs évitables, excès et ratio
---------------------------------------------

**La question.** Cette erreur était-elle pardonnable ? Une même perte de 2 %
est une inattention dans une position évidente, et une faute que tout bon
joueur ferait dans une position où quatre coups se tiennent à quelques
millièmes. Pour l'étude, la difficulté trie les fautes : les erreurs évitables
(inattention, règle mal sue) se corrigent vite et passent en premier ; les
pertes sur des décisions difficiles relèvent du travail de fond.

**Définition.** La difficulté d est la perte qu'aurait subie en moyenne, dans
la même position, un **joueur de référence** qui choisit chaque option i avec
une probabilité d'autant plus faible qu'elle coûte cher :

.. math::

   \pi(i) = \frac{e^{-\Delta_i/\tau}}{\sum_j e^{-\Delta_j/\tau}},
   \qquad d = \sum_i \pi(i)\,\Delta_i,
   \qquad \tau = 0{,}025

Δᵢ est le coût de l'option i par rapport à la meilleure, en équité
normalisée. Les options sont les candidats de l'analyse pour un coup de pions ;
pas de double et double (contre la meilleure réponse) pour le joueur qui a le
videau ; prendre et passer pour celui qui reçoit le double. d est convertie en
MWC comme ℓ : les deux sont dans la même unité, et l'option jouée a pour coût
exactement ℓ.

τ = 0,025 est un a priori de joueur fort : sur deux options séparées de Δ,
d = Δ / (1 + e^{Δ/τ}) est maximale vers Δ ≈ 0,032, entre « douteux » et
« erreur » ; un écart de 0,1 ne coûte plus que 0,002 ; cinq candidats espacés
de 0,01 donnent d ≈ 0,012, l'erreur moyenne d'un joueur de PR 6 environ.

* **Erreur évitable** : la perte atteint le seuil Erreur de la base et
  d ≤ ℓ / 10 — sur deux options, le joueur de référence ne la commettrait pas
  une fois sur dix.
* **Excès** d'un joueur sur un match : Σ(ℓ − d), ce qu'il a perdu au-delà de la
  référence (négatif quand il a fait mieux).
* **Ratio** : Σℓ / Σd ; 1 veut dire « joue comme la référence », 2 « perd deux
  fois plus ». Il n'est pas donné quand Σd < 0,5 % de MWC : sur un match trop
  facile il ne mesure que du bruit, l'excès reste.

**Ce qui compte.** Les décisions dont la perte est notée et convertie en MWC ;
ailleurs la difficulté est absente, jamais nulle. Aucun seuil ne met à part
une décision évidente ou forcée : sa difficulté est proche de zéro d'elle-même.

**Unité.** d et l'excès : un pourcentage de MWC. Le ratio : sans unité.

**Lecture.** Une perte bien au-dessus de sa difficulté est une faute que la
position n'excusait pas ; une perte proche de sa difficulté, une faute
partagée. Excès et ratio corrigent le PR de la difficulté des positions
rencontrées, donc en partie du style de l'adversaire (:ref:`panneau_matchs`).

**Incertitude.** Sur un match, excès et ratio reposent sur peu de décisions et
varient beaucoup : une tendance, pas un classement.

**Limites.** La difficulté dépend d'un modèle de joueur et de τ, fixés une
fois : les changer change tous les chiffres. Elle ne connaît que les candidats
que l'analyse a gardés (quelques-uns chez XG, selon ses filtres chez gnuBG) :
les options absentes ne pèsent rien, et d est alors un minorant. Elle hérite de
l'erreur de l'analyse, surtout à faible profondeur.

**XG et gnuBG.** Ni l'un ni l'autre ne calcule de difficulté. L'idée vient des
indices de vraisemblance des échecs (Regan) ; τ n'est pas étalonné sur une
population de joueurs.


.. _metrique_intervalles:

Intervalles à 95 %
------------------

**La question.** Ce chiffre dit-il quelque chose, ou est-ce du bruit ? Un PR
de 4,2 sur un match et un PR de 4,2 sur quarante matchs ne valent pas la même
chose. L'intervalle évite de bâtir un plan de travail sur une case qui ne
repose sur rien, et de lire un rang dans un match isolé.

**Définition.** L'intervalle rééchantillonne la plus grande unité
indépendante : les **parties** d'un match, les **matchs** d'un agrégat (PR
global, cases par phase, plan de jeu, étiquette et score, L₇ d'un agrégat). Les
parties d'un même match partagent adversaire, séance et fatigue : les
rééchantillonner séparément dans un agrégat rétrécirait l'intervalle à tort.
Sans filtre joueur, les deux sièges d'un match forment une seule unité, car
ils jouent les mêmes positions.

**La formule.** Pour un rapport R = Σeᵤ / Σnᵤ sur G unités (eᵤ la somme des
erreurs, nᵤ le nombre de décisions de l'unité), la variance du bootstrap est
calculée, sans tirage :

.. math::

   \widehat{\mathrm{Var}}(R) = \frac{G}{G-1} \cdot
   \frac{\sum_u (e_u - R\,n_u)^2}{\left(\sum_u n_u\right)^2}

et l'intervalle vaut R ± 1,96 × √Var, la borne basse jamais sous 0, puis la
même transformation que la valeur (× 500 pour le PR). Le L₇ d'un agrégat suit
la même règle, L à la place de e et √N à la place de n ; celui d'un match est
une somme sur ses G parties, de variance G/(G−1) × Σ(l_g − l̄)², l_g la perte de
la partie g. Le calcul est déterministe : la
même base donne toujours le même intervalle, sur les deux moteurs de
stockage.

**Ce qu'il faut.** Au moins **trois unités** et une largeur non nulle. Avec
deux, la dispersion repose sur un seul degré de liberté et l'intervalle serait
plusieurs fois trop étroit ; des unités toutes au même rapport disent qu'elles
s'accordent, pas que la valeur est exacte. Sans intervalle, la case est
grisée, son effectif visible (:ref:`stats`).

**Lecture.** Deux intervalles qui se chevauchent largement ne départagent
rien. Une famille dont l'intervalle reste au-dessus de votre PR global est une
faiblesse établie.

**Limites.** L'approximation normale est grossière sur peu d'unités, et les
erreurs ont une distribution asymétrique (quelques grosses fautes) : avec trois
ou quatre unités, l'intervalle est indicatif.

**XG et gnuBG.** Aucun des deux ne donne d'intervalle.


.. _metrique_bilan_match:

Bilan du match
--------------

**La question.** Après un match : qu'est-ce que je revois, ai-je perdu à cause
des dés ou du jeu, et mes erreurs viennent-elles de la précipitation ou d'une
lacune ? Les écrans : :ref:`bilan du match <bilan_match>`.

**Trois décisions à revoir.** Parmi les erreurs du joueur (seuil de la base)
dont la perte est chiffrée, les trois de plus grande **part évitable**
ℓ − d = ℓ × (1 − d/ℓ), positive : la perte multipliée par son caractère
évitable. Une difficulté inconnue compte zéro ; à égalité, la plus grosse
perte, puis l'ordre du match. Une erreur que le joueur de référence ferait
aussi ne vaut pas une séance.

**Résultat ajusté de la chance**, pour un match terminé, en points de MWC du
joueur :

.. math::

   A = R - C, \quad R = \text{issue} - \mathrm{MWC}_\text{départ}, \quad
   C = \sum \text{chance de ses jets} - \sum \text{chance des jets adverses}

L'issue vaut 1 gagné, 0 perdu ; la MWC de départ est celle du score initial
(50 % à 0-0). Chaque jet est converti en MWC comme une perte, au score et au
videau de sa position. Sous jeu parfait, l'espérance de A est la perte de
l'adversaire moins la vôtre : le bilan donne l'*écart des erreurs* à côté pour
recouper. A > 0 sur un match perdu : perdu par les dés ; A < 0 : perdu par le
jeu. La chance est celle que l'analyseur a écrite (:ref:`stats_parity`) : sans
chance, pas de ligne, et la couverture (jets mesurés sur jets joués) est
affichée, car une chance partielle biaise A. Pas de ligne non plus pour une
partie d'argent ou un match inachevé.

**Erreurs précipitées et réfléchies.** Par joueur et par type (pions,
videau), le seuil est la médiane des durées connues de ses décisions chiffrées
de ce type dans le match : une erreur jouée plus vite est précipitée, les
autres réfléchies. La médiane est la sienne, dans ce match : elle ne dépend pas
de la cadence, et sans lien entre vitesse et erreur les erreurs se partagent
moitié-moitié. Une majorité précipitée appelle de la discipline ; une majorité
réfléchie, de la connaissance.

**Limites.** Un match est un petit échantillon : PR et L₇ y portent leur
intervalle par parties (trois parties au moins). Le résultat ajusté dépend de
la MET et de la chance du fichier. La durée n'existe que pour un match qui l'a
gardée (un match joué contre un bot).

**XG et gnuBG.** XG et gnuBG affichent la chance totale de chaque joueur, mais
ni résultat ajusté en MWC, ni sélection des décisions à revoir, ni partage par
la durée.


.. _metrique_plan_etude:

Plan d'étude
------------

**La question.** Que dois-je travailler maintenant ? Classer ses erreurs par
coût dit où l'on a le plus perdu, pas où l'étude rapporte le plus : une
famille de positions réellement difficiles coûte à tout le monde, et trois
erreurs ne font pas une tendance. Les écrans : :ref:`plan_etude`.

**Définition.** Une **famille** est un triplet (plan de jeu, nature de la
décision, thème) des erreurs récurrentes ; ses membres sont les erreurs du
filtre, décisions comptées qui coûtent au moins le seuil Erreur. Une erreur
sans thème n'entre dans aucune famille. Un membre est **chiffré** s'il porte ℓ
et d ; une erreur en partie d'argent est comptée « non chiffrée » et n'entre
dans aucune priorité.

**La formule.** Le **MWC récupérable** et son intervalle :

.. math::

   R = \sum_i (\ell_i - d_i), \qquad
   R \pm 1{,}96 \sqrt{\textstyle\sum_i (\ell_i - d_i)^2}

R est la fréquence de la famille multipliée par sa perte moyenne au-delà de la
difficulté : ce qu'on regagnerait en jouant ces positions comme le joueur de
référence. La somme est signée, sans écrêtage. L'intervalle traite le nombre
de membres comme un comptage de Poisson.

**Ce qui entre au plan.** Au moins **5 membres chiffrés** et une **borne basse
strictement positive**. Le plan est classé par borne basse décroissante : à
récupérable égal, la famille la mieux établie passe devant. Les autres sont
« à confirmer », sans rang.

**Unité.** Un pourcentage de MWC, cumulé sur le filtre.

**Lecture.** Une famille d'erreurs évitables monte ; une famille de positions
où tout le monde se trompe descend. Un nouveau joueur voit d'abord des
familles « à confirmer » : le plan ne pousse pas vers du bruit.

**Limites.** Le thème vient du classificateur de blunderDB : une erreur mal
classée tombe dans une mauvaise famille, une erreur sans thème n'est dans
aucune. R hérite des limites de ℓ et d (MET, troncature des candidats).

**XG et gnuBG.** Ni l'un ni l'autre ne propose de plan d'étude.


.. _metrique_positions_reference:

Positions de référence
----------------------

**La question.** Quelles positions mettre dans une collection pour travailler
le plan ? Les dix pires erreurs donnent souvent dix variantes d'un même
problème. Une bonne position de référence est au centre de beaucoup de vos
erreurs évitables, porte une leçon nette et n'est pas déjà étudiée. Les
écrans : :ref:`positions de référence <positions_reference>`.

**Définition.** Les candidates sont les erreurs chiffrées et thématisées du
filtre, réunies par position ; l'excès d'une position est la somme des excès
positifs ℓ − d de ses erreurs. Deux candidates d'une même famille — et, pour
le videau, du même score — sont **voisines** à au plus ρ = 12 pions-pas de
distance ``like`` (un coup de dés en vaut 8 à 16).

**La formule.** Le gain d'une candidate p est

.. math::

   G(p) = c(p) \times \sum_{q \,\in\, \text{voisinage}(p),\ q\ \text{non couverte}} \text{excès}(q)

en fraction de MWC : la représentativité, la fréquence et le coût évitable en
une seule quantité. c(p) part de 1 et est multiplié par ½ si la leçon est
**serrée** (le second choix coûte moins de la moitié du seuil Erreur), et
encore par ½ si elle est **instable** (une autre profondeur d'analyse du
videau ou un rollout rend un autre verdict) : ¼ quand les deux se cumulent.
Un rollout qui confirme ne relève rien : il
départage à gain égal.

**Sélection.** Gloutonne par gain marginal : chaque position retenue couvre
ses voisines, qui ne rapportent plus rien aux suivantes, et ses quasi-doublons
sont écartés même hors de sa famille. Une position déjà traitée (commentaire,
carte Anki, collection, marque « étudiée ») n'est jamais proposée et compte
comme déjà retenue.

**Unité.** Le MWC couvert, en pourcentage.

**Limites.** Le rayon et les facteurs sont fixés, pas appris. La distance
``like`` compare des structures de pions, pas des idées : deux positions de
même leçon mais de structure éloignée ne se couvrent pas.

**XG et gnuBG.** Pas d'équivalent.


.. _metrique_avant_apres:

Avant/après l'étude
-------------------

**La question.** Ce que j'ai travaillé me coûte-t-il moins en match réel ?
Sans mesure, l'étude se juge à l'impression ; une famille qui ne bouge pas
malgré le travail dit qu'il faut changer de méthode. Les écrans :
:ref:`avant_apres_etude`.

**Définition.** Une famille du plan est **étudiée** au jour (UTC) de la
première action d'étude sur l'une de ses positions : marque « étudiée »,
révision Anki quelle que soit la note, réponse de quiz. Créer une carte ou
ranger une position en collection n'en est pas une. Fenêtre *avant* : les
matchs datés strictement avant ce jour ; *après* : strictement après ; le jour
même et les matchs sans date sont écartés.

**La formule.** Dans chaque fenêtre, le taux de perte r = Σℓ / N : ℓ les pertes
des erreurs de la famille, N les décisions comptées en match du même plan de
jeu et de la même nature (le thème n'existe que pour une erreur). Le gain et
son intervalle :

.. math::

   \Delta = r_\text{avant} - r_\text{après}, \qquad
   \Delta \pm 1{,}96 \sqrt{\frac{\sum \ell^2_\text{avant}}{N^2_\text{avant}}
   + \frac{\sum \ell^2_\text{après}}{N^2_\text{après}}}

**Verdict.** Au moins **30 décisions** dans chaque fenêtre, sinon
« insuffisant ». « En progrès » si la borne basse de Δ est positive, « en
recul » si la borne haute est négative, « indéterminé » sinon.

**Unité.** Points de MWC pour 100 décisions.

**Limites.** Un changement, pas un effet. Une famille est étudiée *parce
qu'elle* coûtait : la régression vers la moyenne gonfle le gain. Rien ne
contrôle les adversaires, le format ni les dés.

**XG et gnuBG.** Pas d'équivalent.


.. _metrique_biais:

Biais signés
------------

**La question.** Dans quel sens est-ce que je me trompe ? « Vous prenez
trop » se corrige mieux qu'un PR : la règle à revoir est nommée. Les écrans :
:ref:`biais_signes`.

**La formule.** Chaque décision comptée porte x = +1 (erreur par excès
d'audace), −1 (par excès de prudence) ou 0. Avec P et M les deux comptes sur N
décisions :

.. math::

   B = \frac{P - M}{N}, \qquad
   B \pm 1{,}96 \sqrt{\frac{(P + M)/N - B^2}{N}}

Au moins **20 décisions** ; « trop » si la borne basse est positive, « pas
assez » si la borne haute est négative, « équilibré » sinon. Le coût de P et de
M (en mpt) accompagne les comptes. Le signe et le coût sont ceux du coup joué
dans son match.

* **Prise / refus** : sur les réponses au videau ; P = prises fautives,
  M = refus fautifs. B est exactement votre taux de prise moins celui du bot
  sur les mêmes positions.
* **Doubles** : sur les décisions de videau du porteur comptées au PR ;
  P = doubles prématurés (pas de double, ou trop bon), M = doubles manqués. Le
  biais est aussi donné par case de score (vos points restants, ceux de
  l'adversaire ; post-Crawford compté à 1 ; la partie d'argent à part).
* **Blots** : sur les coups de pions avec contact ; x est le signe de (vos blots
  après le coup joué − vos blots après le meilleur coup). Un coup est reconnu
  par le plateau qu'il laisse ; un coup introuvable sort du compte, et sa part
  est affichée.

**Unité.** Une différence de proportions, sans unité.

**Limites.** La plupart des cases de score restent muettes : c'est la réponse
honnête. Le biais des blots compte les blots sans peser leur exposition, et
les coups écartés ne sont pas un tirage neutre.

**XG et gnuBG.** Pas d'équivalent ; XG et gnuBG comptent les erreurs de
videau par type (double manqué, prise fautive…) sans intervalle ni verdict.


.. _metrique_bilan_tournoi:

Bilan du tournoi
----------------

**La question.** Ai-je joué à mon niveau, et sinon, où cela a-t-il cédé : les
dernières rondes, les longues fins de match, les scores où un point décide,
la pendule ? Les écrans : :ref:`bilan du tournoi <bilan_tournoi>`.

**Définition.** Le **niveau habituel** est le joueur sur ses matchs en points
des **365 jours** qui précèdent le tournoi (sa date, à défaut celle de son
premier match), ceux du tournoi exclus ; il faut au moins **5 matchs**. Le
tournoi et le niveau habituel ont chacun leur PR et leur L₇, avec l'intervalle
par matchs (:ref:`metrique_intervalles`).

**La formule.** Δ = tournoi − habituel, d'écart-type √(σₜ² + σₕ²), chaque σ lu
sur la demi-largeur haute de son intervalle. « Moins bien » si la borne basse
de Δ est positive, « mieux » si sa borne haute est négative, « dans
l'habitude » sinon. Le verdict demande un intervalle et au moins **20
décisions** de chaque côté ; sinon « insuffisant », chiffres visibles.

**Ventilations**, en PR, chaque case face à la même case du niveau habituel :
par ronde ; par rang de la décision dans le match (tranches de 30 : la
fatigue) ; par score (DMP, partie Crawford, post-Crawford, autres) ; par
rythme (de part et d'autre de la médiane de vos durées dans le match). Au plus
trois familles du plan d'étude restreint au tournoi.

**Limites.** Un tournoi de trois matchs ne dit presque jamais « moins bien » :
c'est voulu. Les deux côtés sont traités comme indépendants, ce qui élargit
l'intervalle : la règle est prudente. La réserve de la pendule n'est pas
enregistrée coup par coup : la pression du temps restant ne se mesure pas.

**XG et gnuBG.** Pas d'équivalent.
