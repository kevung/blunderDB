.. _glossaire:

Glossaire
=========

Les termes ci-dessous reviennent dans tout le reste de la documentation —
manuel, guide utilisateur, FAQ, CLI. Cette page les rassemble en un seul
endroit plutôt que de les redéfinir à chaque section.

Position
    Un point de décision au backgammon : plateau, videau, dés, score et
    marques du match. Une position importée deux fois ne compte qu'une
    fois — c'est son identité, calculée automatiquement, qui la reconnaît,
    jamais le fichier d'où elle vient.

ArkID
    La notation textuelle d'une position propre à blunderDB, lisible et
    prononçable : ``[34]65:x2(ySQmLhfA)S3Qm3L3hf3`` est la position de départ,
    à 3 points et 4 points du but, 6-5 à jouer, le videau à 4 chez le joueur
    au trait. Elle s'écrit, se lit, s'affiche, se copie et se colle partout où
    l'on sait le faire d'un XGID, d'un OGID ou d'un ID GNUbg, et une position
    collée en ArkID retombe sur la même ligne que l'original.

    La forme est ``[score]dés:videau(portes)reste``. Le **score** donne les
    scores « à combien » (*Away score*) du joueur au trait puis de l'adversaire,
    ``[$]`` en partie d'argent, un ``c`` marquant la partie Crawford
    (``[14c]``). Les **dés** sont absents pour une décision de videau. Le
    **videau** est ``x`` s'il appartient au joueur au trait, ``X`` à
    l'adversaire, suivi de son exposant s'il dépasse 1 (``x`` : à 2, ``x2`` :
    à 4), et n'est pas écrit au centre. Les **portes** sont les points tenus par
    deux pions ou plus ; le **reste** donne, pour chaque case, le nombre de
    pions au-delà de deux (1 pour un pion seul, le nombre de pions pour la
    barre), omis quand il vaut 1. Les pions sortis se déduisent.

    Les lettres sont vues du côté du joueur au trait : les minuscules sont ses
    pions, les majuscules ceux de l'adversaire, tous deux numérotés comme le
    joueur au trait numérote son plateau. ``A`` à ``W`` sont les points 1 à 23,
    ``Y`` le point 24, ``Z`` la barre ; ``X`` est réservé au videau. Chaque
    lettre est l'initiale d'un animal, qui aide à retenir le point :

    .. csv-table::
       :header: "Lettre", "Point", "Animal"
       :widths: 8, 10, 60

      "A", "1", "Araignée — tapie dans un coin"
      "B", "2", "Bernard-l'ermite — dans sa coquille, point mort"
      "C", "3", "Crocodile — en eau basse, happe les pions"
      "D", "4", "Dauphin — deuxième meilleur point"
      "E", "5", "Éléphant — très puissant, indéplaçable"
      "F", "6", "Fourmi — gros tas, réservoir"
      "G", "7", "Gorille — le gros bras qui bloque la sortie"
      "H", "8", "Hérisson — piquant et solide"
      "I", "9", "Ibis — sur une patte, bâtisseur isolé"
      "J", "10", "Jaguar — prêt à bondir"
      "K", "11", "Kangourou — traverse l'outfield d'un bond"
      "L", "12", "Loup — dans la bergerie adverse"
      "M", "13", "Mouton — le troupeau du milieu"
      "N", "14", "Nasique — met le nez dehors"
      "O", "15", "Oie — en pleine migration pour rentrer"
      "P", "16", "Papillon — fragile, éphémère"
      "Q", "17", "Quetzal — oiseau rare, on y est rarement"
      "R", "18", "Rhinocéros — du très solide"
      "S", "19", "Souris — dure à attraper"
      "T", "20", "Tortue — carapace indestructible"
      "U", "21", "Urubu — vautour qui attend la frappe"
      "V", "22", "Vipère — tapie dans l'herbe basse"
      "W", "23", "Wombat — terrier profond"
      "Y", "24", "Yack — loin en altitude, lent à redescendre"
      "Z", "barre", "Zèbre — derrière les barreaux"

    À la lecture, blunderDB accepte la forme compacte et la forme développée
    (``[34]65:x2(ymhf)m3hf3(SQLA)S3QL3`` : d'abord tous les pions du joueur au
    trait, puis ceux de l'adversaire), un ordre libre et plusieurs groupes de
    parenthèses ; à l'écriture, il produit toujours la forme compacte
    canonique, la développée sur demande. Deux écritures de la même position
    par blunderDB sont donc identiques au caractère près. Un ArkID ne dit ni
    les règles de la session (Jacoby, castor), ni le plafond du videau : comme
    l'OGID, il ne les porte pas et ne les suppose pas. Il est refusé, en le
    disant, s'il compte plus de quinze pions par camp, une case tenue par les
    deux camps, une lettre inconnue, un score ou des dés invalides.

Position importée individuellement (*individually imported position*)
    Une position entrée seule dans la base — écrite depuis le plateau, ou
    lue depuis un fichier de position — plutôt qu'apportée par un match.
    Cette propriété est durable : une position ainsi importée le reste même
    si un match qui la contient est importé ensuite.

Position marquée (*flagged position*)
    Une position que l'utilisateur a signalée comme digne d'étude *dans le
    logiciel dont provient le match* — aujourd'hui seul eXtreme Gammon
    enregistre cette marque, coup par coup. Une décision de videau marquée
    marque les deux positions qui en découlent : le double et la
    prise/passe.

Analyse (*analysis*)
    Ce qu'un moteur (eXtreme Gammon, GNUbg, BGBlitz ou l'évaluateur
    gammonNet embarqué) a conclu sur une position, enregistré et relu plus
    tard. Une position porte une analyse par moteur ; importer une analyse
    n'écrase jamais celle d'un autre moteur, ni une analyse déjà présente
    (voir :ref:`import_regles`).

