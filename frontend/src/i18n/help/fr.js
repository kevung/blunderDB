// GENERATED FILE — do not edit by hand, and do not translate it here.
//
// Produced by `go run ./cmd/help-gen` (make help) from:
//   - doc/source/raccourcis.rst  → the "shortcuts" tab
//   - doc/source/cmd_mode.rst    → the "commands" tab
//   - doc/source/locale/<lang>/LC_MESSAGES/*.po for the eight translations
//   - frontend/src/i18n/help/prose/<lang>.html → the "manual" and "about" tabs
//
// Fix the documentation (and its .po catalogues), or the prose fragment, then
// run `make help`. TestHelpBundlesAreCurrent fails if this file is stale.
export default {
    manual: `
<h3>Introduction</h3>
<p>
    blunderDB est un logiciel permettant de créer des bases de données de positions de backgammon. Sa principale force est d'offrir un endroit unique où regrouper les positions qu'un joueur a
    rencontrées (en ligne, en tournoi) et de pouvoir réétudier ces positions en les filtrant selon divers filtres combinables de façon arbitraire. blunderDB peut aussi être utilisé pour créer des
    catalogues de positions de référence.
</p>
<p>Les positions sont stockées dans une base de données représentée par un fichier .db.</p>

<h3>Interactions principales</h3>
<p>Les principales interactions possibles avec blunderDB sont :</p>
<ul>
    <li>ajouter une nouvelle position,</li>
    <li>modifier une position existante,</li>
    <li>copier le plateau sous forme d'image PNG dans le presse-papiers (<strong>Ctrl+X</strong>), ou le plateau avec son analyse (<strong>Ctrl+X, Ctrl+X</strong>),</li>
    <li>supprimer une position existante,</li>
    <li>rechercher une ou plusieurs positions,</li>
    <li>importer des matchs depuis diverses sources (XG, GNUbg, BGBlitz, Jellyfish), y compris les commentaires des fichiers XG,</li>
    <li>parcourir les coups d'un match importé,</li>
    <li>organiser les positions en collections,</li>
    <li>organiser les matchs en tournois,</li>
    <li>analyser en lot, depuis un terminal, les positions dépourvues d'analyse grâce à l'évaluateur gammonNet embarqué (commande <strong>analyze</strong> de blunderDB).</li>
</ul>
<p>L'utilisateur peut librement étiqueter les positions et les annoter avec des commentaires.</p>

<h3>Description de l'interface</h3>
<p>L'interface de blunderDB est structurée de haut en bas de la manière suivante :</p>
<ul>
    <li>[en haut] la barre d'outils, qui rassemble toutes les opérations principales pouvant être effectuées sur la base de données,</li>
    <li>[au milieu] la zone d'affichage principale, qui permet d'afficher ou d'éditer des positions de backgammon,</li>
    <li>[en bas] la barre d'état, qui intègre la ligne de commande et présente diverses informations sur la position courante.</li>
</ul>
<p>Des panneaux peuvent être affichés pour :</p>
<ul>
    <li>afficher les données d'analyse associées à la position courante (de XG, GNUbg ou BGBlitz),</li>
    <li>afficher, ajouter ou modifier des commentaires,</li>
    <li>parcourir les matchs importés et naviguer dans leurs coups (panneau Match),</li>
    <li>gérer des collections de positions (panneau Collection),</li>
    <li>étudier des positions par répétition espacée (panneau Anki),</li>
    <li>gérer des tournois (panneau Tournoi),</li>
    <li>afficher des statistiques de performance (panneau Stats),</li>
    <li>évaluer n'importe quelle position avec le moteur intégré, et calculer l'EPC d'une position de sortie (panneau Eval),</li>
    <li>consulter les filtres de recherche enregistrés (panneau Bibliothèque de filtres),</li>
    <li>consulter l'historique des recherches (panneau Historique des recherches).</li>
</ul>
<p>La zone d'affichage principale fournit à l'utilisateur :</p>
<ul>
    <li>un plateau pour afficher ou éditer une position de backgammon,</li>
    <li>le niveau et le propriétaire du videau,</li>
    <li>le pip count de chaque joueur,</li>
    <li>le score de chaque joueur,</li>
    <li>les dés à jouer. Si aucune valeur n'est affichée sur les dés, la position des dés indique quel joueur a le trait et que la position est une décision de videau.</li>
</ul>
<p>La barre d'état affiche de gauche à droite :</p>
<ul>
    <li>la ligne de commande (appuyez sur <strong>Espace</strong> pour l'ouvrir),</li>
    <li>un message d'information lié à la dernière opération effectuée,</li>
    <li>l'indice de la position courante, suivi du nombre total de positions (ou des informations de coup/partie lors de la navigation dans un match).</li>
</ul>
<p>Dans le cas de positions issues d'une recherche de l'utilisateur, le nombre de positions indiqué dans la barre d'état correspond au nombre de positions filtrées.</p>

<h3>Parcourir les positions</h3>
<p>Par défaut, blunderDB vous permet de :</p>
<ul>
    <li>faire défiler les différentes positions de la bibliothèque courante,</li>
    <li>afficher les informations d'analyse associées à une position,</li>
    <li>afficher, ajouter et modifier des commentaires sur une position.</li>
</ul>

<h3>Éditer des positions</h3>
<p>
    Appuyer sur la touche <strong>Tab</strong> ouvre le panneau de recherche et permet d'éditer une position sur le plateau afin de l'ajouter à la base de données ou de définir une structure de
    position à rechercher. La répartition des pions, le videau, le score et le trait peuvent être modifiés à l'aide de la souris.
</p>

<h3>Ligne de commande</h3>
<p>
    La ligne de commande, intégrée dans la barre d'état, permet d'exécuter toutes les fonctionnalités de blunderDB : opérations sur la base de données, navigation dans les positions, affichage des
    analyses et des commentaires, recherche de positions avec filtres... Après vous être familiarisé avec l'interface, il est recommandé d'utiliser progressivement la ligne de commande, qui permet une
    utilisation puissante et fluide de blunderDB, en particulier pour les fonctionnalités de recherche de positions.
</p>
<p>
    Pour ouvrir la ligne de commande, appuyez sur la touche <strong>Espace</strong>. Une invite apparaît dans la barre d'état. Tapez votre commande et appuyez sur <strong>Entrée</strong> pour
    l'exécuter. Appuyez sur
    <strong>Échap</strong>
    pour annuler.
</p>
<p>
    blunderDB exécute les requêtes envoyées par l'utilisateur dès lors qu'elles sont valides et modifie immédiatement l'état de la base de données si nécessaire. Aucune action d'enregistrement
    explicite n'est requise de la part de l'utilisateur.
</p>
<p>
    Pour affiner une recherche au sein de positions préalablement filtrées, utilisez la commande <strong>ss</strong> suivie de filtres (par ex. <strong>ss nc</strong>). Cela restreint la recherche aux
    seules positions actuellement affichées, ce qui permet de réduire progressivement les résultats. Le panneau de recherche (<strong>Ctrl+F</strong>) propose aussi une case « Rechercher dans les
    résultats courants » pour la même fonctionnalité.
</p>

<h3>Panneau Eval</h3>
<p>
    Le panneau <strong>Eval</strong> évalue la position posée sur le plateau, quelle qu'elle soit : probabilités de gain, de gammon et de backgammon, équité, coups candidats classés, et la seule
    décision que la position appelle — jouer un coup, ou doubler. Le calcul est fait par gammonNet, embarqué : ni eXtreme Gammon ni GNU Backgammon ne sont requis.
</p>
<p>
    Pour l'ouvrir, appuyez sur <strong>Ctrl+E</strong>, cliquez sur l'onglet Eval du panneau inférieur ou tapez <strong>epc</strong> dans la ligne de commande. Le plateau s'ouvre sur une configuration
    de sortie standard (15 pions), à moins qu'une position de la base n'y ait été envoyée. Les pions s'ajoutent et se retirent librement à la souris ; l'évaluation suit chaque geste.
</p>
<p>
    Sur une position de sortie, le panneau <strong>se spécialise</strong> : un second tableau, par joueur, porte le calcul de l'EPC (Effective Pip Count) à partir de la base de sortie unilatérale à 6
    points de GNUbg —
</p>
<ul>
    <li><strong>EPC</strong> : le nombre moyen de pips nécessaires pour sortir tous les pions,</li>
    <li><strong>Pip Count</strong> : le pip count brut,</li>
    <li><strong>Wastage</strong> : la différence entre l'EPC et le pip count,</li>
    <li><strong>Avg Rolls</strong> : le nombre moyen de jets pour sortir tous les pions,</li>
    <li><strong>Std Dev</strong> : l'écart-type du nombre de jets.</li>
</ul>
<p>Lorsque les deux joueurs ont des pions dans leur jan intérieur, une section de comparaison affiche les différences d'EPC et de pip count.</p>
<p>
    Sur une position de course pure, un tableau supplémentaire affiche les probabilités de gain des deux joueurs et, lorsque la position est couverte par une base two-sided (table à 6 pions par joueur
    calculée au premier lancement, table étendue à 11 pions calculée depuis l'onglet Bearoff de la configuration), les équités money exactes et la meilleure décision de videau. Hors de ce domaine, la
    probabilité de gain est estimée (badge « estimé » avec sa marge d'erreur) et aucune décision n'est affichée. Le joueur au trait s'édite en cliquant le rectangle sortie/score d'un joueur, la
    position du videau en cliquant le videau du plateau.
</p>
<p>
    La case <strong>Défi</strong> masque les résultats à chaque modification de la position ; cliquez une zone pour la révéler — idéal pour s'entraîner à estimer une équité, un EPC ou une décision de
    videau avant de vérifier.
</p>
<p>Pour fermer le panneau Eval, appuyez de nouveau sur <strong>Ctrl+E</strong> ou passez à un autre onglet.</p>

<h3>Navigation dans les matchs</h3>
<p>
    blunderDB permet de parcourir les coups des matchs importés. Ouvrez le panneau Match avec <strong>Ctrl+Tab</strong> et double-cliquez sur un match (ou appuyez sur <strong>Entrée</strong>) pour
    charger ses positions.
</p>
<p>
    Lors de la navigation dans un match, la dernière position visitée est automatiquement enregistrée et restaurée. Utilisez les touches <strong>Gauche</strong>/<strong>Droite</strong> pour vous
    déplacer entre les positions, et <strong>PageUp</strong>/<strong>PageDown</strong> pour sauter d'une partie à l'autre.
</p>
<p>
    Le panneau d'analyse (<strong>Ctrl+L</strong>) affiche l'analyse de chaque coup, le coup joué étant mis en évidence. Appuyez sur <strong>d</strong> pour basculer entre l'analyse de pions et
    l'analyse de videau.
</p>

<h3>Collections</h3>
<p>
    Les collections permettent d'organiser les positions en groupes personnalisés. Ouvrez le panneau Collection avec <strong>Ctrl+B</strong>, puis double-cliquez sur une collection pour parcourir ses
    positions. Les collections et les positions qu'elles contiennent peuvent être réordonnées par glisser-déposer.
</p>

<h3>Anki (répétition espacée)</h3>
<p>Le panneau Anki (<strong>Ctrl+K</strong>) offre la répétition espacée pour étudier des positions de backgammon à l'aide de l'algorithme FSRS.</p>
<p>
    <strong>Créer des paquets :</strong> Cliquez sur <em>Nouveau paquet</em> pour créer un paquet à partir d'une collection ou des résultats de recherche courants. Les paquets basés sur une recherche
    se synchronisent automatiquement lorsque l'onglet Anki est activé.
</p>
<p>
    <strong>Réviser :</strong> Sélectionnez un paquet et cliquez sur <em>Étudier</em> (ou double-cliquez sur un paquet) pour commencer à réviser les cartes à échéance. Chaque carte affiche la position
    correspondante sur le plateau. Notez votre rappel avec les touches <strong>1</strong> (À revoir), <strong>2</strong> (Difficile), <strong>3</strong> (Bien) ou <strong>4</strong> (Facile). Appuyez
    sur <strong>Échap</strong> pour arrêter et revenir à la liste des paquets.
</p>
<p>
    <strong>Limiter la séance :</strong> Dans les Paramètres du paquet, vous pouvez borner une séance à un nombre de cartes. La séance s'arrête alors en le disant, et l'entraînement libre reste
    disponible pour continuer sans toucher au planning. Une limite de <em>0</em> ne sert aucune carte — ce n'est pas la même chose que « pas de limite ».
</p>
<p>
    <strong>Rétention :</strong> La rétention cible est votre choix sur le compromis charge/qualité. Les Paramètres affichent en regard la rétention <em>mesurée</em> sur vos révisions — une
    information, jamais un pilotage. Changer la cible n'est pas rétroactif : chaque carte adopte le nouveau rythme à sa prochaine révision.
</p>
<p>
    <strong>Afficher la réponse :</strong> La carte pose une question ; réfléchissez, puis appuyez sur <strong>Espace</strong> (ou cliquez sur la zone masquée) pour dévoiler l'analyse enregistrée de
    la position. Elle s'affiche sous les boutons de notation, qui restent à portée. Rien ne vous oblige à la dévoiler pour noter, et elle se remasque à la carte suivante — pas si vous changez
    simplement d'onglet.
</p>
<p>
    <strong>Arrêter/Reprendre :</strong> Vous pouvez arrêter une session de révision à tout moment en appuyant sur <strong>Échap</strong>. Le bouton devient <em>Reprendre</em> et affiche votre
    progression. Cliquez dessus pour continuer là où vous vous êtes arrêté.
</p>
<p>
    <strong>Gestion des paquets :</strong> Utilisez les boutons d'action pour renommer, synchroniser, réinitialiser ou supprimer des paquets. Les paramètres FSRS (rétention cible, intervalle maximal,
    fuzz) peuvent être configurés par paquet dans les Réglages (icône d'engrenage).
</p>

<h3>Tournois</h3>
<p>
    Les tournois permettent de regrouper les matchs par événement. À l'import, un match entre dans le tournoi que son fichier nomme, créé au besoin ; un match déjà rangé n'est jamais déplacé. Ouvrez
    le panneau Tournoi avec <strong>Ctrl+Y</strong> pour gérer les tournois et leur affecter des matchs.
</p>

<h3>Stats</h3>
<p>
    Le panneau Stats (<strong>Ctrl+D</strong>) affiche des statistiques de performance (PR et coût en MWC) calculées à partir de toutes les positions importées. Utilisez la barre de filtres pour
    restreindre l'analyse par joueur, tournoi, plage de dates, type de décision ou longueur de match. Cliquez sur n'importe quel indicateur pour explorer en détail les positions correspondantes.
    L'onglet <strong>Joueurs</strong> liste, par joueur, le nombre de matchs, le bilan, les décisions, le PR (pions et videau), le Snowie, les blunders et la chance mesurée sur les jets connus.
</p>

<h3>Filigrane et export protégé</h3>
<p>Lors d'un export (<strong>export_db</strong> ou la boîte de dialogue Exporter), deux protections indépendantes peuvent être activées librement, l'une, l'autre, ou les deux à la fois :</p>
<ul>
    <li>
        <strong>Filigrane :</strong> marque le fichier exporté de son origine (qui l'a produit, une note facultative). Le filigrane est signé par votre identité d'émetteur : il ne peut être ni modifié
        ni contrefait au nom de quelqu'un d'autre — mais il n'est pas ineffaçable et n'empêche aucune copie.
    </li>
    <li>
        <strong>Mot de passe :</strong> place l'export dans un conteneur chiffré <strong>.dbx</strong>. Il protège le fichier pendant son transport, pas la base elle-même — celui à qui vous donnez le
        mot de passe peut l'ouvrir — et l'origine reste lisible sans lui.
    </li>
</ul>
<p>
    Votre identité d'émetteur, la clé qui signe vos filigranes, se crée automatiquement au premier export marqué de son origine. Consultez-la, exportez-la ou régénérez-la depuis l'onglet
    <strong>Identité d'émetteur</strong> de la configuration.
</p>
`,
    shortcuts: `
<h3>Base de données</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-N</td>
<td>Créer une nouvelle base de données.</td>
</tr>
<tr>
<td>CTRL-O</td>
<td>Ouvrir une base de données existante.</td>
</tr>
<tr>
<td>CTRL-MAJ-I</td>
<td>Fusionner une base de données dans celle-ci.</td>
</tr>
<tr>
<td>CTRL-MAJ-S</td>
<td>Exporter la base de données.</td>
</tr>
<tr>
<td>CTRL-Q</td>
<td>Fermer blunderDB.</td>
</tr>
<tr>
<td>CTRL-M</td>
<td>Modifier les métadonnées de la base de données.</td>
</tr>
</tbody>
</table>
<h3>Position</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-I</td>
<td>Importer une ou plusieurs positions/matchs par fichier (xg, xgp, sgf, mat, txt, bgf).</td>
</tr>
<tr>
<td>CTRL-MAJ-F</td>
<td>Importer récursivement un dossier de fichiers de matchs/positions.</td>
</tr>
<tr>
<td>CTRL-C</td>
<td>Copier une position dans le presse-papier.</td>
</tr>
<tr>
<td>CTRL-X</td>
<td>Copier l'image du board dans le presse-papier (PNG).</td>
</tr>
<tr>
<td>CTRL-X CTRL-X</td>
<td>Copier l'image du board avec l'analyse dans le presse-papier (PNG).</td>
</tr>
<tr>
<td>CTRL-V</td>
<td>Coller une position depuis le presse-papier (détection automatique du format).</td>
</tr>
<tr>
<td>CTRL-S</td>
<td>Enregistrer une position.</td>
</tr>
<tr>
<td>CTRL-U</td>
<td>Mettre à jour une position.</td>
</tr>
<tr>
<td>Del</td>
<td>Supprimer la position courante (confirmation demandée).</td>
</tr>
<tr>
<td>RETOUR ARRIERE</td>
<td>Réinitialiser le board, le cube, le score et les dés.</td>
</tr>
<tr>
<td>CTRL-G</td>
<td>Afficher les métadonnées de la position.</td>
</tr>
</tbody>
</table>
<h3>Navigation</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-R</td>
<td>Recharger toutes les positions de la base de données.</td>
</tr>
<tr>
<td>PageUp, h</td>
<td>Première position / Partie précédente (navigation match).</td>
</tr>
<tr>
<td>GAUCHE, k</td>
<td>Position précédente.</td>
</tr>
<tr>
<td>DROITE, j</td>
<td>Position suivante.</td>
</tr>
<tr>
<td>HAUT, k</td>
<td>Coup précédent (lorsqu'un coup est sélectionné dans l'analyse).</td>
</tr>
<tr>
<td>BAS, j</td>
<td>Coup suivant (lorsqu'un coup est sélectionné dans l'analyse).</td>
</tr>
<tr>
<td>PageDown, l</td>
<td>Dernière position / Partie suivante (navigation match).</td>
</tr>
<tr>
<td>r</td>
<td>Charger une position aléatoire.</td>
</tr>
</tbody>
</table>
<h3>Affichage</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-GAUCHE</td>
<td>Orientation du board à gauche.</td>
</tr>
<tr>
<td>CTRL-DROITE</td>
<td>Orientation du board à droite.</td>
</tr>
<tr>
<td>p</td>
<td>Afficher/cacher le compte de course.</td>
</tr>
</tbody>
</table>
<h3>Actions</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>TAB</td>
<td>Ouvrir le panneau de recherche (éditeur de position).</td>
</tr>
<tr>
<td>ESPACE</td>
<td>Ouvrir la ligne de commande.</td>
</tr>
</tbody>
</table>
<h3>Outils</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-L</td>
<td>Afficher/cacher l'analyse.</td>
</tr>
<tr>
<td>CTRL-P</td>
<td>Afficher/cacher les commentaires.</td>
</tr>
<tr>
<td>CTRL-K</td>
<td>Afficher/cacher le panneau Anki (répétition espacée).</td>
</tr>
<tr>
<td>CTRL-F</td>
<td>Afficher/cacher le panneau de recherche.</td>
</tr>
<tr>
<td>CTRL-Tab</td>
<td>Afficher/cacher le panneau des matchs.</td>
</tr>
<tr>
<td>CTRL-B</td>
<td>Afficher/cacher le panneau des collections.</td>
</tr>
<tr>
<td>CTRL-Y</td>
<td>Afficher/cacher le panneau des tournois.</td>
</tr>
<tr>
<td>CTRL-D</td>
<td>Afficher/cacher le panneau Stats.</td>
</tr>
<tr>
<td>CTRL-E</td>
<td>Afficher/cacher le panneau Eval.</td>
</tr>
<tr>
<td>?</td>
<td>Afficher/cacher l'aide.</td>
</tr>
</tbody>
</table>
<h3>Onglets de vues</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-T</td>
<td>Créer une nouvelle vue (copie de la vue courante).</td>
</tr>
<tr>
<td>CTRL-W</td>
<td>Fermer la vue courante.</td>
</tr>
<tr>
<td>CTRL-PageUp, MAJ-J</td>
<td>Vue précédente.</td>
</tr>
<tr>
<td>CTRL-PageDown, MAJ-K</td>
<td>Vue suivante.</td>
</tr>
<tr>
<td>CTRL-1 … CTRL-9</td>
<td>Aller directement à la n-ième vue.</td>
</tr>
<tr>
<td>Double-clic sur l'onglet</td>
<td>Renommer la vue.</td>
</tr>
</tbody>
</table>
<h3>Ligne de commande</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>HAUT</td>
<td>Parcourir l'historique des commandes vers le haut.</td>
</tr>
<tr>
<td>BAS</td>
<td>Parcourir l'historique des commandes vers le bas.</td>
</tr>
</tbody>
</table>
<h3>Historique de recherche</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Sélectionner/désélectionner une recherche (afficher la position).</td>
</tr>
<tr>
<td>Double-clic</td>
<td>Exécuter la recherche.</td>
</tr>
</tbody>
</table>
<h3>Bibliothèque de filtres</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Sélectionner/désélectionner un filtre (afficher la position).</td>
</tr>
<tr>
<td>Double-clic</td>
<td>Exécuter la recherche du filtre.</td>
</tr>
</tbody>
</table>
<h3>Panneau d'analyse</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Sélectionner/désélectionner un coup (afficher/cacher les flèches).</td>
</tr>
<tr>
<td>HAUT, k</td>
<td>Sélectionner le coup précédent (lorsqu'un coup est sélectionné).</td>
</tr>
<tr>
<td>BAS, j</td>
<td>Sélectionner le coup suivant (lorsqu'un coup est sélectionné).</td>
</tr>
<tr>
<td>d</td>
<td>Basculer entre l'analyse des coups et du cube (navigation match uniquement).</td>
</tr>
<tr>
<td>Esc</td>
<td>Désélectionner le coup. Si aucun coup sélectionné, fermer le panneau.</td>
</tr>
</tbody>
</table>
<h3>Panneau Eval</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Sélectionner/désélectionner un coup (afficher/cacher les flèches).</td>
</tr>
<tr>
<td>HAUT, k</td>
<td>Sélectionner le coup précédent (lorsqu'un coup est sélectionné).</td>
</tr>
<tr>
<td>BAS, j</td>
<td>Sélectionner le coup suivant (lorsqu'un coup est sélectionné).</td>
</tr>
<tr>
<td>Esc</td>
<td>Désélectionner le coup.</td>
</tr>
</tbody>
</table>
<h3>Panneau des matchs</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Sélectionner un match.</td>
</tr>
<tr>
<td>Double-clic</td>
<td>Naviguer dans le match.</td>
</tr>
<tr>
<td>HAUT, k</td>
<td>Sélectionner le match précédent.</td>
</tr>
<tr>
<td>BAS, j</td>
<td>Sélectionner le match suivant.</td>
</tr>
<tr>
<td>ENTREE</td>
<td>Charger le match sélectionné.</td>
</tr>
<tr>
<td>Del</td>
<td>Supprimer le match sélectionné.</td>
</tr>
<tr>
<td>Esc</td>
<td>Désélectionner/fermer le panneau.</td>
</tr>
</tbody>
</table>
<h3>Panneau Anki (répétition espacée)</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>ESPACE, Clic</td>
<td>Afficher la réponse (l'analyse enregistrée de la position).</td>
</tr>
<tr>
<td>1</td>
<td>Évaluer : À revoir (échec, revoir bientôt).</td>
</tr>
<tr>
<td>2</td>
<td>Évaluer : Difficile.</td>
</tr>
<tr>
<td>3</td>
<td>Évaluer : Bien.</td>
</tr>
<tr>
<td>4</td>
<td>Évaluer : Facile.</td>
</tr>
<tr>
<td>p</td>
<td>Afficher/cacher le compte de course (identique au raccourci général, disponible pendant la révision).</td>
</tr>
<tr>
<td>Esc</td>
<td>Arrêter la révision et revenir à la liste des paquets (reprise possible).</td>
</tr>
</tbody>
</table>
<h3>Panneau des tournois</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic, Double-clic</td>
<td>Sélectionner un tournoi (afficher son détail).</td>
</tr>
<tr>
<td>HAUT, k</td>
<td>Sélectionner le tournoi précédent.</td>
</tr>
<tr>
<td>BAS, j</td>
<td>Sélectionner le tournoi suivant.</td>
</tr>
<tr>
<td>Double-clic (sur un match du tournoi)</td>
<td>Naviguer dans le match.</td>
</tr>
<tr>
<td>Esc</td>
<td>Annuler l'édition en cours, sinon effacer la recherche d'ajout de match, sinon désélectionner le tournoi, sinon fermer le panneau (par paliers).</td>
</tr>
</tbody>
</table>
<h3>Panneau des collections</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Ajouter/retirer la position courante de la collection survolée.</td>
</tr>
<tr>
<td>Double-clic</td>
<td>Ouvrir la collection.</td>
</tr>
<tr>
<td>Del</td>
<td>Retirer la position courante (ou les positions cochées) de la collection ouverte.</td>
</tr>
<tr>
<td>Esc</td>
<td>Revenir à la liste des collections, sinon désélectionner la collection, sinon fermer le panneau (par paliers).</td>
</tr>
</tbody>
</table>
<h3>Panneau d'aide</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>GAUCHE, h</td>
<td>Onglet précédent.</td>
</tr>
<tr>
<td>DROITE, l</td>
<td>Onglet suivant.</td>
</tr>
<tr>
<td>HAUT, k</td>
<td>Défiler vers le haut.</td>
</tr>
<tr>
<td>BAS, j</td>
<td>Défiler vers le bas.</td>
</tr>
<tr>
<td>ESPACE</td>
<td>Page suivante.</td>
</tr>
<tr>
<td>PageUp</td>
<td>Haut du contenu.</td>
</tr>
<tr>
<td>PageDown</td>
<td>Bas du contenu.</td>
</tr>
<tr>
<td>?, CTRL-F, Esc</td>
<td>Fermer l'aide.</td>
</tr>
</tbody>
</table>
`,
    commands: `
<p>La ligne de commande, située dans la barre d'état, s'ouvre en appuyant sur la touche <em>ESPACE</em>. Lors de la saisie d'une commande, une liste de suggestions apparaît automatiquement : la touche <em>TAB</em> (ou <em>MAJ-TAB</em>) parcourt les propositions et complète la commande, tandis que <em>ÉCHAP</em> referme la liste (un second <em>ÉCHAP</em> ferme la ligne de commande). Les touches <em>HAUT</em> et <em>BAS</em> restent réservées à l'historique des commandes.</p>
<h3>Opérations globales</h3>
<table>
<thead>
<tr>
<th>Commande</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>new, ne, n</td>
<td>Crée une nouvelle base de données.</td>
</tr>
<tr>
<td>open, op, o</td>
<td>Ouvre une base de données existante.</td>
</tr>
<tr>
<td>import_db, idb</td>
<td>Importe et fusionne une autre base de données.</td>
</tr>
<tr>
<td>export_db, edb</td>
<td>Exporte la sélection courante vers une nouvelle base de données.</td>
</tr>
<tr>
<td>quit, q</td>
<td>Ferme blunderDB.</td>
</tr>
<tr>
<td>help, he, h</td>
<td>Ouvre l'aide de blunderDB.</td>
</tr>
<tr>
<td>tutorial, tour</td>
<td>Ouvre le catalogue des visites guidées de l'interface.</td>
</tr>
<tr>
<td>demo</td>
<td>Charge une base d'exemple (matchs, tournoi, collections, commentaires, paquet Anki, analyses) pour découvrir l'outil.</td>
</tr>
<tr>
<td>meta</td>
<td>Affiche les métadonnées de la base de données.</td>
</tr>
<tr>
<td>epc</td>
<td>Ouvre le panneau Eval (Effective Pip Count, probabilité de gain et verdict de videau en bearoff). <code>epc</code> est l'ancien nom de ce panneau, conservé.</td>
</tr>
<tr>
<td>met</td>
<td>Ouvre la table d'équité de match Kazaross-XG2.</td>
</tr>
<tr>
<td>cm</td>
<td>Ouvre la matrice du videau : le verdict de la position courante à tous les scores d'un match de 5, 7 ou 9 points.</td>
</tr>
<tr>
<td>tags</td>
<td>Ouvre le vocabulaire de tags : les tags utilisés dans cette base, avec le nombre de positions, cliquables pour lancer la recherche.</td>
</tr>
<tr>
<td>train</td>
<td>Lance une session de micro-entraînement. Prend un argument : <code>train pips</code> (compte de pions), <code>train epc</code>, <code>train tp</code> (point de prise au score). Cinq questions, chronométrées, corrigées sur-le-champ.</td>
</tr>
<tr>
<td>tp2</td>
<td>Ouvre la table des takepoints avec videau à 2.</td>
</tr>
<tr>
<td>tp2_live</td>
<td>Ouvre la table des takepoints avec videau à 2 pour les courses longues.</td>
</tr>
<tr>
<td>tp2_last</td>
<td>Ouvre la table des takepoints avec videau à 2 mort.</td>
</tr>
<tr>
<td>tp4</td>
<td>Ouvre la table des takepoints avec videau à 4.</td>
</tr>
<tr>
<td>tp4_live</td>
<td>Ouvre la table des takepoints avec videau à 4 pour les courses longues.</td>
</tr>
<tr>
<td>tp4_last</td>
<td>Ouvre la table des takepoints avec videau à 4 mort.</td>
</tr>
<tr>
<td>gv1</td>
<td>Ouvre la table des valeurs de gammon avec videau à 1.</td>
</tr>
<tr>
<td>gv2</td>
<td>Ouvre la table des valeurs de gammon avec videau à 2.</td>
</tr>
<tr>
<td>gv4</td>
<td>Ouvre la table des valeurs de gammon avec videau à 4.</td>
</tr>
</tbody>
</table>
<h3>Positions et navigation</h3>
<table>
<thead>
<tr>
<th>Commande</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>import, i</td>
<td>Importe une ou plusieurs positions/matchs par fichier (xg, xgp, sgf, mat, txt, bgf). Avec un argument — <code>import XGID=…</code> ou <code>import OGID=…</code> — lit l'identifiant plutôt que d'ouvrir un sélecteur de fichiers, pour le cas où il arrive d'un message, d'un forum ou d'un script.</td>
</tr>
<tr>
<td>delete, del, d</td>
<td>Supprime la position courante (confirmation demandée) ; la suppression passe par la corbeille et reste annulable trente jours.</td>
</tr>
<tr>
<td>trash</td>
<td>Ouvre la corbeille : ce qui a été supprimé, avec de quoi le restaurer.</td>
</tr>
<tr>
<td>[number]</td>
<td>Aller à la position d'indice indiqué.</td>
</tr>
<tr>
<td>list, l</td>
<td>Afficher l'analyse de la position courante.</td>
</tr>
<tr>
<td>comment, co</td>
<td>Afficher/écrire des commentaires.</td>
</tr>
<tr>
<td>history, hi</td>
<td>Ouvrir le panneau de recherche (l'historique de recherche se trouve dans son onglet <em>Historique</em>).</td>
</tr>
<tr>
<td>stats, st</td>
<td>Afficher/masquer le panneau de statistiques.</td>
</tr>
<tr>
<td>match, ma</td>
<td>Afficher/cacher le panneau des matchs.</td>
</tr>
<tr>
<td>collection, coll</td>
<td>Afficher/cacher le panneau des collections.</td>
</tr>
<tr>
<td>#tag1 tag2 ...</td>
<td>Etiqueter la position courante.</td>
</tr>
<tr>
<td>e</td>
<td>Charger toutes les positions de la base de données.</td>
</tr>
<tr>
<td>blunders, bl [n]</td>
<td>Charger les pires erreurs (équité/MWC) dans la vue d'analyse, selon le filtre courant des statistiques. Un nombre optionnel choisit combien en charger (<code>bl 50</code>) ; par défaut 10.</td>
</tr>
<tr>
<td>m</td>
<td>Naviguer dans le dernier match visité.</td>
</tr>
</tbody>
</table>
<h3>Édition et recherche</h3>
<table>
<thead>
<tr>
<th>Commande</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>write, wr, w</td>
<td>Enregistre la position courante.</td>
</tr>
<tr>
<td>write!, wr!, w!</td>
<td>Mettre à jour la position courante.</td>
</tr>
<tr>
<td>s</td>
<td>Chercher des positions avec des filtres.</td>
</tr>
<tr>
<td>ss</td>
<td>Chercher parmi les positions actuellement filtrées.</td>
</tr>
</tbody>
</table>
<h3>Filtres de recherche</h3>
<p>Cette table est la référence de la grammaire de recherche : la ligne de commande, la bibliothèque de filtres et le drapeau <code>--query</code> de <code>blunderdb search</code> lisent tous les mêmes jetons. La colonne <em>Équivalent CLI</em> donne, quand il existe, le drapeau de <code>search</code> qui fait la même chose (voir Interface en ligne de commande (CLI)) ; un tiret signale un filtre que seule la grammaire exprime.</p>
<p>Cinq jetons ne portent pas leur valeur : ils la lisent sur le plateau de recherche. <code>cube</code> et <code>score</code> reprennent le videau et le score qui y sont posés, <code>d</code> le type de décision, <code>D</code> et <code>D1</code> les dés, <code>x</code> la structure dessinée dans l'onglet <em>Sauf</em>. Un lancer ne s'écrit donc jamais dans le jeton : <code>D65</code> n'existe pas, seule la forme d'exclusion porte ses chiffres (<code>xD65</code>). En ligne de commande, où il n'y a pas de plateau, ces jetons se comparent à un plateau vide ; ce sont les drapeaux de la troisième colonne qu'il faut y employer.</p>
<p>Les erreurs et les équités se comptent en <strong>millièmes d'équité</strong> — les <em>millipoints</em> de la table ci-dessous : <code>E&gt;100</code> retient les coups qui ont coûté au moins un dixième de point, un point valant 1000 millièmes.</p>
<p>Deux recherches complètes :</p>
<ul>
<li><code>s p&gt;30 w40,60 xco</code> — plus de 30 pips de retard, entre 40 % et 60 % de chances de gain, aucun commentaire.</li>
<li><code>s ph:race E&gt;50 co:xg</code> — en course, un coup ayant coûté au moins 50 millièmes, et un commentaire venu d'eXtreme Gammon.</li>
</ul>
<table>
<thead>
<tr>
<th>Requête</th>
<th>Action</th>
<th>Équivalent CLI</th>
</tr>
</thead>
<tbody>
<tr>
<td>cube, cub, cu, c</td>
<td>La position vérifie la configuration du cube.</td>
<td><code>--cube</code></td>
</tr>
<tr>
<td>score, sco, sc, s</td>
<td>La position vérifie le score.</td>
<td><code>--score1</code> <code>--score2</code></td>
</tr>
<tr>
<td>d</td>
<td>La position vérifie le type de décision (pion ou cube).</td>
<td><code>--decision</code></td>
</tr>
<tr>
<td>D</td>
<td>La position vérifie le lancer de dés (les deux dés, peu importe l'ordre).</td>
<td><code>--dice 6,5</code></td>
</tr>
<tr>
<td>D1</td>
<td>La position vérifie le lancer de dés sur le premier dé uniquement (la valeur du premier dé apparaît sur l'un des deux dés de la position).</td>
<td><code>--dice 6</code></td>
</tr>
<tr>
<td>xD65</td>
<td>La position n'a <strong>pas</strong> été jouée avec le lancer 6-5 (peu importe l'ordre). La valeur est indiquée dans le jeton ; répétable pour exclure plusieurs lancers (<code>xD65 xD54</code>).</td>
<td>—</td>
</tr>
<tr>
<td>nc</td>
<td>La position est sans contact.</td>
<td>—</td>
</tr>
<tr>
<td>ph:race</td>
<td>La position est dans une phase de jeu donnée : <code>opening</code> (ouverture), <code>middlegame</code> (milieu de partie), <code>race</code> (course) ou <code>bearoff</code> (sortie des pions). Répétable (<code>ph:race ph:bearoff</code>). L'étiquette est calculée à partir du plateau, jamais modifiable ; la commande <code>blunderdb repair</code> la recalcule.</td>
<td><code>--phase</code></td>
</tr>
<tr>
<td>#prime</td>
<td>La position porte ce <strong>tag</strong> dans l'un de ses commentaires. Un tag est un <code>#mot</code> écrit dans la prose ; rien ne le déclare. La comparaison est délimitée, donc <code>#prime</code> ne trouve pas <code>#priming</code> — c'est toute la différence avec le filtre de texte, qui cherche une sous-chaîne. Répétable, et les tags se <strong>cumulent</strong> (<code>#prime #backgame</code> demande les deux) : une position porte plusieurs tags, donc en nommer deux veut dire « les deux ».</td>
<td>—</td>
</tr>
<tr>
<td>M</td>
<td>La position ou celle miroir vérifie les filtres.</td>
<td>—</td>
</tr>
<tr>
<td>i</td>
<td>La position a été importée seule, et non apportée par l'import d'un match.</td>
<td><code>--individual</code></td>
</tr>
<tr>
<td>fl</td>
<td>La position a été marquée (<em>flag</em>) dans le logiciel d'origine, lors de l'import d'un match eXtreme Gammon.</td>
<td><code>--flagged</code></td>
</tr>
<tr>
<td>x</td>
<td>La position ne contient aucun pion de la structure d'exclusion (onglet <em>Sauf</em> du panneau de recherche).</td>
<td>—</td>
</tr>
<tr>
<td>p&gt;x</td>
<td>Le joueur a au moins x pips de retard à la course.</td>
<td><code>--pip-min</code></td>
</tr>
<tr>
<td>p&lt;x</td>
<td>Le joueur a au plus x pips de retard à la course.</td>
<td><code>--pip-max</code></td>
</tr>
<tr>
<td>px,y</td>
<td>Le joueur a entre x et y pips de retard à la course.</td>
<td><code>--pip-min</code> <code>--pip-max</code></td>
</tr>
<tr>
<td>P&gt;x</td>
<td>Le joueur a une course au moins de x pips.</td>
<td>—</td>
</tr>
<tr>
<td>P&lt;x</td>
<td>Le joueur a une course au plus de x pips.</td>
<td>—</td>
</tr>
<tr>
<td>Px,y</td>
<td>Le joueur a une course entre x et y pips.</td>
<td>—</td>
</tr>
<tr>
<td>e&gt;x</td>
<td>L'équité (en millipoints) de la position est supérieure à x.</td>
<td>—</td>
</tr>
<tr>
<td>e&lt;x</td>
<td>L'équité (en millipoints) de la position est inférieure à x.</td>
<td>—</td>
</tr>
<tr>
<td>ex,y</td>
<td>L'équité (en millipoints) de la position est comprise entre x et y.</td>
<td>—</td>
</tr>
<tr>
<td>E&gt;x</td>
<td>L'erreur du coup joué par le joueur 1 (en millipoints) est supérieure à x.</td>
<td><code>--move-error-min</code></td>
</tr>
<tr>
<td>E&lt;x</td>
<td>L'erreur du coup joué par le joueur 1 (en millipoints) est inférieure à x.</td>
<td><code>--move-error-max</code></td>
</tr>
<tr>
<td>Ex,y</td>
<td>L'erreur du coup joué par le joueur 1 (en millipoints) est comprise entre x et y.</td>
<td><code>--move-error-min</code> <code>--move-error-max</code></td>
</tr>
<tr>
<td>w&gt;x</td>
<td>Le joueur a des chances de gain supérieures à x %.</td>
<td><code>--winrate-min</code></td>
</tr>
<tr>
<td>w&lt;x</td>
<td>Le joueur a des chances de gain inférieures à x %.</td>
<td><code>--winrate-max</code></td>
</tr>
<tr>
<td>wx,y</td>
<td>Le joueur a des chances de gain comprises à x % et y %.</td>
<td><code>--winrate-min</code> <code>--winrate-max</code></td>
</tr>
<tr>
<td>g&gt;x</td>
<td>Le joueur a des chances de gammon supérieures à x %.</td>
<td>—</td>
</tr>
<tr>
<td>g&lt;x</td>
<td>Le joueur a des chances de gammon inférieures à x %.</td>
<td>—</td>
</tr>
<tr>
<td>gx,y</td>
<td>Le joueur a des chances de gammon comprises à x % et y %.</td>
<td>—</td>
</tr>
<tr>
<td>b&gt;x</td>
<td>Le joueur a des chances de backgammon supérieures à x %.</td>
<td>—</td>
</tr>
<tr>
<td>b&lt;x</td>
<td>Le joueur a des chances de backgammon inférieures à x %.</td>
<td>—</td>
</tr>
<tr>
<td>bx,y</td>
<td>Le joueur a des chances de backgammon comprises à x % et y %.</td>
<td>—</td>
</tr>
<tr>
<td>W&gt;x</td>
<td>L'adversaire a des chances de gain supérieures à x %.</td>
<td>—</td>
</tr>
<tr>
<td>W&lt;x</td>
<td>L'adversaire a des chances de gain inférieures à x %.</td>
<td>—</td>
</tr>
<tr>
<td>Wx,y</td>
<td>L'adversaire a des chances de gain comprises à x % et y %.</td>
<td>—</td>
</tr>
<tr>
<td>G&gt;x</td>
<td>L'adversaire a des chances de gammon supérieures à x %.</td>
<td>—</td>
</tr>
<tr>
<td>G&lt;x</td>
<td>L'adversaire a des chances de gammon inférieures à x %.</td>
<td>—</td>
</tr>
<tr>
<td>Gx,y</td>
<td>L'adversaire a des chances de gammon comprises à x % et y %.</td>
<td>—</td>
</tr>
<tr>
<td>B&gt;x</td>
<td>L'adversaire a des chances de backgammon supérieures à x %.</td>
<td>—</td>
</tr>
<tr>
<td>B&lt;x</td>
<td>L'adversaire a des chances de backgammon inférieures à x %.</td>
<td>—</td>
</tr>
<tr>
<td>Bx,y</td>
<td>L'adversaire a des chances de backgammon comprises à x % et y %.</td>
<td>—</td>
</tr>
<tr>
<td>o&gt;x</td>
<td>Le joueur a au moins x pions sortis.</td>
<td><code>--off1-min</code></td>
</tr>
<tr>
<td>o&lt;x</td>
<td>Le joueur a au plus x pions sortis.</td>
<td>—</td>
</tr>
<tr>
<td>ox,y</td>
<td>Le joueur a entre x et y pions sortis.</td>
<td>—</td>
</tr>
<tr>
<td>O&gt;x</td>
<td>L'adversaire a au moins x pions sortis.</td>
<td><code>--off2-min</code></td>
</tr>
<tr>
<td>O&lt;x</td>
<td>L'adversaire a au plus x pions sortis.</td>
<td>—</td>
</tr>
<tr>
<td>Ox,y</td>
<td>L'adversaire a entre x et y pions sortis.</td>
<td>—</td>
</tr>
<tr>
<td>k&gt;x</td>
<td>Le joueur a au moins x pions arriérés.</td>
<td>—</td>
</tr>
<tr>
<td>k&lt;x</td>
<td>Le joueur a au plus x pions arriérés.</td>
<td>—</td>
</tr>
<tr>
<td>kx,y</td>
<td>Le joueur a entre x et y pions arriérés.</td>
<td>—</td>
</tr>
<tr>
<td>K&gt;x</td>
<td>L'adversaire a au moins x pions arriérés.</td>
<td>—</td>
</tr>
<tr>
<td>K&lt;x</td>
<td>L'adversaire a au plus x pions arriérés.</td>
<td>—</td>
</tr>
<tr>
<td>Kx,y</td>
<td>L'adversaire a entre x et y pions arriérés.</td>
<td>—</td>
</tr>
<tr>
<td>z&gt;x</td>
<td>Le joueur a au moins x pions dans la zone.</td>
<td>—</td>
</tr>
<tr>
<td>z&lt;x</td>
<td>Le joueur a au plus x pions dans la zone.</td>
<td>—</td>
</tr>
<tr>
<td>zx,y</td>
<td>Le joueur a entre x et y pions dans la zone.</td>
<td>—</td>
</tr>
<tr>
<td>Z&gt;x</td>
<td>L'adversaire a au moins x pions dans la zone.</td>
<td>—</td>
</tr>
<tr>
<td>Z&lt;x</td>
<td>L'adversaire a au plus x pions dans la zone.</td>
<td>—</td>
</tr>
<tr>
<td>Zx,y</td>
<td>L'adversaire a entre x et y pions dans la zone.</td>
<td>—</td>
</tr>
<tr>
<td>bo&gt;x</td>
<td>Le joueur a au moins x blots dans l'outfield.</td>
<td>—</td>
</tr>
<tr>
<td>bo&lt;x</td>
<td>Le joueur a au plus x blots dans l'outfield.</td>
<td>—</td>
</tr>
<tr>
<td>box,y</td>
<td>Le joueur a entre x et y blots dans l'outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BO&gt;x</td>
<td>L'adversaire a au moins x blots dans l'outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BO&lt;x</td>
<td>L'adversaire a au plus x blots dans l'outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BOx,y</td>
<td>L'adversaire a entre x et y blots dans l'outfield.</td>
<td>—</td>
</tr>
<tr>
<td>bj&gt;x</td>
<td>Le joueur a au moins x blots dans le jan.</td>
<td>—</td>
</tr>
<tr>
<td>bj&lt;x</td>
<td>Le joueur a au plus x blots dans le jan.</td>
<td>—</td>
</tr>
<tr>
<td>bjx,y</td>
<td>Le joueur a entre x et y blots dans le jan.</td>
<td>—</td>
</tr>
<tr>
<td>BJ&gt;x</td>
<td>L'adversaire a au moins x blots dans le jan.</td>
<td>—</td>
</tr>
<tr>
<td>BJ&lt;x</td>
<td>L'adversaire a au plus x blots dans le jan.</td>
<td>—</td>
</tr>
<tr>
<td>BJx,y</td>
<td>L'adversaire a entre x et y blots dans le jan.</td>
<td>—</td>
</tr>
<tr>
<td><code>t'mot1;mot2;...'</code></td>
<td>Les commentaires de la position contiennent au moins un des mots.</td>
<td>—</td>
</tr>
<tr>
<td>co</td>
<td>La position porte un commentaire, quel qu'en soit le contenu.</td>
<td><code>--has-comment</code></td>
</tr>
<tr>
<td>xco</td>
<td>La position ne porte aucun commentaire.</td>
<td><code>--no-comment</code></td>
</tr>
<tr>
<td>co:user</td>
<td>La position porte un commentaire d'une provenance donnée : <code>user</code> (écrit par vous), <code>xg</code>, <code>gnubg</code>, <code>bgf</code> (apporté par l'import d'un match) ou <code>unknown</code>. Répétable (<code>co:xg co:gnubg</code>).</td>
<td><code>--comment-origin</code></td>
</tr>
<tr>
<td><code>m'motif1,motif2,...'</code></td>
<td>Les meilleurs coups de pions contenant au moins un des motifs.</td>
<td>—</td>
</tr>
<tr>
<td><code>m'ND,DT,DP,...'</code></td>
<td>Les meilleures décisions de videau de No Double/Take, Double Take, Double Pass.</td>
<td>—</td>
</tr>
<tr>
<td>T&gt;x</td>
<td>Date d'ajout de la position après x (AAAA/MM/JJ).</td>
<td>—</td>
</tr>
<tr>
<td>T&lt;x</td>
<td>Date d'ajout de la position avant x (AAAA/MM/JJ).</td>
<td>—</td>
</tr>
<tr>
<td>Tx,y</td>
<td>Date d'ajout de la position entre x et y (AAAA/MM/JJ).</td>
<td>—</td>
</tr>
<tr>
<td>max</td>
<td>Rechercher dans le match d'identifiant x (ex: ma3).</td>
<td><code>--match-ids</code></td>
</tr>
<tr>
<td>max,y</td>
<td>Rechercher dans les matchs d'identifiants x à y (ex: ma2,5).</td>
<td><code>--match-ids</code></td>
</tr>
<tr>
<td>tnx</td>
<td>Rechercher dans le tournoi d'identifiant x (ex: tn1).</td>
<td><code>--tournament-ids</code></td>
</tr>
<tr>
<td>tnx,y</td>
<td>Rechercher dans les tournois d'identifiants x à y (ex: tn1,3).</td>
<td><code>--tournament-ids</code></td>
</tr>
<tr>
<td>idx</td>
<td>Rechercher la position d'identifiant x (ex: id12).</td>
<td><code>--position-ids</code></td>
</tr>
<tr>
<td>idx,y</td>
<td>Rechercher les positions d'identifiants x à y (ex: id5,10).</td>
<td><code>--position-ids</code></td>
</tr>
<tr>
<td><code>pl'nom'</code></td>
<td>Rechercher les positions issues d'un match impliquant le joueur indiqué, sur l'un ou l'autre camp (ex: <code>pl'Alice'</code>). La casse est ignorée.</td>
<td>—</td>
</tr>
</tbody>
</table>
<h3>Commandes diverses</h3>
<table>
<thead>
<tr>
<th>Commande</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>clear, cl</td>
<td>Efface l'historique des commandes.</td>
</tr>
</tbody>
</table>
`,
    about: `
<h3>Version</h3>
<p>Version de l'application : {appVersion}</p>
<p>Version de la base de données : {dbVersion}</p>
<p>
    <a href="https://kevung.github.io/blunderDB/fr/" target="_blank" rel="noopener noreferrer">Documentation en ligne</a> ·
    <a href="https://kevung.github.io/blunderDB/fr/historique.html" target="_blank" rel="noopener noreferrer">Historique des versions</a>
</p>

<h3>Auteur</h3>
<p><strong>Kévin Unger &lt;blunderdb@proton.me&gt;</strong></p>
<p>Vous pouvez aussi me retrouver sur Heroes sous le pseudo <strong>postmanpat</strong>.</p>
<p>
    J'ai développé blunderDB au départ pour mon usage personnel, afin de détecter des schémas dans mes erreurs. Mais il est très agréable de recevoir des retours, surtout lorsqu'on a passé beaucoup
    d'heures sur la conception, le code, le débogage... Alors n'hésitez pas à m'écrire pour partager vos retours.
</p>
<p>Voici plusieurs façons de me contacter :</p>
<ul>
    <li>Rejoignez le serveur Discord de blunderDB : <a href="https://discord.gg/DA5PpzM9En" target="_blank" rel="noopener noreferrer">discord.gg/DA5PpzM9En</a>,</li>
    <li>Discutez avec moi si nous nous croisons en tournoi,</li>
    <li>Envoyez-moi un e-mail,</li>
</ul>
<h3>Licence</h3>
<p>
    blunderDB est distribué sous licence MIT. Cela signifie que vous êtes libre d'utiliser, copier, modifier, fusionner, publier, distribuer, sous-licencier et/ou vendre des copies du logiciel, à
    condition que la notice de copyright originale et cette notice d'autorisation soient incluses dans toutes les copies ou parties substantielles du logiciel.
</p>
<h3>Remerciements</h3>
<p>Je dédie ce petit logiciel à ma compagne <strong>Anne-Claire</strong> et à notre chère fille <strong>Perrine</strong>. Je tiens à remercier tout particulièrement quelques amis :</p>
<ul>
    <li>
        <strong>Tristan Remille</strong>, pour m'avoir initié au backgammon avec joie et bienveillance ; pour m'avoir montré la Voie dans la compréhension de ce jeu merveilleux ; pour continuer à me
        soutenir malgré mes piètres tentatives de mieux jouer.
    </li>
    <li>
        <strong>Nicolas Harmand</strong>, un joyeux compagnon depuis plus d'une décennie dans de grandes aventures, et un fantastique partenaire de jeu depuis qu'il a attrapé le virus du backgammon.
    </li>
</ul>
<h3>Crédits</h3>
<p>blunderDB embarque du code, des données et des polices d'autres personnes. L'essentiel :</p>
<ul>
    <li>
        Le réseau de neurones <strong>strehl-prob5-512-512-256-128</strong> est l'œuvre d'<strong>Alexander Strehl</strong> (<em>alexstrehl/backgammon-ai-engine</em>, MIT). La recherche, le modèle de
        videau et la table d'équité de match qui l'entourent forment la configuration propre de <strong>gammonNet</strong> (<a
            href="https://github.com/kevung/gammonNet"
            target="_blank"
            rel="noopener noreferrer"
            >github.com/kevung/gammonNet</a
        >, MIT).
    </li>
    <li>La table d'équité de match Kazaross-XG2 (MET) est l'œuvre de <strong>Neil Kazaross</strong>.</li>
    <li>Les tables de take points et de valeurs de gammon sont tirées du livre <em>The Theory of Backgammon</em> de <strong>Dirk Schiemann</strong>.</li>
    <li>
        Les bases de bearoff unilatérale (6 points, 15 pions, pour l'EPC) et bilatérale (6 points, 6 pions, pour les verdicts de videau en course) ont été générées avec
        <strong>GNU Backgammon</strong> (GNUbg). GNUbg est un logiciel libre sous licence GPL ; ces tables sont des données qu'il a produites, créditées comme telles.
    </li>
    <li>Les fichiers de match sont lus par <em>xgparser</em>, <em>gnubgparser</em> et <em>bgfparser</em> (MIT).</li>
    <li>Côté Go : <em>modernc.org/sqlite</em> (BSD-3-Clause), <em>pgx</em>, <em>Wails</em> et <em>go-fsrs</em> (MIT).</li>
    <li>Côté interface : <em>Svelte</em>, <em>two.js</em>, <em>Chart.js</em> et <em>driver.js</em> (MIT).</li>
    <li>Les polices <em>Nunito</em> et <em>Noto Sans JP</em> (SIL Open Font License 1.1).</li>
</ul>
<p>
    L'inventaire complet, avec le texte des licences, est le fichier <strong>THIRD_PARTY.md</strong> livré avec blunderDB (<a
        href="https://github.com/kevung/blunderDB/blob/main/THIRD_PARTY.md"
        target="_blank"
        rel="noopener noreferrer"
        >github.com/kevung/blunderDB</a
    >).
</p>
`
};
