.. _manuel_entrainement:

Entraînement, Anki et duel
==========================

.. _panneau_anki:

Panneau Anki
------------

Le panneau **Anki** (*CTRL-K*) permet d'étudier par répétition espacée, en
utilisant l'algorithme FSRS. Une carte pose une question : le plus souvent une
position, tirée d'une collection ou d'une recherche ; ce peut aussi être un
score.

**Création de paquets :** Cliquez sur **+ Nouveau paquet** pour créer un paquet à partir
d'une collection ou des résultats de recherche courants. Les paquets basés sur
une recherche se synchronisent automatiquement à l'activation de l'onglet Anki.

**Un paquet de fiches de score.** La troisième source, *Fiches de score*, ne
demande rien d'autre qu'un nom : blunderDB remplit le paquet avec les 36 scores
non ordonnés de 2 à 9 away, et la carte d'un score est la fiche que
l':ref:`exercice Scores <panneau_entrainement>` affiche — points de prise et
valeurs de gammon, les deux faces. Ce paquet n'existe que si vous le créez :
36 cartes dues le premier jour sont une dette de révision, et elle se contracte
volontairement. Le bouton de synchronisation le régénère.

Les deux endroits ne font pas le même travail. L'exercice Scores fait
retrouver ces nombres **sous la pendule** et mesure la vitesse ; le paquet les
fait **tenir dans le temps** et n'en mesure rien. Les deux histoires restent
séparées : le journal de l'Entraînement ignore les révisions Anki, et les
statistiques d'Anki ignorent les sessions d'Entraînement.

**Un paquet sélectionné prend l'en-tête du panneau** : son nom, le nombre de ses
positions, qu'un clic ouvre, et le nombre de cartes dues ; à droite, les
paramètres (⚙), la remise à zéro (↻), *Bachoter* et *Étudier*. ← revient à la
liste des paquets.

**Exporter vers Anki :** le bouton ⤓ d'une ligne de la liste écrit le paquet
en fichier **.apkg**, que l'application Anki importe sur un ordinateur ou un
téléphone (AnkiDroid, AnkiMobile). Chaque position devient une carte
recto-verso : au recto le plateau, le score vu du joueur au trait (placé en
bas), le videau et les dés ; au verso le meilleur coup ou la bonne décision de
videau, son équité et l'erreur du coup joué. Les libellés suivent la langue de
l'interface. Le paquet s'exporte tel qu'il est, sans synchronisation, et
l'export n'écrit rien dans la base. Réimporter un export plus récent dans Anki
met les cartes à jour au lieu de les dupliquer. En ligne de commande :
``blunderdb anki export``, qui exporte aussi une collection (voir :doc:`cli`).

**Révision :** Sélectionnez un paquet puis cliquez sur *Étudier* (ou double-cliquez
sur un paquet) pour commencer la révision des cartes dues. Une carte de position
affiche la position sur le plateau ; une carte de score annonce le score et
laisse le plateau tel qu'il est. Évaluez votre rappel avec les touches
*1* (À revoir), *2* (Difficile), *3* (Correct), ou *4* (Facile). Appuyez sur *Esc*
pour arrêter et revenir à la liste des paquets.