Évaluation (*evaluation*)
    Le résultat d'un calcul que l'évaluateur embarqué effectue sur la
    position actuellement au plateau, enregistrée ou non. Une évaluation
    n'est valable que tant que la position ne change pas ; si elle est
    enregistrée, elle devient une Analyse.

Référentiel (*referential*)
    L'échelle dans laquelle un chiffre à propos d'une position s'exprime.
    *Money* répond à « combien de points cela vaut-il », un gammon valant
    toujours deux points. *Match* répond à « combien vaut-ce vis-à-vis du
    match », sur une échelle normalisée où un gammon vaut ce que le score
    en fait — presque tout à 2-away/4-away, rien pour un meneur à 1-away.
    Le référentiel est une propriété de la position, jamais une préférence
    d'affichage : c'est le score away qui le fixe.

Équité normalisée (*normalised equity*)
    L'équité de match, exprimée sur l'échelle ``2 × MWC − 1`` où ±1
    représente le videau courant gagné ou perdu. C'est l'échelle que
    blunderDB affiche et stocke à un score de match, jamais les probabilités
    de victoire brutes.

EMG
    Le sigle sous lequel eXtreme Gammon et GNUbg désignent l'équité
    normalisée ci-dessus, et sous lequel les statistiques de blunderDB
    comptent les erreurs : « une erreur de 0,100 EMG » est une erreur de
    0,100 sur cette échelle. Les deux termes nomment la même grandeur.

Erreur / blunder (*error*, *blunder*)
    Une décision dont le coût atteint le **seuil d'erreur** de la bibliothèque
    est une *erreur* ; une erreur dont le coût atteint son **seuil de blunder**
    est un *blunder*. Tout blunder est donc une erreur. Les deux seuils se
    règlent dans l'onglet *Bibliothèque* de la configuration, valent 0,050 et
    0,100 EMG par défaut, et suivent le fichier plutôt que la machine — ils ne
    voyagent pas dans un export.

Score away (*away score*)
    Le nombre de points qu'il reste à un joueur pour gagner le match, de son
    propre point de vue. Deux valeurs sont des sentinelles plutôt que des
    comptes : ``-1`` signifie qu'il n'y a pas de match — le référentiel est
    *money* — et ``0``/``1`` distinguent la partie Crawford (``1``) de la
    partie qui la suit (``0``), pour la même distance à la victoire.

Régime (*regime*)
    Quel type de réponse le panneau donne sur une course, toujours annoncé
    à l'écran plutôt que laissé à deviner :

    - *exact* — lu dans une base de bearoff two-sided, jamais estimé ;
    - *évalué* — calculé par l'évaluateur embarqué, qui rejoue la
      trajectoire ;
    - *estimé* — un résumé calibré d'une trajectoire, valable pour une
      probabilité de gain, jamais pour un verdict de videau.

