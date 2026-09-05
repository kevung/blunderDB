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

Position importée individuellement
    Une position entrée seule dans la base — écrite depuis le plateau, ou
    lue depuis un fichier de position — plutôt qu'apportée par un match.
    Cette propriété est durable : une position ainsi importée le reste même
    si un match qui la contient est importé ensuite.

Position marquée
    Une position que l'utilisateur a signalée comme digne d'étude *dans le
    logiciel dont provient le match* — aujourd'hui seul eXtreme Gammon
    enregistre cette marque, coup par coup. Une décision de videau marquée
    marque les deux positions qui en découlent : le double et la
    prise/passe.

Analyse
    Ce qu'un moteur (eXtreme Gammon, GNUbg, BGBlitz ou l'évaluateur
    gammonNet embarqué) a conclu sur une position, enregistré et relu plus
    tard. Une position porte une analyse par moteur ; importer une analyse
    n'écrase jamais celle d'un autre moteur, ni une analyse déjà présente
    (voir :ref:`panneau_epc`).

Évaluation
    Le résultat d'un calcul que l'évaluateur embarqué effectue sur la
    position actuellement au plateau, enregistrée ou non. Une évaluation
    n'est valable que tant que la position ne change pas ; si elle est
    enregistrée, elle devient une Analyse.

Référentiel
    L'échelle dans laquelle un chiffre à propos d'une position s'exprime.
    *Money* répond à « combien de points cela vaut-il », un gammon valant
    toujours deux points. *Match* répond à « combien vaut-ce vis-à-vis du
    match », sur une échelle normalisée où un gammon vaut ce que le score
    en fait — presque tout à 2-away/4-away, rien pour un meneur à 1-away.
    Le référentiel est une propriété de la position, jamais une préférence
    d'affichage : c'est le score away qui le fixe.

Équité normalisée
    L'équité de match, exprimée sur l'échelle ``2 × MWC − 1`` où ±1
    représente le videau courant gagné ou perdu. C'est l'échelle que
    blunderDB affiche et stocke à un score de match, jamais les probabilités
    de victoire brutes.

Score away
    Le nombre de points qu'il reste à un joueur pour gagner le match, de son
    propre point de vue. Deux valeurs sont des sentinelles plutôt que des
    comptes : ``-1`` signifie qu'il n'y a pas de match — le référentiel est
    *money* — et ``0``/``1`` distinguent la partie Crawford (``1``) de la
    partie qui la suit (``0``), pour la même distance à la victoire.

Régime
    Quel type de réponse le panneau donne sur une course, toujours annoncé
    à l'écran plutôt que laissé à deviner :

    - *exact* — lu dans une base de bearoff two-sided, jamais estimé ;
    - *évalué* — calculé par l'évaluateur embarqué, qui rejoue la
      trajectoire ;
    - *estimé* — un résumé calibré d'une trajectoire, valable pour une
      probabilité de gain, jamais pour un verdict de videau.

Base bearoff
    Une table de réponses de course two-sided (le verdict de videau) ou
    one-sided (l'EPC). blunderDB en génère elle-même une partie au premier
    lancement (jusqu'à 6 pions par joueur) ; les domaines plus larges se
    génèrent depuis le panneau *Bearoff* ou la CLI. Une base générée est dite
    *vérifiée* quand son empreinte correspond exactement à celle produite
    par GNUbg.

EPC (Effective Pip Count)
    Une estimation classique, un-côté, du nombre de coups nécessaires pour
    sortir tous les pions d'un joueur — corrigée du gaspillage (*wastage*)
    par rapport au simple compte de pions.

Fait de position
    Une grandeur qui appartient au plateau lui-même, indépendante de tout
    choix d'un joueur : compte de pions, EPC, chances de gain/gammon/
    backgammon, équité cubeless. Vrai avant le jet de dés.

Ligne avant le jet
    Le vecteur des chances de gain/gammon/backgammon et de l'équité
    cubeless, affiché en tête de la liste des coups candidats une fois les
    dés jetés. L'écart entre cette ligne et un coup mesure la chance du
    lancer, jamais le mérite du coup joué.

Décision
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

Trop bon
    Le verdict qui dit que la position est si favorable que doubler
    inciterait l'adversaire à passer alors que jouer sans doubler rapporte
    davantage — un videau qu'il vaut mieux garder.

Chance
    L'apport du lancer de dés à l'équité d'un coup, lu depuis le fichier
    source à l'import (eXtreme Gammon, GNUbg) et jamais recalculé par
    blunderDB. Les formats qui ne la transportent pas (BGF, Jellyfish
    ``.mat``) n'en fournissent jamais, et l'import n'est pas rétroactif.

Blunder
    Un coup ou une décision de videau dont l'erreur (l'écart d'équité avec
    le meilleur choix) dépasse un seuil ; la commande ``bl`` (ou
    ``blunders``) charge directement les pires erreurs d'une recherche.

PR (Performance Rate)
    Le coût moyen des erreurs d'un joueur, en millièmes de point d'équité
    par décision comptée, coups et videau confondus. Voir
    :ref:`stats_parity` pour les règles de comptage exactes (parité avec
    eXtreme Gammon et GNUbg).

Snowie Error Rate
    Le même numérateur que le PR (la somme des erreurs), mais rapporté au
    nombre de coups joués plutôt qu'au nombre de décisions comptées — plus
    stable d'un logiciel à l'autre. Voir :ref:`stats_parity`.

Coût MWC
    Une erreur de videau exprimée en points de pourcentage de chances de
    gagner le match (*Match Winning Chance*), plutôt qu'en équité — la
    grandeur naturelle pour comparer un même joueur à des scores de match
    différents.

Collection
    Un ensemble nommé et ordonné de positions que l'utilisateur assemble à
    la main, glisser-déposer compris. L'appartenance à une collection est
    un geste explicite, contrairement à la position importée
    individuellement (voir :ref:`panneau_collections`).

Paquet Anki
    Un ensemble de positions transformées en cartes à réviser par
    répétition espacée (algorithme FSRS). Voir :ref:`panneau_anki`.

Carte de révision
    Une position d'un paquet Anki, présentée comme une question : le
    plateau est la question, l'analyse enregistrée est la réponse, masquée
    jusqu'au geste qui la révèle.

Commentaire
    Un texte libre attaché à une position, qu'il ait été tapé par
    l'utilisateur ou repris tel quel d'un fichier importé — les deux sont
    indiscernables. Un `#mot-clé` dans un commentaire est une étiquette,
    cherchable comme n'importe quel autre texte.

Joueur
    Un nom exactement tel qu'il apparaît dans un match importé. blunderDB ne
    devine pas la personne derrière le nom : la même personne signant sous
    deux orthographes est deux joueurs distincts, jusqu'à une fusion
    explicite (« Fusionner les joueurs »).

Filigrane
    La déclaration signée, apposée par qui exporte une base, de ce qu'elle
    est et d'où elle vient. Infalsifiable mais pas inamovible, elle ne
    consigne jamais rien du côté de qui reçoit la base. Voir
    :ref:`diffusion_controlee`.

Identité d'émetteur
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