**Annuler une note.** Le bouton *Annuler*, sous les boutons de note, ou la
touche *u*, reprend la dernière note de la séance : la carte retrouve exactement
l'état qu'elle avait avant (échéance, intervalle, stabilité, difficulté,
nombre de révisions et d'oublis), la note quitte le journal des révisions, et la
carte revient à l'écran pour être notée de nouveau. Répété, il remonte la
séance note après note. Quand la séance s'achève, *Annuler* reste dans la bande
du paquet jusqu'à la séance suivante : la dernière note se reprend aussi, et la
révision reprend sur cette carte. Une révision enregistrée avant la version 2.42.0 du
schéma ne peut pas être annulée : l'état d'avant n'était pas conservé. En ligne
de commande : ``blunderdb anki card --action undo`` (voir :doc:`cli`).

Deux comptes portent des noms distincts : la colonne **Échues** de la liste
compte toutes les cartes dont l'échéance est passée, y compris les cartes
suspendues ou enterrées ; le compte *dues* de l'en-tête ne retient que celles qui
sont disponibles maintenant, et peut donc être plus petit.

Les comptes d'Anki sont des liens : un clic sur les cartes dues, nouvelles, en
cours ou en révision de la bande, ou sur une valeur des colonnes **Cartes**,
**Nouvelles** et **Échues** de la liste, charge les positions correspondantes.
De même, dans un tournoi ouvert, le nombre de positions ouvre exactement ces
positions.

**Les décisions de videau font deux cartes, enchaînées.** Une décision de
videau est deux questions — « double ? », puis « prend ? » — et blunderDB les
enregistre depuis toujours comme deux positions. Un paquet qui n'en
sélectionne qu'une moitié reçoit l'autre : la décision est complétée, pas
augmentée. Et quand les deux sont dues, la seconde vient **immédiatement**
après la première.

Chacune garde sa propre note et son propre calendrier : ce ne sont pas deux
temps d'une même carte, ce sont deux cartes. L'enchaînement n'avance aucune
échéance — il ordonne les cartes déjà dues, rien de plus. Les deux naissant
ensemble, elles sont dues ensemble la première fois, et c'est là qu'il sert.

**Afficher la réponse :** La carte pose une question — quel coup jouer, quelle
action de videau, ou quels nombres porte un score. Réfléchissez, puis appuyez sur
*ESPACE* (ou cliquez sur la zone masquée) pour dévoiler la réponse : l'analyse
enregistrée de la position, telle que l'onglet Analyse la présente, ou la fiche
du score, entière. Sur une fiche il n'y a rien à cocher : Anki planifie une
mémoire, il ne mesure pas un calcul — c'est l'Entraînement qui compte les
fautes. Elle apparaît sous les boutons d'évaluation,
qui restent à leur place et à portée. Cliquer sur un coup de la liste le montre
sur le plateau.

Rien ne vous oblige à dévoiler la réponse pour évaluer : si vous êtes sûr de
vous, les touches *1* à *4* restent actives. La réponse se remasque à la carte
suivante, mais pas si vous changez simplement d'onglet — allez consulter le
panneau Eval ou le commentaire de la position, elle vous attendra au retour.

Une position dépourvue d'analyse enregistrée l'indique directement, sans zone
masquée.

**Répondre au damier.** Par défaut, vous vous notez vous-même. Dans les
Paramètres d'un paquet de positions, cochez *Répondre au damier* : pour une
carte de pions, vous jouez alors le coup sur le damier, comme dans l'exercice
Décision (un clic sur un pion le joue du premier dé non joué, un clic sur les
dés intervertit les dés restants, le clic droit reprend les pas), puis le coup
complet se valide par un clic sur les dés ou par ENTRÉE. Le moteur juge le coup contre l'analyse enregistrée,
dévoile la réponse et **propose une note** : *Facile* pour une bonne réponse
rapide, *Correct* pour une bonne réponse plus lente, *Difficile* pour une erreur
sous le seuil du blunder, *À revoir* pour un blunder ou un coup illégal. La note
proposée est en surbrillance ; vous gardez la main et notez ce que vous voulez
avec *1* à *4*. Un coup légal que l'analyse ne classe pas ne propose rien. Les
cartes de videau, les cartes de score et les paquets de fiches de score restent
en auto-notation. Dévoiler la réponse sans jouer abandonne le coup.

**Limiter la séance.** Par défaut, une séance de révision va jusqu'au bout des
cartes dues. Vous pouvez la borner à un nombre de cartes, par paquet, dans les
Paramètres : cochez *Limiter la séance* et indiquez combien de cartes une séance
doit servir. Quand la limite est atteinte, la séance s'arrête en le disant — le
message distingue « limite atteinte, tant de cartes encore dues » d'une file
réellement épuisée. Pour continuer malgré tout, l'entraînement libre est là : il
sert d'autres positions sans rien modifier au planning.

Une limite de **0** ne sert aucune carte : c'est un état à part entière, utile
pour geler un paquet le temps de préparer un tournoi, et ce n'est pas la même
chose que « pas de limite ». Le bouton *Étudier* est alors inactif.

La limite porte sur la **séance**, pas sur la journée. Un paquet blunderDB est
bâti sur une collection ou une recherche : c'est un corpus fini, introduit en
quelques séances, dont le volume quotidien est déjà borné par sa taille. Un
plafond par jour n'y mordrait jamais, ou bien créerait un retard sur un paquet
qui tenait en une séance.

**Entraînement libre (cram) :** Le bouton *Bachoter*, à côté de *Étudier*, lance une
session d'entraînement libre : des positions aléatoires du paquet vous sont
présentées sans tenir compte de l'échéancier FSRS. Ce mode **ne modifie jamais
le planning de révision espacée** — idéal pour s'échauffer avant un tournoi ou
réviser intensément un paquet thématique sans perturber son ordonnancement. Une
pastille *Libre* remplace l'état de la carte et un bouton *Suivant* (touches *1*
à *4*) fait défiler les positions. *Esc* revient à la liste sans enregistrer de
session interrompue.

**Écarter une carte, sans la noter.** Pendant une révision, un clic droit sur
l'en-tête de la carte ouvre trois gestes qui la sortent de la séance sans rien
dire au planificateur :