Base bearoff (*bearoff database*)
    Une table de réponses de course two-sided (le verdict de videau) ou
    one-sided (l'EPC). blunderDB en génère elle-même une partie au premier
    lancement (jusqu'à 6 pions par joueur) ; les domaines plus larges se
    génèrent depuis l'onglet *Bearoff* de la configuration ou la CLI. Une base générée est dite
    *vérifiée* quand son empreinte correspond exactement à celle produite
    par GNUbg.

EPC (Effective Pip Count)
    Une estimation classique, un-côté, du nombre de coups nécessaires pour
    sortir tous les pions d'un joueur — corrigée du gaspillage (*wastage*)
    par rapport au simple compte de pions.

Fait de position (*position fact*)
    Une grandeur qui appartient au plateau lui-même, indépendante de tout
    choix d'un joueur : compte de pions, EPC, chances de
    gain/gammon/backgammon, équité cubeless. Vrai avant le jet de dés.

Ligne avant le jet (*pre-roll baseline*)
    Le vecteur des chances de gain/gammon/backgammon et de l'équité
    cubeless, affiché en tête de la liste des coups candidats une fois les
    dés jetés. L'écart entre cette ligne et un coup mesure la chance du
    lancer, jamais le mérite du coup joué.

Décision (*decision*)
    La réponse à la question que le plateau pose : le classement des coups
    candidats quand des dés sont jetés, ou les trois options de videau (pas
    de double, double-prend, double-passe) et leur verdict sinon. Une
    position ne pose jamais les deux questions à la fois.

Verdict
    Laquelle des options d'une décision de videau est la bonne : *pas de
    double*, *double/prend*, *double/passe* ou *trop bon (pour doubler)*.
    Deux cas particuliers ne sont jamais laissés vides : *pas de décision*
    (le régime n'y est pas habilité) et *refusé* (le moteur a décliné la
    position, typiquement un score hors de portée de sa table).

Trop bon (*too good*)
    Le verdict qui dit que la position est si favorable que doubler
    inciterait l'adversaire à passer alors que jouer sans doubler rapporte
    davantage — un videau qu'il vaut mieux garder.

Chance (*luck*)
    L'apport du lancer de dés à l'équité d'un coup, lu depuis le fichier
    source à l'import (eXtreme Gammon, GNUbg) et jamais recalculé par
    blunderDB. Les formats qui ne la transportent pas (BGF, Jellyfish
    ``.mat``) n'en fournissent jamais, et l'import n'est pas rétroactif.

Blunder
    Un coup ou une décision de videau dont l'erreur (l'écart d'équité avec
    le meilleur choix) dépasse un seuil ; la commande ``bl`` (ou
    ``blunders``) charge directement les pires erreurs retenues par le
    filtre des statistiques.

Millipoint (mpt)
    Un millième de point d'équité : une erreur de 0,100 d'équité vaut
    100 mpt. C'est l'unité dans laquelle se lisent les grandeurs d'une
    position — le seuil de blunder, les filtres de recherche ``e>`` et
    ``E>``, l'histogramme des magnitudes d'erreur, la chance moyenne par
    lancer. Le PR ne se lit pas sur cette échelle : son facteur est 500 et
    non 1000, si bien qu'un PR de 5,0 vaut 10 mpt d'erreur moyenne par
    décision.

PR (*Performance Rating*)
    Le coût moyen des erreurs d'un joueur, coups et videau confondus :
    l'erreur moyenne d'équité par décision comptée, multipliée par 500
    comme le font eXtreme Gammon et GNUbg. Un PR de 5,0 vaut donc 0,010
    d'équité perdue par décision. Voir :ref:`stats_parity` pour les règles
    de comptage exactes (parité avec eXtreme Gammon et GNUbg).

Snowie Error Rate
    Le même numérateur que le PR (la somme des erreurs), mais rapporté au
    nombre de coups joués plutôt qu'au nombre de décisions comptées — plus
    stable d'un logiciel à l'autre. Voir :ref:`stats_parity`.

Coût MWC (*MWC cost*)
    Une erreur de videau exprimée en points de pourcentage de chances de
    gagner le match (*Match Winning Chance*), plutôt qu'en équité — la
    grandeur naturelle pour comparer un même joueur à des scores de match
    différents.

Collection
    Un ensemble nommé et ordonné de positions que l'utilisateur assemble à
    la main, glisser-déposer compris. L'appartenance à une collection est
    un geste explicite, contrairement à la position importée
    individuellement (voir :ref:`panneau_collections`).

Paquet Anki (*Anki deck*)
    Un ensemble de positions transformées en cartes à réviser par
    répétition espacée (algorithme FSRS). Voir :ref:`panneau_anki`.

Carte de révision (*review card*)
    Une position d'un paquet Anki, présentée comme une question : le
    plateau est la question, l'analyse enregistrée est la réponse, masquée
    jusqu'au geste qui la révèle.

Commentaire (*comment*)
    Un texte libre attaché à une position, tapé par l'utilisateur ou repris
    d'un fichier importé. Une position peut en porter plusieurs, chacun signé
    de son auteur ; un commentaire importé garde sa provenance (``XG``,
    ``GNU BG``, ``BGF``). Un ``#mot`` dans un commentaire est un tag, que le
    filtre ``#mot`` retrouve (voir :ref:`panneau_commentaires`).

Joueur (*player*)
    Un nom exactement tel qu'il apparaît dans un match importé. blunderDB ne
    devine pas la personne derrière le nom : la même personne signant sous
    deux orthographes est deux joueurs distincts, jusqu'à une fusion
    explicite (« Gérer les joueurs »).

Filigrane (*watermark*)
    La déclaration signée, apposée par qui exporte une base, de ce qu'elle
    est et d'où elle vient. Infalsifiable mais pas inamovible, elle ne
    consigne jamais rien du côté de qui reçoit la base. Voir
    :ref:`diffusion_controlee`.

Identité d'émetteur (*issuer identity*)
    La clé qui signe les filigranes d'une personne — créée d'elle-même au
    premier filigrane apposé, et déplaçable d'une machine à l'autre par un
    seul fichier.

Fichier protégé (.dbx)
    Une base exportée et chiffrée par mot de passe, ouverte une fois puis
    ordinaire. Elle protège le fichier en transit, pas son contenu : qui
    reçoit le mot de passe peut l'ouvrir.

Tenant
    En mode serveur, le propriétaire d'un ensemble de positions, matchs et
    collections. Rien de ce qu'un tenant stocke n'est jamais visible à un
    autre. Sur le poste de bureau il n'y en a qu'un seul, implicite : la
    personne dont c'est le fichier. Voir :ref:`headless`.