* **Suspendre** — la carte garde son échéancier et ne remonte plus jamais tant
  qu'elle est suspendue. C'est la manière de mettre de côté une carte fausse,
  ou pas encore utile, sans perdre l'historique qui y est attaché.

* **Enterrer** — la carte disparaît jusqu'au lendemain. Contrairement à la
  suspension, cela ne dit rien de sa valeur : c'est pour celle que l'on vient
  de voir ailleurs, ou que l'on préfère ne pas croiser deux fois dans la soirée.

* **Retirer** — la carte quitte le paquet, après confirmation. La position,
  elle, reste dans la base : un paquet est une liste d'étude sur la
  bibliothèque, jamais une copie de celle-ci.

Aucun de ces trois gestes n'enregistre de note : une carte écartée n'est pas
une carte répondue, et elle ne compte pas dans le décompte de la séance.

**Journal des révisions.** Dans les Paramètres d'un paquet, le bouton *Journal
des révisions* montre ce que le planificateur a été **dit** — date, sujet
(le numéro de la position, ou le score), note, état, intervalle accordé — par
opposition à ce qu'il prévoit. C'est le
seul endroit où une note entrée par erreur se voit. Elle ne s'y corrige pas :
l'échéancier reste hors de portée, et cette règle est précisément ce qui rend
le journal utile — on ne peut pas réécrire le passé, mais on peut savoir ce
qu'il a été.

**Arrêt/Reprise :** Vous pouvez interrompre une session de révision à tout moment
avec *Esc*. Le bouton change en *Reprendre* et affiche votre progression.
Cliquez dessus pour reprendre là où vous vous êtes arrêté.

**Gestion des paquets :** Utilisez les boutons d'action pour renommer,
synchroniser, réinitialiser ou supprimer des paquets (confirmation demandée
pour ces deux dernières actions). Les paramètres FSRS (rétention cible,
intervalle maximum, aléa) peuvent être configurés par paquet dans les
Paramètres (icône engrenage).

**Rétention : la cible et la mesure.** La *rétention cible* est votre choix sur
le compromis entre charge de travail et qualité du rappel : plus elle est haute,
plus les intervalles raccourcissent et plus vous révisez. En regard, les
Paramètres affichent la **rétention mesurée** sur vos propres révisions — une
information, jamais un pilotage : blunderDB ne modifie pas votre cible pour
poursuivre votre taux de réussite. Sous une vingtaine de révisions, la mesure
n'est pas affichée : elle se lirait comme un fait alors qu'elle n'est que du
bruit.

Changer la rétention **n'est pas rétroactif** : chaque carte adopte le nouveau
rythme à sa prochaine révision, et les échéances déjà fixées ne bougent pas.
L'effet est donc progressif, et invisible le jour même.

L'*intervalle maximum* borne l'espacement. Un paquet créé récemment démarre à
un an : une position que l'algorithme reporterait de plusieurs années a quitté
le paquet sans que vous l'ayez décidé, et votre propre jeu change plus vite que
cela. Les paquets plus anciens conservent la valeur qu'ils avaient.

.. _panneau_entrainement:

Panneau Entraînement
--------------------

Le panneau **Anki** fait réviser ce qui se **retient** ; le panneau
**Entraînement** fait travailler ce qui se **calcule**, sous la pendule. Il
s'ouvre par ``CTRL-J``, par le bouton de la barre d'outils placé juste après
« Position aléatoire », ou par la commande ``train``. La fiche de score relève
des deux : elle se calcule ici et se retient dans un
:ref:`paquet de fiches de score <panneau_anki>`.

Au repos, le panneau montre le lanceur et le bilan des sessions passées.

Le lanceur
~~~~~~~~~~

Trois réglages, un par ligne ; « Démarrer » est au bout de la bande d'en-tête :

* l'**exercice** — *Scores*, *Comptage des pips*, *Bearoff*, *Évaluation*, *Décision* ou *Ouverture* ;
* la **source** de la question, quand l'exercice en a plusieurs — *Vivier*
  (des formes canoniques de l'exercice), *Plateau* (la position telle qu'elle
  est) ou *Base* (une position de la liste parcourue) ;
* pour *Ouverture*, à la place de la source, le **jet** posé — *1st roll* (le
  coup d'ouverture) ou *2nd roll* (la réponse) ; *1st roll* par défaut ;
* pour *Ouverture* encore, le **mode**, exclusif : *Catégories XG* (par défaut) tire
  parmi les **catégories** cochées — *Money*, *DMP*, *Gammon save* (l'ouvreur à 1 away
  Crawford, le répondeur à 2 away) et *Gammon go* (l'inverse), cochées d'emblée ;
  *Match en 5* tire un score à égalité parmi ceux du livre où chaque joueur est à 1 à
  5 away, sans choix de score. Une catégorie, ou le mode *Match en 5*, que le livre ne
  tient pas est grisé ;
* la **limite par question** — aucune, 15, 30 ou 60 secondes.

La source, ou le jet, choisi est mémorisé pour chaque exercice, d'une session à l'autre.

La liste parcourue peut venir des *erreurs récurrentes* du panneau Stats : un
clic sur « Quiz sur ce groupe » la remplace par les positions du groupe et
démarre l'exercice Décision.

``train scores``, ``train pips``, ``train bearoff``, ``train evaluation``,
``train decision`` et ``train opening`` ouvrent le panneau et démarrent directement ; ``train tp`` et ``train takepoint``
sont des synonymes de ``train scores``, ``train epc`` de ``train bearoff``,
``train quiz`` de ``train decision``, ``train ouverture`` et ``train book`` de
``train opening``.

Les six exercices
~~~~~~~~~~~~~~~~~

**Scores** tire au sort l'un des 36 scores non ordonnés de 2 à 9 away et
affiche une **fiche de score** : deux colonnes — *Vous* et *L'adversaire* — et
sept lignes — le point de prise au videau 2 puis au videau 4, chacun en course
longue et au dernier lancer, puis la valeur du gammon aux videaux 1, 2 et 4.

L'infobulle de « Révéler » rappelle le geste — estimer chaque nombre de tête,
puis *Révéler*, puis cliquer ceux qu'on a ratés —, et le plateau montre le score
tiré sur une table vide.

Chaque colonne ne porte que les cases que les tables de référence — celles
du :ref:`panneau Données de référence <panneau_reference>` — définissent pour
sa face : trois
nombres à 2a-2a,
quatorze au plus, et une seule colonne à score égal. Une ligne qu'aucune des
deux faces ne définit ne figure pas sur la fiche — il n'y a donc aucune case
« sans objet » à deviner. Les deux faces sont là parce qu'une décision de
videau au score a besoin des deux : le point de prise corrigé combine les
valeurs de gammon des deux joueurs, et c'est le point de prise de l'adversaire
qui dit si votre double passe.

**Comptage des pips** demande le compte de pions des **deux** camps. Le pipcount du
plateau est masqué tant que la question est ouverte ; « Révéler » l'affiche —
**même si vous aviez masqué le pipcount** avec ``p``, sans quoi la réponse
resterait invisible et l'exercice invérifiable. C'est un masque et non un
réglage : votre choix n'est pas modifié, et il reprend la main dès la question
suivante. La source *Plateau*
pose une question sur la position affichée, et une seule ; la source *Base*
tire une nouvelle position à chaque question et l'amène sur le plateau.

**Bearoff** demande l'**EPC des deux camps** — le compte de pions effectif,
celui qui ajoute au pipcount le gaspillage des pions qui sortent en trop.
C'est le domaine où le moteur est exact, et celui où l'EPC se distingue
vraiment du compte de pions.

Chaque question est **engendrée** : le moteur part d'une graine et joue
quelques lancers, et c'est l'instantané qui vous est posé. Un placement au
hasard n'aurait pas les trous, les piles basses et les asymétries d'un vrai
bearoff. La graine vient du *Vivier* (une position de bear-in, puis zéro à dix
plis), du *Plateau* (la position affichée, puis un à quatre plis — jamais zéro,
puisque vous venez de la voir) ou de la *Base* (une position de la liste
parcourue, telle quelle : elle est déjà réelle).

Le domaine de l'exercice : les **deux camps entièrement dans leur jan**, de
**4 à 15 pions** par camp et le reste sorti, videau au centre, en partie
d'argent. Une graine qui n'y entre pas est **refusée en le disant**, et rien ne
démarre — aucune adaptation silencieuse : jouer jusqu'à ce que le contact se
rompe vous donnerait une position que vous n'avez pas choisie. Un plateau vide
fait exception : la question vient alors du vivier, et le panneau dit pourquoi.

L'exercice a besoin de la table de bearoff à un camp ; tant qu'elle s'engendre
en arrière-plan (voir :ref:`configuration`), il le dit plutôt que de poser une
question sans réponse.

**Évaluation** demande ce que vaut une position : les **chances de gain du
joueur au trait**, en pourcentage, et l'**action de videau** — *Pas de double*,
*Double, prend* ou *Double, passe*. Ce sont les deux nombres que montre le
panneau Eval, demandés avant d'être montrés. Le domaine est **toute position** :
une course comme une position de contact, en **partie d'argent**.

Comme pour Bearoff, la question est **engendrée** : la graine vient du *Vivier*
(une position où le contact vient de se rompre, puis zéro à dix plis joués par
le moteur), du *Plateau* (la position affichée, puis un à quatre plis ; la
question se pose en partie d'argent, videau au centre, quel que soit le score de
la graine) ou de la *Base* (une position de la liste parcourue, telle quelle).
Une position de la base ne convient que si elle est une décision de videau en
partie d'argent — sans dés, le videau au centre ou au joueur au trait — ; le
tirage passe sinon à la suivante, et quand aucune ne convient, l'exercice le
dit. Un plateau qui n'est pas une position de partie — pas quinze pions par
camp, ou une partie finie — est **refusé en le disant** ; un plateau vide fait
venir la question du vivier.

La vérité est celle du moteur : la base de bearoff à deux camps quand la
position y figure, gammonNet à sa profondeur canonique partout ailleurs, et le
panneau dit laquelle a répondu. Rien n'est estimé : une position que le moteur
n'évalue pas n'est pas posée.

**Décision** pose une décision **déjà analysée** : une position de la liste
parcourue — sa seule source —, avec la décision qu'elle porte, coup de pions ou
action de videau, et c'est l'analyse enregistrée qui juge. Une position sans
analyse ne pose pas de question, et une position posée ne revient pas dans la
session ; quand la liste est épuisée, le panneau le dit. Sans base ouverte, ou
sans position analysée dans la liste, l'exercice **refuse en le disant** et rien
ne démarre.

**Ouverture** pose **l'un des deux premiers jets d'une partie à un score**,
selon le jet choisi au lanceur : en *1st roll*, l'ouverture du joueur au trait ;
en *2nd roll*, la réponse de l'adversaire dans la position que laisse un
**coup d'ouverture à égalité avec le meilleur à ce score**, tiré parmi ceux que
le livre suit et déjà joué au plateau. Le coup
est jugé contre le **livre d'ouverture** embarqué dans blunderDB, calculé par ses propres rollouts : pour
chaque score de sa grille — de 2 à 7 points de part et d'autre, à un point
d'écart au plus, et la partie Crawford à 1 away contre 2 ou 4 away, dans les
deux sens —, les quinze jets d'ouverture, puis les réponses après les coups
d'ouverture à égalité avec le meilleur à ce score. Le livre est la seule source
de l'exercice.

Les scores se tirent par **catégories** (*Money*, *DMP*, *Gammon save*, *Gammon
go*), parce que, de 4 à 7 away, les meilleurs coups changent à peine d'un score à
l'autre, alors que ces quatre situations jouent différemment. La consigne
nomme la catégorie à côté du score.

Tant qu'une question d'*Évaluation*, de *Décision* ou d'*Ouverture* attend sa
réponse, le panneau Analyse est masqué : il porte la réponse.

Répondre
~~~~~~~~

Le mode de réponse est une propriété de l'exercice, jamais un réglage : ce qui
se **compte ou se récite** se déclare, ce qui s'**estime** se saisit — parce
que là, la taille de l'erreur est la leçon.

*Scores* et *Comptage des pips* se **déclarent** : vous calculez de tête, vous cliquez
« Révéler », et la vérité s'affiche. Chaque nombre est alors **juste par
défaut** — vous cliquez celui que vous avez raté pour le marquer **faute**
(*Tab* puis *Espace* fait le même geste au clavier), et un second clic annule
la marque. Rien ne se tape : un compte de pions ou une case de table est juste
ou faux, et l'écrire n'apprend rien de plus que de le lire.

*Bearoff* se **saisit** : vous écrivez les deux EPC, « Valider » les juge à un
demi-pion près — la granularité à laquelle l'EPC change une décision de course
— et la vérité s'affiche à côté de ce que vous avez écrit. C'est l'application
qui juge, il n'y a rien à cocher. L'écart est enregistré **avec son signe** :
surestimer n'est pas sous-estimer, et c'est le bilan qui en fait une moyenne.

*Évaluation* mêle les deux gestes dans une même question. Les chances de gain
se **saisissent** et se jugent à cinq points près, écart signé compris ;
l'action de videau se **choisit** — le clic retient le bouton sans rien juger,
et « Valider » juge les deux en une fois. Le videau n'a pas de tolérance : seul
le bouton que le verdict du moteur rend juste est juste, et une position *trop
bonne pour doubler* se répond *Pas de double*. *Entrée* dans le champ mène à
l'action de videau tant qu'elle n'est pas choisie, puis valide. Après la
réponse, le panneau affiche la vérité — le verdict en quatre issues —, sa
source, et l'**EPC des deux camps** quand la position en a un exact ; l'EPC
n'est jamais demandé ici, il a son propre exercice. Le journal compte les deux
nombres à part : on peut bien estimer une position et mal lire son videau.

*Décision* se **choisit**. Sur une décision de pions, **jouez le coup sur le
damier** : un clic sur un pion le joue du premier dé non joué (le dé de gauche,
puis celui de droite), ou du suivant si le premier ne le déplace pas ; si aucun
dé ne le peut, rien ne bouge. Vous pouvez aussi glisser le pion. Un clic sur les
dés intervertit les dés restant à jouer, et fait juger le coup une fois complet.
Le clic droit sur le damier reprend les pas joués. Dans le panneau, « Annuler le
pas » revient d'un dé et « Recommencer » remet la position telle que la question
la pose. Le champ de notation accepte aussi le coup tapé (``13/7 8/7``, la
notation de la transcription) : les pas se posent sur le damier à chaque frappe,
un champ rougi dit qu'un pas n'est pas jouable, et ENTRÉE valide le coup
complet. Le panneau ayant le focus, RETOUR ARRIÈRE défait un pas et ÉCHAP
recommence le coup ; ENTRÉE valide le coup complet où que soit le focus, hors
d'un bouton. Sur une décision de videau, cliquez *Pas
de double*, *Double, prend* ou *Double, passe* : le clic est la réponse.

La correction distingue trois issues, et les confondre mentirait. Un **coup
illégal** n'est pas un coup mal choisi — c'est une faute de règle. Un **coup
légal que le moteur n'a pas classé** n'est pas une erreur de jugement : il n'a
simplement pas de prix, et il ne coûte rien. Un coup classé coûte ce que
l'analyse dit qu'il coûte, en millipoints. Seul un coup classé et sans coût est
juste ; le meilleur coup s'affiche dans tous les cas.

*Ouverture* se **choisit** aussi, coup joué sur le damier comme en *Décision*
— mêmes gestes, même champ de notation. Le panneau dit le jet à jouer et le
score, *vous* d'abord. Un coup est **juste** quand il est **à égalité avec le
meilleur** : il n'y a pas un seul bon coup, mais un groupe de tête que le livre
ne sait pas départager (voir :ref:`metrique_livre_ouverture`). Le coup joué,
le panneau marque *juste* ou *faux* et montre le tableau des candidats du
livre : l'**écart** de chacun au meilleur, l'**IC 95 %** de cet écart et la
mention **à égalité**, votre coup surligné — ajouté en bas, sans écart, quand
le livre ne le classe pas. Le nombre de parties et le réglage du rollout qui
ont produit ces chiffres sont dans l'infobulle du tableau. En *2nd roll*, la
consigne nomme le coup d'ouverture déjà joué. Le journal compte l'ouverture et
la réponse à part : on peut connaître ses ouvertures et mal répondre.

Le chronomètre part à l'affichage de la question et s'arrête à « Révéler », à
« Valider », au clic d'une action de videau de *Décision* ou au coup joué d'*Ouverture* ; la question suivante se prépare pendant que vous répondez, elle
n'est donc jamais chronométrée avec la vôtre. Cocher ses fautes n'est pas
chronométré non plus.
Avec une limite, une question restée
sans réponse à l'échéance se révèle seule et compte **hors délai** : tous ses
nombres sont faux, et son temps n'entre pas dans la médiane — on ne mesure pas
une réponse qui n'a pas été donnée. Une décision hors délai affiche le meilleur
coup, et n'entre pas dans le PR de la session.

« Suivante » enregistre la question et en pose une autre ; ENTRÉE, hors d'un
champ ou d'un bouton, fait de même une fois la réponse affichée, ou « Terminer »
s'il n'y a pas de question suivante. La session n'a pas de
longueur fixée : elle dure jusqu'à « Terminer », qui l'écrit au journal, ou
« Quitter », qui la jette. Tous les boutons sont dans la bande d'en-tête du
panneau, avec l'exercice, le numéro de la question et le chronomètre :
« Quitter », « Terminer », puis, toujours au bout, l'action de la question
(« Révéler », « Valider » ou « Suivante »). Le plateau montre la question et sa
réponse, il ne porte aucune commande.

Tant qu'une question est posée sur le plateau, révélée ou non, les touches qui
parcourent la liste ne la font pas défiler : la question garde le plateau
jusqu'à « Suivante », « Terminer » ou « Quitter ».

Le journal et le bilan
~~~~~~~~~~~~~~~~~~~~~~

Les sessions terminées sont conservées dans la base elle-même — elles suivent
donc le fichier — et sans plafond. Au repos, le panneau affiche une ligne par
exercice : le nombre de sessions, le taux de fautes, le temps médian et,
au-delà de dix sessions, la **tendance**, c'est-à-dire l'écart entre le taux de
fautes des dix dernières sessions et celui de toutes — négatif, vous
progressez.

Pour *Décision*, la ligne donne aussi le **PR** de la dernière session, calculé
par la formule que les statistiques appliquent au jeu réel — 500 × erreur
moyenne en équité normalisée, sur les décisions jugées. Un PR de 6 à
l'entraînement et un PR de 6 en match mesurent la même chose sur la même
échelle.

Cliquer le nom de l'exercice déplie le détail **par type de nombre** :
« Point de prise 4 · dernier lancer, 6 / 9 ». C'est ce détail qui fait
l'intérêt du journal, et il compte par type et non par face : la même case de
la même table, vue d'un côté ou de l'autre, est une seule faiblesse.

Une question de *Décision* garde au journal sa position, la réponse donnée et
son coût en millipoints. Le détail de *Décision* porte donc trois boutons qui
agissent sur toutes les positions ratées, chacune une fois, la plus récemment
ratée d'abord — une question hors délai compte comme ratée :

* **Reprendre mes ratés** — les positions ratées deviennent la liste parcourue
  et une session *Décision* repart dessus ;
* **Paquet Anki des ratés** — un paquet Anki de ces positions ;
* **Collection des ratés** — une collection de ces positions.

Le paquet et la collection sont nommés « Ratés à Décision » suivis de la date
du jour. En ligne de commande, ``training missed`` rend la même liste et en
fait un paquet (``--deck``) ou une collection (``--collection``), et
``training sessions`` relit le journal (voir :ref:`cli_training`).

.. _panneau_duel:

Panneau Duel
------------

Le panneau **Duel** fait jouer un match entier contre le Bot, sous l'arbitrage
de blunderDB : il lance les dés, impose les règles, tient le score et les
horloges. Il s'ouvre par ``CTRL-H``, par le bouton « Jouer » de la barre
d'outils, ou par la commande ``duel``. Une base doit être ouverte : le Duel s'y
écrit après chaque décision.

Sans Duel ouvert, le panneau montre le formulaire, mémorisé d'un Duel à
l'autre, et, s'il y en a, la liste des Duels en suspens. La bande d'en-tête
porte le **Départ**, les choix que la position de départ appelle, la
**cadence**, et « Jouer » à son bout :

* **Départ** : la position initiale, ou la position au plateau. Partant du
  plateau, celui-ci s'édite comme dans le panneau Eval (même plateau
  brouillon, mêmes gestes) : les pions, le videau et son propriétaire, les dés
  — posés du côté d'un joueur, ils lui donnent le trait — et le score.
* **Dés** (départ du plateau portant des dés) : « Dés du plateau », par défaut,
  joue le jet posé ; « Relancer » le fait tirer à nouveau par l'Arbitre.
* **Décision de videau** (départ du plateau sans dés à jouer, quand le joueur
  au trait peut doubler ou qu'un double est offert au plateau) : « Avant le
  videau », par défaut, commence par cette décision — le joueur au trait
  double ou lance, ou l'adversaire répond au double offert ; « Après le
  videau » passe au lancer, un double offert étant alors pris.
* **Cadence** : sans cadence, **Standard** (2 min par point + 12 s),
  **Speed** (0,4 min par point, soit 2 min pour un match en 5 points,
  + 10 s), ou une cadence enregistrée.

Le formulaire, un réglage par ligne :

* **Match** de 1 à 25 points, ou **session en argent** (Jacoby au choix).
* **Score** : ce qu'il manque à chaque joueur, par défaut la longueur du match
  (début de match) ; « 1 (Crawford) » est la partie Crawford, « 1
  (post-Crawford) » une partie qui la suit. Partant du plateau, ces champs et
  le score du plateau ne font qu'un : changer l'un change l'autre, et un
  plateau en argent fait une session en argent. **Une seule partie** arrête le
  Duel à la fin de sa première partie, pour s'entraîner à un score : le Match
  écrit ne compte que cette partie.
* **Cadence** (une cadence choisie) : **minutes par point** et **délai** en
  secondes. Chaque joueur dispose de minutes par point × longueur du match
  (moins, à un score de départ : la longueur moyenne restant à jouer) ; à
  chaque tour, le délai s'écoule d'abord, puis la réserve. Une réserve épuisée
  reste à zéro, et le temps écoulé continue en le notant, ou fait perdre le
  match. Un nom et « Enregistrer » gardent les deux nombres comme cadence du
  menu, dans les réglages de l'utilisateur ; « Supprimer » retire la cadence
  enregistrée choisie. Une session en argent se joue sans cadence.
* **Côté joué** (joueur 1 ou 2) et **niveau du Bot** (``instant``, ``normal``,
  ``thorough``, ceux de l'analyse). Le sélecteur dit la profondeur de chaque
  niveau, par exemple « instant (0 coup) » ou « normal (2 coups, élagué) » ; plus profond, c'est plus
  fort et plus lent.
* **Votre nom** et **Enregistrer le match** : décoché, le Duel terminé est
  jeté au lieu de devenir un Match.
* **Pipcount** : décoché, le pipcount reste masqué pendant le Duel, même par
  le bouton de la barre d'outils ou ``P`` ; il revient à la sortie du Duel.

« Reprendre » rouvre un Duel en suspens au même point, avec les mêmes dés à
venir ; ses horloges étaient arrêtées.

Un Duel en suspens ne quitte pas sa base : l'export de la base ne l'emporte
pas, parce que ses dés à venir ne doivent sortir par aucune voie avant la fin.
Un Duel terminé part comme tout Match.

Le Duel se joue au plateau. Le panneau montre la feuille de match en deux
colonnes, comme la Transcription, et les horloges quand une cadence court ;
sa bande d'en-tête nomme les joueurs, dit ce qui est attendu, et porte à
droite « Abandonner le match », « Mettre en pause », « Annuler le match ». Un match en points ne s'enregistre qu'entier :
il n'y a pas d'arrêt qui garde un match inachevé. Le score et le videau sont
ceux du plateau ; le score et les horloges restent dans la barre d'état quand
l'onglet est replié.
L'empreinte SHA-256 du germe des dés, publiée par l'Arbitre dès la création,
se lit dans l'infobulle de l'invite de la bande d'en-tête, puis avec le germe dans l'origine du Match terminé. Le plateau
passe en mode **DUEL** : la bibliothèque ne se parcourt plus, l'édition, le
panneau Eval et les autres onglets ne s'ouvrent pas, et le moteur se tait —
aucune évaluation, aucun candidat. Seules restent la Pile (``B``), le
pipcount (``P``, sauf s'il est masqué pour le Duel) et l'aide.

* Avant le lancer, un clic sur le plateau, sur les dés ou sur un pion les
  lance ; seul un clic sur le videau propose de doubler, et le plateau demande « Doubler » ou « Annuler ». Quand
  le videau n'est pas disponible, le lancer est automatique.
* Face à un double du Bot, le plateau demande « Prendre » ou « Passer ».
* Un clic sur un pion le joue aussitôt avec le dé de gauche encore libre, ou
  avec l'autre quand celui-ci ne peut pas le jouer ; un pion sur la barre
  entre, un double se joue en quatre clics. Un pion peut aussi se glisser
  vers sa destination. Seuls passent les pas d'un coup légal.
* Un dé se grise quand il est joué ; sur un double, chaque dé vaut deux pas
  et se voile à moitié au premier. Le coup complet grise tous les dés, y
  compris ceux que la règle ne permet pas de jouer.
* Tant que le coup n'est pas complet, un clic sur les dés intervertit les
  dés restant à jouer. Le clic droit sur le plateau reprend tous les pions
  joués (``RETOUR ARRIÈRE`` aussi) ; sans pion joué, il ouvre le menu du Duel.
* Le coup complet se valide par un clic sur les dés, par « Valider » sur le
  plateau, ou par ``ENTRÉE`` ou ``ESPACE``. Rien ne se reprend après.
* Le Bot répond aussitôt ; ses coups sont rejoués au plateau, lentement.
* Le clic droit hors du plateau, ou sur le plateau hors de son coup ou sans
  pion joué, ouvre le menu du Duel : mettre la position sur la Pile ou l'en retirer, abandonner la
  partie pour un simple, un gammon ou un backgammon (à son tour, après
  confirmation), abandonner le match, le mettre en pause, l'annuler. Ce menu
  n'offre ni évaluation ni édition.
* « Abandonner le match » cède le match entier, à tout moment et après
  confirmation : la partie en cours va à l'adversaire pour les points qui le
  portent à la longueur, et le Match s'écrit gagné par lui. En argent, la
  partie en cours est perdue en simple, à la valeur du videau (avec ou sans
  règle Jacoby), et la session se clôt. Sous une cadence qui fait perdre le
  match, une réserve épuisée vaut l'abandon du match par ce joueur.
* « Mettre en pause » met le Duel en suspens, horloges arrêtées ; il se
  reprend au même point. « Annuler le match » le jette, après confirmation :
  rien n'en est écrit.
* Un double-clic hors du plateau met la position sur la Pile, ou l'en retire,
  comme ``B`` ; seul le marque-page au coin du plateau le montre.

Chaque décision porte sa durée, avec ou sans cadence. Un abandon n'a pas de
durée enregistrée dans le Match.

À la fin, le Match est écrit, son analyse se lance et l'onglet Matchs s'ouvre
sur lui. En ligne de commande, ``blunderdb duel`` pilote le même Duel (voir
:ref:`cli_duel`).


.. _panneau_reference:

Panneau Données de référence
----------------------------

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
