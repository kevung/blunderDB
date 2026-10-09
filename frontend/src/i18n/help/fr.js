// GENERATED FILE — do not edit by hand, and do not translate it here.
//
// Produced by `go run ./cmd/help-gen` (make help) from:
//   - doc/source/manuel.rst      → the "manual" tab
//   - doc/source/raccourcis.rst  → the "shortcuts" tab
//   - doc/source/cmd_mode.rst    → the "commands" tab
//   - doc/source/locale/<lang>/LC_MESSAGES/*.po for the eight translations
//   - frontend/src/i18n/help/prose/<lang>.html → the "about" tab
//
// Fix the documentation (and its .po catalogues), or the prose fragment, then
// run `make help`. TestHelpBundlesAreCurrent fails if this file is stale.
export default {
    manual: `
<h3>Introduction</h3>
<p>blunderDB est un logiciel pour constituer des bases de données de positions de backgammon. Sa force principale est de fournir un lieu unique pour agréger les positions qu'un joueur a rencontrées (en ligne, en tournoi) et de pouvoir les réétudier en les filtrant selon divers filtres arbitrairement combinables. blunderDB peut également être utilisé pour créer des catalogues de positions de référence.</p>
<p>Les positions sont stockées dans une base de données représentée par un fichier <em>.db</em>. L'application de bureau ouvre ce fichier directement, jamais une adresse réseau : le mode serveur est un autre mode du même binaire, et l'on passe de l'un à l'autre en exportant ou en migrant la base, pas en pointant l'application vers une URL.</p>
<h3>Interactions principales</h3>
<p>Les principales interactions possibles avec blunderDB sont:</p>
<ul>
<li>ajouter une nouvelle position,</li>
<li>modifier une position existante,</li>
<li>copier l'image du plateau dans le presse-papier (PNG) via <strong>CTRL-X</strong>, ou avec l'analyse complète via <strong>CTRL-X CTRL-X</strong>,</li>
<li>supprimer une position existante,</li>
<li>rechercher une ou plusieurs positions,</li>
<li>importer des matchs depuis différentes sources (XG, GNUbg, BGBlitz, Jellyfish, HedgeHog), y compris les commentaires depuis les fichiers XG,</li>
<li>naviguer dans les coups d'un match importé,</li>
<li>organiser les positions en collections,</li>
<li>organiser les matchs en tournois.</li>
</ul>
<p>L'utilisateur peut étiqueter librement les positions à l'aide de tags et les annoter via des commentaires.</p>
<h3>Description de l'interface</h3>
<p>L'interface de blunderDB est constituée de haut en bas par:</p>
<ul>
<li>[en haut] la barre d'outils, qui rassemble l'ensemble des principales opérations réalisables sur la base de données,</li>
<li>[au milieu] la zone d'affichage principale, qui permet d'afficher ou d'éditer des positions de backgammon,</li>
<li>[en bas] la barre d'état, qui présente différentes informations sur la base de données ou la position courante, et intègre la ligne de commande.</li>
</ul>
<p>Des panneaux peuvent être affichés pour:</p>
<ul>
<li>afficher les données d'analyse associées à la position courante issues d'eXtreme Gammon (XG), GNUbg, ou BGBlitz,</li>
<li>afficher, ajouter ou modifier des commentaires,</li>
<li>rechercher et filtrer des positions selon des critères combinables,</li>
<li>afficher et gérer les collections de positions (panneau collections),</li>
<li>afficher la liste des matchs importés et naviguer dans les coups d'un match (panneau matchs),</li>
<li>afficher et gérer les tournois (panneau tournois),</li>
<li>afficher les statistiques de performance (panneau Stats),</li>
<li>calculer l'EPC (Effective Pip Count) d'une position de bearoff (panneau Eval),</li>
<li>s'entraîner sur des exercices de calcul (panneau Entraînement),</li>
<li>étudier les positions par répétition espacée (panneau Anki),</li>
<li>transcrire un match à la main (panneau Transcription),</li>
<li>afficher les métadonnées de la base de données (panneau métadonnées).</li>
</ul>
<p>La hauteur du panneau se règle en tirant sa poignée ; elle reste la même d'un onglet à l'autre.</p>
<p>Des fenêtres modales peuvent s'afficher pour:</p>
<ul>
<li>afficher l'aide de blunderDB, dont le champ de recherche (<em>/</em> y place le curseur) sélectionne les occurrences du texte saisi, sans égard aux majuscules ni aux accents (<em>ENTREE</em> passe à la suivante, <em>MAJ-ENTREE</em> à la précédente),</li>
<li>afficher le catalogue des visites guidées (voir Visites guidées et base d'exemple),</li>
<li>paramétrer l'export de la base de données,</li>
<li>configurer blunderDB, notamment la langue de l'interface (voir Configuration).</li>
</ul>
<p>La zone d'affichage principale met à disposition à l'utilisateur:</p>
<ul>
<li>un plateau afin d'afficher ou d'éditer une position de backgammon,</li>
<li>le niveau et le propriétaire du cube,</li>
<li>le compte de course de chaque joueur,</li>
<li>le score de chaque joueur,</li>
<li>les dés à jouer. Si aucune valeur n'est affichée sur les dés, la position des dés indique quel joueur a le trait et que la position est une décision de cube. Lorsque la décision de cube est une réponse à un doublement (prise/passe), le videau proposé est affiché au centre du plateau, à la valeur offerte.</li>
</ul>
<p>Un clic droit sur le plateau ouvre un menu contextuel proposant : évaluer la position affichée dans le panneau Eval, évaluer son miroir, copier l'image du plateau avec son analyse dans le presse-papier (l'équivalent de <em>CTRL-X CTRL-X</em>, moins facile à découvrir), <strong>enregistrer l'image dans un fichier</strong> en SVG ou en PNG, ouvrir une nouvelle vue sur cette position, et — si la position vient déjà de la base — l'ajouter à un paquet Anki (répétition espacée) ou classer ses <strong>positions voisines</strong> (voir Panneau Recherche). Pendant un coup joué au plateau (quiz, Transcription), le menu commence par <em>Recommencer</em>, qui remet le coup à zéro sans toucher la position. En édition et en Eval, où le clic droit sur le plateau pose des pions, le menu s'ouvre d'un clic droit hors du plateau — hors des dés, du videau et des scores — et propose seulement <em>Effacer la position</em> (comme <em>RETOUR ARRIERE</em>) et <em>Position de départ</em>, qui pose les pions d'une partie neuve.</p>
<p>Le presse-papier est le geste courant ; enregistrer est l'autre besoin — l'illustration d'un article, d'un message de forum, d'une leçon. Le <strong>SVG</strong> y est proposé parce que le plateau en est un : c'est la forme qui survit à un agrandissement, celle qu'on met dans un document sans la flouter. Le PNG en dérive, comme la copie dans le presse-papier : un seul rendu, trois destinations, donc aucune ne peut diverger des autres. Ce menu n'apparaît pas dans le panneau Eval ni dans le panneau Recherche, où le bouton droit sert déjà à poser les pions de l'autre couleur. Voir Amener une position dans le panneau Eval pour amener une position dans le panneau Eval.</p>
<p>La barre d'état est structurée de gauche à droite par les informations suivantes:</p>
<ul>
<li>la ligne de commande, accessible en appuyant sur la touche <em>ESPACE</em>,</li>
<li>un message d'information lié à une opération réalisée par l'utilisateur,</li>
<li>l'index de la position courante, suivi du nombre de positions dans la bibliothèque courante (ou les informations de coup/partie lors de la navigation dans un match),</li>
<li>le <strong>compteur de bibliothèque</strong> — « 412 positions · 38 blunders · 5 matchs » — où chaque nombre <strong>ouvre ce qu'il compte</strong> : les positions, la recherche <code>E&gt;</code> au seuil de la bibliothèque (lancée aussitôt, comme si on l'avait saisie, et rangée dans l'historique des recherches), ou la liste des matchs. Quand la liste à l'écran n'est pas la bibliothèque entière (recherche, collection, match), le nombre de blunders s'écrit « 12 / 340 blunders » : les blunders de cette liste, puis ceux de la bibliothèque, au même seuil ; une liste de plus de 20 000 positions n'est pas comptée et garde le seul total. Un chiffre qu'on ne peut pas suivre est une décoration. Le seuil des blunders est celui de la bibliothèque, réglé dans l'onglet <em>Bibliothèque</em> de la configuration et partagé avec les statistiques : deux seuils feraient dire deux choses au même mot. Le compteur promet exactement ce que le lien ouvre, y compris pour une position jouée de plusieurs façons, qui vaut son coût le plus élevé. Sur une très grande bibliothèque (plus de 200 000 lignes), le compteur ne balaie pas les tables : un nombre précédé de « ≈ » est une estimation (une borne haute, les lignes supprimées laissant des trous). Si les positions sont estimées, le nombre de blunders, qui n'a pas d'estimation honnête, s'affiche « ? » — le lien lance la recherche, qui donne le compte exact. Les deux nombres de blunders sont des liens distincts : le premier lance la sous-recherche <code>ss E&gt;</code> dans la liste à l'écran, le second la recherche dans toute la bibliothèque ; quand seul le total s'affiche, il n'y a qu'un lien.</li>
</ul>
<div class="admonition note">
<p>Dans le cas de positions issues d'une recherche par l'utilisateur, le nombre de positions indiqué dans la barre d'état correspond au nombre de positions filtrées.</p>
</div>
<p>L'onglet <strong>Anki</strong> porte un <strong>badge</strong> quand des cartes sont à réviser, tous paquets confondus. Ce chiffre est la raison d'ouvrir l'onglet ; il n'a donc rien à faire derrière lui. Zéro n'affiche rien : un badge qui dit « 0 » est du bruit.</p>
<p>La commande <code>log</code> ouvre le <strong>journal d'activité</strong> : les deux cents dernières lignes du fichier de journal, un bouton pour les copier — de quoi joindre un rapport à un signalement — et un autre pour ouvrir le dossier qui les contient. Le journal n'est ni filtré ni reformaté : un journal qu'on embellit est un journal qu'on ne peut plus citer.</p>
<p>La commande <code>grid</code> ouvre la <strong>planche-contact</strong> : la liste parcourue — résultats d'une recherche, bibliothèque, collection — en grille de mini-plateaux, dessinés comme le plateau, par pages de vingt-quatre. Elle s'ouvre sur la page de la position courante, dont la vignette est encadrée ; un clic ou <em>ENTREE</em> sur une vignette ouvre sa position sur le plateau et referme la planche, et elle se parcourt entièrement au clavier (voir Planche-contact). Elle ne s'ouvre ni en mode édition ni dans un match, qui se parcourt par ses coups.</p>
<p>Dans l'<strong>historique de recherche</strong> du panneau Recherche, chaque jeton d'une commande enregistrée s'affiche en pastille nommée — <em>Sans contact</em>, <em>Erreur de coup</em> — plutôt qu'en jeton nu. La commande exacte reste en infobulle, car c'est elle qu'on relance ; et un jeton que blunderDB ne reconnaît pas s'affiche <strong>tel quel</strong> plutôt que traduit au plus proche.</p>
<h3>Onglets de vues</h3>
<p>Sous la barre d'outils, une barre d'onglets permet de travailler avec plusieurs <strong>vues</strong> en parallèle. Chaque vue est un espace de travail indépendant qui conserve sa propre liste de positions, l'index de la position courante, la position affichée, l'analyse et le coup sélectionné, le panneau actif, le commentaire en cours ainsi que le contexte de navigation dans un match. Il est ainsi possible, par exemple, de garder une recherche ouverte dans une vue tout en parcourant un match dans une autre.</p>
<ul>
<li><strong>Créer une vue</strong> : cliquer sur le bouton <em>+</em> de la barre d'onglets (nommé <em>+ Nouvelle vue</em> tant qu'il n'y a qu'une vue) ou appuyer sur <em>CTRL-T</em>. La nouvelle vue démarre comme une copie de la vue courante.</li>
<li><strong>Fermer une vue</strong> : cliquer sur la croix de l'onglet ou appuyer sur <em>CTRL-W</em>. La dernière vue ne peut pas être fermée.</li>
<li><strong>Changer de vue</strong> : cliquer sur un onglet, appuyer sur <em>CTRL-PageUp</em> / <em>CTRL-PageDown</em> (ou <em>MAJ-J</em> / <em>MAJ-K</em>) pour passer à la vue précédente / suivante, ou <em>CTRL-1</em> à <em>CTRL-9</em> pour atteindre directement la n-ième vue.</li>
<li><strong>Renommer une vue</strong> : double-cliquer sur l'onglet, saisir le nouveau nom et valider avec <em>ENTREE</em>.</li>
</ul>
<p>Les vues sont enregistrées avec l'état de session de la base de données et restaurées à sa réouverture.</p>
<h3>Configuration</h3>
<p>Le bouton de configuration (icône en forme de rouage) situé dans la barre d'outils, à gauche du bouton d'aide, ouvre la fenêtre de configuration de blunderDB. Elle est organisée en neuf onglets :</p>
<ul>
<li><strong>Interface</strong> — thème, langue, échelle d'affichage, position du panneau, pas de PageUp / PageDown (10, 50, 100, 500 ou 1 000 positions, ou 10 % de la liste), journaux, vérification des mises à jour, votre nom et positions voisines ;</li>
<li><strong>Couleurs</strong> — les couleurs du plateau ;</li>
<li><strong>Bibliothèque</strong> — ce qui appartient à la base ouverte : les seuils d'erreur et de blunder, le compactage et la réparation, décrits ci-dessous ;</li>
<li><strong>Corpus</strong> — les doublons à l'import, les alias de joueurs et d'événements et la recherche des doublons probables (voir l'import de matchs) ;</li>
<li><strong>Bearoff</strong> — les tables de sortie utilisées par le panneau Eval ;</li>
<li><strong>gammonNet</strong> — les réglages de l'évaluateur embarqué, décrits ci-dessous ;</li>
<li><strong>Dossier surveillé</strong> — l'import automatique des matchs qui arrivent dans un dossier, décrit ci-dessous ;</li>
<li><strong>Assistant et MCP</strong> — le serveur MCP local et l'assistant interne, décrits ci-dessous ;</li>
<li><strong>Identité d'émetteur</strong> — la clé qui signe vos filigranes, décrite à la section Diffuser une base : origine et mot de passe.</li>
</ul>
<p>L'onglet <em>Interface</em> propose d'abord un <strong>thème</strong> : <em>suivre le système</em>, <em>clair</em>, <em>sombre</em>, <em>contraste élevé</em> ou <em>imprimable</em>. Le thème règle les couleurs de l'interface et <strong>propose une palette de plateau</strong> — une interface sombre autour d'un plateau clair n'est pas un thème sombre, c'est la moitié d'un, puisque le plateau occupe l'essentiel de la fenêtre.</p>
<p>Vous gardez le dernier mot, et le mécanisme le garantit plutôt que de le promettre : l'onglet <em>Couleurs</em> continue de régler le plateau directement, et une couleur choisie après le thème est la vôtre. Au démarrage, seuls les jetons de l'interface sont appliqués, jamais la palette du plateau — celle que vous avez réglée est déjà chargée, et la réécrire à chaque lancement effacerait votre travail une session à la fois. Voir <code>ADR-0038 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0038-a-named-theme-carries-the-board-palette-and-the-user-still-has-the-last-word.md&gt;</code>__.</p>
<p><em>Suivre le système</em> est le réglage par défaut : il obéit à la préférence clair/sombre du bureau, y compris lorsqu'elle change en cours de session. Un outil n'impose pas son clair ou son sombre à un bureau qui a déjà tranché.</p>
<p>L'onglet <em>Interface</em> permet aussi de choisir la langue parmi l'anglais, le français, l'allemand, l'italien, l'espagnol, le finnois, le japonais, le grec et le russe. L'ensemble de l'interface (barre d'outils, panneaux, messages, aide) est traduit dans la langue sélectionnée. Le choix de la langue est enregistré et conservé d'une session à l'autre.</p>
<p>L'onglet <em>Bibliothèque</em> réunit ce qui appartient au fichier ouvert et non à la machine. Il est vide tant qu'aucune base n'est ouverte, et il le dit.</p>
<p>Il porte d'abord les deux <strong>seuils</strong> qui décident du vocabulaire de toute l'application : une décision est une <strong>erreur</strong> dès que son coût atteint le seuil d'erreur, et cette erreur est un <strong>blunder</strong> dès qu'il atteint le seuil de blunder. Tout blunder est une erreur, donc le premier seuil ne peut pas dépasser le second, et blunderDB refuse la paire inversée. Les valeurs sont saisies en équité — « 0,080 » — l'unité de toutes les tables ; la ligne de commande, elle, parle en millipoints, si bien que le seuil 0,080 s'écrit <code>E&gt;80</code> dans une recherche.</p>
<p>Ces seuils suivent le fichier, pas l'ordinateur : la même base compte les mêmes blunders partout où on l'ouvre, <code>blunderdb info</code> les affiche, et <code>blunderdb edit --error-threshold</code> / <code>--blunder-threshold</code> les règle. Ils ne voyagent pas dans un export : un seuil est une habitude de lecture, pas un fait des positions.</p>
<p>Trois <strong>préréglages</strong> sont proposés d'un clic, chacun sous le nom du programme qui a tracé cette ligne : blunderDB (0,050 / 0,100), XG (0,020 / 0,080) et gnubg (0,040 / 0,080). Par défaut, une bibliothèque lit 0,050 et 0,100.</p>
<p>Ce qu'ils changent, à l'écran : le nombre de blunders du compteur de la barre d'état et la recherche que son lien prépare, les colonnes « Erreurs » et « Blunders » des statistiques et du tableau des joueurs, les marques des coups dans la fiche d'un match (Panneau Matchs), et la liste des positions que blunderDB propose de revoir après un import.</p>
<p>Le même onglet propose aussi le bouton <strong>Compacter la base</strong>, qui récupère l'espace disque laissé par les suppressions (matchs, tournois, purges) : la base de données ne rétrécit jamais toute seule quand on supprime des données, il faut demander explicitement ce compactage. L'opération peut prendre du temps sur une grosse base. La copie compactée est écrite à côté du fichier, puis le remplace : il faut, temporairement, environ la taille de la base en espace disque libre sur son volume, et non en mémoire vive. Si un autre programme a la base ouverte (une seconde instance en lecture seule, par exemple), le fichier n'est pas remplacé : la base est compactée sur place, ce qui demande environ deux fois sa taille en espace libre, et un fichier temporaire va dans le dossier temporaire du système, ou dans celui que désigne la variable d'environnement <code>SQLITE_TMPDIR</code> : sur un petit <code>/tmp</code>, pointez-la vers le volume de la base. blunderDB refuse de démarrer, avec un message qui dit ce qui manque (espace disque, ou moins de 512 Mo de mémoire disponible), plutôt que de risquer un compactage interrompu. Une confirmation est demandée avant de lancer l'opération. Le résultat — l'espace gagné, en mégaoctets — s'affiche ensuite dans la barre d'état. La même opération est disponible en ligne de commande via <code>blunderdb vacuum</code> (voir Interface en ligne de commande (CLI)).</p>
<p>Après un compactage sur place, blunderDB vérifie que le fichier a bien rétréci. Sous Windows, le système refuse de raccourcir un fichier qu'un autre programme garde ouvert de certaines façons : le compactage se termine alors par un message qui le dit — fermez l'autre programme et relancez-le — au lieu d'annoncer un gain qui n'a pas eu lieu.</p>
<p>Le bouton <strong>Ouvrir le dossier des journaux</strong>, dans l'onglet <em>Interface</em>, ouvre le dossier contenant le journal de l'application — utile pour joindre des détails à un signalement de problème, en particulier quand blunderDB est lancé depuis un raccourci ou un double-clic, sans terminal attaché pour afficher quoi que ce soit.</p>
<p>La case <strong>Vérifier les mises à jour au démarrage</strong>, du même onglet, désactivée par défaut, interroge une fois la page des dernières versions du dépôt GitHub à chaque lancement et affiche, dans la barre d'état, un message si une version plus récente est disponible — jamais une fenêtre qui bloque l'utilisation. Cette vérification reste désactivée automatiquement sur une installation passée par un gestionnaire de paquets (Flatpak, Homebrew, un paquet de distribution…) : c'est ce canal-là qui gère alors les mises à jour, pas blunderDB lui-même.</p>
<p>Deux réglages de l'onglet <em>Interface</em> gouvernent le classement des <strong>positions voisines</strong> (jeton <code>like</code>, voir Panneau Recherche) : le nombre de voisines rendues, et la distance maximale au-delà de laquelle une position cesse d'en être une. Cette distance vaut zéro par défaut, c'est-à-dire aucun plafond : l'échelle dépend de la phase de la partie, et une valeur choisie ici se lirait comme une mesure. Le jeton <code>like&lt;12</code> impose la sienne pour une recherche, sans toucher au réglage.</p>
<p>L'onglet <em>Couleurs</em> permet de personnaliser les couleurs du plateau. Chaque élément dispose de son propre sélecteur de couleur : le fond, la bordure, les flèches claires et foncées, les pions du joueur 1 et du joueur 2, les dés, les points des dés et le videau. Le bouton <em>Réinitialiser</em> rétablit l'ensemble des couleurs par défaut. Comme la langue, les couleurs choisies sont conservées d'une session à l'autre.</p>
<p>L'onglet <em>Bearoff</em> gère les tables de sortie du panneau Eval (voir Panneau Eval). Elles ne sont <strong>ni embarquées dans l'exécutable, ni téléchargées</strong> : blunderDB les calcule sur la machine qui s'en sert, et le résultat est identique octet pour octet à ce que produit gnubg — l'empreinte SHA-256 est vérifiée avant que la table ne soit acceptée.</p>
<p>Les deux tables ordinaires (TS-06-06 pour le verdict de videau, OS-06 pour l'EPC) sont calculées au premier lancement, en arrière-plan et sans rien demander : environ six secondes sur un cœur, pendant lesquelles l'application s'utilise normalement. Le panneau Eval ne le signale que si l'on y pose une position qui a besoin d'une table pas encore prête.</p>
<p>L'onglet affiche le domaine actif et son origine, l'état de la table une face que lit l'EPC, le dossier où tout cela vit, et la liste des tables présentes avec leur taille et leur verdict. Chaque ligne se supprime individuellement, après confirmation.</p>
<p><strong>Vérifiée ou non vérifiée.</strong> Une table <em>vérifiée</em> a exactement les octets que gnubg produit pour son domaine : son empreinte SHA-256 figure dans blunderDB et a été retrouvée. Les empreintes enregistrées pour les tables une face (OS-06 à OS-10) sont celles que produit l'outil <code>makebearoff</code> de GNUbg 1.08. Une table <em>non vérifiée</em> est bien formée mais son domaine n'a pas d'empreinte enregistrée — rien ne lui est reproché, simplement personne ne l'a comparée à la référence. Une table <em>corrompue</em> se contredit elle-même et n'est jamais lue ; elle est recalculée.</p>
<p><strong>Calculer une table plus large.</strong> Le domaine se choisit dans une liste à deux familles, avec le nombre de cœurs à y consacrer (par défaut tous sauf un, pour que la machine reste utilisable) :</p>
<ul>
<li><strong>videau exact (deux faces)</strong>, de TS-06-06 à TS-06-15 : élargit le domaine où la probabilité de gain et le verdict de videau sont lus plutôt qu'estimés ;</li>
<li><strong>EPC hors du jan (une face)</strong>, de OS-06 à OS-10 : élargit la distance à laquelle un pion peut se trouver sans que le bloc EPC se taise. Ce balayage ne lit que des positions plus petites que celle qu'il calcule, donc il est séquentiel par construction et le nombre de cœurs ne lui sert à rien — le sélecteur le dit en se grisant.</li>
</ul>
<p>Avant de lancer quoi que ce soit, l'onglet annonce trois chiffres pour le domaine choisi : la taille sur le disque, la mémoire nécessaire pendant le calcul, et le temps que cela devrait prendre <em>sur cette machine</em>. Ce dernier commence par une estimation, puis devient une mesure : chaque calcul assez large relève sa propre vitesse et la conserve. Un domaine que la mémoire disponible ne permet pas est proposé grisé, avec la raison — « il faudrait 24 Go, il en reste 12 » est une réponse, une ligne absente n'en serait pas une.</p>
<p>À titre d'ordre de grandeur, sur une machine à seize fils : TS-06-09 pèse 191 Mo et demande une dizaine de secondes, TS-06-11 pèse 1,2 Go et quelques minutes, TS-06-13 dépasse ce que la plupart des machines peuvent tenir en mémoire. Du côté une face, sur un cœur : OS-07 pèse 4,9 Mo et prend 17 s, OS-08 15 Mo et 1 min 20, OS-10 117 Mo et une demi-heure.</p>
<p><strong>Pause et reprise.</strong> Pendant le calcul, la progression affiche le temps restant <em>mesuré</em>, et deux boutons distincts : <em>Pause</em> et <em>Annuler</em>. La pause écrit l'état du calcul à côté de la table ; le relancer reprend là où il s'est arrêté au lieu de tout recommencer. Annuler ne garde rien. Fermer la fenêtre de configuration n'interrompt rien — le calcul continue en arrière-plan.</p>
<p>Un calcul mis en pause se retrouve au lancement suivant, nommé et chiffré (« TS-06-09 interrompue à 43 % »), avec <em>Reprendre</em> et <em>Supprimer</em>. Rien ne redémarre tout seul : c'est l'utilisateur qui a demandé l'arrêt.</p>
<p>L'onglet permet enfin de pointer vers un fichier <code>.bd</code> two-sided externe, par exemple une base produite par gnubg lui-même : la table au domaine le plus large l'emporte.</p>
<p>L'onglet <em>Bibliothèque</em> porte enfin <strong>Réparer les analyses</strong> : les colonnes d'analyse que la recherche et les statistiques interrogent sont une projection des analyses stockées, lesquelles restent intactes. Un défaut de projection se répare donc sans rien réimporter. C'est explicite et jamais automatique — réécrire les colonnes d'analyse de quelqu'un au seul motif qu'il ouvre sa base n'est pas une chose qu'un outil doit faire dans son dos. Le même <code>blunderdb repair</code> est disponible en ligne de commande.</p>
<p>L'onglet <strong>gammonNet</strong> règle l'évaluateur embarqué (voir <code>ADR-0011 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0011-gammonnet-is-ported-to-go-and-the-representation-boundary-sits-at-the-evaluator-s-edge.md&gt;</code>__). Deux profondeurs de recherche y sont réglables, nommées et conservées séparément — abaisser l'une ne modifie jamais l'autre :</p>
<ul>
<li><strong>Profondeur d'affichage</strong> — le confort interactif pendant l'édition du plateau ; jamais écrite en base.</li>
<li><strong>Profondeur d'analyse</strong> — ce que le lot d'analyse après import écrit dans l'Analyse d'une position.</li>
</ul>
<p>Les deux valent par défaut <strong>2-ply</strong>, la configuration canonique. L'onglet propose aussi l'<strong>élagage</strong> (par défaut <code>k=12</code>) et le <strong>nombre de coups candidats affichés</strong> (par défaut 10), ainsi qu'une case <strong>analyser automatiquement après import</strong> qui, une fois activée, vérifie après chaque import s'il reste des positions <strong>sans aucune analyse</strong> (ni gammonNet, ni XG, ni GNUbg, ni BGBlitz — la règle est « une évaluation ne comble qu'un trou », jamais un remplacement) et, le cas échéant, lance en tâche de fond une analyse gammonNet à la profondeur d'analyse configurée. Un bouton <strong>Analyser maintenant</strong> relance manuellement le même rattrapage, utile pour une bibliothèque constituée avant l'existence de cette fonctionnalité.</p>
<p>Une prise ou un refus est évalué par gammonNet sur la décision du doubleur, avant le double. Un verdict gammonNet écrit sur une telle réponse par une version antérieure de blunderDB jugeait le redouble du receveur : il est supprimé à l'ouverture de la bibliothèque, ou à l'import d'une base qui le porte. La prise ou le refus redevient une position sans analyse, absente des statistiques jusqu'à ce que <strong>Analyser maintenant</strong> (ou l'analyse automatique après import) la réévalue.</p>
<p>Un second bouton, <strong>Ré-analyser les positions périmées</strong>, couvre le cas inverse : une position déjà analysée par gammonNet, mais dont l'analyse stockée a été écrite par une version de moteur plus ancienne que celle en cours d'exécution, ou à une profondeur différente de la profondeur d'analyse configurée ci-dessus, y est signalée comme périmée et réévaluée. Une position portant en plus une analyse XG, GNUbg ou BGBlitz n'est jamais touchée par ce bouton, quel que soit son contenu gammonNet — la protection d'<code>ADR-0013 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md&gt;</code>__ reste inconditionnelle. Le nombre affiché à côté de chaque bouton (positions sans analyse, positions périmées) est purement informatif ; le lot recalcule sa propre liste au moment de démarrer.</p>
<p>Les deux lots sont <strong>bornés, visibles et annulables, jamais un démon silencieux</strong> : leur progression (<code>positions analysées / total</code>) et un bouton d'annulation apparaissent dans la barre de statut pendant toute leur durée, et disparaissent une fois terminés au profit d'un message résumant le résultat — combien de positions ont été <strong>analysées</strong>, combien ont été <strong>refusées</strong> (une position que gammonNet décline d'évaluer, comme un score de match hors de portée de sa table de match, ce qui n'est jamais une panne) et combien ont <strong>échoué</strong> (retentées, inchangées, au prochain lancement). Fermer l'application pendant l'un ou l'autre ne perd rien : chaque position analysée est écrite au fil de l'eau, et un prochain lancement reprend exactement là où l'analyse s'était arrêtée, sans aucun journal à tenir.</p>
<p>Le même onglet porte le <strong>réglage des rollouts</strong> lancés depuis le panneau Analyse (voir Rollouts), conservé d'une session à l'autre : <strong>Rapide</strong> (216 parties, tronquées à 7 demi-coups), <strong>Standard</strong> (1296 parties, tronquées à 11 demi-coups) ou <strong>Libre</strong>, où tous les paramètres s'éditent — troncature, parties minimum et maximum (multiples de 36), limite de JSD, profondeur (ply), nombre de candidats, graine et nombre de processus.</p>
<p><strong>Table d'équité de match de la base.</strong> Au bas de l'onglet, la liste <strong>Table d'équité de match (MET) de la base</strong> choisit la table avec laquelle gammonNet valorise les scores de match : Kazaross-XG2, intégrée, par défaut, ou une table importée d'un fichier <code>.xml</code> de GNUbg par le bouton <strong>Importer une MET .xml…</strong>. Le choix appartient à la base, pas à l'application : ouvrir une autre base, c'est retrouver sa table. Seules les tables explicites sont lues (pas les tables paramétriques <code>zadeh</code> ou <code>mec</code>) ; au-delà de la longueur d'une table importée, la table intégrée prend le relais. Une table s'identifie par ses valeurs, non par son nom : importer le <code>Kazaross-XG2.xml</code> de GNUbg n'ajoute rien, c'est la table intégrée. Chaque analyse calculée par gammonNet enregistre la table avec laquelle elle l'a été ; les analyses importées (XG, GNUbg, BGBlitz) sont réputées calculées avec Kazaross-XG2. Une analyse à un score de match calculée avec une autre table que la table courante est marquée <strong>MET différente</strong> dans le panneau d'analyse et sortie des statistiques (moyennes d'erreur, PR, classements, face-à-face) ; une analyse en money game ne l'est jamais. Changer de table ne réécrit aucune analyse : seul change ce que les comparaisons retiennent.</p>
<p>L'évaluation en direct du panneau et la grille de videau sont valorisées avec la table de la base, comme les analyses qu'elle enregistre. Une table voyage avec les analyses qui la citent : l'export d'une base l'emporte, et l'import d'une base l'ajoute à la base qui reçoit, sans doublon (une table déjà présente est reconnue à ses valeurs) et sans la rendre courante. Une analyse gammonNet remplacée par celle d'un autre moteur (XG, GNUbg) redevient réputée calculée avec Kazaross-XG2.</p>
<p><strong>Un match importé sans analyse obtient ainsi un PR.</strong> C'est le cas d'un match joué en ligne, ou d'un fichier Jellyfish <code>.mat</code>, que personne n'a fait passer par XG : blunderDB en connaissait les positions et les coups joués, mais aucune analyse ne disait ce qu'ils valaient. Une fois le lot passé, le coup effectivement joué est comparé au classement de gammonNet et l'écart alimente le PR, le taux d'erreur, les pires décisions et tous les autres indicateurs, exactement comme un match analysé par XG. La comparaison ne s'invente rien : le coup joué vient de la table des coups du match, écrite à l'import, que le fichier ait porté une analyse ou non.</p>
<p>Une base analysée avec une version antérieure à celle-ci n'a pas besoin d'être réévaluée : <code>blunderdb repair</code> recalcule les colonnes à partir des analyses et des coups déjà en base et rend leur PR à ces matchs (voir repair).</p>
<p>Une réserve honnête : une position est identifiée par sa structure, donc une position rencontrée deux fois — bien jouée une fois, mal l'autre — ne porte qu'un seul écart, celui de sa première occurrence enregistrée. Ce n'est pas propre à ce calcul : une bibliothèque XG a exactement la même forme.</p>
<h4>Dossier surveillé</h4>
<p>L'onglet <strong>Dossier surveillé</strong> demande à blunderDB de regarder un dossier pendant qu'il tourne et d'importer chaque fichier de match qui y <strong>apparaît</strong>. Jouer une session dans eXtreme Gammon, revenir à blunderDB, et trouver les matchs déjà là.</p>
<p>Rien n'est deviné. Tant qu'aucun dossier n'est désigné, il n'y a pas de surveillance : blunderDB ne se met pas à lire un répertoire parce qu'il a supposé où vivent vos matchs. Le bouton <strong>Proposer</strong> cherche les emplacements habituels sur cette machine et n'en propose un que s'il existe réellement ; sinon il le dit, et c'est à vous de désigner le dossier.</p>
<p>Trois points méritent d'être connus avant d'activer la case :</p>
<ul>
<li><strong>Seuls les fichiers qui apparaissent sont importés.</strong> Ce que le dossier contient déjà au moment où la surveillance démarre est enregistré comme connu et laissé tranquille : pointer une surveillance sur quatre ans de matchs ne doit pas les importer tous. Pour importer ce qui est là, utilisez l'import de dossier, qui existe pour cela — et les deux se composent très bien, l'import d'abord, la surveillance ensuite.</li>
<li><strong>Un fichier n'est importé qu'une fois sa taille stable.</strong> Un match qu'un autre programme est en train d'écrire grossit d'un coup d'œil à l'autre ; l'importer à moitié écrit donnerait une erreur d'analyse syntaxique sur laquelle personne ne peut agir. blunderDB attend donc de voir deux fois le même fichier inchangé.</li>
<li><strong>L'import est silencieux.</strong> Vous étiez en train d'étudier une position quand vos matchs sont arrivés : vous reprendre l'écran serait le pire moment. Le mode, la recherche active, l'onglet et la position affichée ne bougent pas ; la liste des positions n'est pas rechargée et montre les nouveaux matchs au prochain rechargement. L'import se fait sans fenêtre, et la barre d'état affiche un bandeau donnant le compte des matchs importés, ignorés (doublons) et en échec, avec un bouton qui ouvre le compte rendu complet si vous le souhaitez. Tout le reste est identique à un import manuel : mêmes doublons détectés, même lot d'import, même analyse automatique si elle est activée.</li>
</ul>
<p>L'intervalle par défaut est de dix secondes ; le plancher est de deux. Le dossier n'est pas parcouru récursivement : un dossier surveillé est l'endroit où un outil dépose ses matchs, pas une arborescence à explorer. Un partage réseau démonté n'arrête pas la surveillance et ne fait pas non plus passer son contenu pour nouveau à son retour.</p>
<p>La même surveillance existe en ligne de commande, avec <code>blunderdb import --type batch --dir &lt;dossier&gt; --watch</code> (voir Interface en ligne de commande (CLI)) : c'est la forme qu'un serveur, une tâche planifiée ou un script peuvent utiliser.</p>
<h4>Assistant et MCP</h4>
<p>L'onglet <strong>Assistant et MCP</strong> règle deux choses, toutes deux désactivées par défaut.</p>
<p>Le <strong>serveur MCP local</strong> offre les outils de blunderDB (recherche, lecture d'une position et de son analyse, statistiques d'un joueur, quiz…) à un assistant qui parle le Model Context Protocol, comme Claude Desktop ou Claude Code, tant que la fenêtre est ouverte. Il n'écoute que sur cette machine, à l'adresse <code>http://127.0.0.1:&lt;port&gt;/mcp</code> (port 8765 par défaut), et refuse une requête venue d'une page web. Il ajoute deux outils d'affichage : ouvrir une vue sur une recherche et montrer une position. Les outils ne font que lire, sauf si <strong>Autoriser l'écriture</strong> est cochée : ils peuvent alors enregistrer une position, la commenter, créer et remplir une collection, noter une carte Anki et enregistrer un rollout à côté de l'analyse d'une position ; rien n'efface. Sans fenêtre ouverte, <code>blunderdb mcp</code> sert les mêmes outils (voir Interface en ligne de commande (CLI)).</p>
<p>L'<strong>assistant interne</strong> est un client de ces mêmes outils. Aucun modèle n'est embarqué : il s'adresse à un fournisseur compatible OpenAI — Ollama sur cette machine par défaut, ou Groq, OpenRouter, Gemini, Anthropic, ou une autre adresse. Est distante toute adresse hors de cette machine, quel que soit le fournisseur choisi : vos phrases et les résultats des outils la quittent alors, l'onglet le dit et attend votre accord pour cette adresse précise ; une autre adresse le redemande. La clé d'API va dans le trousseau du système, jamais dans la base ni dans le fichier de réglages. Le modèle proposé par défaut pour Ollama, <code>qwen2.5:7b</code>, est un point de départ, pas une recommandation : aucune recommandation de modèle n'est faite sans un score du banc de mesure livré avec le code source.</p>
<p>Une fois activé, l'assistant est un sous-onglet <strong>Assistant</strong> du panneau Recherche. Une phrase — « mes erreurs de plus de 50 millipoints dans la course » — ouvre une nouvelle vue nommée d'après elle, y lance la recherche et y bascule ; ses jetons restent dans l'historique des recherches. Ce que les outils renvoient vient de la base ; ce que le modèle écrit est marqué <strong>Texte libre du modèle</strong>. L'assistant ne propose de modification que si la case <strong>Laisser l'assistant proposer des modifications</strong> est cochée — un réglage distinct de l'écriture du serveur MCP local — et chaque modification qu'il prépare s'affiche et n'est faite qu'après <strong>Confirmer</strong>.</p>
<p>La fenêtre de configuration regroupe également des réglages d'affichage de l'interface. Un curseur d'<strong>échelle de l'interface</strong> permet d'agrandir ou de réduire l'ensemble des éléments, ce qui est utile sur les écrans à haute densité ou pour améliorer la lisibilité. Un menu <strong>position des panneaux</strong> détermine l'emplacement des panneaux (recherche, matchs, analyse) par rapport au plateau : <em>en bas</em>, <em>sur le côté</em> ou <em>automatique</em> (le côté est alors choisi sur les écrans larges afin de mieux exploiter l'espace disponible). Comme les autres réglages, ces choix sont conservés d'une session à l'autre.</p>
<h3>Visites guidées et base d'exemple</h3>
<p>Pour faciliter la prise en main, blunderDB propose des <strong>visites guidées</strong> de l'interface. Le catalogue des visites s'ouvre depuis la barre d'outils ou avec la commande <code>tutorial</code> (alias <code>tour</code>). Sept visites sont disponibles : un tour général de l'interface, et des visites dédiées à la recherche de positions, à la revue des matchs, à la revue des tournois, au panneau Eval, à la révision Anki et aux statistiques. Chaque visite met en évidence les éléments concernés de l'interface, étape par étape, ouvre au passage le panneau dont elle parle, et peut être rejouée à tout moment. Au premier démarrage, le tour général est proposé automatiquement.</p>
<p>La commande <code>demo</code> charge une <strong>base d'exemple</strong> permettant de découvrir les fonctionnalités de l'outil sans importer ses propres parties : trois matchs (dont deux regroupés dans un tournoi) analysés par eXtreme Gammon, BGBlitz et gammonNet, trois collections thématiques, des commentaires étiquetés (<code>#blunder</code>, <code>#cube</code>) et un paquet Anki avec son journal de révisions. Les joueurs, le tournoi et le lieu sont fictifs. Les visites guidées s'appuient sur cette base lorsqu'aucune base n'est ouverte.</p>
<h3>Navigation dans les positions</h3>
<p>Par défaut, blunderDB permet de:</p>
<ul>
<li>faire défiler les différentes positions de la bibliothèque courante — qui n'est jamais chargée d'un bloc : blunderDB n'en tient que la liste des identifiants et charge les positions par fenêtres de cinquante autour de celle qui est affichée, si bien qu'une base de plusieurs dizaines de milliers de positions s'ouvre aussi vite qu'une petite,</li>
<li>afficher les informations d'analyse associées à une position,</li>
<li>afficher, ajouter et modifier les commentaires d'une position.</li>
</ul>
<p>Le bouton <strong>Aller à la position</strong> de la barre d'outils ouvre une fenêtre où saisir directement l'indice d'une position pour y sauter, sans avoir à défiler. C'est l'équivalent graphique de la commande <code>[number]</code> en ligne de commande (voir Positions et navigation).</p>
<div class="admonition tip">
<p>Se référer à Raccourcis clavier pour les raccourcis disponibles.</p>
</div>
<h3>Édition de positions</h3>
<p>L'appui sur la touche <em>TAB</em> ouvre le panneau de recherche et permet d'éditer une position sur le plateau pour l'ajouter à la base de données ou pour définir une structure de position à rechercher. La distribution des pions, du videau, du score, et du trait peuvent être modifiés à l'aide de la souris (voir Editer une position).</p>
<div class="admonition tip">
<p>Se référer à Raccourcis clavier pour les raccourcis disponibles.</p>
</div>
<h3>La ligne de commande</h3>
<p>La ligne de commande, intégrée dans la barre d'état, permet de réaliser l'ensemble des fonctionalités de blunderDB disponibles à l'interface graphique: opérations générales sur la base de données, navigation de position, affichage de l'analyse et/ou des commentaires, recherche de positions selon des filtres... Après une première prise en main de l'interface, il est recommandé de progressivement utiliser la ligne de commande qui permet une utilisation puissante et fluide de blunderDB, notamment pour les fonctionnalités de recherche de positions.</p>
<p>Pour ouvrir la ligne de commande, appuyer sur la touche <em>ESPACE</em>. Pour envoyer une requête et fermer la ligne de commande, appuyer sur la touche <em>ENTREE</em>.</p>
<p>blunderDB exécute les requêtes envoyées par l'utilisateur sous réserve qu'elles soient valides et modifie immédiatement l'état de la base de données le cas échéant. Il n'y a pas d'actions de sauvegarde explicite de la part de l'utilisateur.</p>
<div class="admonition tip">
<p>Se référer à la liste des commandes pour la liste de commandes disponible en ligne de commande.</p>
</div>
<h3>La palette de commandes</h3>
<p>La palette de commandes (<em>CTRL-MAJ-P</em>) retrouve par un nom approché ce que l'on ne sait plus où chercher : une commande de la ligne de commande, un onglet, un filtre de la bibliothèque ou un match. Les lettres tapées doivent apparaître dans l'ordre, pas forcément côte à côte, et sans égard aux majuscules ni aux accents : « mtrc » trouve la matrice du videau, « lyon » les matchs d'un tournoi de Lyon.</p>
<p>Les flèches choisissent, <em>ENTREE</em> lance, <em>ECHAP</em> referme. Une commande se lance comme si elle avait été tapée ; <code>s</code> et <code>ss</code> ouvrent la ligne de commande pour y écrire les filtres ; un filtre se lance comme d'un double-clic dans la bibliothèque ; un match s'ouvre comme d'un double-clic dans le panneau Matchs.</p>
<p>Quand une Direction est ouverte, la palette y ajoute le tournoi : joueurs, tables, matchs en cours et épreuves (voir la recherche rapide).</p>
<h3>Panneau Analyse</h3>
<p>Le panneau <strong>Analyse</strong> (<em>CTRL-L</em>) affiche les données d'analyse de la position courante importées depuis eXtreme Gammon (XG), GNUbg, BGBlitz ou gammonNet. Il présente les meilleures alternatives (coups de pions ou décisions de videau) avec leurs valeurs d'équité et les erreurs correspondantes. La touche <em>d</em> bascule entre l'analyse des coups de pions et l'analyse du cube. Lors de la navigation dans un match, le coup effectivement joué est mis en évidence dans la liste des alternatives. Appuyer sur <em>CTRL-L</em> ou exécuter la commande <code>list</code> pour afficher ou masquer le panneau.</p>
<p>Sous les tableaux, une <strong>phrase</strong> dit parfois ce que la décision jouée a coûté et pourquoi : « Vous perdez 120 mp : le coup joué laisse trois blots là où 13/7 8/7 n'en laisse qu'un. » Elle est produite par six règles mesurables — l'exposition, un point du jan fait ou manqué, les chances de gammon abandonnées, une sécurité qui coûte plus qu'elle ne rapporte, et les deux sens d'une erreur de videau (doubler trop tard ou trop tôt, prendre trop large ou passer trop serré).</p>
<p>La règle qui compte est celle du <strong>silence</strong> : la phrase n'apparaît que si une règle s'applique de façon confiante, et sur une erreur qui dépasse le seuil à partir duquel les moteurs s'accordent à dire qu'elle en est une. Le reste du temps, il n'y a pas de phrase — ni cadre vide, ni « nous ne savons pas ». Une explication fausse coûte plus cher que pas d'explication : elle apprend quelque chose d'inexact.</p>
<p>La même phrase accompagne l'erreur là où vous venez de la commettre : au dos d'une <strong>carte Anki</strong>, sous l'analyse dévoilée, et dans le <strong>verdict du quiz</strong> de l'exercice Décision, sous le coût en mp. Les mêmes règles de silence y valent : un coup juste, ou une erreur qu'aucune règle n'explique, n'ajoute rien.</p>
<p>Lorsqu'une position a été jugée par <strong>plusieurs moteurs</strong>, une bande en tête du panneau les met côte à côte : une ligne par moteur, avec sa profondeur et sa réponse — le verdict de videau, ou son propre meilleur coup. Elle dit d'abord s'ils sont d'accord, et c'est le désaccord qui la justifie : « XG dit double, prend ; gammonNet dit pas de double » se lit d'un coup d'œil, là où il fallait comparer deux tableaux en diagonale.</p>
<p>Le meilleur coup d'un moteur est le meilleur <strong>de ce moteur</strong> : la liste des coups candidats est triée par équité, tous moteurs confondus, et son premier élément n'est donc le meilleur coup d'aucun d'eux en particulier.</p>
<p>La bande n'apparaît que s'il y a effectivement plusieurs moteurs, et elle n'existe que dans ce panneau : le panneau Eval présente <strong>une</strong> décision, celle du moteur embarqué (<code>ADR-0017 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0017-the-panel-shows-position-facts-plus-the-one-decision-the-board-asks.md&gt;</code>__), et une comparaison n'y aurait pas sa place.</p>
<p>Les coups sont écrits comme on les lit sur le plateau, ici comme dans le panneau Eval : le pion le moins avancé bouge d'abord, et <strong>un pion qui enchaîne plusieurs dés ne s'écrit qu'une fois</strong> — un 64 joué avec le même pion se lit <code>24/14</code>, et <code>24/14*</code> s'il frappe en arrivant. Le détail de l'enchaînement ne réapparaît que lorsqu'il dit quelque chose de plus : une frappe <em>en cours de route</em> conserve son point de passage, <code>24/18* 18/14</code>, sans quoi la frappe en 18 disparaîtrait de la notation.</p>
<p>L'équité d'une analyse importée suit la même règle que le panneau Eval : la colonne annonce son référentiel, « Équité (money) » ou « Équité (match) » selon le score de la position analysée, jamais un simple « Équité » muet sur l'échelle. Les règles <strong>Jacoby</strong> et <strong>Beaver</strong> actives sur une position en money game s'affichent, elles aussi, en badges sous le tableau de décision de videau.</p>
<h4>Rollouts</h4>
<p>Le panneau <strong>Analyse</strong> sait <strong>rouler</strong> une position : jouer des centaines de parties à partir de chaque coup choisi, ou de chaque action de videau, pour départager deux choix que l'évaluation directe sépare à peine. Dans la table des coups candidats, <em>Ctrl+clic</em> ajoute ou retire un coup de la sélection et <em>Maj+clic</em> l'étend jusqu'au coup cliqué ; le clic simple sélectionne un seul coup, comme d'ordinaire. Un <strong>clic droit</strong> sur la sélection — ou sur un autre coup, qui devient la sélection — ouvre un menu dont l'entrée <strong>Rollout (Standard)</strong> lance aussitôt le rollout des coups sélectionnés avec le réglage choisi. Sur une décision de videau, le clic droit roule la décision entière. La touche <em>r</em> du panneau fait de même (sans sélection, elle roule les meilleurs candidats de la position), comme la commande <code>rollout</code> (alias <code>ro</code>). Pendant le rollout, une barre fine sous la table suit les parties jouées ; <strong>Annuler</strong>, l'entrée <strong>Annuler le rollout</strong> du menu, <em>Échap</em> ou <em>r</em> de nouveau l'arrêtent sans rien écrire. Le réglage — <strong>Rapide</strong>, <strong>Standard</strong> ou <strong>Libre</strong> — se choisit dans l'onglet <strong>gammonNet</strong> de la configuration (voir configuration).</p>
<p>Le même menu propose <strong>Copier la position et l'analyse</strong>, l'image que copie <em>CTRL-X CTRL-X</em> : le plateau et le haut de la liste des coups, ou l'analyse de videau entière. Dès qu'un coup est sélectionné, une seconde entrée, <strong>Copier la position et les coups sélectionnés</strong>, s'y ajoute : l'image ne garde que ces coups, dans l'ordre du classement, chacun précédé de son rang parmi tous les candidats et avec son écart au meilleur coup de la position, non au meilleur de la sélection. Sur une décision de videau, seule la première figure. Les cellules des tables d'analyse ne se sélectionnent pas comme du texte : les clics de sélection n'y laissent pas de surlignage.</p>
<p>Le résultat est <strong>stocké à côté de l'analyse, jamais à sa place</strong> : une analyse importée n'est pas modifiée. Chaque coup roulé porte son résultat dans sa propre ligne de la table, dans une colonne <strong>Rollout</strong> qui n'apparaît que lorsque la position a un rollout : l'équité et la demi-largeur de son intervalle de confiance à 95 %, sur l'échelle de la colonne d'équité. Survoler la cellule donne le détail : l'écart-type, la <strong>JSD</strong> (l'écart au meilleur coup en écarts-types de la différence : à partir de la limite, le coup est tranché et cesse d'être joué), le nombre de parties et la <strong>Configuration</strong> — le moteur et la signature complète des paramètres : deux rollouts de même signature sont les mêmes nombres. Le rollout s'arrête dès que les coups sont départagés. Le rollout d'une décision de videau, qui n'a pas de ligne de coup, s'affiche en tableau sous la décision. Un rollout joue le videau dans ses parties : le classement est fiable, l'équité absolue un peu moins, ce que le bloc rappelle. Il ne joue pas le beaver : sa ligne <em>double, prend</em> est celle d'une simple prise, même sur une position en money sous la règle Beaver, là où l'analyse directe compte le beaver. Une position qui n'est pas dans la base se roule, mais ne se stocke pas.</p>
<p>La commande <code>ro search</code> roule, l'une après l'autre, les positions de la liste affichée — résultats de recherche, match ou collection — qui n'ont pas encore ce rollout ; une confirmation donne le total avant de commencer. Chaque position est écrite dès qu'elle est finie : annuler garde ce qui est fait, et relancer reprend où l'on s'est arrêté. L'avancement survit à la fermeture du panneau.</p>
<p>Une base ouverte <strong>en lecture seule</strong>, parce qu'une autre instance de blunderDB la tient, refuse d'emblée un rollout qui serait stocké : un message le dit avant que la première partie ne soit jouée.</p>
<h3>Panneau Commentaires</h3>
<p>Le panneau <strong>Commentaires</strong> (<em>CTRL-P</em>) affiche, ajoute et modifie les commentaires associés à la position courante. Une position peut en porter plusieurs, écrites par des personnes différentes : ils sont tous affichés en fil, du plus récent au plus ancien, chacun avec le nom de son auteur, et les vôtres passent en tête. Les commentaires importés depuis les fichiers XG sont automatiquement associés aux positions correspondantes. Appuyer sur <em>CTRL-P</em> ou exécuter la commande <code>comment</code> pour afficher ou masquer le panneau.</p>
<p>Le nom qui signe vos commentaires se règle dans les préférences, onglet <em>Interface</em>, champ <strong>Votre nom</strong> ; vide, vos commentaires ne sont pas signés. Les commentaires d'un import XG sont signés <code>XG</code>, ceux d'un fichier de match du nom de son transcripteur. Réécrire un commentaire le signe de votre nom. La recherche <code>au"Alice"</code> retient les positions qu'Alice a commentées ; hors de l'interface, <code>blunderdb comment add --author</code> écrit un commentaire signé et <code>blunderdb comment list</code> les relit (voir Interface en ligne de commande (CLI)).</p>
<p>Chaque commentaire venu d'un fichier porte une <strong>étiquette de provenance</strong> (<code>XG</code>, <code>GNU BG</code>, <code>BGF</code>, ou <em>importé</em> lorsque la provenance n'a pas été enregistrée). Les commentaires que vous avez écrits n'en portent pas : c'est le cas courant, et le signaler à chaque ligne serait du bruit. Modifier un commentaire importé vous l'attribue : après la modification, la phrase est la vôtre.</p>
<p>Cette distinction a une conséquence visible ailleurs : supprimer un match n'efface plus une position sur laquelle <strong>vous</strong> aviez écrit. Une note reprise du fichier source, elle, disparaît avec le match qui l'a apportée.</p>
<h4>Les tags</h4>
<p>Un <strong>tag</strong> est un <code>#mot</code> écrit dans un commentaire. Rien ne le déclare, aucune table ne le porte, et c'est voulu : le vocabulaire est votre prose, et exiger une déclaration avant de pouvoir taguer transformerait une habitude en paperasse.</p>
<p>Ce qui manquait, c'était l'autre moitié : <strong>voir</strong> le vocabulaire qu'on s'est construit, et cliquer un tag plutôt que se rappeler comment on l'écrivait. La commande <code>tags</code>, ou le bouton <code>#</code> de la zone de saisie, ouvre la fenêtre du vocabulaire : les tags de cette base, chacun avec le <strong>nombre de positions</strong> qui le portent, cliquables pour lancer la recherche correspondante. Sous la liste figurent les tags recommandés que la base n'utilise pas encore — un vocabulaire tiré de la littérature du backgammon (<code>#blitz</code>, <code>#prime</code>, <code>#holding</code>, <code>#backgame</code>, <code>#containment</code>, <code>#crunch</code>, <code>#ace-point</code>, <code>#timing</code>…), suggéré et jamais imposé : un tag absent de cette liste vaut exactement autant qu'un tag qui y figure.</p>
<p>Pendant la frappe, taper <code>#</code> propose les tags que <strong>cette base</strong> utilise déjà, puis les recommandés. C'est ce qui évite d'écrire <code>#back-game</code> un jour et <code>#backgame</code> le lendemain, ce que rien d'autre ne rattraperait.</p>
<p>La recherche par tag s'écrit <code>#prime</code> dans la ligne de commande. Elle est <strong>délimitée</strong> : <code>#prime</code> ne trouve pas <code>#priming</code>, là où une recherche de texte ordinaire, qui cherche une sous-chaîne, ne sait pas les distinguer. Plusieurs tags se <strong>cumulent</strong> — <code>s #prime #backgame</code> demande les positions qui portent les deux — parce qu'une position porte plusieurs tags : en nommer deux ne peut vouloir dire que « les deux ». C'est l'inverse du filtre de phase ou de provenance, où une position n'a qu'une valeur et où nommer deux valeurs ne peut vouloir dire que « l'une ou l'autre ».</p>
<p>La même liste s'obtient hors de l'interface avec <code>blunderdb list --type tags</code> (voir Interface en ligne de commande (CLI)).</p>
<h3>La corbeille</h3>
<p>Supprimer une position, une collection, un commentaire, une carte Anki ou un match passe par une <strong>corbeille</strong> : la suppression a bien lieu, mais une copie de ce qui disparaît est gardée trente jours. La commande <code>trash</code> ouvre la fenêtre qui les liste, avec pour chacune <em>Restaurer</em> et <em>Supprimer</em>, et un bouton <em>Vider la corbeille</em>.</p>
<p>Une position restaurée revient avec <strong>son analyse et ses commentaires</strong> — la rendre nue serait une restauration de nom seulement —, sa marque « étudiée » et les réponses d'entraînement données sur elle. Elle reprend son ancien numéro ; si la même position a été enregistrée entre-temps, c'est celle-là qui est gardée, car une position n'existe qu'une fois : elle reçoit la marque, les réponses et les commentaires qui lui manquent, et garde son analyse si elle en a une. Si son numéro est pris par une autre, elle en reçoit un nouveau. Une collection revient avec sa liste ; les positions qu'elle contenait, elles, n'avaient jamais été supprimées — une collection est une vue sur elles.</p>
<p>Un match supprimé emporte ses parties, ses coups, leurs analyses et les positions que plus rien d'autre ne retient, avec les notes venues du fichier source : la corbeille garde tout cela. Restauré, il revient sous son numéro et sa date d'import d'origine, avec ses parties, ses coups et ses statistiques, et le journal de son import le nomme de nouveau ; les positions que la suppression avait purgées reviennent comme une position restaurée, celles restées dans la bibliothèque restent telles qu'elles sont devenues. Il reprend sa place dans son tournoi ; si le tournoi a été supprimé, il revient hors tournoi. Il reprend sa place de Direction si elle existe encore, oppose les mêmes joueurs et est libre ; sinon elle reste au directeur et le match revient sans elle. Dans ces cas, la barre d'état le signale. La restauration est refusée, et l'entrée reste dans la corbeille, si un autre match occupe son numéro, ou si un match de même empreinte de fichier a été importé entre-temps ; un match sans empreinte, saisi ou joué dans blunderDB, n'est pas comparé ainsi.</p>
<p>Ce qui a plus de trente jours est supprimé par la commande <code>vacuum</code>, jamais à l'ouverture d'une base : ne pas faire de <code>vacuum</code>, c'est tout garder.</p>
<div class="admonition note">
<p>La corbeille ne voyage pas : un export ne l'emporte pas.</p>
</div>
<h3>Panneau Recherche</h3>
<p>Le panneau <strong>Recherche</strong> (<em>CTRL-F</em> ou <em>TAB</em>) permet de filtrer les positions selon des critères combinables librement : structure de pions, type de décision de videau, magnitude d'erreur, dates, tags, etc. La touche <em>TAB</em> ouvre simultanément le panneau de recherche et l'éditeur de position, permettant de définir une structure de pions à rechercher sur le plateau.</p>
<p>Les filtres se règlent dans le sous-onglet <strong>Critères</strong>. Quand une recherche ne trouve rien, le panneau l'affiche (« Aucune position ne correspond ») avec un bouton <strong>Effacer les filtres</strong>, sans se contenter de la barre d'état.</p>
<p>Pour chercher parmi les positions affichées, utiliser la commande <code>ss</code> suivie de filtres (ex: <code>ss nc</code>, <code>ss E&gt;40</code>). <code>ss</code> cherche dans la liste à l'écran : les résultats de la recherche précédente, la collection ouverte ou les positions du match en cours de revue, que la commande soit tapée directement ou depuis le panneau de recherche (<em>TAB</em>). La case à cocher <em>Rechercher dans les résultats actuels</em> du panneau suit la même règle ; la case <em>Ouvrir dans un nouvel onglet</em> affiche les résultats dans une nouvelle vue (voir Onglets de vues) plutôt que dans la vue courante. En collection et en match, <code>s</code> est refusé : il chercherait dans toute la bibliothèque et remplacerait la liste affichée.</p>
<p>Les résultats d'une recherche <code>ss</code> lancée depuis une collection ou un match se quittent avec <em>Esc</em>, en un seul appui dès que ni un champ ni le panneau qui a le focus n'a quelque chose à fermer (un coup sélectionné dans l'analyse, par exemple) : blunderDB revient à la collection entière, ou au match sur le coup étudié, et à la position quittée. Ce retour ne suit que <code>ss</code> : <code>s</code>, lancé depuis le panneau de recherche ouvert sur une collection ou un match, cherche dans toute la bibliothèque, et <em>Esc</em> ne ramène plus à la liste quittée.</p>
<p>Une recherche sur une grande base se montre avant d'être comptée : la première page de résultats s'affiche aussitôt, et la barre d'état indique « Recherche… » avec le temps écoulé tant que le nombre total n'est pas connu ; il remplace alors la longueur provisoire de la liste. Une seule recherche court à la fois : en lancer une autre abandonne la précédente. <em>Esc</em> interrompt la recherche en cours et arrête son balayage de la base ; si la première page était déjà affichée, elle reste, seule, et la barre d'état le dit.</p>
<p>Le panneau propose un contrôle explicite du <strong>type de décision</strong> recherché : <em>Indifférent</em> (aucun filtre), <em>Pions</em> (décisions de coup) ou <em>Videau</em> (décisions de cube). Lorsque <em>Videau</em> est sélectionné, une seconde liste précise le sous-type : <em>Tous</em>, <em>Double / Pas de double</em> (le joueur au trait doit décider de doubler) ou <em>Prise / Passe</em> (réponse à un doublement adverse). Le contrôle est synchronisé avec le plateau : modifier les dés ou le videau sur le plateau met à jour le type de décision, et inversement. En mode <em>Prise / Passe</em>, le videau est affiché au centre du plateau à la valeur offerte ; cette valeur reste éditable.</p>
<p>La <strong>phase de partie</strong> — ouverture, milieu de partie, course, sortie des pions — est une étiquette calculée par blunderDB à partir du plateau seul, jamais modifiable, et disponible en recherche par le jeton <code>ph:</code> de la ligne de commande (<code>ph:race</code>, répétable : <code>ph:race ph:bearoff</code>). Trois de ses quatre frontières sont celles que GNU Backgammon emploie pour aiguiller ses réseaux ; la quatrième, où s'arrête l'ouverture, est une convention de blunderDB : une position en est encore à l'ouverture tant qu'aucun des deux camps n'a déplacé plus de quatre pions de leurs points de départ, qu'aucun pion n'est sorti et qu'aucun n'est sur la barre.</p>
<div class="admonition note">
<p>L'étiquette est recalculée par la commande <code>blunderdb repair</code>. Sur une base ouverte pour la première fois avec cette version, le calcul est fait une fois, à l'ouverture. Une base dont les phases n'ont jamais été calculées ne renvoie rien pour <code>ph:</code> — rien, plutôt qu'une réponse fausse.</p>
</div>
<p>Le jeton <code>like</code> <strong>classe</strong> au lieu de restreindre : sa présence ordonne le résultat par distance croissante à une position cible — <code>like</code> la position courante, <code>like42</code> celle d'indice 42 — et les autres jetons restreignent l'ensemble ainsi classé, si bien que <code>s like42 E&gt;80</code> se lit « les voisines de la 42 où j'ai fauté ». La distance est une distance de transport en pions-pas, la quantité de mouvement de pions qui sépare deux positions, vue du joueur au trait.</p>
<p>Une voisine est le même <strong>problème</strong>, pas le même dessin : le classement se prend dans la classe de la cible — même type de décision, même régime (argent ou match) pour une décision de videau, et un match différent du sien, car les positions qui l'entourent dans sa propre partie sont ses structures les plus proches sans jamais être ses voisines. Les dés, le score et le videau restent hors classe ; les jetons ordinaires les filtrent quand on le veut. <code>like42*</code> élargit la classe à tous les types de décision et aux deux régimes, jamais au match de la cible ; <code>like&lt;12</code> écarte ce qui est à plus de douze pions-pas. Un classement qui ne trouve rien rend une liste vide et le dit, plutôt que dix positions sans rapport.</p>
<p>En mode <strong>édition</strong>, <code>s like</code> prend pour cible le plateau <strong>dessiné</strong> : on dessine à peu près la position dont on se souvient, on lance, et la bibliothèque répond — là où la recherche par structure exige le dessin juste. Le plateau est alors lu comme une position et non comme un motif : un point laissé vide compte comme des pions sortis, ce qui est exact pour une position réelle et fausse le calcul pour un dessin laissé à moitié.</p>
<p>Chaque voisine porte sa distance sous les tableaux d'analyse, avec la position dont elle est proche. C'est ce qui permet de juger si l'on regarde une voisine ou une coïncidence, et c'est la raison d'être du plafond. Le classement se lance aussi sans passer par la ligne de commande : <em>CTRL-MAJ-L</em>, ou l'entrée <strong>Positions voisines</strong> du menu contextuel du plateau.</p>
<p>Le jeton <code>n</code> compte les <strong>rencontres</strong> : <code>n&gt;3</code> retient les positions rencontrées au moins trois fois dans la base, tous matchs et tous joueurs confondus. C'est une autre question que « qu'ai-je raté » — une position rencontrée vingt fois et bien jouée dix-neuf reste celle qu'il faut savoir par cœur. Le compte porte sur les décisions, pas sur les matchs : la même position deux fois dans un match compte pour deux, parce que c'étaient deux décisions. Combiné à un filtre de joueur, il ne compte plus que les occurrences de ce joueur : <code>n&gt;3 pl!"Alice"</code> retient les positions qu'Alice a eu à jouer au moins trois fois, et <code>pl"Alice"</code> celles des matchs qu'elle a disputés ; <code>op"Bob"</code> restreint de même le compte aux matchs contre Bob.</p>
<p>Le <strong>plan de jeu</strong> est une seconde étiquette dérivée, à côté de la phase, et elle répond à la question qu'un paquet de filtres sauvegardés ne sait pas poser : « montre-moi mes erreurs en holding game ». Jeton <code>gt:</code>, répétable (<code>gt:holding gt:mutualholding</code>), du point de vue du <strong>joueur au trait</strong> — le plan dans lequel se prenait la décision.</p>
<p>Les dix plans reconnus, dans l'ordre où les règles les épuisent, du plus spécifique au plus général :</p>
<ul>
<li><code>race</code> — les pions les plus arriérés des deux camps se sont croisés : aucun contact n'est plus possible. Frontière de GNU Backgammon.</li>
<li><code>bearin</code> — le joueur au trait rentre ses pions alors que l'adversaire tient encore une ancre dans son jan.</li>
<li><code>crunch</code> — le joueur au trait a au plus six pions hors de ses points 1 et 2. Règle de GNU Backgammon, seuil de son auteur.</li>
<li><code>backgame</code> — deux ancres ou plus dans le jan adverse.</li>
<li><code>acepoint</code> — une seule ancre, sur le point 1 adverse, avec au moins vingt pions de retard.</li>
<li><code>blitz</code> — trois points du jan faits ou plus, et l'adversaire à la barre ou avec un blot à frapper dans ce jan.</li>
<li><code>primevprime</code> — les deux camps tiennent une amorce d'au moins quatre points, et chacun a un pion enfermé derrière celle de l'autre.</li>
<li><code>mutualholding</code> — les deux camps tiennent une ancre haute.</li>
<li><code>holding</code> — le joueur au trait tient une ancre haute, l'adversaire non.</li>
<li><code>contact</code> — contact, et aucun des plans ci-dessus. L'ouverture atterrit ici.</li>
</ul>
<p>Trois de ces règles sont celles de GNU Backgammon et sont sourcées ; les autres sont des <strong>conventions de blunderDB</strong>. La littérature du backgammon décrit les plans de jeu sans en chiffrer les frontières, et aucune mesure d'accord entre classificateurs n'est publiée pour ce problème. Les seuils non sourcés — trois points du jan pour un blitz, quatre points pour une amorce, vingt pions de retard pour un ace-point game — sont donc énoncés ici plutôt que cachés dans le code, et ils sont versionnés : les changer et relancer <code>blunderdb repair</code> ré-étiquette toute la base.</p>
<div class="admonition note">
<p>Une seule étiquette est conservée par position, celle du joueur au trait. Une étiquette dérivée n'est jamais modifiable, jamais exportée comme une vérité, et une base dont les plans n'ont jamais été calculés ne renvoie rien pour <code>gt:</code> — comme pour <code>ph:</code>.</p>
</div>
<p>Le filtre <strong>Marquée</strong> retient les positions que vous avez marquées (<em>flag</em>) dans le logiciel d'origine du match. Seul eXtreme Gammon produit cette information, enregistrée coup par coup dans le fichier <code>.xg</code> ; blunderDB la lit à l'import et la conserve. Une décision de videau marquée donne deux positions marquées, le double et la prise/passe, blunderDB scindant en deux ce que le fichier source enregistre comme une seule décision.</p>
<div class="admonition note">
<p>Le marquage n'est pas rétroactif : les matchs déjà présents dans la base ne portent pas cette information, puisqu'elle n'existe que dans les fichiers source. Il suffit de réimporter le fichier <code>.xg</code> concerné — l'import détecte le doublon et n'ajoute rien d'autre que les marques, sans toucher aux commentaires ni aux analyses existants. Le marquage ne peut ni être posé ni être retiré depuis blunderDB : pour une liste de travail temporaire, utilisez plutôt une collection.</p>
</div>
<p>Le filtre <strong>Commentaire</strong> interroge les commentaires attachés aux positions selon trois modes exclusifs. <em>contient le texte</em> recherche un ou plusieurs mots dans le texte des commentaires (champ de saisie, mots séparés par <code>;</code>, au moins un doit correspondre) ; <em>a un commentaire</em> retient toute position portant un commentaire, quel qu'en soit le contenu ; <em>sans commentaire</em> retient au contraire les positions non annotées — utile, combiné à un filtre d'erreur ou de date, pour dresser la liste de ce qu'il reste à commenter.</p>
<div class="admonition note">
<p>Les commentaires importés depuis un fichier de match (XG, GNUbg) comptent comme des commentaires. Pour ne retenir que les vôtres, ajoutez le jeton <code>co:user</code> sur la ligne de commande (<code>co:xg</code>, <code>co:gnubg</code>, <code>co:bgf</code> et <code>co:unknown</code> désignent les autres provenances). Par ailleurs, les commentaires attachés à un <em>match</em> ou à un <em>tournoi</em> ne sont pas concernés : ils annotent le match ou le tournoi, non ses positions.</p>
</div>
<p>Le filtre <strong>Matchs &amp; Tournois</strong> s'appuie sur un sélecteur commun (fenêtre modale) plutôt que sur la saisie d'identifiants numériques : deux listes à cocher, une pour les matchs et une pour les tournois, chacune filtrable par texte (joueur, date, événement pour les matchs ; nom, date, lieu pour les tournois), avec des boutons <em>Tout</em> / <em>Aucun</em> qui n'agissent que sur le sous-ensemble actuellement filtré. Cocher un tournoi coche automatiquement (et grise) ses matchs membres dans la liste des matchs, rendant visible le fait qu'un tournoi équivaut à l'ensemble de ses matchs.</p>
<p>Le panneau de recherche comporte trois onglets sur son bord gauche : <em>Critères</em> (les filtres), <em>Historique</em> et <em>Enregistrés</em> — et un quatrième, <em>Assistant</em>, quand l'assistant interne est activé. L'onglet <strong>Historique</strong> liste les recherches passées avec leur date et leur commande : un clic sélectionne une recherche et affiche la position associée sur le plateau, un double-clic la ré-exécute. Chaque entrée peut être enregistrée dans la bibliothèque de filtres (icône signet, en donnant un nom au filtre) ou supprimée. L'onglet <strong>Enregistrés</strong> contient la <strong>bibliothèque de filtres</strong> : double-cliquer sur un filtre enregistré pour relancer la recherche correspondante (voir Annexe: Utilisation avancée des filtres). La commande <code>history</code> (alias <code>hi</code>) ouvre le panneau de recherche.</p>
<p>L'étoile d'un filtre de la bibliothèque l'<strong>épingle</strong>. Les filtres épinglés s'affichent en pastilles en haut du panneau, quel que soit l'onglet ouvert, numérotées dans l'ordre de la bibliothèque : un clic sur une pastille lance le filtre, et <code>ALT-1</code> … <code>ALT-9</code> lancent le filtre épinglé de ce rang depuis n'importe quel écran en mode NORMAL ou EDIT, sans ouvrir le panneau. Le filtre pose alors la même question qu'au double-clic, structure de pions comprise. L'épingle appartient à la base : elle suit le filtre renommé, disparaît avec le filtre supprimé et ne voyage pas avec l'export de la bibliothèque.</p>
<p>Une recherche relancée garde son classement : <code>s like42</code> classe contre la position 42, et <code>s like</code> contre le plateau enregistré avec la recherche — celui qu'on feuilletait ou qu'on avait dessiné. Une entrée qui n'a pas gardé de plateau n'est pas relancée contre celui de l'écran, et la barre d'état le dit.</p>
<div class="admonition tip">
<p>Se référer à la liste des commandes pour la liste des filtres disponibles.</p>
</div>
<h3>Panneau Collections</h3>
<p>Dans les panneaux Collections, Tournois, Anki et Transcription, le bouton <strong>+</strong> de l'en-tête, suivi du nom de ce qu'il crée (<strong>+ Nouvelle collection</strong>, <strong>+ Nouveau tournoi</strong>, <strong>+ Nouveau paquet</strong>, <strong>+ Nouvelle transcription</strong>), est l'unique geste de création : il ouvre le champ de saisie, que <em>Échap</em> ou <strong>Annuler</strong> referme dans les panneaux Collections et Tournois. Dans la liste des matchs, l'icône ⌨ ouvre la transcription du match et l'icône ✎ en corrige les métadonnées.</p>
<p>Le panneau <strong>Collections</strong> (<em>CTRL-B</em>) permet de gérer des collections de positions. Les collections peuvent être créées, renommées et supprimées. Des positions peuvent y être ajoutées ou retirées (touche <em>Suppr</em>, confirmation demandée). Double-cliquer sur une collection pour parcourir ses positions avec les touches <em>GAUCHE</em> et <em>DROITE</em>. La commande <code>ss</code> cherche parmi les positions de la collection ouverte ; <em>Esc</em> ramène ensuite à la collection (voir Panneau Recherche). L'ordre des collections et des positions au sein des collections peut être modifié par glisser-déposer. Appuyer sur <em>CTRL-B</em> ou exécuter la commande <code>collection</code> pour afficher ou masquer le panneau.</p>
<p>La <strong>Pile</strong> est la collection du geste « à revoir plus tard » : un double-clic hors du plateau, la touche <em>b</em> ou le bouton marque-page de la barre d'outils met la position affichée sur la Pile, et le même geste l'en retire. Un marque-page au coin du plateau dit que la position y est, sans autre message. Le geste vaut partout où une position est affichée : revue, recherche, édition, Eval, Transcription, quiz, Duel. La remise à zéro du plateau passe par le menu du clic droit (voir menu du plateau). La Pile est une collection ordinaire — renommée, réordonnée, exportée, vidée comme une autre ; elle est créée au premier usage, et recréée si elle a été supprimée. Une position qui n'est pas encore dans la bibliothèque (un brouillon du plateau de recherche ou d'évaluation) y est écrite sur-le-champ, comme une position apportée seule, puis mise sur la Pile. La ligne de commande fait le même geste par <code>collection pile</code>.</p>
<p>Une collection peut être <strong>vivante</strong> : sa composition n'est plus une liste faite à la main mais le résultat d'une <strong>recherche</strong>, réévalué chaque fois qu'on l'ouvre. Le bouton ◇ en tête de la collection la rend vivante avec la dernière recherche lancée ; ◈ signale qu'elle l'est déjà, et le même bouton la rend à sa liste. Rien n'est détruit en la rendant vivante : les positions qu'elle contenait sont toujours là quand on revient en arrière.</p>
<p>Le bouton ❄, visible sur une collection vivante, la <strong>fige</strong> : les positions que la recherche sélectionne à cet instant deviennent la composition d'une collection ordinaire, dans l'ordre de la recherche, et la requête s'efface. Les positions qu'elle contenait avant d'être vivante sont remplacées.</p>
<p>Une collection vivante dont la requête porte un jeton que cette version ne connaît plus <strong>refuse de s'ouvrir</strong> en le disant, plutôt que de renvoyer toute la base. C'est la seule panne qu'un filtre enregistré ne doit pas avoir : s'élargir en silence.</p>
<p>Le bouton <strong>Proposer…</strong> de l'en-tête ouvre les <strong>positions de référence</strong> : une courte liste de positions à étudier, choisie parmi vos erreurs pour que chacune en résume beaucoup. Étudier les dix pires erreurs donne souvent dix variantes d'un même problème, ou des positions singulières qui n'apprennent rien sur les autres ; une position de référence est au centre de plusieurs de vos erreurs, de structure voisine, et sa leçon sert à toutes.</p>
<ul>
<li>Les erreurs sont celles du plan d'étude : chiffrées en MWC et rangées par famille (plan de jeu, nature, thème). Deux erreurs d'une famille sont <strong>voisines</strong> à au plus 12 pions-pas de distance <code>like</code> ; une erreur de videau n'a pour voisines que celles du même score.</li>
<li>Chaque position vaut le <strong>MWC récupérable</strong> (perte moins difficulté) de ses voisines et d'elle-même : ce que sa leçon rapporterait si elle servait à toutes. Une leçon <strong>serrée</strong> (le second choix coûte moins d'une demi-erreur) ou un verdict <strong>instable</strong> (une autre profondeur ou un rollout dit autre chose) compte pour moitié ; un rollout qui confirme est signalé. Le chiffre en tête de chaque ligne est le MWC couvert, la raison donne le chiffre compté quand la leçon compte pour moitié.</li>
<li>La liste est <strong>variée</strong> : une position retenue couvre ses voisines, qui ne rapportent plus rien aux suivantes, et aucune position proposée n'est un quasi-doublon d'une autre. Une position <strong>déjà traitée</strong> — commentée, dans une collection, une carte Anki ou marquée « étudiée » — n'est jamais proposée, et ses voisines sont tenues pour couvertes.</li>
</ul>
<p>La <strong>portée</strong> est toute la base, le match en cours, un tournoi ou le filtre du panneau Statistiques ; le joueur est celui de ce filtre. La <strong>taille</strong> est de 10, 20 ou 50 positions. Chaque ligne donne sa famille, son score s'il s'agit du videau, le MWC qu'elle couvre et sa raison : combien d'erreurs et de matchs elle résume, la taille de sa famille parmi vos décisions, son propre récupérable, l'écart avec le second choix. Cliquer sur une ligne ouvre la position ; les cases cochées deviennent en un clic une <strong>collection</strong>, un <strong>paquet Anki</strong> ou un <strong>quiz</strong>. En ligne de commande : <code>blunderdb collection suggest</code> (voir collection — Gérer les collections).</p>
<h4>Leçons</h4>
<p>Une <strong>leçon</strong> est une suite d'étapes qu'un coach écrit une fois pour un élève et lui remet dans un fichier de base (voir la commande <code>lesson export</code> de cli). Chaque étape a un titre, un texte et peut montrer une collection, une position, les deux ou aucune. La commande <code>le</code> liste les leçons de la base dans la barre d'état ; <code>le 2</code> ouvre la leçon 2 ; <code>le edit</code> ouvre l'éditeur de leçons.</p>
<p>Une <strong>barre de lecture</strong> apparaît alors au-dessus du plateau : nom de la leçon, numéro de l'étape, titre, puis le texte. <em>Précédente</em> et <em>Suivante</em> changent d'étape ; l'étape amène sur le plateau la collection ou la position qu'elle montre, que l'on parcourt ensuite par les gestes habituels. <em>Fermer</em> quitte la leçon. Une étape dont la collection ou la position a été supprimée garde son texte.</p>
<p>La case <em>Étape faite</em> de la barre marque l'étape courante comme faite ; un second clic retire la marque, et la barre compte les étapes faites. C'est le seul geste qui écrive quelque chose : lire une leçon, changer d'étape ou l'ouvrir n'enregistre rien, ni l'étape atteinte ni l'ouverture. La marque s'écrit dans la base ouverte, celle de l'élève, et aucun export ne l'emporte. Importer un fichier qui contient une leçon la crée ; une leçon de même nom déjà présente n'est pas touchée.</p>
<p><strong>L'éditeur de leçons</strong> s'ouvre par <code>le edit</code> (<code>le edit 2</code> sur la leçon 2) ou par le bouton <em>Modifier</em> de la barre de lecture. À gauche, la liste des leçons et un champ pour en créer une ; à droite, le nom et la description de la leçon choisie, puis ses étapes. Chaque étape a un titre, un texte, une collection choisie dans la liste et une position : <em>Position courante</em> y attache la position affichée sur le plateau, <em>Détacher la position</em> l'en retire. <em>Enregistrer l'étape</em> écrit ses changements ; les flèches la déplacent d'un cran ; <em>Ajouter une étape</em> en ajoute une à la fin. <em>Lire</em> ferme l'éditeur et ouvre la leçon à sa première étape ; <em>Supprimer</em> efface la leçon et ses étapes, sans toucher aux collections ni aux positions qu'elles montraient. Les leçons se créent et se modifient aussi par la ligne de commande ou par l'API (Les leçons).</p>
<p>Une sauvegarde — l'export de toute la bibliothèque depuis la fenêtre d'export ou par la ligne de commande — emporte toutes les leçons. Dans la fenêtre d'export, la case <em>Inclure les leçons</em> les choisit une à une : chaque leçon cochée part avec les collections et les positions que ses étapes montrent, et le fichier peut être filigrané ou protégé par mot de passe (<code>.dbx</code>) comme tout export. Sans cette case, un export partiel (une sélection de positions, de collections ou de matchs) n'emporte pas les leçons.</p>
<h3>Import : ce qui est écrit, ce qui ne l'est jamais</h3>
<p>Importer un match, une position ou une autre base ajoute ce qui manque ; cela ne remplace pas ce qui est déjà là.</p>
<ul>
<li><strong>Une position n'est jamais dupliquée.</strong> C'est son identité — pions, videau, dés, score — qui la reconnaît, jamais le fichier d'où elle vient : la même position rencontrée dans deux matchs reste une seule ligne.</li>
<li><strong>Une analyse par moteur.</strong> eXtreme Gammon, GNUbg, BGBlitz et l'évaluateur embarqué cohabitent sur une même position, et le panneau Analyse indique l'origine de chacune. Importer l'une n'efface pas l'autre.</li>
<li><strong>Une analyse importée n'est jamais recalculée.</strong> blunderDB la range telle quelle, avec son étiquette de niveau (« 3-ply », « XG Roller++ », « Book »), ses équités, ses erreurs, ses probabilités et la chance du lancer. La règle est « une évaluation ne comble qu'un trou » : l'analyse automatique après import ne visite que les positions sans <strong>aucune</strong> analyse, et <em>Ré-analyser les positions périmées</em> laisse intacte toute position portant une analyse importée (voir Configuration).</li>
<li><strong>Réimporter un match déjà présent n'apporte que du plus profond.</strong> Le match est reconnu à son jeu (joueurs, longueur, dés, coups, videau), pas à son analyse : aucun match, partie ni coup n'est réécrit. Les marques posées dans le logiciel d'origine sont ajoutées, et une analyse plus profonde que celle rangée la remplace, position par position — une version Roller++ du même match remplace la version 3-ply, quel que soit l'ordre d'import ; à profondeur égale ou moindre, l'analyse rangée reste. Réimporter le même fichier ne réécrit donc rien. Le rapport d'import distingue les doublons qui n'apportaient rien de ceux qui ont approfondi des analyses. En ligne de commande, <code>--skip-duplicates</code> ignore un doublon sans rien en reprendre d'autre que les marques. Un match tronqué puis complété (plus de parties) n'est pas le même match : il est importé comme un second match.</li>
<li><strong>Une analyse illisible n'empêche pas l'import du match.</strong> Une décision dont l'analyse contient une valeur qui n'est pas un nombre fini (NaN ou infini, que portent certains fichiers XG) est importée sans cette analyse ; le reste du match entre normalement, et le rapport d'import compte les décisions concernées.</li>
<li><strong>Un match déjà présent sous d'autres noms est signalé, jamais fusionné.</strong> Les deux empreintes de match portent les noms des joueurs : « Martin A. » et « Alice Martin » font deux matchs. L'import compare aussi les dés (longueur, score initial, dés de chaque partie) et signale, sous la ligne du fichier et dans le rapport, le match déjà en base sous d'autres noms dont il a les dés. <code>blunderdb repair --duplicates</code> liste ces paires dans une base existante, ainsi que les matchs tronqués et leur version plus longue.</li>
<li><strong>Un nom connu sous une autre graphie est enregistré sous son nom canonique.</strong> Un <em>alias</em> dit que « Martin A. » est une autre graphie d'« Alice Martin » (ou qu'un nom d'événement en est un autre). À l'import, les noms des joueurs et de l'événement sont remplacés par leur nom canonique, et le match rangé dans le tournoi de l'événement canonique. Les empreintes du match gardent les noms du fichier : un fichier importé avant que son alias existe se reconnaît donc toujours. Un match dont les dés sont ceux d'un match en base, et dont les noms ne diffèrent que par des alias connus, n'est pas un second match : ses analyses enrichissent celui qui est rangé.</li>
</ul>
<p>Les alias se gèrent dans l'onglet <strong>Corpus</strong> des paramètres : choisir <strong>Joueurs</strong> ou <strong>Événements</strong>, saisir l'alias et le nom canonique, ou retirer un alias de la liste. <strong>Proposer</strong> liste les noms qui ne diffèrent que par la casse, les accents, la ponctuation ou l'ordre des mots ; rien n'est appliqué sans un clic sur <strong>Appliquer</strong>. Le même onglet porte la case <strong>Ignorer les doublons à l'import</strong> (l'équivalent de <code>--skip-duplicates</code>, pour la session) et la recherche des <strong>doublons probables</strong> d'une base existante, celle de <code>blunderdb repair --duplicates</code> : chaque paire est donnée par ses numéros de match, rien n'est fusionné. Sous une paire aux mêmes dés, <strong>Créer ces alias</strong> enregistre en un clic les alias de joueur qui feraient nommer aux deux matchs les mêmes joueurs, la graphie du match le plus récent devenant l'alias : une seule proposition quand un nom est commun aux deux, sinon deux — siège pour siège, puis croisée — dont on choisit celle qui dit qui est qui. En ligne de commande : <code>blunderdb players alias</code> et <code>blunderdb events alias</code>.</p>
<ul>
<li><strong>Un dossier s'importe en parallèle.</strong> Les fichiers sont lus sur plusieurs cœurs à la fois et écrits par groupes, toujours dans l'ordre du dossier : les numéros de match ne dépendent pas de la machine. Un fichier identique, octet pour octet, à un fichier déjà lu du même dossier n'est pas relu : il est compté comme doublon. Annuler arrête l'import au groupe en cours ; ce qui était déjà écrit reste.</li>
<li><strong>La progression se lit en positions par seconde.</strong> La fenêtre d'import donne le pourcentage lu, le débit, le temps restant estimé et les comptes en cours (importés, doublons, en erreur). <strong>Réduire</strong> la range dans la barre de statut, d'où une pastille la rouvre : l'import continue pendant que vous travaillez, et la fenêtre revient d'elle-même avec le rapport à la fin. Les cent premières erreurs sont listées ; les suivantes sont comptées, et le journal de l'application les nomme toutes. En ligne de commande, <code>blunderdb import --type batch</code> affiche la même progression sur la sortie d'erreur, et <code>--format json</code> la rend dans l'objet final (<code>progress</code>).</li>
<li><strong>Chaque fichier d'un import est journalisé, et un import interrompu se reprend.</strong> Pour chaque fichier, le journal du lot garde le chemin, la taille, la date de modification, l'empreinte SHA-256 et le résultat : match nouveau (avec son numéro), doublon (avec le match qui le couvre), match enrichi, ou erreur (avec le message). Un import annulé ou coupé se continue sans reprendre ce qui est déjà décidé : un fichier de même chemin, même taille et même date est sauté sans être lu ; un fichier de même contenu est lu mais pas analysé ; un fichier en erreur est retenté. Dans l'application, la fenêtre de fin d'import liste le journal (un fichier par ligne, avec son résultat et le message d'une erreur ; un fichier retenté montre sa dernière issue), et une ligne qui a donné un match l'ouvre. Un import <strong>annulé</strong> laisse la fenêtre ouverte avec le bouton <strong>Reprendre</strong>. En ligne de commande, <code>blunderdb import --type batch --dir &lt;dossier&gt; --resume &lt;lot&gt;</code> reprend le lot dont le numéro a été affiché au départ de l'import, et <code>--format json</code> rend le journal dans l'objet final (<code>journal</code>). Le serveur accepte <code>resume</code> dans la requête de <code>imports.batch</code> et rend le journal par <code>imports.files</code>. Le journal est une donnée de l'import : ouvrir ou lire une base n'y écrit rien.</li>
<li><strong>Un import coupé par l'arrêt de l'application se reprend depuis la fenêtre.</strong> La commande <code>:resume</code> liste les imports que rien n'a terminé ; on choisit celui à continuer, puis on désigne à nouveau le dossier ou les fichiers, comme <code>--dir</code> en ligne de commande. Le journal du lot décide de ce qui n'est pas relu.</li>
<li><strong>Un gros dossier s'importe en mode masse.</strong> À partir de 200 fichiers, blunderDB écrit avec un cache plus grand et moins de points de contrôle. Si la base ne contient encore aucune position, il va plus loin : les index de recherche ne sont reconstruits qu'à la fin, et les écritures ne sont plus synchronisées sur le disque. Une coupure de courant pendant un tel import peut alors abîmer la base : il faut la recréer et relancer l'import. Un arrêt brutal du programme, lui, laisse seulement des index absents, que l'ouverture suivante reconstruit (le journal le signale).</li>
<li><strong>Ce que blunderDB n'écrit jamais</strong> : une chance recalculée — elle est lue dans le fichier source, ou reste inconnue — et un rollout qu'il n'a pas lancé lui-même : les données d'un rollout contenues dans un fichier <code>.xg</code> ne sont pas ouvertes. Seuls les rollouts que blunderDB produit (Rollouts) sont stockés, à côté de l'analyse.</li>
</ul>
<h3>Panneau Matchs</h3>
<p>Le panneau <strong>Matchs</strong> (<em>CTRL-Tab</em>) liste les matchs importés. Double-cliquer sur un match (ou appuyer sur <em>ENTREE</em>) pour naviguer dans ses coups. La commande <code>m</code> reprend la navigation dans le dernier match visité.</p>
<p>Le champ de filtre, en haut du panneau (<em>/</em> pour y aller, <em>Esc</em> pour l'effacer), ne garde que les matchs dont un joueur, l'événement, le lieu, le tournoi ou la date contient le texte saisi. Le filtre et le tri des colonnes sont faits par la base : la liste se charge par pages au fil du défilement, et le compteur « n / N matchs » indique la part chargée. Corriger un joueur, une date ou un tournoi dans la liste ne met à jour que la ligne éditée.</p>
<p>Quand la liste est vide, le panneau propose <strong>Importer… (Ctrl+I)</strong> ; sans base ouverte, il propose <strong>Ouvrir une base…</strong> à la place, avec <strong>Retour à l'accueil</strong>. Quand c'est le filtre de texte qui a vidé la liste, il propose <strong>Effacer le filtre</strong>. Les panneaux Stats, Collections et Anki vides offrent les mêmes boutons.</p>
<p>L'utilisateur peut:</p>
<ul>
<li>parcourir les coups d'un match en utilisant les touches <em>GAUCHE</em> et <em>DROITE</em>,</li>
<li>passer d'une partie à l'autre à l'aide des touches <em>PageUp</em> et <em>PageDown</em>,</li>
<li>afficher l'analyse des coups (pions et cube) en appuyant sur <em>CTRL-L</em>,</li>
<li>basculer entre l'analyse des coups de pions et du cube avec la touche <em>d</em>,</li>
<li>voir le coup effectivement joué mis en évidence dans l'analyse,</li>
<li>chercher parmi les positions du match avec la commande <code>ss</code> (ex: <code>ss E&gt;80</code>) ; <em>Esc</em> ramène ensuite au coup étudié (voir Panneau Recherche).</li>
</ul>
<p>La dernière position visitée dans chaque match est mémorisée et restaurée automatiquement. Appuyer sur <em>CTRL-Tab</em> ou exécuter la commande <code>match</code> pour afficher ou masquer le panneau.</p>
<p>Le bouton <strong>⊕</strong> d'une ligne enrichit ce match depuis un fichier. Il n'y a rien de nouveau derrière : réimporter le même match dans un autre format l'enrichit déjà en place — l'empreinte canonique reconnaît qu'il s'agit du même match, et les analyses et commentaires du second fichier viennent compléter le premier. Ce que le bouton apporte, c'est qu'on le trouve : personne ne devine qu'un import est aussi un enrichissement. Le compte rendu qui suit dit lequel des deux a eu lieu — « enrichis : 1 » plutôt que « importés : 1 ».</p>
<p>Chaque match peut être exporté en transcription Jellyfish <code>.mat</code> via le bouton ⬇ de la liste des matchs ou le bouton <em>.mat</em> de la fiche du match.</p>
<p>Un clic sur un match ouvre sa fiche. Son onglet <strong>Transcription</strong> liste les coups partie par partie, et un clic sur un coup y amène la revue. Chaque coup y porte sa <strong>gravité</strong> : <code>?</code> pour une erreur, <code>??</code> pour un blunder, un filet de couleur en marge de la ligne, et le coût du coup en équité au survol de la marque. Les seuils sont ceux de la base (Configuration), ceux que comptent les statistiques. Le coup est jugé tel qu'il a été joué : une même position jouée deux fois dans le match reçoit deux jugements. Un coup que l'analyse ne note pas ne porte aucune marque.</p>
<p>L'en-tête de chaque partie compte ses marques, qu'elle soit dépliée ou non : on voit sans l'ouvrir dans quelle partie se trouvent les blunders.</p>
<p>Quand le match est analysé, l'onglet ajoute la colonne <strong>MWC</strong>, les chances de gagner le match que la décision a coûtées, en pourcentage : <code>0</code> pour une décision sans perte, un tiret pour une décision que l'analyse ne note pas ou dont le score n'a pas de valeur dans la table d'équité du match (jeu en money, décision hors statistiques). Un clic sur l'en-tête <strong>MWC</strong> trie les coups de chaque partie de la perte la plus lourde à la plus légère, puis l'inverse, puis rend l'ordre du match ; les décisions non notées passent en dernier, et ce tri remplace celui de la durée. Au-dessus des parties, un résumé donne pour chaque joueur sa perte MWC totale (celle de la liste des matchs) et le nombre de décisions notées, avec deux graphiques sur le même axe de décisions que celui des durées : une barre par décision, puis la perte cumulée de chaque joueur, qui s'arrête sur son total (trait plein pour le premier joueur, pointillé pour le second). Une décision non notée porte un petit repère à la base de la barre, sans hauteur. Survoler une décision, ou la parcourir avec les flèches (les touches Début et Fin mènent aux extrémités), la marque sur les deux graphiques et sur sa ligne de la transcription et affiche la partie, le coup, le joueur et la perte ; un clic, ou Entrée, y amène la revue et fait défiler la transcription jusqu'à la ligne. Il en va de même pour le graphique des durées. Hors de l'interface, <code>match --format json</code> ajoute <code>decision_losses</code> (une entrée par coup, <code>mwc_loss</code> valant <code>null</code> pour un coup non noté), <code>--format text</code> une ligne « MWC loss » par coup noté et <code>--format summary</code> le total par joueur.</p>
<p><strong>Difficulté et erreurs évitables.</strong> Une perte dit ce qu'un coup a coûté, pas s'il était difficile de trouver le bon. La colonne <strong>Diff.</strong>, à côté de <strong>MWC</strong>, donne la <em>difficulté</em> de la décision : la perte de chances de gagner le match qu'un <em>joueur de référence</em> subirait en moyenne dans la même position. Ce joueur de référence ne joue pas parfaitement : il choisit chaque option avec une probabilité d'autant plus faible qu'elle coûte cher, π(i) proportionnelle à exp(−Δᵢ/τ), où Δᵢ est le coût de l'option i par rapport à la meilleure, en équité normalisée. La difficulté vaut d = Σ π(i)·Δᵢ, convertie en MWC au score et au videau de la décision comme la perte : elle s'exprime dans la même unité, un pourcentage de chances de gagner le match. Les options sont les candidats de l'analyse pour un coup de pions ; pas de double et double (contre la meilleure réponse) pour qui a le videau ; prendre et passer pour qui reçoit le double. La température τ vaut 0,025 : la décision à deux options la plus difficile est celle dont l'écart est d'environ 0,032, et une position à cinq candidats espacés de 0,01 donne l'erreur moyenne d'un joueur de PR 6 environ ; c'est un a priori de joueur fort, fixé avant tout examen de résultats.</p>
<p>À lire ainsi : une décision évidente, ou forcée, a une difficulté proche de <code>0</code> sans qu'aucun seuil la mette à part ; une décision serrée, où plusieurs options se tiennent à quelques millièmes, en a une grande. Une perte bien supérieure à la difficulté est une faute que la position n'excusait pas ; une perte proche de la difficulté, une faute qu'un bon joueur ferait aussi. Une décision est une <strong>erreur évitable</strong>, marquée d'un <code>!</code> à côté de sa perte et d'un point au-dessus de sa barre, quand sa perte atteint le seuil d'erreur de la base et que sa difficulté n'en dépasse pas le dixième : sur une décision à deux options, le joueur de référence ne la commettrait pas une fois sur dix. Sur le graphique par décision, la difficulté est un trait horizontal sur chaque barre : une barre qui monte loin au-dessus de son trait signale une erreur évitable.</p>
<p>Le tableau au-dessus des graphiques ajoute, pour chaque joueur et sur les décisions qui ont une perte et une difficulté : la <strong>difficulté</strong> totale, l'<strong>excès</strong> Σ(perte − difficulté), en MWC, ce que le joueur a perdu au-delà du joueur de référence (négatif quand il a fait mieux), le <strong>ratio</strong> perte totale / difficulté totale (1 : il joue comme le joueur de référence ; 2 : il perd deux fois plus) et le nombre d'<strong>erreurs évitables</strong>. Le ratio n'est pas donné quand la difficulté totale est sous 0,5 % : sur un match trop facile il ne mesurerait que du bruit, l'excès reste.</p>
<p>Pourquoi la regarder : le PR et la perte MWC mêlent deux choses, la qualité du jeu et la difficulté des positions rencontrées. Un adversaire qui crée des positions complexes fait monter l'erreur de l'autre ; un match de course pure la fait baisser. La difficulté corrige ces deux effets : l'excès et le ratio comparent le joueur à ce qu'on pouvait attendre dans <em>ses</em> positions. Pour l'étude, elle trie les fautes : les erreurs évitables sont de l'inattention ou une règle mal sue, à revoir en premier et souvent faciles à corriger ; les pertes sur des décisions difficiles relèvent du travail de fond (rollouts, principes, positions de référence) et pèsent moins sur le jugement d'un match.</p>
<p>Incertitude et limites : la difficulté dépend d'un modèle de joueur et de τ, fixés une fois ; changer l'un change tous les chiffres. Elle ne connaît que les candidats que l'analyse a gardés (quelques-uns chez XG, selon ses filtres chez GNU Backgammon) : les options absentes ne pèsent rien et la difficulté est alors un minorant. Elle hérite de l'erreur de l'analyse elle-même, surtout à faible profondeur. Sur un seul match, l'excès et le ratio reposent sur peu de décisions et varient beaucoup d'un match à l'autre : ils indiquent une tendance, pas un classement. La difficulté n'est calculée que pour les décisions dont la perte est notée ; ailleurs elle vaut un tiret. Hors de l'interface, <code>match --format json</code> porte <code>difficulty</code> et <code>avoidable</code> sur chaque entrée de <code>decision_losses</code> et le résumé par joueur sous <code>difficulty_summary</code> ; <code>--format text</code> ajoute les lignes « Difficulty » et « Avoidable error », <code>--format summary</code> la difficulté, l'excès, le ratio et le nombre d'erreurs évitables.</p>
<p><strong>Bilan du match.</strong> Au-dessus des graphiques, un encart répond, pour chaque joueur, aux trois questions qu'on se pose après un match : qu'est-ce que je revois, ai-je perdu à cause des dés ou du jeu, et mes erreurs viennent-elles de la précipitation ou d'une lacune ? Les seuils ont été fixés avant tout examen de résultats.</p>
<ul>
<li><strong>PR et perte MWC (éq. 7 pts), avec leur intervalle à 95 %</strong>, calculé en rééchantillonnant les parties du match. Il faut trois parties ; un match plus court montre un tiret. Un intervalle large dit qu'un match ne suffit pas à juger un niveau : comparez-le à vos autres matchs plutôt que de conclure sur une valeur.</li>
<li><strong>Résultat ajusté de la chance</strong>, pour un match terminé. Le <em>résultat</em> est l'issue (100 % gagné, 0 % perdu) moins les chances au départ (50 % à 0-0) ; la <em>chance nette</em> additionne la chance de vos jets moins celle des jets adverses, chaque jet converti en MWC au score et au videau de sa position, comme une perte. Le résultat ajusté est le résultat moins la chance nette. S'il est positif sur un match perdu, ce sont les dés qui l'ont décidé ; négatif, c'est le jeu. L'<em>écart des erreurs</em> (perte de l'adversaire moins la vôtre) est ce que le résultat ajusté estime : les deux chiffres se recoupent quand la chance est complète. La couverture est indiquée quand des jets n'ont pas de chance mesurée (un fichier sans chance, un jet non analysé) ; sans aucun jet mesuré, ou pour une partie d'argent, la ligne n'apparaît pas.</li>
<li><strong>Trois décisions à revoir</strong> : parmi vos erreurs (seuil de la base), celles dont la <em>part évitable</em> de la perte, perte − difficulté, est la plus grande — c'est-à-dire la perte multipliée par son caractère évitable. Une erreur que le joueur de référence aurait faite aussi ne vaut pas une séance : on revoit d'abord ce qui était à sa portée. Un clic sur la décision l'affiche, comme une ligne du relevé.</li>
<li><strong>Erreurs précipitées et réfléchies</strong>, quand le match a gardé la durée des décisions : une erreur jouée plus vite que la médiane de vos décisions du même type (pions ou videau) dans ce match est précipitée, les autres réfléchies. La médiane est la vôtre, dans ce match : elle ne dépend pas de la cadence, et sans lien entre vitesse et erreur les erreurs se partageraient moitié-moitié. Une majorité précipitée appelle de la discipline (ralentir sur ces positions) ; une majorité réfléchie, de la connaissance (étudier la famille de positions).</li>
</ul>
<p><code>match --format summary</code> imprime le même bilan, et le serveur le sert par <code>/v1/stats.matchReview</code>.</p>
<p>Quand le match a gardé la durée de ses décisions (un match joué contre un bot), l'onglet ajoute trois colonnes alignées sur les chiffres : <strong>Videau</strong> (la décision de videau, prise avant le lancer ou sur la ligne du videau elle-même), <strong>Jeu</strong> (le coup de pions) et <strong>Horloge</strong>, le temps que le joueur au trait a consommé depuis le début du match, ce coup compris, quel que soit le tri. Sous une cadence, la colonne devient <strong>Pendule</strong> et donne le temps qui reste à la pendule du joueur après le coup, calculé comme l'arbitre du Duel le compte (délai par tour, réserve ramenée à zéro au dépassement). Les informations du match et son en-tête rappellent la cadence (préréglage, délai, comportement au dépassement) et la banque de temps de chaque joueur, ramenée au score de départ quand la réserve est comptée par point restant. Un clic sur l'en-tête <strong>Jeu</strong> trie les coups de chaque partie du plus long au plus court, puis du plus court au plus long, puis rend l'ordre du match ; une case sans durée (pas de décision de videau à ce tour, ou coup joué par l'Arbitre seul) porte un tiret, et ces coups passent en dernier. L'Horloge part de zéro dès le premier coup. Au-dessus des parties, un résumé donne pour chaque joueur le total, la moyenne par coup de pions et par décision de videau, et, si le match a une cadence, une marque pour le joueur dont la réserve s'est épuisée en premier (le Duel n'enregistre que celui-là) ; un graphique place la durée de chaque décision au fil du match. En revue, la durée de la décision jouée se lit discrètement sous l'analyse. La recherche la filtre avec <code>tm&gt;30</code> (en secondes), qui se combine avec <code>E&gt;x</code> : <code>s tm&gt;30 E&gt;80</code> retient les coups longuement réfléchis et pourtant faux, la durée et l'erreur étant celles du même coup joué. L'onglet Statistiques, sous les erreurs récurrentes, croise le temps et l'erreur : pour chaque joueur et chaque tranche de durée connue (moins de 5 s, 5 à 15 s, 15 à 30 s, plus de 30 s), le nombre de décisions, l'erreur moyenne et la part de blunders. Une décision dont l'erreur n'est pas enregistrée est comptée sans entrer dans la moyenne.</p>
<p>Un match joué ici, issu d'un Duel, porte son origine en tête de la transcription, sur une ligne, même s'il n'a aucun coup : « Joué ici », puis, s'il y a lieu, « Perdu au temps » avec le joueur dont la réserve s'est épuisée sous une cadence qui fait perdre le match, ou « Arrêté avant la fin » pour un Match qu'un Duel arrêté a laissé inachevé, la cadence, le joueur dont la réserve s'est épuisée en premier et le niveau du Bot avec la version de gammonNet dont il a joué la politique. Un clic déplie le départ (la position initiale ou le XGID choisi), le germe des dés révélé et son SHA-256, à comparer avec l'empreinte publiée à la création du Duel : s'ils concordent, le germe permet de recalculer chaque lancer sans faire confiance à blunderDB. Un match qui n'a pas été joué ici n'a pas d'origine et n'affiche pas cette ligne.</p>
<p>L'onglet <strong>Infos</strong> de la fiche rappelle l'en-tête du match. Il y ajoute ce que le fichier source dit des joueurs et de la session, quand il le dit — un fichier eXtreme Gammon le dit toujours : le classement Elo de chaque joueur et son expérience entre parenthèses, le transcripteur, les règles Jacoby et Beaver d'une partie libre, et le programme qui a écrit le fichier. Ces informations sont exportées avec le match. Réimporter un fichier déjà présent les donne au match qui ne les avait pas, sans rien remplacer de ce qu'il porte déjà. La commande <code>match</code> de la ligne de commande les affiche aussi. Les commentaires d'en-tête et de pied de match d'un fichier eXtreme Gammon deviennent le commentaire du match, signé du nom de son transcripteur, ou <code>XG</code> si le fichier n'en nomme pas ; l'horloge et la table d'équité ne sont pas importées. La fiche affiche cet auteur à côté du commentaire, et l'export le copie avec lui ; modifier le commentaire le signe de <strong>Votre nom</strong> (réglages).</p>
<p>Un match transcrit depuis une vidéo en porte la <strong>source</strong> : une URL http(s) (YouTube, par exemple) ou le chemin d'un fichier. La ligne <strong>Vidéo</strong> de l'onglet <strong>Infos</strong> la modifie : saisir une URL ou un chemin puis <em>ENTREE</em>, <strong>Fichier…</strong> pour choisir une vidéo, <strong>Détacher</strong> pour retirer la source. Un match qui en a une affiche l'icône 🎞 dans la barre de sa fiche, et chaque décision qui porte un repère de la vidéo la même icône dans sa ligne de l'onglet <strong>Transcription</strong>. Cliquer l'icône, ou appuyer sur <em>v</em> quand la décision est celle de la revue, amène la vidéo une seconde avant le jet des dés : pour un fichier, la vidéo s'ouvre à côté du plateau, ou au-dessus de la fiche si elle a été remise dans le panneau, et <em>[</em> et <em>]</em> en règlent la vitesse ; pour une source YouTube, le navigateur s'ouvre sur le lien horodaté. Un fichier introuvable se relocalise depuis le volet, et un format que le webview ne lit pas y est signalé avec les paquets à installer (voir Téléchargement et installation). Un match sans source ne change pas.</p>
<p>Le bouton <strong>Fusionner les joueurs</strong> de la barre d'outils du panneau ouvre une fenêtre listant tous les noms de joueurs de la base avec leur nombre de matchs : sélectionner les variantes d'orthographe d'un même joueur, choisir le nom canonique à conserver, puis fusionner. La fusion crée un alias par variante : les matchs gardent les noms de leurs fichiers, mais les statistiques, la table Joueurs et la recherche <code>pl"…"</code> lisent toutes les variantes comme un seul joueur, et les imports suivants enregistrent le nom canonique. Retirer l'alias dans l'onglet <strong>Corpus</strong> des paramètres défait la fusion.</p>
<p>Lorsqu'un match est ouvert, une <strong>barre d'informations</strong> apparaît au-dessus du plateau : elle rappelle les joueurs en présence (<em>joueur 1</em> contre <em>joueur 2</em>) ainsi que le contexte du match (événement, lieu, ronde, date et longueur du match, lorsque ces informations sont disponibles). Cette barre s'affiche aussi en dehors du mode match : lorsqu'une position étudiée (issue d'une recherche, d'une collection ou d'un accès direct) provient d'un ou de plusieurs matchs, elle en indique la <strong>provenance</strong> — le premier match concerné et, le cas échéant, un badge « +N » listant les autres au survol. Une position importée seule, qu'aucun match ne référence, n'affiche rien.</p>
<p>Les onglets <strong>Recherche</strong> et <strong>Eval</strong> remplacent le plateau par un plateau de travail : un bandeau en haut du plateau le dit (« Plateau de recherche », « Plateau d'évaluation »), et la barre d'informations est masquée tant qu'elle décrirait une position qui n'est pas à l'écran. Le retour à l'analyse restaure la position étudiée.</p>
<p>À l'ouverture d'une base contenant des matchs, le panneau <strong>Matchs</strong> est affiché d'emblée et la revue débute directement sur la première position, afin de commencer immédiatement la navigation.</p>
<div class="admonition note">
<p>Une base de données ne peut être ouverte en écriture que par une seule fenêtre à la fois. Si vous ouvrez une base déjà ouverte dans une autre fenêtre de blunderDB, elle s'ouvre en <strong>lecture seule</strong> : la navigation, la recherche et l'analyse restent possibles, mais toute modification est désactivée et la barre de titre affiche « [lecture seule] ».</p>
</div>
<div class="admonition tip">
<p>Se référer à Raccourcis clavier pour les raccourcis disponibles.</p>
</div>
<h3>Panneau Transcription</h3>
<p>Le panneau <strong>Transcription</strong> (<em>CTRL-MAJ-T</em>, commande <code>transcribe</code> ou <code>tr</code>) sert à taper un match qu'on a sous les yeux — une feuille de match, un enregistrement vidéo — pour en faire un match de la bibliothèque. Ce qui se tape est un <strong>brouillon</strong> : il vit dans la base, se referme et se rouvre, et n'entre ni dans les statistiques ni dans les recherches tant qu'il n'a pas été enregistré en match.</p>
<p>Le panneau s'ouvre sur la <strong>liste des brouillons</strong> de la base : date de dernière modification, joueurs, longueur, nombre d'actions, et le match déjà produit (<code>#</code> suivi de son identifiant) ou la mention « aucun match ». Un clic ouvre un brouillon, et le bouton <strong>Brouillons</strong> de la barre y ramène. Le bouton <strong>Nouvelle transcription</strong> déplie le formulaire de création.</p>
<p>La liste se parcourt aussi au clavier : <em>BAS</em> et <em>HAUT</em> (ou <em>j</em> et <em>k</em>) déplacent le surlignage, <em>ENTREE</em> ouvre le brouillon surligné, <em>n</em> déplie le formulaire. Le premier brouillon est surligné à l'ouverture et c'est le plus récemment modifié : reprendre le travail de la veille tient donc en deux touches, <em>CTRL-MAJ-T</em> puis <em>ENTREE</em>.</p>
<p>Le formulaire ne demande qu'une chose : la <strong>longueur du match</strong>. La valeur <code>0</code> désigne une partie d'argent et fait apparaître les cases <em>Jacoby</em> et <em>Beaver</em>. Le champ s'ouvre sur la longueur du dernier brouillon modifié, ou sur 7 lorsque la base n'en contient aucun. Les noms des joueurs ne sont pas demandés : le brouillon désigne les camps par <em>Joueur 1</em> et <em>Joueur 2</em>, et la liste affiche « Sans nom ».</p>
<p>Tout ce qui se déduit de ce qui a été tapé — la longueur du match (ou « Argent »), le score, la mention <em>Crawford</em> lorsque la partie en cours l'est, le numéro de la partie, l'état du videau — sa valeur, centré ou au nom de celui qui le possède — et le camp au trait — s'affiche dans la <strong>barre de match</strong>, au-dessus du plateau : c'est là que le regard se trouve déjà quand on se demande qui est au trait. L'action attendue, elle, est écrite en toutes lettres dans la <strong>barre d'état</strong> : « dés de Kévin », « réponse d'Alice au double ».</p>
<p>La <strong>barre du brouillon</strong>, en tête du panneau, se lit en trois groupes. À gauche, <strong>Brouillons</strong> ramène à la liste. Au centre, ce qui sert pendant la saisie : le sens du plateau, les deux flèches d'annulation, <strong>Vidéo</strong> et <strong>Métadonnées</strong>. À droite, les gestes qui font sortir le brouillon de lui-même et le menu <strong>⋯</strong>. Dans un panneau étroit, la saisie passe sur une seconde ligne.</p>
<p>Le <strong>joueur 1 reste en bas du plateau</strong>, quel que soit le camp au trait. Un match qui se transcrit est une partie qui se déroule : le trait change à chaque demi-coup, et suivre le trait retournerait le damier d'un tour sur l'autre — les pions qu'on vient de regarder passeraient en haut et l'œil referait le trajet à chaque jet. Le trait se lit aux <strong>dés</strong>, qui changent de côté. Le bouton ⇅ de la barre retourne le plateau et montre le joueur 2 en bas ; il ne modifie pas le brouillon, et l'affichage revient à l'endroit à la fermeture. À ne pas confondre avec le bouton <em>Inverser les joueurs</em> du volet Métadonnées, qui échange les deux joueurs dans le document lui-même.</p>
<p>Le bouton <strong>Métadonnées</strong> de la barre déplie l'en-tête du brouillon, à tout moment : les noms des deux joueurs — autocomplétés depuis les joueurs de la base —, l'événement, le lieu, la ronde, la date (celle du jour par défaut), le transcripteur (l'utilisateur de la base par défaut) et le tournoi auquel le match sera rattaché lors de l'enregistrement. Aucun champ n'est obligatoire : un brouillon sans noms s'enregistre et s'exporte, avec des en-têtes vides. Le bouton <strong>Inverser les joueurs</strong> échange les deux noms, donne toutes les actions au camp d'en face et retourne le plateau : c'est le même match, lu de l'autre côté.</p>
<p>La <strong>longueur du match</strong> se change dans ce même volet, à tout moment : le score, la partie Crawford et le référentiel — les parties d'argent lorsque la longueur vaut <code>0</code>, et les cases <em>Jacoby</em> et <em>Beaver</em> apparaissent alors — sont recalculés d'un bout à l'autre du brouillon, et les actions postérieures à la victoire sont marquées « au-delà de la fin » sans qu'aucune ne soit supprimée. La longueur entre dans l'identité des positions : après un enregistrement, la changer puis réenregistrer écrit des positions neuves, à analyser, et les anciennes disparaissent dès que plus rien ne les retient.</p>
<p>Sous la barre, le brouillon occupe trois régions : la <strong>palette</strong> des cibles souris, les <strong>coups candidats</strong> et le <strong>transcript</strong>. Elles se placent selon la largeur du panneau. Dans un panneau large — le dock du bas — les trois sont de front, la palette à gauche. Dans un panneau moyen, les candidats occupent le haut et le transcript vient à côté du triangle des jets, dans la place que celui-ci laisse à sa droite. Dans un panneau étroit, les trois se suivent : les candidats, la palette, le transcript — le triangle et le transcript n'y tiennent pas côte à côte sans amputer le transcript de sa seconde colonne, et élargir le dock de quelques dizaines de pixels suffit à les réunir. Dans tous les cas la règle est la même : rien ne s'intercale entre les deux cases du jet et la première ligne des candidats, et cinq candidats au moins se lisent sans faire défiler quoi que ce soit.</p>
<p>La palette montre les deux dés au fur et à mesure de leur saisie ; un clic dessus les efface, comme <em>RETOUR ARRIERE</em>. Une partie commence par son premier coup, joué par le gagnant du jet d'ouverture : ses deux dés se tapent comme ce jet, dé du joueur 1 puis dé du joueur 2, et le plus fort donne le coup à son camp, qui joue les deux dés. Le coup se choisit ensuite parmi les candidats, comme tout autre. Les égalités, rejouées à la table, ne se transcrivent pas ; un premier coup tapé en double est enregistré tel quel et marqué « dés incohérents », aucun jet d'ouverture n'étant un double, et un double de videau avant le premier coup est marqué « action de videau impossible ». Une action de videau inscrite après la fin d'une partie porte la même marque : elle reste dans la partie terminée, n'en ouvre pas de nouvelle, et se supprime à la main.</p>
<p>Dès que le second dé tombe, tous les <strong>coups légaux</strong> du jet sont listés, classés par le moteur embarqué, le premier présélectionné et ses flèches posées sur le plateau. La liste donne le coup, son équité et son écart au meilleur : transcrire, c'est reconnaître le coup qu'on a vu jouer, pas le juger — le panneau <strong>Évaluation</strong> est là pour cela. Ce classement est une évaluation : il est affiché, il n'est jamais écrit dans la base. Lorsque le moteur n'est pas disponible, les coups sont listés sans classement et la liste le dit en tête.</p>
<p>La <strong>molette</strong> sélectionne le candidat suivant ou précédent, au-dessus de la liste comme au-dessus du damier : le regard reste sur le plateau et les flèches défilent, ce qui reconnaît un coup plus vite que la lecture de sa notation. Un clic sur une ligne la sélectionne, un double-clic la valide.</p>
<p>Le triangle des vingt et un jets est posé sous les deux cases du jet, à côté du clavier et non à sa place : deux chiffres restent deux fois plus rapides qu'un clic, et le triangle est là pour qui transcrit la souris à la main. Une case par jet, jamais deux : 3-1 et 1-3 sont le même jet.</p>
<p>Le coup joué au plateau dispense de lire les dés. Tant qu'aucun dé n'est saisi, un clic sur un pion puis sur sa destination — ou un glissé de l'un à l'autre — joue le coup sur le damier, contraint aux coups légaux ; les destinations offertes par le pion choisi s'allument. Les deux dés se déduisent des pas : jouer 13/7 puis 8/7 dit 6-1 sans qu'un chiffre ait été tapé, et l'action est enregistrée dès que le coup est achevé. Retour arrière défait le dernier pas, un chiffre abandonne le coup et revient à la saisie par les dés, et <em>Recommencer</em>, en tête du menu du clic droit, le reprend depuis le début. Quand plusieurs jets produisent le même coup — une sortie que plusieurs dés couvrent, un dé qui n'est pas jouable — rien n'est enregistré et le triangle ne laisse cliquables que ces jets-là : le jet n'est jamais deviné à la place de celui qui regarde la partie.</p>
<p>Les deux dés saisis, le plateau joue aussi, contraint aux coups légaux de ce jet — en bout de document comme sur une action relue, dont le curseur a chargé les dés. Chaque pas joué ne garde dans la liste que les candidats qui le contiennent, le premier d'entre eux présélectionné : c'est le geste du coup lointain, là où descendre au douzième candidat coûte treize touches. Un coup légal achevé est enregistré aussitôt, avec les dés tels qu'ils ont été tapés ; sur une action relue, il la remplace.</p>
<p>Un coup illégal se transcrit tel qu'il a été joué, sans bouton ni changement de mode. Les dés saisis, un glissé qu'aucun coup légal n'offre pose le pion là où il est lâché — y compris depuis un point d'où aucun coup légal ne part, pourvu qu'il porte un pion du camp au trait. Le coup sort alors des règles : la suite se joue librement, au clic comme au glissé, la liste des candidats cède la place à une ligne qui le rappelle, et rien n'est enregistré avant ENTREE, qui écrit les dés saisis, les pas et le plateau obtenu. Retour arrière défait le dernier pas ; défaire le seul pas hors des règles rend la liste. Sans dés saisis, le glissé reste contraint : un coup illégal ne dit pas quel jet l'a produit.</p>
<p>Le coup se tape aussi au clavier, dans le transcript. Un double-clic sur la cellule d'un coup — ou d'une danse, d'un coup non consigné — la change en champ, pré-rempli de sa notation. On n'y tape que le coup, <code>13/7 8/7*</code>, <code>bar/22</code> ou <code>6/off</code> : les dés sont ceux de la cellule. ENTREE l'enregistre à la place du coup écrit, ÉCHAP referme la cellule sans rien écrire, et un texte qui ne dit aucun coup laisse le champ ouvert. La cellule en pointillés de la saisie en cours s'ouvre de même, dès que ses deux dés sont saisis.</p>
<p>Un coup saisi par le glissé libre ou par la notation qui se trouve être légal reste un coup ordinaire — la comparaison se fait sur le plateau obtenu, jamais sur la provenance du geste ; sinon il est marqué « coup illégal » dans le transcript, et l'export <code>.mat</code> avertit avant d'écrire le fichier, sans jamais refuser.</p>
<p>Sur la ligne des dés, la rangée <strong>Doubler</strong>, <strong>Prendre</strong>, <strong>Passer</strong>, <strong>Abandonner</strong> reprend à la souris les quatre gestes de videau : ce sont, avec les deux dés, les cinq réponses possibles à une seule question — qu'a fait le camp au trait ? Elle dit de qui est le tour : le camp au trait annonce — doubler, abandonner — ou le camp d'en face répond — prendre, passer ; jamais les quatre à la fois, et un bouton dont le geste ne répondrait à rien reste éteint. Le clavier, lui, ne refuse jamais rien : un bouton éteint est une cible qu'on n'offre pas, pas un geste interdit. « Abandonner » n'enregistre rien encore : la rangée devient les trois niveaux — simple, gammon, backgammon — et « Annuler », qui reprend la touche ÉCHAP. Le videau dessiné sur le plateau est la seconde cible de ces gestes : un clic dessus propose un double. Devant une offre il ne répond pas — la prise et la passe sont deux réponses symétriques et vivent ensemble dans la rangée, un clic chacune.</p>
<p>La <strong>barre d'état</strong> dit ce que le brouillon attend, en un mot : la danse enregistrée d'office, le premier coup d'une partie, la réponse attendue à un double, le niveau attendu après une résignation, la correction en place, le coup « à revoir » dont le jet a changé. Elle y répond aussi aux gestes qui n'ont rien à faire — « rien à annuler », « aucune action sous le curseur » — le temps d'une seconde et demie. L'incohérence qu'une action a laissée derrière elle, elle, est signalée en tête du transcript, là où se trouve la cellule fautive.</p>
<p>Une partie se termine par une passe, par une résignation ou par la sortie du quinzième pion (simple, gammon ou backgammon, multiplié par la valeur du videau). Le score, la partie Crawford et la fin du match paraissent alors dans la barre de match, et le premier coup de la partie suivante est attendu.</p>
<p>Le score d'une partie est celui que donnent les parties précédentes, sauf s'il a été annoncé autrement à la table. Un double-clic sur le score de l'en-tête d'une partie, dans le transcript, le change en champ pré-rempli : on y tape le score auquel la partie a été jouée — <code>3-2</code>, <code>3–2</code> ou <code>3 2</code> —, ENTREE l'enregistre, ÉCHAP referme le champ sans rien écrire, et un champ vidé puis validé revient au score dérivé. La partie est jouée à ce score : la partie Crawford, la fin du match et les parties suivantes en découlent, et le match enregistré comme le fichier <code>.mat</code> le portent. Un score qui diffère du score dérivé est marqué, l'info-bulle donne celui-ci, et la première action de la partie porte l'incohérence « score annoncé incohérent ». En argent, il n'y a pas de score à annoncer.</p>
<p>Le transcript occupe la moitié droite du panneau : une colonne par joueur, une ligne par tour, l'action de videau et la fin de partie dans la colonne de celui qui agit. La cellule du curseur est encadrée ; déplacer le curseur ramène le plateau à la position de l'action visée et affiche ses candidats, le coup joué sélectionné. Une incohérence (coup illégal, double trait, videau impossible, action au-delà de la fin du match, dés incohérents, coup non consigné, score annoncé incohérent) décore sa cellule et se nomme dans une info-bulle. Le coup non consigné est le cas d'un fichier <code>.mat</code> relu : gnubg y écrit <code>???</code> quand il n'a pas gardé le coup joué, le jet est connu et le coup ne l'est pas, et poser le curseur sur cette cellule propose les coups de ce jet pour le renseigner. Un double trait laisse une case vide, encadrée de pointillés, dans la colonne du camp dont le tour manque : le curseur s'y arrête, un clic l'y mène, et c'est là que se tape le tour manquant — une décision supprimée, par exemple. Les parties se replient ; celle du curseur est ouverte.</p>
<p><strong>Ce qui est en train d'être tapé se dessine dans le transcript</strong>, en pointillés, à la place exacte où il sera écrit : les dés au fur et à mesure qu'ils tombent, la notation du coup dès qu'il est sélectionné, et dans la colonne du camp à qui l'action revient. Une correction recouvre la cellule qu'elle remplace, une insertion ouvre une cellule entre ses deux voisines, une saisie neuve paraît au bas de la partie en cours. Rien n'est écrit dans le brouillon avant la validation ; ce que l'on lit et ce que le document dira ne divergent jamais.</p>
<p>Corriger, c'est taper sur la cellule où l'on est. Le curseur posé sur une action, un chiffre en recommence le jet sur place, et les quatre gestes de videau valent eux aussi comme corrections : sur une passe, <em>t</em> — ou le bouton <strong>Prendre</strong> — écrit une prise <strong>à la place</strong> de la passe, sans qu'il faille la supprimer puis insérer. La partie reprend alors son cours : une cellule s'ouvre juste après la prise, au camp du doubleur, et la suite de la partie s'y tape comme d'habitude, insérée devant le premier coup de la partie suivante, jusqu'à ce que la partie se termine. Ce premier coup garde le score auquel sa partie commençait, désormais annoncé : si la fin de la partie reprise en donne un autre, l'écart est marqué. Les mêmes touches remplissent la cellule qu'une insertion vient d'ouvrir : <em>i</em> puis <em>d</em> insère un double devant l'action visée.</p>
<p>Revenir sur la <strong>dernière</strong> action, c'est revenir là où la transcription s'écrit. Un chiffre tapé sur elle en corrige encore le jet, mais une fois ce jet retapé, le chiffre suivant la valide et ouvre la décision d'après ; ENTREE la valide de même. Les décisions suivantes s'ajoutent alors à la suite, comme la première fois.</p>
<p>Taper un <strong>autre jet</strong> sur le <strong>premier coup</strong> d'une partie décide à nouveau qui commence : le dé du joueur 1 se tape en premier, celui du joueur 2 ensuite, et c'est le plus fort qui l'emporte — gros dé d'abord, le coup revient au joueur 1, en bas du plateau ; petit dé d'abord, il revient au joueur 2, en haut. Le premier candidat du nouveau jet est présélectionné et le coup est à revoir. Retaper le <strong>même jet</strong>, dans un ordre ou dans l'autre, ne change rien. Pour donner le premier coup à l'autre joueur sans changer de jet, c'est <em>s</em> : l'ordre des dés suit le camp, et un premier coup joué par le camp que cet ordre ne désigne pas est marqué « dés incohérents ». La suite de la partie garde ses camps.</p>
<p>Une <strong>insertion au milieu du document continue d'insérer</strong> : la validation ouvre une cellule vide à la suite, et l'action suivante s'insère à son tour au lieu d'écraser celle d'après. C'est ce qui permet de rattraper toute la fin d'une partie — une passe qui aurait dû être une prise — sans perdre ce qui a déjà été tapé de la partie suivante. La fin de la partie, ou un déplacement du curseur, met fin à l'insertion : le curseur se pose alors sur le premier coup de la partie suivante.</p>
<p><strong>Suppr</strong> (ou <em>x</em>) retire la décision en cours d'édition et recule sur la précédente, prête à être corrigée : sur une cellule écrite, l'action disparaît ; sur une insertion ouverte ou un jet tapé en bout de document, c'est la saisie qui est abandonnée. Suppr pressée à répétition remonte ainsi le transcript en effaçant. Les actions suivantes gardent leur camp, et le double trait qu'une suppression laisse est marqué sans que le curseur y soit ramené.</p>
<p>Un clic droit sur une cellule ouvre les corrections de cette action — insérer avant, insérer après, supprimer, changer de camp — et amène le curseur dessus au passage ; ce sont les mêmes gestes que les touches <em>i</em>, <em>a</em>, <em>x</em> et <em>s</em>, et le menu du navigateur n'est retiré que là. Ils n'ont pas de boutons ailleurs : un bouton qui agirait sur « l'action du curseur » viserait une cellule que l'on ne voit pas forcément, quand le clic droit désigne la sienne.</p>
<p>La barre du brouillon porte ses deux seules sorties. « <strong>Terminer</strong> » (CTRL-ENTREE) écrit le match dans la bibliothèque et libère le brouillon ; l'analyse des seules positions nouvelles démarre aussitôt, avec sa progression et son annulation dans la barre d'état. « <strong>Abandonner</strong> » supprime le brouillon sans match ; la confirmation n'est demandée que pour un brouillon qui n'a jamais été terminé, puisqu'il emporte tout ce qui y est écrit. À côté, la barre dit ce que Terminer fera — un nouveau match, ou le remplacement du match #<em>n</em>. Elle ne dit rien du salut du brouillon lui-même : il est écrit dans la base après chaque action, et revenir à la liste le laisse à reprendre plus tard.</p>
<p>Le menu <strong>⋯</strong> de la barre tient « <strong>Texte .mat</strong> », qui ouvre le fichier Jellyfish tel qu'il serait écrit, dans une fenêtre assez large pour que ses colonnes restent alignées, avec un bouton pour le copier, et « <strong>Exporter .mat</strong> », qui écrit ce même fichier sur le disque. Les deux flèches <strong>↶</strong> et <strong>↷</strong> annulent et rétablissent, comme <em>CTRL-Z</em> et <em>CTRL-MAJ-Z</em>.</p>
<p>Si l'analyse d'un match transcrit a été interrompue — l'application fermée pendant le lot —, la barre d'état le signale à la prochaine ouverture de la base et propose de la terminer. Rien n'est retenu de cette interruption : la proposition revient tant qu'il reste des positions à analyser, et le lot relancé ne porte que sur ce match, jamais sur toute la bibliothèque.</p>
<p>Un brouillon qui porte des incohérences est terminé tout de même, après un avertissement : rien n'est refusé. Un coup illégal est exporté tel qu'il a été joué, avec l'avertissement que gnubg et XG le signaleront (« Invalid move ») et divergeront ensuite.</p>
<p>Pour corriger un match de la bibliothèque, le bouton ⌨ de la liste des matchs ou « <strong>Éditer la transcription</strong> » de sa fiche ouvre un brouillon depuis ce match — ou rouvre celui qui y est déjà ouvert : un seul brouillon par match. Terminer ce brouillon remplace le match sous le même identifiant ; les positions des actions inchangées gardent leurs commentaires, leurs analyses et leurs cartes. Un match importé (XG, GnuBG, BGF) porte des analyses et des commentaires qu'un <code>.mat</code> ne porte pas : avant d'ouvrir, un dialogue dit jusqu'à combien, et que terminer le brouillon peut les perdre.</p>
<p>Un match se transcrit aussi <strong>depuis une vidéo</strong>. Le bouton <strong>Vidéo</strong> de la barre du brouillon ouvre un menu : <strong>Fichier local…</strong> pour choisir une vidéo sur le disque, <strong>Lien YouTube…</strong> pour coller une adresse dans le champ qui s'ouvre dans le menu, et <strong>Détacher</strong> pour retirer la source. <em>ECHAP</em> ou un clic ailleurs ferme le menu et rend la saisie au panneau. Tant qu'aucune source n'est attachée, le panneau reste tel qu'il est décrit plus haut : ni volet, ni touche de plus. Une fois la source attachée, le bouton porte son nom court — le nom du fichier, ou « YouTube » — et la vidéo s'affiche à côté du plateau. Un fichier introuvable se relocalise depuis la vidéo par <strong>Choisir le fichier…</strong>. Sur une vidéo YouTube déjà attachée, <strong>Lien YouTube…</strong> s'ouvre sur son adresse, sélectionnée : un collage la remplace. Un brouillon rouvert retrouve sa vidéo en pause là où on l'avait laissée sur ce poste, à défaut quelques secondes avant son dernier repère ; le curseur reste en fin de transcript, prêt pour la saisie, et ne suit la lecture qu'une fois qu'elle entre dans une autre action.</p>
<p><strong>La vidéo à côté du plateau.</strong> Tant qu'une vidéo est ouverte, la zone du plateau se partage : la vidéo à gauche, le plateau à droite, séparés par une barre verticale qu'on tire pour agrandir l'une ou l'autre ; la largeur reste la même d'une session à l'autre. La vidéo garde ses proportions et occupe au mieux la place qu'on lui donne. Le bouton placé dans son coin supérieur droit la remet dans le panneau, au-dessus du transcript, et la ramène à côté du plateau ; ce choix aussi est retenu. Dans le panneau, la hauteur de la vidéo se règle en tirant la barre placée sous elle, sans jamais repousser la saisie hors de vue. Passer d'une place à l'autre ne relance pas un fichier : la lecture continue au même instant ; une vidéo YouTube reprend à l'instant où elle était, en pause. Cliquer dans la vidéo, sur ses commandes ou sur la barre de séparation laisse le clavier au panneau : les touches de saisie, les touches de la vidéo et <em>CTRL-GAUCHE</em> / <em>CTRL-DROITE</em>, qui tournent le plateau, gardent leur effet.</p>
<p><strong>Le côté de la vidéo.</strong> La vidéo se met à gauche du plateau au départ ; elle peut passer à droite. Le bouton ⇄, à côté de celui qui la remet dans le panneau, change de côté d'un clic. On peut aussi saisir la petite poignée en haut de la vidéo et la glisser : les deux moitiés de la zone s'éclairent, et lâcher dans l'une y place la vidéo. <em>ÉCHAP</em>, ou lâcher hors de la zone, annule. Le plateau, lui, ne se déplace pas : y glisser déplace les pions. Le côté est retenu d'une session à l'autre, et la largeur reste celle de la vidéo, de quelque côté qu'elle soit. La poignée et le bouton laissent le clavier au panneau.</p>
<p>Avec une vidéo, chaque action <strong>nouvelle</strong> porte des <strong>repères</strong> : l'instant du jet, posé par la première touche de dé, et l'instant de l'action, posé par la validation, tous deux lus sur la vidéo au moment du geste. Une correction en place ne touche pas aux repères. Seule une validation explicite pose l'instant de l'action : <em>ENTREE</em>, le double-clic sur un candidat, ou le coup achevé au plateau. Le chiffre du jet suivant, ou un geste de videau, valide aussi le coup, mais ne pose aucun instant : le coup reste sans instant d'action plutôt que d'en recevoir un faux, et la barre d'état le rappelle. Un coup achevé au plateau sans dés tapés avant n'a pas d'instant de jet. Pour horodater la décision de pions, on valide donc par <em>ENTREE</em> au moment où le coup est fini à l'image. <em>v</em> pose après coup l'instant courant comme instant de l'action du curseur, <em>MAJ-V</em> comme instant du jet. <em>ESPACE</em> lance ou met en pause la vidéo ; <em>MAJ-GAUCHE</em> et <em>MAJ-DROITE</em> la déplacent de 5 secondes, <em>CTRL-MAJ-GAUCHE</em> et <em>CTRL-MAJ-DROITE</em> d'une seconde. <em>[</em> ralentit la lecture et <em>]</em> l'accélère, par pas de 0,25 entre 0,25× et 4× ; une vidéo YouTube ne propose que les vitesses de son lecteur, jusqu'à 2×. La vitesse s'affiche dans le coin de la vidéo quand elle n'est pas de 1×, et une nouvelle source repart à 1×.</p>
<p>Des repères se déduisent les <strong>durées</strong> de décision : la décision de pions va du jet à la fin du coup, la décision de videau de l'action précédente au jet, un double ou une réponse de l'action précédente à la leur. Une cellule qui porte un repère le signale par un point discret ; son info-bulle donne le jet, le coup et la durée (« jet 12:34, coup 12:51, 17 s »), et le panneau d'analyse affiche la durée de l'action du curseur comme pour un match joué en Duel. Un repère antérieur à celui qui le précède est marqué « repère à rebours », comme toute incohérence, et laisse inconnues les durées qui en dépendent. Placer le curseur sur une cellule (clic sur le coup ou sur sa durée, <em>h</em>, <em>l</em>) amène la vidéo une seconde avant le jet de cette action, ou avant son action si le jet n'a pas d'instant ; un lecteur en pause le reste.</p>
<p>Dès que le brouillon a une vidéo ou un repère, le transcript gagne une colonne <strong>durée</strong> à droite de chaque joueur : la durée de la décision, en secondes (en minutes et secondes au-delà d'une minute), suivie, après un point médian, de la décision de videau qui précède le jet. Un coup sans instant d'action n'est pas laissé vide : sa durée s'estime du jet au premier instant du coup suivant (son jet, ou un geste de videau), et s'affiche comme une durée mesurée ; son infobulle la dit estimée. C'est un majorant, qui compte le ramassage et le lancer des dés. Aucune décision de videau ne s'estime : sans l'instant d'action du coup précédent, sa durée reste inconnue, et la barre d'état propose alors <em>v</em> sur ce coup pour la mesurer. Pendant la lecture, le curseur <strong>suit la vidéo</strong> dans la partie déjà horodatée : chaque action couvre la vidéo depuis son jet, ou à défaut depuis l'action précédente, jusqu'au début de la suivante, et le curseur se pose sur l'action où entre la lecture ; au-delà du dernier repère, il revient une fois en fin de transcript. Il ne bouge ni pendant une saisie ni quand la vidéo est en pause, et ce déplacement n'entre pas dans l'historique d'annulation. Placé à la main, le curseur reste sur l'action choisie tant que la lecture n'en est pas sortie : on clique sur un coup sans instant d'action, on lance la vidéo, on appuie sur <em>v</em> à la fin du coup, et c'est ce coup qui reçoit l'instant.</p>
<p>L'application ne décode pas la vidéo elle-même : les formats lus sont ceux du webview. Un format qu'il ne lit pas est signalé dans le volet, avec son conteneur et, sous Linux, les greffons GStreamer à installer — <code>gstreamer1.0-plugins-good</code> et <code>gstreamer1.0-libav</code> sous Debian et Ubuntu, <code>gstreamer1-plugins-good</code> et <code>gstreamer1-plugin-libav</code> sous Fedora, <code>gst-plugins-good</code> et <code>gst-libav</code> sous Arch (voir Téléchargement et installation).</p>
<div class="admonition tip">
<p>Se référer à Raccourcis clavier pour les raccourcis disponibles.</p>
</div>
<h3>Panneau Tournois</h3>
<p>Le panneau <strong>Tournois</strong> (<em>CTRL-Y</em>) permet de regrouper des matchs en tournois pour un suivi organisé et une analyse statistique par événement. Les tournois peuvent être créés, renommés et supprimés ; les matchs peuvent leur être assignés. Les statistiques du panneau Stats peuvent être filtrées par tournoi. Appuyer sur <em>CTRL-Y</em> pour afficher ou masquer le panneau.</p>
<p><strong>Nouveau tournoi</strong> ouvre le champ de création, qui prend le focus ; <em>ÉCHAP</em> ou <strong>Annuler</strong> le referme. Un clic surligne une ligne, un double-clic ou <em>ENTRÉE</em> ouvre le tournoi : ses notes, puis ses matchs, un par ligne, qu'un double-clic ouvre, que ▲ et ▼ réordonnent, que ⇄ échange de joueurs et que × retire du tournoi. Le champ <strong>Ajouter un match…</strong> y range un match de la base, et ← ramène à la liste des tournois.</p>
<p>Les tournois se remplissent d'eux-mêmes à l'import. Les fichiers XG, GnuBG et BGF nomment leur événement ; à l'import d'un match nouveau, blunderDB le classe dans le tournoi de ce nom et crée celui-ci s'il n'existe pas encore. La date et le lieu du tournoi restent vides — c'est ici qu'on les renseigne. Un match déjà présent dans la base n'est jamais reclassé : réimporter son fichier ne défait pas le rangement fait à la main.</p>
<p>Les colonnes <strong>PR</strong> et <strong>MWC</strong> de chaque tournoi affichent le PR et la perte de MWC du <strong>joueur de référence</strong> — c'est-à-dire le joueur présent dans le plus grand nombre de matchs du tournoi (en cas d'égalité, celui ayant pris le plus de décisions). Le PR ne mélange donc pas votre jeu avec celui de vos adversaires : pour vos propres tournois, il reflète votre performance seule. Le nom du joueur de référence apparaît en infobulle au survol de la valeur.</p>
<p><strong>Bilan du tournoi.</strong> Le bouton <strong>Bilan</strong>, dans l'en-tête d'un tournoi ouvert, répond à la question qu'on se pose en rentrant d'une épreuve : ai-je joué à mon niveau, et sinon, où cela a-t-il cédé ? Le joueur est par défaut le plus présent dans les matchs du tournoi ; le menu <strong>Joueur</strong> en choisit un autre. Les seuils ont été fixés avant tout examen de résultats.</p>
<ul>
<li><strong>Le niveau habituel</strong> est ce joueur sur ses matchs en points des 365 jours qui précèdent le tournoi (sa date, à défaut celle de son premier match), ceux du tournoi exclus. Seul l'avant compte : c'est le niveau qu'on avait en arrivant. Il faut au moins 5 matchs ; sinon il est déclaré inconnu.</li>
<li><strong>PR et perte MWC (éq. 7 pts)</strong> du tournoi face au niveau habituel, chacun avec son intervalle à 95 % sur les matchs. Le verdict — <em>moins bien</em>, <em>mieux</em>, <em>dans l'habitude</em> — porte sur l'écart et son propre intervalle ; il n'est rendu qu'avec un intervalle et au moins 20 décisions de chaque côté, sinon il se lit <em>insuffisant</em>. Un tournoi de trois matchs dit rarement « moins bien » : les chiffres restent visibles, le verdict attend la preuve.</li>
<li><strong>Par ronde</strong> : chaque match dans l'ordre du tournoi, avec l'adversaire, son PR (intervalle sur les parties) et sa perte éq. 7 pts, face au PR habituel.</li>
<li><strong>Par rang de la décision dans le match</strong>, par tranches de 30 décisions : si la fin des matchs longs coûte plus que d'habitude, c'est la fatigue qu'il faut travailler (pauses, rythme), pas une famille de positions.</li>
<li><strong>Par score</strong> : DMP, partie Crawford, post-Crawford et autres scores, face aux mêmes scores du niveau habituel — les points où une erreur décide du match.</li>
<li><strong>Par rythme</strong> : décisions rapides ou posées, de part et d'autre de la médiane de vos durées pour ce type de décision dans le match. C'est ainsi que la pendule se lit : la réserve de temps n'est pas enregistrée coup par coup.</li>
<li><strong>Familles d'erreurs du tournoi</strong> : au plus trois familles du plan d'étude (voir Plan d'étude) restreint aux erreurs du tournoi, avec les mêmes règles (5 erreurs chiffrées, borne basse positive) ; les autres sont comptées « à confirmer ». Ce sont les séances à prévoir en premier.</li>
</ul>
<p><code>stats tournament</code> imprime le même bilan, et le serveur le sert par <code>/v1/stats.tournamentReview</code>.</p>
<h3>Diriger un tournoi</h3>
<p>blunderDB sait <strong>diriger</strong> un tournoi, et pas seulement le ranger. La direction est portée par le moteur <strong>Nicomaque</strong>, de Nicolas Harmand : c'est lui qui tient le format, les appariements, les tableaux et le classement ; blunderDB lui donne son interface et garde ses matchs. Le bouton <strong>ⓘ</strong> de l'en-tête de la Direction rappelle ce crédit et mène au dépôt et à la documentation du moteur.</p>
<p>Un tournoi dirigé se choisit dans le panneau Panneau Tournois (<em>CTRL-Y</em>, commande <code>direct</code>) : ouvrir un tournoi, puis <strong>Diriger ce tournoi</strong>. Un tournoi déjà dirigé porte son état à côté de son nom et le bouton devient <strong>Ouvrir la direction</strong>. Tant qu'une direction est ouverte, la zone principale montre le tournoi <strong>à la place du plateau</strong> — c'est la seule exception de blunderDB à cette règle ; passer sur n'importe quel autre onglet ramène le plateau. <strong>Quitter la direction</strong>, dans l'en-tête de la vue, ou <strong>Fermer la direction</strong>, dans le panneau, la ferment.</p>
<p>Une direction a trois états : <strong>en préparation</strong> tant qu'aucun match n'a été lancé, <strong>en cours</strong> ensuite, <strong>terminé</strong> une fois le tournoi clos et le classement figé. Rouvrir un tournoi clos est possible, et demande une confirmation : le classement final cesse d'être final.</p>
<p>Tout ce qui est décidé est écrit dans un <strong>journal</strong>, et rien d'autre ne l'est. Le classement, les arbres, les propositions et les avertissements sont rejoués depuis ce journal à chaque ouverture : une coupure de courant ne coûte rien, et une correction n'efface jamais ce qui s'est passé — elle s'ajoute.</p>
<h4>La page Direction</h4>
<p>C'est là que le directeur passe l'essentiel de son temps. Elle porte, de haut en bas : les <strong>avertissements</strong> du moteur, qui restent visibles et ne bloquent jamais rien ; la <strong>grille des tables</strong>, dont l'en-tête porte le bouton <strong>Imprimer la feuille</strong> d'appariements ; la <strong>dernière décision</strong> ; la <strong>file des propositions</strong> ; et les joueurs libres. La grille précède la file : une file longue ne la pousse jamais hors de l'écran. Au clavier, le premier arrêt de <em>TAB</em> dans la page est « Aller à la file », qui pose le focus sur la file sans traverser la grille ; la file rappelle sous son titre ses raccourcis (<em>J</em> et <em>K</em>, <em>ENTRÉE</em>, <em>GAUCHE</em> et <em>DROITE</em>).</p>
<p>La vue occupe toute la largeur de la zone principale, et chaque onglet défile seul : quitter un onglet puis y revenir, ou passer d'une épreuve à l'autre, retrouve la position où on l'avait laissé. Les boutons et les champs font au moins 44 pixels de haut, pour se viser sans précision au comptoir ; le nombre de colonnes de la grille suit la largeur de la zone, non celle de la fenêtre. Dans les <strong>Réglages</strong>, chaque section se replie sur son titre, et le bouton <strong>Ouvrir dans le navigateur</strong> des Réglages et le bouton <strong>Page murale</strong> de l'en-tête ouvrent la page murale en un clic dès qu'un dossier de sortie est choisi ; la barre d'état donne le chemin du fichier écrit. Inscription, retardataire, phase suivante, tirage, lancement d'un match et clôture y laissent aussi un retour (« Sophie Martin inscrit — 16 inscrits »).</p>
<p>Une Direction s'ouvre sur ses <strong>Joueurs</strong> tant que le tournoi n'est pas lancé, sur l'onglet <strong>Direction</strong> ensuite, et sur l'onglet où on l'avait laissée quand on la rouvre. Après un rechargement ou une relance de l'application, la dernière Direction ouverte se rouvre d'elle-même, sur le même onglet. Un second clic sur <em>Diriger</em> ou <em>Ouvrir la direction</em> pendant que la vue se charge est ignoré.</p>
<p>Une ronde proposée s'annonce avant d'être lancée : <strong>Ronde à venir…</strong>, à côté de <em>Imprimer la feuille</em>, demande la date et l'heure à imprimer (« lundi 21/09, 20 h ») et imprime la feuille des appariements de la file, marquée « annoncée ». Rien n'est lancé ni écrit au journal : la ronde se lance le jour venu, à son heure. Un appariement qui attend une table libre porte un tiret à la place du numéro de table.</p>
<p>En tête de la vue tournoi, la <strong>bande d'horloge</strong> tient en une ligne : l'heure, le temps écoulé depuis le premier match lancé, les matchs joués et en cours, le rythme observé en minutes par point face au rythme prévu, les matchs lents, la prochaine pause et la <strong>fin estimée</strong>. La fin estimée est la prévision du moteur : il rejoue le journal, termine le tournoi quinze fois au rythme prévu, et la bande en donne la médiane, repoussée après les pauses déclarées. Une nuit qui n'est pas déclarée comme pause compte donc comme du jeu. Une heure qui n'est pas celle du jour porte son jour.</p>
<p>À partir du deuxième jour, le temps écoulé laisse la place au <strong>jour de jeu</strong> (le jour du premier match lancé est le jour 1) et au <strong>temps de jeu</strong> : le temps pendant lequel au moins un match était en cours, sans les nuits ni les intervalles où aucune table ne jouait. Un tournoi clos n'a plus de bande d'horloge.</p>
<p>Une proposition se confirme d'<strong>un clic</strong> sur <em>Lancer</em>. <strong>Tout lancer</strong> confirme en deux clics les propositions qui ont une table, après en avoir montré la liste ; <em>Confirmer</em> est en tête de cette liste. Les appariements sans table restent dans la file, marqués « aucune table libre » : en mode rondes, une ronde reste ouverte tant que tous ses joueurs n'y sont pas engagés. Un repêchage entre ex æquo attend aussi le choix du directeur, et le passage de phase avec lui. « Ignorer pour l'instant » n'écrit rien : le moteur est déterministe, et la proposition revient identique au prochain appel. <em>Apparier à la main</em> reste offert en permanence — le moteur propose, le directeur décide.</p>
<p>Un match apparié à la main sans numéro de table prend la première table libre ; une table où un match est en cours est refusée. Si toutes les tables sont prises, il est lancé quand même et sa case apparaît au bout de la grille, « sans table », jusqu'à ce qu'on le déplace sur une table.</p>
<p>Une proposition peut porter une remarque du moteur : aucune table libre, ou une fin de match attendue pendant une pause. Elle reste lançable dans les deux cas. Lorsqu'une phase fonctionne par <strong>micro-rondes</strong>, la file montre le temps qui reste avant le prochain lot ; à l'échéance les propositions apparaissent d' elles-mêmes, et rien ne se lance tout seul.</p>
<h4>La fiche de résultat</h4>
<p>Un clic sur une table occupée ouvre la fiche du match. Elle montre deux grandes cibles : les <strong>noms des deux joueurs</strong>. Cliquer celui qui a gagné enregistre le résultat — deux clics en tout, table comprise. Le vainqueur est la seule chose exigée ; le score est libre, l'un et l'autre ou aucun des deux. Au clavier, <em>GAUCHE</em> ou <em>DROITE</em> choisit le vainqueur et <em>ENTRÉE</em> l'enregistre. La fiche ne se ferme qu'une fois le résultat écrit : un échec la laisse ouverte, avec son message.</p>
<p>Le bouton <strong>⋯</strong> de la fiche déplie ce qui sert rarement : le forfait — chaque bouton nomme l'absent et celui qui gagne —, une remarque libre (« tombé au temps », « abandonné pour raison de… »), le déplacement du match sur une autre table, et son annulation. Le forfait et l'annulation se confirment ; le forfait par un bouton <strong>Déclarer forfait</strong>, qui propose aussi de retirer le perdant dans la même fiche. Déplacé sur une table occupée, le match échange sa table avec celui qui l'occupe : deux matchs ne partagent jamais une table, et le même geste les remet en place. Si un ancien journal en a laissé deux sur une table, la grille montre les deux cases, signalées, jusqu'à ce qu'on en déplace un.</p>
<p>Une erreur de saisie vue aussitôt se reprend en deux clics sous la grille : <strong>Corriger</strong> la dernière décision, puis le bon vainqueur (<em>CTRL-Z</em> ouvre la même reprise). Une correction plus ancienne se fait depuis l'historique.</p>
<h4>Les menus contextuels</h4>
<p>Un clic droit, la touche <em>MENU</em> ou <em>MAJ-F10</em> sur un objet de la page Direction ouvre ses actions courantes sans passer par la fiche : une case de la grille (libre ou occupée), un joueur (onglet <strong>Joueurs</strong>, joueurs libres), une place de l'arbre, un emplacement, une proposition de la file, une ligne de l'historique. Le menu s'ouvre sur l'objet ; <em>HAUT</em> et <em>BAS</em> le parcourent, <em>ENTRÉE</em> choisit, <em>ÉCHAP</em> le ferme et rend le focus à l'objet.</p>
<ul>
<li>Case occupée : saisir le résultat, forfait de l'un ou de l'autre, changer de table (viser une table occupée échange les deux matchs), annuler le match, historique de chaque joueur.</li>
<li>Case libre : lancer ici la proposition sélectionnée, mettre la table hors service, ou la remettre en service (tables d'un événement). Une table réservée à une autre épreuve ne propose rien.</li>
<li>Joueur : saisir le résultat de son match en cours, aller à sa table, historique, apparier à la main avec un autre joueur libre, aller dans l'autre épreuve où il joue aussi, marquer absent ou présent, retirer maintenant ou après son match, réinscrire, corriger la fiche.</li>
<li>Emplacement sans match : rattacher un match importé que cette place attend.</li>
<li>Proposition : lancer, lancer à la table…, changer la longueur…, apparier autrement (ces trois entrées ouvrent l'appariement à la main avec les deux joueurs, la longueur et la table de la proposition), ignorer pour l'instant, imprimer la feuille de la ronde.</li>
<li>Ligne d'historique : corriger ou annuler, ajouter une remarque, filtrer sur un des joueurs.</li>
</ul>
<p>Le forfait, l'annulation d'un match et le retrait d'un joueur gardent la confirmation qu'ils ont dans la fiche et dans les boutons des lignes. Pendant qu'une action est en cours, les entrées qui agissent sont grisées, comme les boutons. Un seul menu est ouvert à la fois : en ouvrir un second ferme le premier.</p>
<p>Au clavier, la grille ne prend qu'un arrêt de <em>TAB</em> : chaque case se focalise, libre comprise, et <em>GAUCHE</em>, <em>DROITE</em>, <em>HAUT</em>, <em>BAS</em>, <em>DÉBUT</em> et <em>FIN</em> passent d'une case à l'autre. Un chiffre ouvre la fiche de la table de ce numéro ; pour une table au-delà de 9, le second chiffre se tape dans les 0,4 s. <em>M</em> ouvre la fiche sur le champ de table, et <em>X</em> aussi, que la fiche soit déjà ouverte ou non : viser une table occupée échange les deux matchs.</p>
<p>À la souris, on <strong>glisse</strong> une case occupée sur une autre : sur une case libre le match change de table ; sur une case occupée, une ligne « Table 3 ↔ Table 7 ? » demande de confirmer l'<strong>échange</strong> des deux matchs. Un fantôme suit le pointeur et la case visée s'entoure ; <em>ÉCHAP</em> annule le geste, et rien n'est écrit tant que le pointeur n'est pas relâché sur une case. Une table hors service est refusée, et la barre d'état en donne le motif. En ligne de commande, <code>blunderdb tournament move</code> fait le même déplacement ou le même échange (voir Interface en ligne de commande (CLI)).</p>
<h4>Le plein écran</h4>
<p>La touche <em>F11</em> de la page Direction, ou le bouton en bas à droite, la met en plein écran : la barre d'outils, les onglets, le plateau, le panneau et la barre d'état disparaissent, et toute la fenêtre est rendue à la direction. Le mode traverse les onglets de la direction (Direction, Joueurs, Historique, …). Un menu ou une fiche ouverts se ferment d'abord sur <em>ÉCHAP</em> ; un second <em>ÉCHAP</em>, ou <em>F11</em>, sort du plein écran et rend la fenêtre à son état précédent. Quitter la page, en changeant d'onglet de l'application ou en fermant la direction, y met fin aussi.</p>
<h4>La recherche rapide</h4>
<p>La touche <em>/</em> de la page Direction ouvre la palette sur le tournoi seul : joueurs, tables, matchs en cours et épreuves de l'événement ouvert, ou de l'épreuve seule quand elle n'est dans aucun événement. On tape un nom, un club ou un numéro de table (« 4 » ou « t4 ») ; les joueurs à une table passent avant les joueurs libres. <em>ENTRÉE</em> amène à l'objet : un joueur en cours de match, un match ou une table occupée ouvrent la fiche de la table, une table libre se focalise dans la grille, un joueur libre s'affiche dans l'onglet <strong>Joueurs</strong> filtré sur son nom, une épreuve devient l'onglet courant. Un résultat d'une autre épreuve de l'événement change d'abord d'épreuve. <em>ÉCHAP</em> referme sans rien ouvrir, et la touche reste une barre oblique dans un champ de saisie. <em>CTRL-MAJ-P</em> ouvre la palette complète, qui contient aussi le tournoi.</p>
<h4>Les joueurs</h4>
<p>L'onglet <strong>Joueurs</strong> inscrit, corrige et retire. Le champ d'inscription garde le focus et se vide après chaque nom : vingt joueurs s'inscrivent au clavier seul. L'autocomplétion propose les joueurs de la base ; en choisir un fixe l'orthographe exacte que portent ses matchs et pré-remplit sa cote avec son PR.</p>
<p>L'<strong>annuaire</strong> regroupe les inscrits de tous les tournois dirigés de la base, dédoublonnés par nom, avec le club et la cote de leur dernière inscription. Il n'est jamais stocké : supprimer une direction en retire ses inscrits. Reprendre les inscrits d'un tournoi précédent est un clic, quel que soit leur nombre ; l'annuaire se copie en CSV ou s'enregistre dans un fichier (<strong>Enregistrer…</strong>), et se relit collé.</p>
<p>Une épreuve en doubles inscrit des paires : la case <strong>Paire</strong> ajoute le nom, le club et la cote du partenaire. La paire joue sous le nom « A / B », que portent aussi ses matchs ; sa cote est la moyenne des deux, et une valeur saisie dans <strong>Cote de la paire</strong> la remplace. L'annuaire garde les deux personnes, jamais la paire.</p>
<p>Avant <strong>Inscrire</strong>, l'aperçu d'un CSV collé liste les lignes illisibles — sans nom, sans séparateur quand les autres lignes en ont, cote qui n'est pas un nombre — et les doublons, dans le collage ou avec un joueur déjà inscrit. Un doublon n'est pas inscrit, sauf si on coche sa case.</p>
<p>Le champ de filtre de la liste garde son texte quand on change d'épreuve : tant qu'il est actif, une puce <strong>filtre : …</strong> s'affiche à côté, et sa croix l'efface.</p>
<p>Un <strong>retardataire</strong> arrivé après le tirage prend une place d'exemption libre si le tableau en offre une, et l'interface écrit à côté du champ où il entrera avant qu'on valide. Sans place libre, il est inscrit quand même et la vue dit dans quelle phase il entrera. Aucun tirage déjà fait n'est refait.</p>
<p>Un retrait se fait <em>maintenant</em> ou <em>après son match en cours</em>, selon que le joueur part tout de suite ou finit ce qu'il joue ; il se confirme par un bouton <strong>Retirer</strong>, non rouge puisque rien n'est supprimé.</p>
<p>Un joueur qui manque une ronde n'a pas besoin d'être retiré : <strong>Marquer absent</strong>, sur sa ligne, ouvre un petit formulaire sous son nom — <em>jusqu'à</em> une heure (pré-remplie sur l'heure qui suit), ou, quand la phase en cours est un suisse par rondes, <em>jusqu'à la ronde</em> portant son numéro. Le moteur cesse alors de l'apparier, mais son rang, ses vies et sa place au tableau restent ceux qu'il a gagnés — l'absence n'est pas un forfait. <strong>Revenir</strong>, sur sa ligne, lève l'absence en un clic, avant l'échéance déclarée ou après.</p>
<p>Corriger la fiche d'un joueur retiré — son nom, son club, sa cote — le laisse retiré. Son retour est un geste à part : <strong>Réinscrire</strong>, sur sa ligne. Il est de nouveau apparié, avec les résultats et les vies qu'il avait en partant ; les matchs perdus par forfait à son retrait le restent.</p>
<p>Un qualifié de poule qui se retire avant le tirage de la phase suivante laisse une place. La file propose alors un <strong>repêchage</strong> : le suivant de sa poule, ni qualifié ni retiré, au plus grand nombre de victoires de poule, prend sa place. Entre ex æquo, la file propose chacun d'eux et le directeur choisit ; « Phase suivante » sans repêchage laisse la place en exemption. Une fois le tableau tiré, le retiré perd son match par forfait.</p>
<h4>Arbres, emplacements, classement, historique</h4>
<p>L'onglet <strong>Arbres</strong> dessine les tableaux avec leurs traits, de la première ronde à la finale, la consolante à côté du tableau principal, et, pour un suisse, le tableau des vies. Une poule s'y lit en résultats croisés. Un tableau pas encore tiré montre son squelette grisé. Un match déjà joué y porte son résultat ; un match que le moteur signale y est marqué sur place. Une pastille à côté du nom de l'onglet indique qu'un tableau est en cours.</p>
<p><strong>Cliquer une place</strong> (ou Entrée sur une place focalisée) ouvre sur elle la même fiche que sur la grille des tables : le vainqueur d'un match en cours se saisit en deux clics, et un match terminé se corrige en cliquant le nom du vrai vainqueur.</p>
<p>L'onglet <strong>Matchs</strong> (titré « Emplacements ») relie le tournoi à la bibliothèque. Chaque match du tournoi est un emplacement, que l'on peut remplir de deux façons : transcrire le match sur-le-champ (Panneau Transcription), ou y rattacher un match déjà importé. <strong>Rien n'est rattaché par déduction</strong> : une coïncidence de noms est une suggestion à accepter, un appariement partiel n'est même pas suggéré, et si le fichier d'un match rattaché contredit le résultat enregistré, l'écart est montré sans être résolu — pendant un tournoi, la parole du directeur fait foi.</p>
<p>L'onglet <strong>Classement</strong> montre le classement courant, section par section, avec le bilan de chacun (victoires–défaites) et les prix lorsqu'une dotation est réglée. Deux ex æquo partagent la place et le prix. Un joueur retiré garde le rang que son parcours lui vaut, noté « retiré » avec son bilan ou l'endroit du tableau où il s'est arrêté. <strong>Clore le tournoi</strong> fige le classement final. Le classement se copie en CSV, dans la langue de l'interface, avec la section et la dernière phase où chacun est entré, ou s'enregistre dans un fichier : <strong>Enregistrer…</strong> ouvre le dialogue du système sur un nom proposé, le tournoi suivi du mot « classement » et de la date du jour, et le fichier contient exactement le CSV copié. En ligne de commande, <code>blunderdb tournament standings</code> écrit le même CSV. Le nom d'un joueur est un lien : il ouvre l'Historique filtré sur lui, où chacun de ses résultats se corrige.</p>
<p>Clore sans match en cours est un clic ; quand tout est joué, la file propose aussi <strong>Clore le tournoi</strong>, qui se lance comme une autre proposition. Avec des matchs en cours, le Classement dit combien et attend un second clic sur place : clore fige le classement sans eux, et leur résultat ne se saisit plus. <strong>Rouvrir</strong> se confirme de la même façon ; le classement final cesse alors d'être final, et la réouverture reste dans le journal.</p>
<p>L'onglet <strong>Historique</strong> est le journal en clair : une ligne par décision, dans l'ordre, filtrable par joueur ou par match. C'est ce qu'un directeur relit après une contestation, et c'est là qu'une décision ancienne se corrige ou s'annote. Une ligne de résultat nomme les deux joueurs : le vainqueur, puis son adversaire.</p>
<p><strong>Corriger</strong>, sur la ligne d'un résultat, ouvre sous elle la même reprise que la dernière décision : cliquer le bon vainqueur, avec le score s'il faut. Le résultat d'origine reste à sa place dans le journal, la correction s'y ajoute, et le classement en tient compte aussitôt.</p>
<h4>Les réglages</h4>
<p>L'onglet <strong>Réglages</strong> s'ouvre sur des <strong>formats nommés</strong> : six tournois de club prêts à l'emploi, dont le premier est recommandé. En choisir un suffit à commencer ; les champs restent modifiables ensuite.</p>
<p>Se règlent ici : les phases (<strong>Ajouter une phase</strong> l'ajoute après les autres ; une phase déjà ouverte ne se retire pas) et leur longueur de match, celle de la finale, les longueurs tour par tour d'un tableau (« 15, 13, 11 » se lit du dernier tour vers le premier), la <strong>longueur de fin</strong> (des matchs plus longs à partir d'un nombre de joueurs encore en vie), les <strong>micro-rondes</strong> (apparier par lots toutes les N minutes), le nombre de tables, le rythme prévu en minutes par point (8 par défaut ; la bande d'horloge et la fin estimée en partent), les pauses de la journée, la dotation (droit d'entrée, retenue du club, barème par section) et le dossier d'affichage.</p>
<p>Un <strong>tableau</strong> a trois cases : <strong>Consolante</strong> (ses perdants jouent un second tableau, section à part au classement), <strong>Réconciliation</strong> (le vainqueur de la consolante joue celui du tableau principal ; la case n'apparaît qu'avec la consolante) et <strong>Recharge</strong> (en double élimination, le vainqueur du tableau principal doit être battu deux fois ; la case n'apparaît qu'avec la réconciliation). Une consolante n'a de classement à elle qu'avec un barème : tant que celui de la section <em>Consolante</em> est vide, les Réglages le rappellent. Une phase en <strong>Poules</strong> se règle par la taille des poules (4 par défaut) et le nombre de qualifiés par poule (2 par défaut).</p>
<p>La <strong>bascule</strong> d'une phase suisse est la somme des vies restantes à partir de laquelle on passe au tableau. Si la somme des vies de départ (joueurs × vies) est déjà inférieure ou égale à la bascule, la suisse serait sautée : les Réglages le signalent avant le premier lancement.</p>
<p>Les réglages restent accessibles <strong>en cours de tournoi</strong> : baisser la bascule à 22 h pour finir plus tôt, ajouter une consolante le samedi soir, tant que le tirage du tableau n'est pas fait. Ce qui est alors figé est grisé avec sa raison : le format d'une phase ouverte, le nombre de vies qu'elle a distribué et, dès le tirage d'une phase, la taille de ses poules et son nombre de qualifiés. Enregistrer en cours de tournoi montre d'abord la liste de ce qui va changer, et demande confirmation. Ce que le moteur refuse y figure avec sa raison, et rien n'est alors enregistré : c'est le cas de la consolante, de la réconciliation ou de la recharge d'un tableau déjà tiré.</p>
<p>Une table au plateau cassé se déclare dans <strong>Tables hors service</strong> : ses numéros, séparés par des virgules (« 7, 12 »). Le moteur ne l'attribue plus, et la grille la montre indisponible ; baisser le nombre de tables retirerait la dernière, pas la cassée. Si un match est en cours sur une table mise hors service, la liste de ce qui va changer le dit et nomme une table libre où le déplacer, depuis la fiche du match.</p>
<p>Les <strong>têtes de série</strong> sont une option, éteinte par défaut : l'étude du moteur conclut « pas de têtes de série protégées », qui est la culture actuelle du backgammon. Activées, les joueurs sont placés par cote.</p>
<h4>Les épreuves d'un événement</h4>
<p>Plusieurs épreuves jouées sur les mêmes tables — un principal, un speed, des doubles — se regroupent dans un <strong>Événement</strong>, en tête des Réglages (replié sur son titre quand l'épreuve y est rattachée ; le bouton <strong>Réglages de l'événement</strong> de l'onglet <em>Toutes les tables</em> y mène) : <strong>Créer et rattacher…</strong> crée l'événement avec son nombre de tables, <strong>Rattacher…</strong> y ajoute une épreuve dirigée. Rattacher montre d'abord ce qui va changer : les tables de l'épreuve deviennent celles de l'événement. <strong>Détacher de l'événement</strong> rend l'épreuve à elle-même, avec son journal et ses tables ; <strong>Supprimer l'événement</strong>, après confirmation, le met à la corbeille et détache ses épreuves sans en supprimer aucune.</p>
<p>Dans un événement, aucune épreuve ne propose une table où joue une autre : la grille montre ces tables occupées, avec le nom de l'épreuve, et un appariement sans table libre attend. Une table hors service se coche une fois, dans l'événement, et vaut pour toutes ses épreuves ; le nombre de tables, les tables hors service et les pauses modifiés dans les Réglages d'une épreuve valent aussi pour l'événement, et la liste de ce qui va changer nomme les autres épreuves.</p>
<p>Les <strong>propriétés des tables</strong> se règlent dans le panneau Événement : un tableau d'une ligne par table, avec son <strong>nom</strong> (« Stream », par exemple), sa <strong>salle</strong> (un libellé libre), une case <strong>Réservée</strong> et les joueurs auxquels elle est <strong>Attitrée</strong>, choisis parmi les inscrits des épreuves de l'événement. Pour quarante tables, <em>Tables de N à M, salle</em> pose une salle sur toute une plage d'un geste ; <strong>Enregistrer</strong> n'écrit que les tables qui portent une propriété, les autres restent des tables ordinaires. Une table réservée n'est jamais proposée, mais on peut y placer un match à la main, par un lancement, un déplacement ou un glisser-déposer. Une table attitrée reçoit en premier le match de son titulaire, quand elle est libre ; sinon le match reçoit une table ordinaire, et hors des matchs de ses titulaires elle se comporte comme une table réservée. Deux titulaires de tables différentes qui se rencontrent jouent sur la plus petite des deux.</p>
<p>Une <strong>salle</strong> est l'ensemble des tables qui portent le même libellé : « salle A » pour les tables 1 à 20, « salle B » pour les suivantes. Dans les Réglages de chaque épreuve rattachée, <em>Salles où joue cette épreuve</em> coche les salles où elle joue : le DMP en B, le speed en A. Aucune case cochée, c'est toutes les tables ; une épreuve ne reçoit aucune proposition hors de ses salles et ne peut pas y déplacer un match. Retirer une salle qui porte un match en cours de l'épreuve est refusé, en nommant la table. Une épreuve qui joue seule règle les mêmes propriétés de table dans ses propres Réglages, sans salles d'épreuve.</p>
<p>Le <strong>Classement de saison</strong>, dans le panneau Événement, cumule les épreuves closes de l'événement : chaque place rapporte les points du <strong>Barème</strong> (vainqueur en tête, 25, 18, 15, 12, 10, 8, 6, 4, 2, 1 par défaut), et des ex æquo se partagent la moyenne des places qu'ils occupent. Une personne est reconnue d'une épreuve à l'autre par son nom. <strong>Elo de club</strong> ajoute une colonne : chacun part de 1500 et les matchs de la saison sont rejoués dans l'ordre, selon la formule de FIBS. <strong>Calculer</strong> affiche le classement, <strong>Copier en CSV</strong> le copie avec une colonne de points par épreuve. Une épreuve non close ne rapporte rien.</p>
<p>Ouvrir la Direction d'une épreuve d'un événement ouvre aussi les autres : un onglet par épreuve apparaît en haut de la Direction, chacun avec son résumé — propositions en attente, matchs en cours, une alerte s'il y en a. Changer d'épreuve est un clic sur son onglet, sans confirmation ; l'épreuve quittée ne se ferme pas et ne rejoue rien, elle reste telle qu'on l'a laissée. Un tournoi hors de tout événement n'a qu'une épreuve : pas d'onglet à montrer. Changer d'onglet de l'application puis revenir à Tournois rend la Direction telle qu'on l'a quittée : la même épreuve ou <em>Toutes les tables</em>, et le même onglet de la vue.</p>
<p>L'onglet <strong>Toutes les tables</strong>, à gauche des épreuves, montre toutes les tables de l'événement en une seule grille : une case par table, quelle que soit l'épreuve qui l'occupe, marquée du nom et de la couleur de son épreuve. La fiche de résultat, les menus contextuels, le clavier et le glisser-déposer y fonctionnent comme dans la grille d'une épreuve, et chaque geste s'adresse à l'épreuve de la case. Glisser un match sur une table occupée par une autre épreuve échange les deux matchs après une confirmation qui nomme les deux épreuves. Sous la grille, les propositions de toutes les épreuves forment une seule file, chacune marquée de son épreuve et munie de son bouton <em>Lancer</em> : les joueurs qui attendent depuis le plus longtemps passent d'abord, si bien qu'une épreuve n'attend pas la fin de la ronde d'une autre. Un match sans table — toutes les tables prises, ou un joueur retenu par un match d'une autre épreuve — suit la file avec sa raison, sans bouton <em>Lancer</em>, jusqu'à ce qu'une table ou le joueur se libère. Cliquer sur une épreuve ou sur un onglet de vue quitte <em>Toutes les tables</em>.</p>
<p>Quand l'événement a plusieurs salles, la grille se groupe par salle, sous le nom de chacune. Le nom d'une table s'affiche à côté de son numéro, avec un drapeau pour une table réservée et une étoile suivie des titulaires pour une table attitrée ; les propositions nomment aussi la table.</p>
<p>Une même personne peut jouer plusieurs épreuves de l'événement : deux Participants du même nom sont la même personne, et pour une paire de doubles chacun des deux membres compte. Tant qu'elle joue dans une épreuve, les autres ne la proposent pas, et leur liste <em>En attente</em> dit où elle joue : « joue au principal, table 4 ». Un match de tableau qui l'attend reste dans la file, et le lancer est refusé tant qu'elle joue. L'appariement à la main reste permis : le match démarre, et sa case de la grille porte la même mention.</p>
<h4>L'affichage du tournoi</h4>
<p>Un tournoi se regarde. Choisir un <strong>dossier d'affichage</strong> dans les Réglages suffit une fois pour toutes : blunderDB y réécrit une page HTML autonome à chaque changement, et la page se recharge d'elle-même. Elle s'ouvre hors ligne, sur un second écran ou projetée, et ne charge aucune ressource extérieure. <em>Ouvrir dans le navigateur</em> l'affiche immédiatement.</p>
<p>La <strong>feuille d'appariements</strong> se pose sur la table d'accueil : un clic sur <em>Imprimer la feuille</em> ouvre le dialogue d'impression du système. Une ligne par match — les deux joueurs, la longueur, la table, deux cases vides pour le score — et une ronde de trente-deux joueurs tient sur une page A4.</p>
<p>Un événement a son propre dossier de sortie, choisi une fois dans son panneau de Réglages avec le même bouton <em>Choisir un dossier</em> : blunderDB y écrit <code>index.html</code>, la <strong>page murale</strong> de l'événement — une ligne par table, quelle que soit l'épreuve qui l'occupe, avec les rondes annoncées de chaque épreuve et un lien vers sa propre page — et chaque épreuve rattachée écrit la sienne dans un sous-dossier. Un geste dans n'importe quelle épreuve de l'événement régénère la page murale ; le dossier propre d'une épreuve rattachée est gardé mais ignoré tant qu'elle reste dans l'événement.</p>
<p>Quand une épreuve est en phase de tableau et que celui-ci est tiré, sa page murale et celle de l'événement montrent l'arbre en grand, lisible de loin : une colonne par tour, les perdants qui descendent en consolante en pointillé. La page défile d'elle-même, sans script, entre son contenu habituel (les tables pour l'événement) et l'arbre de chaque épreuve en tableau, douze secondes chacun ; elle se recharge toujours toutes les trente secondes et reprend la rotation là où elle en était. Une épreuve sans tableau — une phase suisse, par exemple — n'a pas d'arbre et la page reste celle d'avant.</p>
<p>Sous la rubrique « Est-ce que je joue ? », la page d'une épreuve et la page murale de l'événement nomment les joueurs qui ne jouent pas en ce moment, tels que le moteur les situe : « éliminé(e) » dès qu'il ne reste plus de match à jouer, « qualifié(e) » avec le nom de la phase où le joueur entre, « pas encore fixé » quand son sort dépend de la fin de la phase ou d'un repêchage que le directeur n'a pas tranché, « exempté(e) — entre au tour N » pour un joueur tiré sans adversaire dans le tableau, tant qu'il n'a pas joué, « exempté(e) — rejoue à la ronde N » pour l'exempt d'une ronde suisse, et « vainqueur ». Le classement prend le relais une fois le tournoi terminé.</p>
<p>Hors de l'interface, la sous-commande <code>blunderdb tournament</code> relit un tournoi dirigé sans interface graphique : <code>list</code>, <code>verify</code>, <code>standings</code>, <code>page</code> et <code>export</code> ; <code>proposals</code> affiche la file du moteur, numérotée, et <code>confirm</code> en confirme une, repêchage compris ; <code>ranking --season</code> cumule les tournois clos d'un événement ou d'une période en un classement de saison, par un barème de points par place et, au choix, un Elo de club (deux inscrits de même nom dans une même épreuve close font refuser le classement, qui connaît une personne par son nom) ; <code>page --rencontre</code> écrit la page murale d'un événement au lieu de la page d'une seule épreuve ; <code>hall</code> imprime la grille <em>Toutes les tables</em> d'un événement et sa file, <code>tables</code> les propriétés des tables et les salles des épreuves. Voir Interface en ligne de commande (CLI).</p>
<h3>Panneau Stats</h3>
<h4>Introduction</h4>
<p>Le panneau <strong>Stats</strong> permet d'analyser son niveau de jeu et de suivre sa progression dans le temps à partir des positions importées dans la base de données. Il calcule et affiche les indicateurs <strong>PR</strong> (<em>Performance Rating</em>) et <strong>MWC cost</strong> (Match Winning Chance cost) pour l'ensemble des positions ou un sous-ensemble filtré.</p>
<p>Le panneau Stats est particulièrement utile pour :</p>
<ul>
<li><strong>situer son niveau</strong> par rapport aux bandes de niveau (<em>Classe mondiale</em>, <em>Expert</em>, <em>Avancé</em>…) grâce au PR global ;</li>
<li><strong>suivre sa progression</strong> tournoi après tournoi ou match après match grâce aux graphiques de l'onglet Progression ;</li>
<li><strong>identifier ses points faibles</strong> : onglet Erreurs pour voir la répartition entre coups joués et décisions de videau, et la distribution des magnitudes d'erreur ;</li>
<li><strong>comparer les joueurs de la base</strong> entre eux, une ligne par joueur, grâce à l'onglet Joueurs — utile pour suivre une compétition entière ;</li>
<li><strong>accéder directement aux positions concernées</strong> en cliquant sur n'importe quel indicateur (drill-down).</li>
</ul>
<p>La définition, la formule, l'incertitude et les limites de chaque chiffre du panneau sont réunies dans Métriques et outils d'étude.</p>
<h4>Ouverture du panneau</h4>
<p>Pour ouvrir le panneau Stats :</p>
<ul>
<li>Appuyer sur <em>CTRL-D</em>.</li>
<li>Saisir la commande <code>stats</code> ou <code>st</code> dans la ligne de commande.</li>
</ul>
<div class="admonition note">
<p>Le panneau se rafraîchit automatiquement à chaque modification du filtre. Il ne recalcule pas les statistiques lors d'un simple basculement PR ↔ MWC : les deux métriques sont calculées simultanément par le backend.</p>
</div>
<h4>Barre de filtre</h4>
<p>La barre de filtre, en haut du panneau, permet de restreindre le calcul à un sous-ensemble de positions.</p>
<h5>Perspective joueur</h5>
<p>La liste déroulante <strong>Joueur</strong> permet de filtrer les statistiques selon le joueur analysé. blunderDB sélectionne automatiquement le joueur dont le nom apparaît le plus souvent dans la base de données — modifiable à tout moment.</p>
<div class="admonition tip">
<p>Changer de joueur ne provoque pas de perte de données ; il suffit de re-sélectionner le joueur précédent dans la liste.</p>
</div>
<h5>Filtres disponibles</h5>
<ul>
<li><strong>Tournoi(s)</strong> — restriction à un ou plusieurs tournois. Plusieurs tournois peuvent être sélectionnés simultanément.</li>
<li><strong>Dates</strong> — plage temporelle (<em>De</em> … <em>À</em>). Si seule la date de début est renseignée, les positions plus récentes sont incluses.</li>
<li><strong>Type de décision</strong> — Tous / Coups joués / Décisions de videau.</li>
<li><strong>Longueur de match</strong> — restriction à des longueurs de match précises (1, 3, 5, 7, 9, 11, 13, 15, 21 points). Plusieurs longueurs peuvent être combinées.</li>
<li><strong>Moteur</strong> et <strong>Profondeur min.</strong> — ne garder que les décisions analysées par un moteur donné (gnubg, xg…), ou au moins à une profondeur donnée, en plis.</li>
</ul>
<p>Un bouton <strong>↺ Réinitialiser</strong> remet tous les filtres à zéro (sauf le joueur auto-détecté).</p>
<div class="admonition note">
<p>Les filtres sont persistés dans la configuration de blunderDB (<code>config.yaml</code>) et sont restaurés à la prochaine ouverture.</p>
</div>
<h4>Toggle PR / MWC</h4>
<p>Le bouton <strong>PR / MWC</strong> en haut du panneau bascule la métrique affichée dans tous les onglets.</p>
<p><strong>PR (Performance Rating)</strong></p>
<blockquote>
<p>L'erreur moyenne d'équité par décision comptée, multipliée par 500 comme le font eXtreme Gammon et GNUbg : un PR de 5,0 vaut 0,010 d'équité perdue par décision, soit 10 millipoints (mpt). La règle de comptage exacte — quelles décisions entrent au dénominateur, comment le score est converti — est celle de PR (<em>Performance Rating</em>).</p>
<p>Les bandes de niveau que le panneau dessine derrière la courbe de progression sont un <strong>repère indicatif propre à blunderDB</strong> : aucune publication ne fait autorité sur ces seuils. La borne haute de chaque bande est exclue : un PR de 4 est <em>Avancé</em>, pas <em>Expert</em>.</p>
<table>
<thead>
<tr>
<th>Niveau</th>
<th>PR</th>
</tr>
</thead>
<tbody>
<tr>
<td>Classe mondiale</td>
<td>&lt; 2</td>
</tr>
<tr>
<td>Expert</td>
<td>2 – 4</td>
</tr>
<tr>
<td>Avancé</td>
<td>4 – 6</td>
</tr>
<tr>
<td>Intermédiaire</td>
<td>6 – 9</td>
</tr>
<tr>
<td>Occasionnel</td>
<td>9 – 12</td>
</tr>
<tr>
<td>Débutant</td>
<td>≥ 12</td>
</tr>
</tbody>
</table>
</blockquote>
<p><strong>MWC cost (Match Winning Chance cost)</strong></p>
<blockquote>
<p>Probabilité cumulée de victoire de match perdue à cause des erreurs, sur l'ensemble du jeu de données filtré. Calculé à partir de la MET courante de la base, par défaut la Kazaross-XG2 embarquée dans blunderDB. Une analyse au score de match calculée avec une autre table est « MET différente » et reste hors des statistiques.</p>
<div class="admonition caution">
<p>Le MWC cost <strong>n'est pas applicable</strong> aux positions <em>money-game</em> (sans enjeu de match). Ces positions sont exclues du calcul MWC. Les valeurs MWC dépendent de la MET utilisée ; elles ne sont pas directement comparables entre logiciels utilisant des METs différentes.</p>
</div>
</blockquote>
<p><strong>Perte MWC (éq. 7 pts)</strong></p>
<blockquote>
<p>La probabilité de gagner le match qu'un joueur a perdue sur l'ensemble de ses décisions — pions, videau, prise ou refus, avec videau — ramenée à un match en 7 points. Pour un match en <em>N</em> points où le joueur a perdu <em>L</em> de MWC :</p>
<blockquote>
<p>L₇ = L × √(7 / N)</p>
</blockquote>
<p>Pour un match en 7 points, c'est la perte de MWC du match, celle qu'affiche eXtreme Gammon. La racine vient du modèle de la formule FIBS : à force égale, la perte d'un joueur croît comme la racine de la longueur du match. 7 points est la longueur de référence parce que c'est la plus courante en tournoi.</p>
<p>Elle se lit comme la part de match perdue face à un joueur parfait : 12,3 % veut dire qu'au lieu de 50 % de chances contre le moteur, le joueur n'en avait plus que 37,7 % sur un match en 7 points. L'infobulle en donne la lecture en Elo face au moteur, en inversant la formule FIBS : D = (2000 / √7) × log₁₀(q / (1 − q)), avec q = 0,5 − L₇. Au-delà d'une perte de 49 %, la formule n'a plus de valeur finie : l'Elo affiché est alors un plafond (« ≤ »). Les deux chiffres classent les joueurs dans le même ordre.</p>
<p>Elle complète le PR sans le remplacer : le PR divise les erreurs par un nombre de décisions, et ce nombre dépend de ce que l'on compte comme décision (coups forcés, décisions de videau évidentes). La perte MWC ne compte aucune décision : chaque erreur pèse ce qu'elle a coûté au score où elle a été commise.</p>
<p>Sur plusieurs matchs (statistiques d'un joueur, d'un tournoi), les pertes et les racines des longueurs s'additionnent avant le rapport : L₇ = √7 × ΣL / Σ√N. Un match en 7 points pèse donc plus qu'un match en 1 point, et un seul match donne sa propre valeur.</p>
<p>Un match isolé est très bruité : quelques grosses erreurs suffisent à le faire varier du simple au double. Le badge et le bilan d'un match (bilan du match) donnent l'intervalle à 95 % en rééchantillonnant ses parties ; un agrégat (statistiques d'un joueur, d'un tournoi) le donne en rééchantillonnant ses matchs ; sans filtre joueur, les deux sièges d'un match forment une seule unité, puisqu'ils jouent les mêmes positions. Il faut au moins trois parties, ou trois matchs, et qu'elles ne donnent pas toutes la même perte : sinon pas d'intervalle. Une ligne par match des statistiques n'en a donc pas : c'est une seule unité. Ne classez pas des joueurs sur un seul match.</p>
<div class="admonition caution">
<p>Une partie <em>money-game</em> n'a pas de longueur de match : la perte MWC (éq. 7 pts) n'y est pas définie et le panneau l'indique au lieu d'afficher un nombre. Comme le MWC cost, elle dépend de la MET.</p>
</div>
<p>Le troisième choix du bouton, <strong>MWC 7 pts</strong>, trace cette perte dans l'onglet Progression, et l'affiche sur les cartes du tableau de bord et dans la comparaison pions/videau de l'onglet Erreurs, découpée en pions et videau sur les mêmes matchs (les deux parts s'additionnent). Le classement des joueurs a sa colonne. Les graphiques sans équivalent 7 points (par action de videau) gardent le MWC cost.</p>
</blockquote>
<p>Le basculement PR ↔ MWC est instantané : aucun recalcul backend n'est effectué.</p>
<h4>Le rapport HTML</h4>
<p>Le bouton <strong>Rapport HTML</strong> de l'en-tête du panneau produit un document <strong>autonome</strong> : un seul fichier, sans image externe, sans feuille de style distante, sans script. Les diagrammes y sont des SVG en ligne, dessinés par le même rendu que le plateau à l'écran, avec votre palette. Il s'ouvre dans n'importe quel navigateur, s'envoie par courriel, et <strong>s'imprime en PDF par le navigateur lui-même</strong> — ce qui évite d'embarquer un générateur de PDF pour produire ce que tout le monde a déjà.</p>
<p>Il contient les indicateurs du périmètre courant (positions, matchs, décisions comptées, PR global, pions et videau), puis les <strong>dix décisions les plus coûteuses</strong>, chacune avec son diagramme, son coût, le match d'où elle vient et le meilleur coup lorsqu'une analyse le donne.</p>
<p>Le document est construit par le moteur, pas par l'écran : la ligne de commande (<code>stats report --html</code>, voir stats — Erreurs récurrentes) et le démon HTTP (route <code>stats.report</code>) produisent le même rapport, dans l'une des neuf langues de l'interface. Seule l'application graphique dessine les diagrammes avec la palette du plateau ; les deux autres emploient la palette par défaut.</p>
<p>Le rapport porte le <strong>filtre courant</strong> du panneau Stats. Un rapport qui ne dit pas son périmètre est un rapport dont les chiffres ne veulent rien dire : réglez le filtre — un tournoi, une plage de dates, un joueur — avant de le produire.</p>
<h4>Onglet Tableau de bord</h4>
<p>L'onglet <strong>Tableau de bord</strong> donne une vue synthétique des indicateurs clés.</p>
<h5>Cartes de niveau</h5>
<p>Trois cartes affichent le PR (ou MWC) pour :</p>
<ul>
<li><strong>PR global</strong> — toutes les décisions (pions + videau) ;</li>
<li><strong>PR pions</strong> — décisions de pions seulement ;</li>
<li><strong>PR videau</strong> — décisions de videau seulement.</li>
</ul>
<p>Cliquer sur une carte charge dans le panneau d'analyse les positions du sous-ensemble correspondant (drill-down).</p>
<div class="admonition note">
<p>Le nombre total de décisions est affiché en bas de chaque carte au survol.</p>
</div>
<h5>Plan d'étude</h5>
<p>La carte <strong>Plan d'étude</strong> répond à « que dois-je travailler maintenant ? ». Les erreurs récurrentes (onglet Erreurs) disent où le filtre a perdu le plus ; elles ne disent pas où l'étude rapporte le plus. Une famille de positions réellement difficiles coûte cher à tout le monde, et trois erreurs ne font pas une tendance. Le plan corrige les deux.</p>
<ul>
<li>Une <strong>famille</strong> est un groupe des erreurs récurrentes : un plan de jeu, une nature de décision (pions ou videau) et un thème. Les erreurs sans thème n'en forment pas : elles n'indiquent rien à étudier.</li>
<li>Chaque erreur est chiffrée en <strong>MWC</strong>, comme dans le panneau Match : sa perte ℓ, et sa <em>difficulté</em> d, la perte qu'un joueur de référence aurait subie dans la même position (voir la difficulté par décision du panneau Match). Une erreur en partie libre n'a pas de MWC : elle est seulement comptée, « non chiffrée ».</li>
<li>Le <strong>MWC récupérable</strong> d'une famille est la somme de ℓ − d sur ses erreurs : la fréquence de la famille multipliée par sa perte moyenne au-delà de la difficulté. C'est ce que vous regagneriez en jouant ces positions comme le joueur de référence. Une famille d'erreurs évitables monte, une famille de positions où tout le monde se trompe descend.</li>
<li>L'<strong>intervalle à 95 %</strong> accompagne chaque chiffre. Une famille entre au plan à partir de <strong>5 erreurs</strong> et d'un intervalle entièrement au-dessus de zéro ; le plan est classé par la borne basse de l'intervalle, si bien qu'à récupérable égal la famille la mieux établie passe devant. Les autres sont nommées sous le tableau, <strong>à confirmer</strong>, sans rang : le plan ne vous pousse pas vers du bruit.</li>
</ul>
<p>Chaque famille propose trois gestes : <strong>Étudier</strong> ouvre la file d'étude sur ses positions, l'écart au joueur de référence le plus grand d'abord ; <strong>Quiz</strong> lance l'exercice Décision du panneau Entraînement sur vingt d'entre elles ; <strong>Anki</strong> en fait un paquet de cartes. Au-dessus du tableau, <strong>Quiz sur les trois premières familles</strong> tire vingt positions parmi celles des trois familles de tête. Le plan suit le filtre du panneau : réglez le joueur pour obtenir <em>votre</em> plan. En ligne de commande : <code>blunderdb stats plan</code> (voir stats — Erreurs récurrentes).</p>
<h5>Avant/après l'étude</h5>
<p>La carte <strong>Avant/après l'étude</strong> ferme la boucle du plan : ce que vous avez travaillé coûte-t-il moins en match réel ? Sans elle, l'étude se juge à l'impression ; avec elle, une famille qui ne bouge pas malgré le travail dit qu'il faut changer de méthode, et une famille en progrès peut céder sa place dans le plan.</p>
<ul>
<li>Une famille (celle du plan) est <strong>étudiée</strong> à la date de la première action d'étude sur l'une de ses positions : la marque « étudiée » de la file d'étude, une révision Anki, une réponse en quiz. Créer une carte ou ranger une position en collection n'en est pas une.</li>
<li>Pour chaque famille étudiée, la carte compare deux <strong>fenêtres</strong> : les matchs joués avant ce jour et ceux joués après (le jour même est écarté). Dans chacune, le <strong>taux de perte</strong> est la MWC perdue par les erreurs de la famille, rapportée à toutes les décisions du même plan de jeu et de la même nature, et affichée en points de MWC pour 100 décisions.</li>
<li>Le <strong>gain</strong> est le taux d'avant moins celui d'après, avec son intervalle à 95 %. Le verdict n'est « en progrès » (ou « en recul ») que si chaque fenêtre compte au moins <strong>30 décisions</strong> et que l'intervalle exclut zéro ; sinon il reste « indéterminé » ou « trop peu de décisions ».</li>
</ul>
<p>C'est un changement, pas un effet : une famille est étudiée parce qu'elle coûtait, et une part du gain est une régression vers la moyenne ; rien ne contrôle les adversaires, le format ni les dés. En ligne de commande : <code>blunderdb stats effect</code> (voir stats — Erreurs récurrentes).</p>
<h5>Biais signés</h5>
<p>La carte <strong>Biais signés</strong> dit dans quel sens vous vous trompez, pas seulement combien. « Vous prenez trop » se corrige mieux qu'un PR : la règle à revoir est nommée. Chaque biais est la part de décisions fautives dans un sens moins la part fautive dans l'autre, avec son intervalle à 95 % ; un penchant n'est nommé qu'à partir de <strong>20 décisions</strong> et d'un intervalle qui exclut zéro.</p>
<ul>
<li><strong>Prise / refus</strong> — sur les réponses au videau : prises fautives (le bot refuse) moins refus fautifs (le bot prend). C'est exactement votre taux de prise moins celui du bot sur les mêmes positions.</li>
<li><strong>Doubles</strong> — sur les décisions de doubler : doubles prématurés (le bot ne double pas, ou la position est trop bonne pour doubler) moins doubles manqués. Le même biais est donné <strong>par score</strong> (votre away, celui de l'adversaire, un score post-Crawford compté à 1 point ; la partie libre dans sa propre case), pour les scores qui comptent assez de décisions.</li>
<li><strong>Blots</strong> — sur les coups de pions avec contact : les coups qui laissent plus de blots que le meilleur coup moins ceux qui en laissent moins. Un coup est reconnu par le plateau qu'il laisse, quelle que soit son écriture (« 13/7(2) », « 8/5*/4 ») ; un coup que le générateur de coups ne sait toujours pas rejouer est écarté, et la part des coups écartés est donnée sous le tableau. Le biais compte les blots sans peser leur exposition : un blot hors de portée compte autant qu'un blot exposé.</li>
</ul>
<p>À côté de chaque compte, son coût en millipoints dit si le penchant coûte ; une position jouée dans plusieurs matchs y compte avec le coup joué dans chacun. Les biais suivent le filtre du panneau. En ligne de commande : <code>blunderdb stats biases</code> (voir stats — Erreurs récurrentes).</p>
<h5>PR glissant sur N dernières décisions</h5>
<p>Une ligne de valeurs PR (ou MWC) calculées sur les <em>N</em> dernières décisions (N = 5, 10, 50, 100, 250, 500, 1000) permet de mesurer la tendance récente. Les valeurs grisées correspondent à un N supérieur au nombre de décisions disponibles.</p>
<p>Cliquer sur une valeur charge les <em>N</em> dernières positions correspondantes.</p>
<h5>Top blunders</h5>
<p>La liste des 10 pires erreurs (ou MWC cost), triées par magnitude décroissante. Cliquer sur une ligne charge la position concernée dans le panneau d'analyse.</p>
<h4>Onglet Progression</h4>
<p>L'onglet <strong>Progression</strong> présente l'évolution du niveau dans le temps.</p>
<p>En tête de l'onglet, un <strong>objectif</strong> : « PR &lt; 5 d'ici douze semaines ». Une cible, une échéance, et une tendance qui dit où l'on va — rien de plus. Un objectif qui se mettrait à noter, à féliciter ou à rappeler serait une autre fonctionnalité, et pas celle-ci.</p>
<p>Le bouton <strong>Proposer</strong> suggère une cible à partir du niveau actuel : la borne basse de la bande où vous êtes, c'est-à-dire l'entrée dans la bande suivante. Proposer « un peu mieux » ne s'ancrerait à rien ; proposer un palier en dit un — passer d'intermédiaire à avancé se voit et se raconte.</p>
<p>La <strong>tendance</strong> est un ajustement par les moindres carrés sur le PR de vos matchs, projeté à l'échéance. Elle refuse de se prononcer sous trois matchs : tracer une droite entre deux points serait une affirmation qu'on ne peut pas tenir. Et la phrase le dit à chaque fois — <em>une tendance n'est pas une prédiction</em>.</p>
<p>L'objectif est enregistré dans les <strong>métadonnées de la base</strong>, pas dans la configuration : il porte sur cette bibliothèque-là, et suit donc le fichier plutôt que la machine. Aucun changement de schéma : <code>metadata</code> est déjà une table de clés et de valeurs, lisible par <code>blunderdb info</code> comme par le démon.</p>
<h5>Courbe par tournoi</h5>
<p>Un graphique en ligne affiche le PR (ou MWC) pour chaque tournoi (axe X : ordre des tournois, axe Y : valeur de la métrique). Des bandes de couleur matérialisent les seuils de niveau.</p>
<p>Cliquer sur un point du graphique ouvre un menu contextuel avec deux options :</p>
<ul>
<li><strong>Ouvrir le tournoi</strong> — ouvre le tournoi dans le panneau Tournois.</li>
<li><strong>Ouvrir les positions</strong> — charge toutes les positions du tournoi dans le panneau d'analyse.</li>
</ul>
<h5>Scatter plot par match</h5>
<p>Un nuage de points représente chaque match (axe X : date, axe Y : PR ou MWC). La taille du point est proportionnelle au nombre de décisions dans le match.</p>
<p>Cliquer sur un point ouvre un menu contextuel :</p>
<ul>
<li><strong>Ouvrir le match</strong> — ouvre le match dans le panneau des matchs.</li>
<li><strong>Ouvrir les positions</strong> — charge toutes les positions du match dans le panneau d'analyse.</li>
</ul>
<h4>Onglet Erreurs</h4>
<p>L'onglet <strong>Erreurs</strong> décompose les sources d'erreurs.</p>
<h5>Erreurs récurrentes</h5>
<p>En tête de l'onglet, un tableau regroupe les erreurs du filtre courant — donc celles d'un seul joueur quand un joueur est filtré — par <strong>plan de jeu</strong> et par <strong>thème</strong>, la plus coûteuse d'abord. Il répond à la question « où est-ce que je perds le plus ? » : par exemple, « tenue · trop de blots ».</p>
<ul>
<li>Une <strong>erreur</strong> est une décision comptée dont le coût atteint le seuil <em>Erreur</em> de la bibliothèque.</li>
<li>Le <strong>plan de jeu</strong> est celui du joueur au trait, tel que l'onglet Ventilations le présente.</li>
<li>Le <strong>thème</strong> d'un coup de pions est celui que nomment les règles de la phrase d'explication du panneau Analyse : gammon sous-estimé, trop de blots, point non fait, trop passif. Celui d'une décision de videau est le sens de l'erreur, comme dans la direction des erreurs de videau ci-dessous : double manqué, double prématuré, passe à tort, prise à tort.</li>
<li>Une erreur qu'aucune règle ne nomme avec assurance n'est pas devinée : elle sort du classement et apparaît à part, sous le tableau, en une ligne par plan de jeu (« sans thème identifié : N erreurs, coût X »), cliquable elle aussi. Ce reste est souvent le plus lourd, car l'explication ne se prononce qu'à partir de 60 mp, au-dessus du seuil <em>Erreur</em> : classé avec les autres, il coifferait un tableau qui ne dirait rien.</li>
<li>Le <strong>coût</strong> est la part du PR du filtre que le groupe représente : la formule du PR appliquée aux erreurs du groupe, rapportée à toutes les décisions comptées. Les coûts des groupes ne dépassent donc jamais le PR.</li>
</ul>
<p>Cliquer sur un groupe charge ses positions, de la plus coûteuse à la moins coûteuse. Le thème est recalculé à chaque affichage, jamais enregistré : comme le plan de jeu, c'est une étiquette dérivée, non modifiable. En ligne de commande : <code>blunderdb stats recurring</code> (voir stats — Erreurs récurrentes).</p>
<p>Chaque ligne propose trois gestes pour passer de l'erreur à l'étude : <strong>Quiz sur ce groupe</strong> lance l'exercice Décision du panneau Entraînement sur les positions du groupe, <strong>Paquet Anki</strong> en fait un paquet de cartes, <strong>Collection</strong> les range dans une nouvelle collection. Au-dessus du tableau, <strong>Quiz de mes trois pires groupes</strong> tire vingt positions au hasard parmi celles des trois groupes les plus coûteux. En ligne de commande, <code>stats recurring --quiz</code> tire ces positions et <code>--deck</code> crée le paquet.</p>
<h5>Répartition par action de videau</h5>
<p>Un diagramme en barres affiche le PR (ou MWC) pour chaque type de décision de videau : <em>NoDouble</em>, <em>DoubleTake</em>, <em>DoublePass</em>, <em>TooGood</em>. Chaque barre indique également le nombre de décisions et le taux de blunders en infobulle.</p>
<p>Cliquer sur une barre charge les positions correspondant à cette action de videau, <strong>uniquement celles avec une erreur</strong> (drill-down).</p>
<h5>Direction des erreurs de videau</h5>
<p>La répartition ci-dessus indique <em>combien</em> coûtent les décisions de videau ; ce tableau indique dans <em>quel sens</em> elles se trompent.</p>
<p>Une position de videau porte deux décisions prises par deux joueurs différents, présentées ici en deux lignes :</p>
<ul>
<li><strong>Offrir</strong> — le joueur qui tient le videau double ou ne double pas. Ses erreurs sont les <strong>doubles manqués</strong> (il fallait doubler) et les <strong>doubles prématurés</strong> (il ne fallait pas).</li>
<li><strong>Répondre</strong> — le joueur à qui le videau est offert prend ou passe. Ses erreurs sont les <strong>passes à tort</strong> (une prise correcte a été passée) et les <strong>prises à tort</strong> (une passe correcte a été prise).</li>
</ul>
<p>Les deux lignes restent séparées à dessein : un joueur peut parfaitement doubler tard <em>et</em> prendre large, et un indicateur unique appellerait cela « équilibré » en perdant les deux moitiés de l'information.</p>
<p>Chaque case affiche le nombre de décisions ; l'infobulle donne l'équité perdue cumulée. Cliquer sur une case charge les positions correspondantes. Une case à zéro n'est pas cliquable.</p>
<div class="admonition note">
<p>Ce tableau compte des décisions, il ne porte pas de jugement. À partir de quel écart une tendance mérite d'être nommée dépend de l'effectif et d'un point de référence, qui ne sont pas des données du moteur.</p>
</div>
<h5>Répartition Checker / Cube</h5>
<p>Un diagramme comparatif place côte à côte le PR des coups joués et des décisions de videau. Cliquer sur une barre charge les positions du sous-ensemble avec erreur.</p>
<h5>Histogramme des magnitudes d'erreur</h5>
<p>Un histogramme distribue les erreurs selon leur magnitude en millipoints (mpt, tranches : 0–5, 5–10, 10–25, 25–50, 50–100, ≥ 100). Cliquer sur une barre charge les positions de la tranche.</p>
<h4>Entraînement dans le temps</h4>
<p>L'onglet <strong>Entraînement</strong> met côte à côte, sur les mêmes fenêtres calendaires — la <strong>semaine</strong> ou le <strong>mois</strong>, au choix — trois séries qui mesurent la progression par trois chemins :</p>
<ul>
<li>le <strong>PR du quiz</strong> : celui des sessions de l'exercice Décision du panneau Entraînement, pondéré par le nombre de décisions jugées. Il est calculé sur l'échelle du PR réel, donc comparable à lui ;</li>
<li>le <strong>PR des matchs</strong> du filtre courant, pondéré par le nombre de décisions ;</li>
<li>la <strong>rétention Anki</strong> : la part des révisions de cartes déjà apprises notées <em>Difficile</em> ou mieux, lue sur l'axe de droite (en %).</li>
</ul>
<p>Sous le graphique, le <strong>PR du quiz par plan de jeu</strong> range les décisions du quiz tirées de positions de la bibliothèque, les pires plans d'abord, avec le PR de la dernière fenêtre où le plan a été joué.</p>
<p>Entre parenthèses, le nombre de décisions ou de révisions derrière chaque valeur : une fenêtre sans échantillon n'a pas de valeur — un tiret, pas un zéro. Le filtre ne restreint que les matchs ; le journal du quiz et celui d'Anki sont les vôtres et ne portent pas de joueur. Rien n'est enregistré en plus : les trois séries sont relues dans les journaux existants. En ligne de commande : <code>blunderdb stats training</code> (voir stats — Erreurs récurrentes).</p>
<h4>Onglet Ventilations</h4>
<p>L'onglet <strong>Ventilations</strong> découpe les mêmes décisions que les chiffres globaux selon quatre axes. Aucun d'eux ne redéfinit ce qui compte comme une décision : ce serait un second PR sous le même nom.</p>
<ul>
<li><strong>Par phase de partie</strong> — ouverture, milieu de partie, course, sortie des pions. C'est ce qui répond à « mon PR en course contre mon PR en contact ». L'étiquette est calculée depuis le plateau (voir Panneau Recherche) ; une base dont les phases n'ont jamais été calculées range tout sous <em>Non classée</em>, et <code>blunderdb repair</code> la remplit.</li>
<li><strong>Par plan de jeu</strong> — course, blitz, tenue, backgame, amorce contre amorce… C'est la ventilation pour laquelle le classificateur existe : « où est-ce que je perds le plus ? », plan par plan. Même étiquette dérivée que la phase, mêmes réserves, et <code>blunderdb repair</code> la remplit de la même façon.</li>
<li><strong>Par étiquette</strong> — les <code>#mot</code> écrits dans les commentaires. Une position peut en porter plusieurs : <strong>ces lignes ne s'additionnent pas au total</strong>, et le panneau le dit sous le tableau. Une étiquette qualifie, elle ne partitionne pas.</li>
<li><strong>Par score</strong> — l'écart au but des deux camps, lu du côté du joueur au trait, donc du côté de celui qui décide. La ligne <em>Money</em> est la partie d'argent. Une cellule sans intervalle est <strong>grisée avec son effectif visible</strong> plutôt que cachée (voir ci-dessous).</li>
</ul>
<p>Chaque ligne porte son <strong>intervalle de confiance à 95 %</strong> (colonne <em>IC 95 %</em>), qui remplace le seuil fixe de dix décisions : il dit ce que vaut un PR, pas seulement combien de décisions le portent. Il rééchantillonne les <strong>matchs</strong> de la sélection, pas les décisions ni les parties : les parties d'un match partagent adversaire, séance et fatigue, et les compter comme indépendantes donnerait un intervalle trop étroit ; sans filtre joueur, les deux sièges d'un match forment une seule unité. Une ligne qui repose sur moins de trois matchs, ou dont tous les matchs donnent le même PR, n'a pas d'intervalle et est grisée, même avec beaucoup de décisions : deux matchs ne disent rien de la dispersion, et des matchs identiques disent qu'ils s'accordent, pas que le PR est exact. Pour l'étude : une famille de positions dont l'intervalle reste au-dessus de votre PR global est une faiblesse établie ; une ligne à l'intervalle large ne justifie pas encore un plan de travail. Le PR global du tableau de bord porte le même intervalle, sous sa carte.</p>
<div class="admonition note">
<p>La partie Crawford n'est pas distinguée : blunderDB n'enregistre pas cet indicateur sur une position. L'effet pratique est faible — une partie Crawford n'a aucune décision de videau — mais l'omission est réelle et vaut mieux d'être écrite que laissée à deviner.</p>
</div>
<h4>Étude et jeu réel</h4>
<p>La commande <code>blunderdb list --type study --days 30</code> met côte à côte, plan de jeu par plan de jeu, trois nombres : combien de <strong>positions distinctes</strong> ont été révisées sur la période, quel était le PR <strong>avant</strong> elle, quel est le PR <strong>depuis</strong>.</p>
<p>Trois nombres, et pas un quatrième. Il n'y a <strong>ni colonne de gain ni flèche</strong>, parce que rien ici ne contrôle quoi que ce soit : le joueur a pu rencontrer des adversaires plus forts, changer de format, ou simplement jouer plus de courses ce mois-ci. Le rapprochement est celui du lecteur ; une colonne qui annoncerait un effet affirmerait une causalité que ces données ne portent pas. Les nombres, eux, sont exacts.</p>
<p>Les révisions sont comptées en <strong>positions distinctes</strong> : une carte revue quatre fois dans le mois est une position étudiée, et compter les répétitions ferait passer un mois de bachotage pour un mois de couverture. Les décisions du PR, elles, sont toutes comptées — chacune a été prise une fois. Un PR appuyé sur moins de dix décisions s'affiche <code>—</code>, avec son effectif visible à côté.</p>
<h4>Onglet Joueurs</h4>
<p>Les cinq onglets précédents décrivent <strong>un</strong> joueur ; l'onglet <strong>Joueurs</strong> les compare tous. Il affiche une ligne par joueur de la base, ce qui répond au besoin d'un organisateur suivant une compétition entière plutôt qu'un joueur en particulier.</p>
<p>Colonnes, dans l'ordre :</p>
<table>
<thead>
<tr>
<th>Colonne</th>
<th>Signification</th>
</tr>
</thead>
<tbody>
<tr>
<td>Joueur</td>
<td>Le nom <strong>tel qu'il figure dans les matchs</strong>. Un joueur enregistré sous deux orthographes apparaît sur deux lignes, sauf si l'une est un alias de l'autre (section Corpus des paramètres) : la ligne porte alors le nom canonique.</td>
</tr>
<tr>
<td>Matchs</td>
<td>Nombre de matchs disputés dans la période retenue.</td>
</tr>
<tr>
<td>V–D</td>
<td>Victoires et défaites. Un match inachevé (journal tronqué, abandon) ne compte ni l'une ni l'autre : V + D peut donc être inférieur au nombre de matchs.</td>
</tr>
<tr>
<td>Décisions</td>
<td>Nombre de décisions comptées — le dénominateur du PR. C'est la colonne qui dit ce que valent les taux voisins : un PR calculé sur douze décisions ne signifie rien.</td>
</tr>
<tr>
<td>PR</td>
<td>Performance Rating global.</td>
</tr>
<tr>
<td>PR pions, PR videau</td>
<td>Le PR ventilé par type de décision.</td>
</tr>
<tr>
<td>Snowie</td>
<td>Snowie Error Rate (voir Annexe : Modèle de statistiques — alignement XG / gnuBG / blunderDB).</td>
</tr>
<tr>
<td>Blunders</td>
<td>Nombre d'erreurs atteignant le seuil de blunder de la bibliothèque (0,100 EMG par défaut).</td>
</tr>
<tr>
<td>Chance</td>
<td>Chance moyenne par lancer, en millipoints (mpt), signée : positive si les dés ont été favorables.</td>
</tr>
</tbody>
</table>
<p>Utilisation :</p>
<ul>
<li><strong>Trier</strong> — cliquez sur un en-tête de colonne. Le tableau s'ouvre trié par PR croissant, meilleur joueur en tête. Les joueurs dont rien n'a été mesuré restent en bas quel que soit le sens du tri : un zéro faute de données n'est pas une performance parfaite.</li>
<li><strong>Ouvrir le détail d'un joueur</strong> — cliquez sur une ligne. Le joueur est sélectionné dans la barre de filtres et l'affichage bascule sur l'onglet Tableau de bord.</li>
<li><strong>Restreindre la période</strong> — les filtres de dates, de tournois et de longueur de match s'appliquent normalement, ce qui permet de borner le tableau aux dates d'une compétition.</li>
<li><strong>Comparer deux joueurs</strong> — cochez la case de la première colonne sur deux lignes. Un bloc apparaît au-dessus du tableau et met leurs indicateurs face à face ; cocher un troisième joueur remplace le plus ancien des deux. La case ne sélectionne pas la ligne : cocher compare, cliquer ouvre le détail.</li>
<li><strong>Voir les positions où l'un a mieux joué</strong> — le bouton du bloc de comparaison liste les positions que les deux joueurs ont eu à jouer et où l'un a bien joué (en dessous du seuil Erreur de la bibliothèque) et l'autre non, l'écart le plus large d'abord ; le joueur est jugé sur son pire coup de la position. <em>Ouvrir ces positions</em> les charge dans la vue analyse. La ligne de commande et le serveur donnent la même liste (<code>stats contrast</code>).</li>
<li><strong>Noter les coups d'abord</strong> — la liste ne compare que les coups dont l'erreur est notée, et aucun import ne les note : la notation se fait sur demande. Tant qu'il en reste à noter, le bloc en donne le nombre avec un bouton <em>Noter les coups</em> ; la ligne de commande fait la même chose avec <code>repair --move-errors</code>.</li>
</ul>
<p>Dans ce bloc, <strong>seuls les taux reçoivent un verdict</strong>, et le meilleur des deux est mis en gras. Trois indicateurs n'en reçoivent jamais, et il vaut de dire pourquoi. La <strong>chance</strong> n'est pas une qualité : un joueur plus chanceux n'est pas meilleur. Les <strong>matchs, le bilan et les décisions</strong> situent ce que les taux valent, mais les mettre en compétition ferait gagner celui qui a simplement joué davantage. Le <strong>nombre de blunders</strong> ne se compare pas brut — douze sur mille décisions valent mieux que dix sur cent —, aussi le bloc ajoute-t-il une ligne <em>Blunders / 100 déc.</em> qui, elle, se compare, et laisse le compte à côté comme contexte.</p>
<p>Une égalité n'est pas une victoire : elle n'est mise en gras d'aucun côté. Un taux qui n'a rien derrière lui s'affiche « — » et ne départage rien.</p>
<div class="admonition note">
<p>Dans cet onglet, la liste <strong>Joueur</strong> et le choix du <strong>type de décision</strong> sont désactivés : le tableau montre tous les joueurs, et il ventile déjà les décisions de pions et de videau en colonnes distinctes.</p>
</div>
<p>Trois vues de corpus figurent dans l'onglet <strong>Corpus</strong> du panneau, en ligne de commande et par l'API du démon (voir stats — Erreurs récurrentes) : le <strong>face-à-face</strong> de deux joueurs (matchs communs, PR de chacun, bilan), le <strong>PR par fenêtre calendaire</strong> glissante (1, 3, 6 ou 12 mois) et le <strong>classement</strong> par PR des joueurs qui ont au moins un nombre donné de décisions comptées. L'onglet Corpus calcule chaque vue à la demande, sous le filtre courant. Un <strong>filtre de provenance</strong> (champs <em>Moteur</em> et <em>Profondeur min.</em> de la barre de filtres) restreint les statistiques aux décisions analysées ainsi ; il porte sur chaque décision, et les chiffres qu'il touche sont alors recalculés depuis les décisions. Le face-à-face et le PR par fenêtre ne l'acceptent pas : tant qu'il est actif, ils refusent de se calculer. Comme le reste de la barre, il est conservé d'une session à l'autre.</p>
<div class="admonition important">
<p>Un tiret (« — ») signale une valeur <strong>jamais mesurée</strong>, à ne pas confondre avec zéro. C'est notamment le cas de la colonne Chance pour tout match importé avant la version 2.15.0 du schéma : la chance n'était alors pas conservée, et rien ne permet de la reconstituer après coup. Réimporter le fichier source ne suffit pas : l'import y reconnaît un doublon et n'applique que les marques d'étude nouvellement levées (le rapport en donne le nombre). Il faut supprimer le match, puis le réimporter. Les formats qui ne la transportent pas (BGF, Jellyfish <code>.mat</code>) n'en fourniront jamais.</p>
</div>
<h4>Règle d'agrégation</h4>
<div class="admonition important">
<p>Le PR d'un tournoi (ou d'un sous-ensemble quelconque) est calculé par la règle <strong>somme/somme</strong> — jamais comme moyenne des PR individuels des matchs.</p>
<p>Formule :</p>
<pre class="math">PR_&#123;tournoi&#125; = 500 \\times \\frac&#123;\\sum_&#123;i&#125; \\text&#123;erreur&#125;_i&#125;&#123;\\text&#123;nombre total de décisions&#125;&#125;</pre>
<p><strong>Exemple :</strong> un joueur dispute deux matchs dans un tournoi —</p>
<ul>
<li>Match A : 10 décisions, 0,100 d'équité perdue → PR = 5,0</li>
<li>Match B : 90 décisions, 0,540 d'équité perdue → PR = 3,0</li>
</ul>
<p>Moyenne naïve des PR : (5,0 + 3,0) / 2 = <strong>4,0</strong> <em>(incorrect)</em></p>
<p>Règle somme/somme : 500 × 0,640 / (10 + 90) = <strong>3,2</strong> <em>(correct)</em></p>
<p>La règle somme/somme est la seule qui résiste à la variation de longueur des matchs (un match en 21 points pèse plus qu'un match en 1 point).</p>
</div>
<h4>MWC : limitations</h4>
<ul>
<li>Par défaut, le MWC cost est calculé à partir de la <strong>MET Kazaross-XG2</strong>, table de référence de facto dans le backgammon compétitif. Les résultats ne sont pas directement comparables avec des logiciels utilisant d'autres METs. C'est la même table, lue par le même point d'entrée, que celle dont l'évaluateur embarqué se sert pour ses décisions de videau au score : les statistiques et le moteur ne peuvent pas diverger là-dessus. Elle donne ses valeurs propres jusqu'à 25 points à faire de chaque côté ; au-delà, elle est prolongée par une table de Zadeh calculée comme celle de GNUbg, jusqu'à 64.</li>
<li>Les positions <em>money-game</em> (sans score de match) sont <strong>exclues</strong> du calcul MWC. Si votre base de données contient beaucoup de positions money-game, le MWC cost peut être sous-estimé ou indisponible.</li>
<li>Le MWC cost est cumulatif sur l'ensemble du jeu de données filtré — pas un indicateur par décision. Il mesure l'impact total de vos erreurs sur vos chances de victoire.</li>
</ul>
<h3>Panneau Eval</h3>
<p>Le panneau <strong>Eval</strong> (<em>CTRL-E</em>) évalue en direct la position posée sur le plateau, quelle qu'elle soit ; sur une position de bearoff il se spécialise et calcule en plus l'EPC (Effective Pip Count). Il est activé en appuyant sur <em>CTRL-E</em>, en cliquant sur l'onglet Eval dans le panneau inférieur, ou en exécutant la commande <code>eval</code> ; <code>epc</code>, son ancien nom, l'ouvre aussi. Le panneau s'est appelé <em>EPC</em>, puis <em>Bearoff</em>, avant de devenir <em>Eval</em> — c'est donc ici qu'il faut chercher ce qu'une version antérieure appelait le panneau Bearoff, le nom ne désignant plus que l'onglet de configuration des tables de sortie.</p>
<p>Le panneau montre toujours la <strong>seule décision</strong> que la position posée sur le plateau appelle — jamais deux à la fois — et les faits qui vont avec. Chaque quantité se lit dans l'axe qui lui convient plutôt que dans un axe unique imposé : la probabilité de gain, de gammon, de backgammon et l'équité cubeless de chaque joueur, calculées <em>avant le jet</em>, se lisent <strong>par joueur</strong> (bas, haut, puis Δ), à gauche de la décision de videau, quand aucun dé n'est posé. Les faits et la décision restent côte à côte : la décision de videau ne passe jamais sous les chiffres qui la justifient, quelles que soient la langue de l'interface et la position sur le plateau. Dès que des dés sont posés, ces mêmes valeurs <em>avant le jet</em> changent d'axe : elles se lisent <strong>au trait</strong>, en tête de la liste des coups candidats, sous forme d'une ligne italique <em>avant le jet</em> — pas un coup candidat de plus, un repère contre lequel lire chaque coup. L'écart entre cette ligne et un coup contient la chance du jet, jamais le mérite du coup, et elle ne porte donc aucune colonne d'erreur. Sur une position de bearoff pur, un second tableau, toujours <strong>par joueur</strong> et toujours présent, dés posés ou non, porte l'EPC, le pip count, le wastage, le nombre moyen de lancers et l'écart type ; ces cinq colonnes ne migrent jamais. Les deux tableaux sont empilés et partagent la même grille de colonnes : mêmes bords, mêmes repères de colonne, une seule colonne de pastilles — ils se lisent comme un seul objet à deux étages. Le bouton <em>Ajouter à la base</em>, le badge de régime, l'attribution du moteur (la profondeur de la dernière évaluation y figure aussi) et la case <em>Défi</em> forment une bande à part, alignée à droite au-dessus des tableaux.</p>
<p>Seule la liste des coups candidats défile — la ligne <em>avant le jet</em>, elle aussi, reste épinglée au-dessus d'elle ; le reste du panneau (faits, badge, décision de videau) reste toujours visible, sans réglage particulier de la taille du panneau.</p>
<p>Le tableau de faits et la décision sont calculés par gammonNet, embarqué, sans XG ni gnubg. Le calcul suit la position sans jamais figer l'interface : une profondeur 0-ply s'affiche immédiatement à chaque geste, puis, après une demi-seconde d'immobilité, une évaluation plus profonde (2 plis par défaut, réglable dans l'onglet <em>gammonNet</em> de la configuration) la remplace en arrière-plan — tout nouveau geste annule ce calcul de fond. La profondeur affichée dans la bande de badges, ou au sein du badge de régime sur une position de course, est toujours celle qui a effectivement produit le chiffre montré, jamais celle demandée ; elle ne se répète pas sur chaque ligne, puisqu'une évaluation en direct partage la même profondeur pour tous les coups. L'équité des coups candidats et de la décision de videau suit le score de la position : en money game elle est exprimée en points, à un score de match en <strong>équité normalisée</strong> — la même échelle que XG et GNU Backgammon, où gagner la valeur du videau courant vaut +1 et la perdre −1 — jamais mélangées dans un même tableau. L'en-tête de la colonne le dit explicitement plutôt que de laisser deviner l'échelle : « Équité (money) » en money game, « Équité (match) » à un score de match. Elle tient compte du <strong>videau vivant</strong> : la recherche valorise chaque position finale par le modèle de videau (Janowski, efficacité mesurée) dans l'état du videau de la position, comme le font XG et GNU Backgammon en évaluation <em>cubeful</em>. C'est ce qui rend visibles au score les effets gammon-go et gammon-save — à 4-away/2-away, le joueur mené joue 8/2 6/2 sur un 6-4 d'ouverture parce que son double précoce donnera au gammon la valeur du match, ce qu'une évaluation sans videau ne peut pas voir. La ligne <em>avant le jet</em>, elle, reste une équité <strong>cubeless</strong> : c'est un fait de la position, pas une décision. L'évaluation n'est jamais enregistrée : c'est un calcul, pas une analyse. Cliquer un coup candidat l'affiche sur le plateau sous forme de flèches, exactement comme dans le panneau Analyse. Le bouton <strong>?</strong> discret, dans la bande de badges, mène au dépôt du moteur <code>gammonNet &lt;https://github.com/kevung/gammonNet&gt;</code>_ ; l'attribution complète (réseau Strehl, configuration gammonNet) figure dans les Remerciements de l'aide.</p>
<p>Le panneau <strong>roule</strong> la position comme le panneau Analyse (voir Rollouts) : <em>Ctrl+clic</em> et <em>Maj+clic</em> sélectionnent des coups, le clic droit ouvre le même menu — rollout, copie de la position et de l'évaluation, ou de la position et des coups sélectionnés —, <em>r</em> lance ou arrête le rollout et <em>Échap</em> l'annule ; une barre fine suit les parties jouées, et chaque coup roulé porte son résultat dans la colonne <strong>Rollout</strong> (sans dés, sous la décision de videau). Le plateau étant un brouillon, le rollout n'est jamais enregistré : il se joue en mémoire, sans base ouverte comme sur une base en lecture seule, et son résultat ne s'affiche que tant que le plateau reste le même. Un rollout lancé avant une modification du plateau ne s'affiche pas sur la nouvelle position.</p>
<p>Le bouton <strong>Ajouter à la base</strong>, en tête de la bande de badges, enregistre dans la base la position posée sur le plateau ; <em>CTRL-S</em>, la commande <code>w</code> et le bouton <em>Enregistrer la position</em> de la barre d'outils font de même. Seule la position est écrite, jamais l'évaluation affichée ; si l'analyse automatique gammonNet est activée, son lot démarre ensuite, comme après un import. La barre d'état annonce le numéro de la position, y compris quand elle figurait déjà dans la base : elle y est alors marquée comme importée seule, et le filtre <em>Importée seule</em> (<code>s i</code>) la retrouve. Le panneau reste ouvert sur le même plateau, qui reste un plateau brouillon : on peut déplacer un pion et ajouter la variante, et quitter le panneau ramène à ce qu'on étudiait. Le bouton est désactivé tant qu'aucune base n'est ouverte, ou tant que la position ne peut pas être enregistrée (par exemple la position de départ du panneau, dont le joueur du haut a sorti tous ses pions) ; son infobulle en donne la raison.</p>
<p>L'utilisateur édite la position des pions sur l'ensemble du plateau, exactement comme en mode édition : clic gauche place un pion du joueur du bas, clic droit un pion du joueur du haut. Le second tableau, celui de la course, n'apparaît que lorsque la position obtenue est un bearoff pur (tous les pions des deux joueurs dans leur jan) ; sur toute autre position, seul le tableau des quatre colonnes communes (gain, gammon, backgammon, cubeless) répond, et la décision porte sur les pions ou sur un videau générique selon que des dés sont posés.</p>
<p>Dans chaque tableau de faits, une ligne par joueur — repérée par sa pastille de couleur, le joueur noir étant toujours en bas. Le premier porte, tant qu'aucun dé n'est posé, le gain, le gammon, le backgammon (probabilités, sans le signe %) et l'équité cubeless du joueur ; le second, sur une position de bearoff et dés posés ou non, l'EPC, le pip count, le wastage (différence entre l'EPC et le pip count), le nombre moyen de lancers et l'écart type. Lorsque les deux joueurs ont des valeurs à comparer, une ligne <strong>Δ</strong> donne les différences <em>signées</em> (bas − haut : négatif quand le joueur noir est en avance). Hors position de course, poser des dés fait donc disparaître les tableaux de faits eux-mêmes : les quatre colonnes qu'ils portaient viennent de changer d'axe, au trait, en tête de la liste des coups.</p>
<p>La décision de videau a toujours la même forme, quelle que soit l'origine des chiffres — table exacte, régime évalué ou évaluation gammonNet ordinaire : <strong>une ligne par option</strong>, dans l'ordre <em>pas de double</em>, <em>double/prend</em>, <em>double/passe</em>, avec son équité dans le référentiel de la position et son écart à la meilleure option. L'ordre ne change jamais, contrairement à la liste des coups : les trois options portent un nom, c'est donc le nom qu'on lit, pas le rang. La meilleure se reconnaît à sa mise en valeur et à sa cellule d'écart laissée vide. Lorsque le videau a déjà été retourné, les options se lisent <em>pas de redouble</em>, <em>redouble/prend</em>, <em>redouble/passe</em>.</p>
<p>Une dernière ligne donne le <strong>verdict</strong>. Il prend quatre valeurs : <em>pas de double</em>, <em>double, prend</em>, <em>double, passe</em> et <em>trop bon pour doubler</em>, cette dernière lorsque jouer la position rapporte davantage que d'encaisser le point : doubler serait alors une erreur pour la raison inverse de celle du simple <em>pas de double</em>. C'est aussi le seul endroit où le panneau dit qu'il n'y a <strong>pas</strong> de verdict, plutôt que de laisser croire à un calcul en cours :</p>
<ul>
<li><em>pas de décision</em> — le régime n'y a pas droit ; le verdict de videau n'est jamais estimé (voir le badge <em>estimé</em>) ;</li>
<li><em>non évaluable à ce score</em> — le moteur refuse la position, typiquement un score hors de l'horizon de la table d'équité de match, c'est-à-dire un camp à plus de 64 points à faire ;</li>
<li><em>videau adverse</em> et <em>videau mort (Crawford)</em> — le videau ne peut pas être retourné. Les équités restent affichées, à titre indicatif, mais aucune option ne porte d'écart : une erreur, c'est ce que coûte un choix, et il n'y a pas de choix.</li>
</ul>
<p>En money game, les règles <strong>Jacoby</strong> et <strong>Beaver</strong> actives sur la position apparaissent sous le tableau de videau, en petits badges à côté du verdict qu'elles changent : le verdict <em>pas de double</em> d'une position sous la règle Jacoby n'est pas le même calcul que sans elle, et rien d'autre à l'écran ne le disait.</p>
<p>Sous la règle <strong>Beaver</strong>, la ligne <em>double, prend</em> vaut la meilleure réponse de l'adversaire autre que passer : prendre, ou <em>beaver</em> — redoubler aussitôt en gardant le videau, la partie se jouant à quatre fois l'enjeu —, le doubleur répondant alors par un <em>raccoon</em> (huit fois l'enjeu, videau chez lui) quand il y gagne. Le verdict et les écarts se lisent sur cette ligne, comme sans la règle ; au score, la règle ne s'applique pas.</p>
<p>Un troisième badge, <strong>Videau max</strong>, apparaît lorsque l'identifiant d'origine plafonne le videau — au score comme en money game. Celui-là ne décrit pas le calcul affiché au-dessus : l'évaluateur intégré ne modélise pas de plafond, et le verdict est donc celui d'un videau libre. C'est justement pourquoi le badge est là : un videau plafonné est la seule raison visible pour laquelle blunderDB et eXtreme Gammon peuvent annoncer deux verdicts différents sur la même position.</p>
<p>Le <strong>joueur au trait</strong> et la <strong>position du videau</strong> s'éditent directement sur le plateau, comme en mode édition : cliquer le rectangle bearoff/score d'un joueur lui donne le trait ; cliquer le videau fait tourner centré → possédé bas → possédé haut (clic droit en sens inverse). La valeur du videau reste épinglée — en money game les équités sont exprimées en unités du videau courant, seul son propriétaire compte. L'analyse est recalculée aussitôt. En régime estimé, le badge lui-même est cliquable et ouvre directement l'onglet <em>Bearoff</em> de la configuration ; son infobulle explique pourquoi (verdict de videau non estimable, <code>ADR-0009 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md&gt;</code>__) et comment étendre le domaine exact.</p>
<p>Le <strong>score</strong> s'édite lui aussi directement sur le plateau, comme en mode édition : clic gauche sur le rectangle score d'un joueur décrémente son nombre de points à faire, clic droit l'incrémente. Sortir du score <em>money</em> (-1, -1) en éditant un seul camp aligne automatiquement l'autre camp sur la même valeur plutôt que de laisser un score incohérent. Sur une position de bearoff en régime <em>exact</em>, passer d'un score money à un score de match laisse la probabilité de gain telle quelle (une lecture en base, valable quel que soit le référentiel) mais bascule l'équité et le verdict de videau affichés vers ceux du régime <em>évalué</em> — la table exacte étant money par construction, elle ne sait pas répondre à la question posée au score. Le badge devient alors composite (« exact (gain) · évalué (videau) ») pour le dire explicitement.</p>
<p>Les <strong>dés</strong> s'éditent enfin de la même façon, et ce sont eux qui décident de la question posée : des dés posés font une décision de pions (la liste des coups candidats), pas de dés une décision de videau. Clic gauche sur un dé fait monter sa valeur (6 revient à 1), clic droit la fait descendre (1 revient à 6) ; cliquer un dé sur un plateau qui n'en a pas en pose deux d'un coup — un seul dé ne serait ni une décision de pions ni une décision de videau. Cliquer le rectangle d'un joueur retire les dés pour poser une question de videau, et le clic suivant sur un dé les remet tels qu'ils étaient.</p>
<p><em>RETOUR ARRIERE</em>, ou <em>Effacer la position</em> dans le menu d'un clic droit en dehors du plateau, efface la position : plateau vide, score money (-1, -1), pas de dés posés — des valeurs propres au panneau Eval, différentes de celles utilisées en mode édition (7 partout, dés 3-1), pour rester cohérentes avec ce que le panneau affiche par défaut. <em>Position de départ</em>, dans le même menu, pose les pions d'une partie neuve sur ces mêmes valeurs.</p>
<h4>Matrice du videau</h4>
<p>Une décision de videau n'est pas une propriété du damier. Les mêmes pions, le même compte de pips, se doublent à 2-away/4-away et ne se doublent pas à 4-away/2-away ; un joueur qui a appris la réponse money n'a appris qu'une case d'une grille. Le panneau Eval montre la case que la position porte ; la <strong>matrice du videau</strong> montre la grille entière.</p>
<p>La commande <code>cm</code> l'ouvre sur la position affichée. Chaque case donne le verdict à un score : la ligne est le nombre de points qu'il reste à faire au joueur au trait, la colonne celui qu'il reste à faire à son adversaire. Les quatre verdicts s'écrivent <em>PD</em> (pas de double), <em>DP</em> (double, prend), <em>DR</em> (double, refuse) et <em>TB</em> (trop bon) ; une case que le moteur refuse porte un point d'interrogation et dit pourquoi au survol, qui donne aussi les trois équités de la case. Trois longueurs de match sont proposées : 5, 7 et 9 points.</p>
<p>La case du score que la position porte réellement est encadrée, et ses deux en-têtes de ligne et de colonne soulignés : la lecture part de là, « ma case, et autour d'elle ». Elle l'est dès que les deux scores <em>away</em> de la position tiennent dans la grille affichée ; changer de longueur la déplace ou la retire. Une position money, la partie Crawford, ou un <em>away</em> au-delà de la grille n'en désignent aucune : il n'y a pas de case à montrer, et en montrer une approchante serait faux.</p>
<p>Le score de la position est remplacé par celui de chaque case ; son <strong>videau</strong>, lui, est conservé. La grille répond à « à quel score retournerais-je <em>ce</em> videau », pas à ce que ferait une position centrée. Elle est post-Crawford d'un bout à l'autre : pendant la partie Crawford le videau n'est pas en jeu, et une colonne de « vous ne pouvez pas doubler » ne dirait rien de la position.</p>
<p>Chaque case est une recherche à part entière. Le moteur tient compte du score — il ne joue pas la même partie à 2-away qu'à 7-away —, donc une seule recherche relue à travers des équités de match différentes serait fausse exactement là où le score compte. La grille arrive d'abord en 0-ply, puis se recalcule à la profondeur d'affichage configurée une fois la fenêtre au repos : la même escalade que le reste du panneau, pour une grille de 9 points qui coûte environ une seconde et demie.</p>
<p>La même grille se calcule hors de l'interface, avec la commande cubematrix de la ligne de commande.</p>
<h4>Amener une position dans le panneau Eval</h4>
<p>Le panneau s'ouvre par défaut sur une position de bearoff, mais l'étude part le plus souvent d'une position déjà en main. Trois gestes l'y amènent :</p>
<ul>
<li><strong>Clic droit sur le plateau</strong>, dans un panneau d'analyse ou pendant la navigation d'un match, puis <em>Évaluer cette position</em> : le panneau Eval s'ouvre directement sur cette position, telle qu'elle est affichée ; <em>Évaluer le miroir de cette position</em> l'y ouvre vue de l'autre camp. Le menu contextuel n'apparaît pas dans le panneau Eval ni dans le panneau Recherche, où le bouton droit sert déjà à poser les pions de l'autre couleur.</li>
<li><strong>CTRL-C puis CTRL-V</strong> : copier la position depuis le panneau d'analyse, puis la coller une fois dans le panneau Eval. Le collage accepte aussi un identifiant venu d'ailleurs — un XGID (eXtreme Gammon, GNU Backgammon, une autre instance de blunderDB) ou un OGID (OpenGammon) : il suffit qu'il soit dans le presse-papier.</li>
<li><strong>La commande</strong> <code>import XGID=…</code> (ou <code>import OGID=…</code>) pour le cas où l'identifiant n'est pas dans le presse-papier mais dans un message, sur un forum lu dans un terminal, ou produit par un script. C'est le même verbe qu'<code>import</code> tout court : sans argument il ouvre un sélecteur de fichiers, avec un argument il lit l'identifiant. Le chemin est ensuite identique à celui du collage — même lecture, même déduplication, même ouverture de la position importée.</li>
</ul>
<p>Un OGID ne porte qu'une position : ni évaluation, ni commentaire. La position arrive donc sans analyse, exactement comme un XGID nu, et l'évaluateur intégré peut la combler ensuite.</p>
<p>Dans un OGID, la partie Crawford se lit au <code>C</code> qui suit la longueur du match (<code>7C</code>) : sans lui, un joueur à un point du but est après la Crawford.</p>
<p>Le plateau du panneau Eval est un brouillon : la position y arrive sans son identifiant de base, de sorte qu'aucune modification faite ici ne peut réécrire l'enregistrement dont elle provient. Toutes les éditions habituelles du plateau y restent disponibles (pions, videau, dés, score), et l'évaluation suit chaque modification.</p>
<p>Dans l'autre sens, <em>CTRL-C</em> copie le plateau du panneau Eval dans le presse-papier, avec un XGID recalculé à partir des pions posés — donc collable directement dans eXtreme Gammon ou dans une autre instance de blunderDB. Seule la position voyage : l'évaluation affichée par le panneau n'est pas un enregistrement de la base et n'accompagne pas la copie.</p>
<p>En quittant le panneau Eval, la position consultée auparavant est restaurée : le brouillon n'est jamais enregistré tout seul.</p>
<p>Lorsque la position est un bearoff pur (tous les pions des deux joueurs dans leur jan) et qu'aucun dé n'est posé, la décision de videau affiche, pour le joueur au trait :</p>
<ul>
<li>en régime <em>exact</em> : les équités money (cubeless, sans double, double/prend, double/passe) et le <strong>verdict de videau money</strong> (pas de double, double/prend, double/passe ou trop bon pour doubler) — hors score de match, voir plus haut pour le cas du score,</li>
<li>en régime <em>évalué</em> : les mêmes équités et le même verdict à quatre valeurs, mais <strong>joués par gammonNet</strong> (recherche + modèle de videau Janowski) plutôt que lus dans une table — disponibles <strong>même au score de match</strong>, ce que le régime estimé n'a jamais pu offrir ;</li>
<li>en régime <em>estimé</em> : le verdict de videau n'est alors volontairement pas affiché — seule la probabilité de gain, dans le tableau de faits, accompagnée de sa marge d'erreur, reste disponible.</li>
</ul>
<p>Dès que des dés sont posés sur une position de course, cette décision de videau <em>avant le jet</em> disparaît — le plateau demande alors une décision de pions, pas de videau — mais la probabilité de gain, elle, reste un fait de la position, pas une décision : elle rejoint la ligne <em>avant le jet</em> en tête de la liste des coups, à côté de l'EPC qui, lui, reste affiché juste à gauche.</p>
<p>Un badge indique le régime : <strong>exact</strong> (valeur lue dans une base de données two-sided), <strong>évalué · &lt;profondeur&gt;</strong> (joué par gammonNet — la profondeur affichée est celle qui a effectivement produit le chiffre montré), <strong>estimé ± marge</strong>, ou, au score de match dans le domaine exact, <strong>exact (gain) · évalué (videau)</strong> — voir plus haut. Le régime exact l'emporte partout où il est disponible ; sinon le régime évalué s'affiche dès qu'il a fini de calculer, remplaçant en place le régime estimé montré pendant l'attente. Voir Méthodologie et hypothèses du panneau Eval pour la définition précise des trois régimes et de leurs hypothèses.</p>
<p><strong>Élargir le domaine exact.</strong> La table calculée au premier lancement couvre 6 pions par joueur. Deux moyens d'aller au-delà, dans l'onglet <em>Bearoff</em> de la configuration :</p>
<ul>
<li>calculer une table deux faces plus large — jusqu'à TS-06-15 si la machine a la mémoire pour. L'onglet annonce la taille, la mémoire et le temps sur cette machine avant de commencer, et le calcul se met en pause et se reprend. Un calcul annulé laisse un fichier <code>.part</code> qui n'est jamais lu comme une table ;</li>
<li>indiquer un fichier <code>.bd</code> two-sided de gnubg quelconque. La base au domaine le plus large l'emporte automatiquement.</li>
</ul>
<p><strong>Le plateau du panneau est un brouillon, et il est retenu.</strong> Quitter le panneau Eval puis y revenir retrouve la position sur laquelle on l'a laissé, et non le plateau de sortie par défaut : ce dernier n'est servi qu'à la première ouverture de la session. Envoyer une position de la base vers le panneau l'emporte sur ce souvenir, et <em>RETOUR ARRIERE</em> rend le plateau par défaut à tout moment. Rien n'est enregistré dans la base au passage — le brouillon n'a pas d'identité de position, et son évaluation est recalculée à l'arrivée plutôt que transportée.</p>
<p><strong>Mode défi.</strong> La case <em>Défi</em>, dans la bande de badges, active un mode entraînement : à chaque modification de la position, les valeurs de trois zones sont masquées (remplacées par « ··· ») ; un clic sur une zone révèle cette zone seulement. Sans dés, ce sont la ligne du joueur du bas, la ligne du joueur du haut et la décision de videau — la ligne Δ n'apparaît qu'une fois les deux lignes joueurs révélées. Le bloc de décision garde alors ses trois lignes : ce sont ses valeurs, son verdict et la mise en valeur de la meilleure option qui disparaissent, faute de quoi l'exercice se résoudrait en cherchant la ligne en gras. Dés posés sur une position de course, la ligne EPC de chaque joueur se masque comme avant, mais la troisième zone couvre alors la ligne <em>avant le jet</em> et la liste des coups <strong>ensemble</strong> : la liste étant classée du meilleur coup au pire, la révéler partiellement en donnerait déjà la réponse. Dés posés hors position de course, cette même zone unique couvre à elle seule tout ce que le panneau affiche. On peut ainsi s'entraîner à estimer l'EPC de chaque camp, puis à se prononcer sur le videau ou sur le coup à jouer, avant de vérifier. Le réglage est mémorisé.</p>
<p><strong>Sans base ouverte.</strong> Le panneau Eval n'a pas besoin de base : une fois l'écran d'accueil écarté, on y pose, colle (<em>CTRL-V</em>) et modifie une position, on l'évalue, on la roule, on la copie en texte (<em>CTRL-C</em>) ou en image (<em>CTRL-X</em>, et <em>CTRL-X CTRL-X</em> avec l'évaluation). Le clic droit dans le panneau propose aussi <em>Enregistrer l'image (SVG)…</em> et <em>Enregistrer l'image (PNG)…</em>. Le plateau de l'onglet Recherche est lui aussi un brouillon : il s'édite et reçoit un collage sans base, et l'on passe de l'un à l'autre sans rien perdre ; lancer la recherche, elle, demande une base. Seuls <em>Ajouter à la base</em> et <em>CTRL-S</em> exigent une base dans le panneau, et la barre d'état le dit. Aucune base n'est créée en coulisse. Dans le panneau Eval comme dans l'onglet Recherche, la commande <code>import XGID=…</code> (ou <code>import OGID=…</code>) pose la position sur le plateau brouillon, sans base. L'onglet <strong>Entraînement</strong> s'ouvre lui aussi sans base : <em>Scores</em>, <em>Pions</em>, <em>Bearoff</em> et <em>Évaluation</em> sont disponibles, <em>Décision</em> est désactivé avec la raison affichée, et la session terminée n'est pas enregistrée (le journal l'indique). La Direction de tournoi exige une base ouverte.</p>
<p>Pour fermer le panneau Eval, appuyer sur <em>CTRL-E</em> ou basculer sur un autre onglet.</p>
<h4>Méthodologie et hypothèses du panneau Eval</h4>
<p>Chaque valeur affichée par le panneau repose sur des hypothèses précises, énoncées ici exhaustivement.</p>
<p><strong>Domaine.</strong> La <em>zone course</em> — probabilité de gain et verdict de videau — ne traite que le bearoff pur : tous les pions restants des deux joueurs dans leur jan intérieur. La position est évaluée <em>avant le lancer</em> ; les dés éventuellement posés sont ignorés.</p>
<p>Les <strong>blocs EPC</strong>, eux, vont plus loin : un camp obtient son EPC dès que son pion le plus éloigné tient dans la table une face chargée. Avec la table par défaut (six points) c'est l'ancienne règle du jan ; avec une table à huit points, calculée depuis l'onglet <em>Bearoff</em>, un camp dont un pion est sur la 8 est traité comme les autres. Rien n'est extrapolé : un pion un point trop loin n'a simplement pas d'EPC, exactement comme un pion sur la 7 n'en avait pas avant. Quand la table qui a répondu n'est pas celle à six points, son nom apparaît dans le coin du bloc course (« OS-08 ») — sans lui, on lirait « six » par défaut et on croirait le camp entièrement rentré.</p>
<p><strong>Blocs EPC (toujours exacts).</strong> L'EPC, le nombre moyen de lancers et l'écart type proviennent de la distribution exacte du nombre de lancers pour sortir tous les pions, lue dans la base one-sided de GNUbg (6 à 10 points, 15 pions, calculée sur la machine). EPC = lancers moyens × 49/6 (49/6 ≈ 8,167 est la moyenne exacte de pips par lancer, doubles comptés quatre fois) ; wastage = EPC − pip count. L'unique idéalisation est le <em>jeu one-sided optimal</em> : chaque joueur minimise ses propres lancers en ignorant l'adversaire — c'est la définition standard de l'EPC.</p>
<p><strong>Probabilité de gain, régime exact.</strong> Lecture directe dans la base two-sided disponible la plus large (TS-06-06 calculée au premier lancement, fichier externe, ou TS-06-11 calculée depuis l'onglet <em>Bearoff</em>). Ces bases résultent d'une analyse rétrograde complète sous jeu two-sided optimal des deux camps : aucune hypothèse supplémentaire, erreur limitée à la quantification (&lt; 0,002 %).</p>
<p><strong>Probabilité de gain, régime estimé.</strong> Hors du domaine de la base : la probabilité est obtenue en convoluant les deux distributions one-sided (le joueur au trait gagne si son nombre de lancers est inférieur ou égal à celui de l'adversaire), puis en appliquant une correction polynomiale figée, calibrée hors ligne contre la base TS-06-11. Trois hypothèses :</p>
<ul>
<li><strong>indépendance</strong> des deux processus de sortie — structurelle en course, sans contact il n'y a aucune interaction ;</li>
<li><strong>jeu one-sided optimal des deux camps</strong> — c'est <em>l'approximation</em> : en réalité le joueur mené dévie pour jouer la variance et le meneur pour la sécurité. L'effet mesuré est un biais antisymétrique (la convolution exagère l'avance du meneur) que la correction absorbe statistiquement ;</li>
<li>la <strong>correction</strong> a été calibrée et validée sur le domaine de l'oracle (jusqu'à 11 pions par joueur). Erreur résiduelle mesurée : écart type 0,05 %, 99e centile 0,17 %, maximum observé 0,9 % (en points de probabilité de gain). <strong>Au-delà de 11 pions par joueur, cette borne est extrapolée</strong> — la tendance est monotone mais aucun oracle ne la certifie.</li>
</ul>
<p><strong>Équités et verdict de videau (régime exact seulement).</strong> Les équités affichées sont celles du <strong>money game, sans Jacoby</strong>, dans le référentiel de la littérature du bearoff. Dans le domaine ≤ 11 pions par joueur, les gammons sont impossibles (chaque camp a déjà sorti au moins 4 pions) : ce n'est pas une approximation. Le verdict (pas de double / double, prend / double, passe) est reconstruit exactement des équités stockées, selon la règle de GNUbg, validée trait pour trait contre son analyse.</p>
<div class="admonition note">
<p>Les équités cubeful supposent un <strong>jeu de videau optimal des deux camps jusqu'au bout</strong> : les recubes futurs sont intégralement valorisés (analyse rétrograde complète). Dans les courses très volatiles de fin de partie, la cascade de recubes mange presque tout l'avantage du camp au trait — les équités « sans double » et « double/prend » peuvent alors être proches de zéro là où un moteur comme XG, dont le modèle de videau ne valorise pas cette cascade, affiche des valeurs proches du dead cube (par exemple 2 pions sur le point 3 contre 2 pions sur le point 2 : 62 % de gain, D/T exact +0,006 contre +0,475 chez XG). La <strong>décision</strong> affichée, elle, coïncide avec celle des moteurs.</p>
</div>
<p><strong>Probabilité de gain et verdict, régime évalué.</strong> Hors du domaine exact, la probabilité de gain provient de la sortie brute de gammonNet (recherche 0-ply, puis à la profondeur configurée, jamais lue dans une table), et le verdict d'un « Decide » Janowski appliqué à cette sortie — la recherche <em>joue</em> la trajectoire au lieu d'en résumer un instantané, ce qui est précisément ce que le régime estimé ne pouvait pas faire (voir plus bas) et permet, seul des trois régimes avec l'exact, un verdict <strong>au score de match</strong>.</p>
<p>Ce régime a été mesuré, pas seulement supposé, contre la table two-sided intégrée (<code>TestEvalMeasure</code>, 4000 décisions money échantillonnées, paramètres canoniques 2-plis k=12) : accord de verdict money <strong>93,4 %</strong> (3735/4000), ventilé par distance au point de prise de gammonNet — 61,1 % à moins de 1 % du point de prise (la zone la plus sensible à un pile ou face), 88,3 % entre 1 et 5 %, 91,5 % entre 5 et 10 %, 94,0 % entre 10 et 20 %, 94,4 % au-delà. Écart de probabilité de gain : moyenne 0,85 %, médiane 0,44 %, 95e centile 3,21 %, maximum 8,30 %. Écart d'équité cubeful : moyenne 0,039, médiane 0,018, 95e centile 0,151, maximum 0,406. La forme est celle attendue : l'essentiel du désaccord se concentre exactement au point de prise, où deux méthodes légitimement différentes divergent le plus sur une décision serrée — pas une erreur diffuse qui coûterait de l'équité partout.</p>
<p>Cette mesure porte sur des décisions <strong>money</strong>, en course. Le verdict au score de match — que ce régime est seul à savoir rendre — et les positions de contact n'ont pas de mesure publiée : ce qui précède ne se transporte pas à ces cas.</p>
<p><strong>Pourquoi pas plus profond que 2 plis ?</strong> Parce que la mesure dit que cela ne rapporte rien. Une décision de pions coûte 99 ms à 2 plis et 8,4 s à 3 plis sur la même machine — <strong>85 fois plus</strong>. Sur quarante décisions réelles rejouées aux deux profondeurs, la recherche plus profonde a changé d'avis <strong>deux fois</strong>, et les deux fois le gain qu'elle s'attribuait à elle-même valait au plus 0,0005 d'équité normalisée : deux ordres de grandeur sous 0,020, le seuil à partir duquel eXtreme Gammon parle d'erreur. Par décision, tous cas confondus, le gain est de 0,0000.</p>
<p>Les 2 plis restent donc le défaut ; 3 et 4 plis se choisissent dans l'onglet <em>gammonNet</em> de la configuration. Il ne s'agit pas de dire que 3 plis ne vaut rien en général, mais que sur <em>ce</em> réseau, avec le filtre canonique, il ne paie pas l'attente de quelqu'un devant un panneau. La mesure est reproductible (<code>TestThreePlyMeasure</code>) et la conclusion se rejugera si le réseau change.</p>
<p><strong>Pourquoi le verdict estimé n'existe-t-il pas ?</strong> Ce qui suit vise spécifiquement la méthode par <em>convolution</em> (régime estimé), pas le régime évalué ci-dessus : l'équité cubeful est un problème de <em>trajectoire</em> (quand doubler), qu'aucun résumé statistique de la position ne capture — le meilleur modèle statique mesuré laisse une erreur résiduelle (écart type 0,016 d'équité, maximum 0,20) qui suffit à inverser toutes les décisions serrées. De même, la conversion du verdict au score de match via une table d'équités de match a été mesurée insuffisante (12 % de désaccords avec l'analyse 2-ply de GNUbg, avec de vraies bourdes). Un verdict faux affiché avec aplomb étant pire que pas de verdict, la convolution n'a jamais eu le droit d'afficher de verdict — c'est une recherche qui joue la trajectoire, pas un résumé statistique, qui comble ce trou.</p>
<div class="admonition note">
<p>Les bases de bearoff sont des tables mathématiques immuables. blunderDB les calcule lui-même, à l'identique de l'outil <code>makebearoff</code> de GNUbg — octet pour octet — dans l'onglet <em>Bearoff</em> de la configuration ou avec <code>blunderdb bearoff generate</code>.</p>
</div>
<h3>Panneau Anki</h3>
<p>Le panneau <strong>Anki</strong> (<em>CTRL-K</em>) permet d'étudier par répétition espacée, en utilisant l'algorithme FSRS. Une carte pose une question : le plus souvent une position, tirée d'une collection ou d'une recherche ; ce peut aussi être un score.</p>
<p><strong>Création de paquets :</strong> Cliquez sur <strong>+ Nouveau paquet</strong> pour créer un paquet à partir d'une collection ou des résultats de recherche courants. Les paquets basés sur une recherche se synchronisent automatiquement à l'activation de l'onglet Anki.</p>
<p><strong>Un paquet de fiches de score.</strong> La troisième source, <em>Fiches de score</em>, ne demande rien d'autre qu'un nom : blunderDB remplit le paquet avec les 36 scores non ordonnés de 2 à 9 away, et la carte d'un score est la fiche que l'exercice Scores affiche — points de prise et valeurs de gammon, les deux faces. Ce paquet n'existe que si vous le créez : 36 cartes dues le premier jour sont une dette de révision, et elle se contracte volontairement. Le bouton de synchronisation le régénère.</p>
<p>Les deux endroits ne font pas le même travail. L'exercice Scores fait retrouver ces nombres <strong>sous la pendule</strong> et mesure la vitesse ; le paquet les fait <strong>tenir dans le temps</strong> et n'en mesure rien. Les deux histoires restent séparées : le journal de l'Entraînement ignore les révisions Anki, et les statistiques d'Anki ignorent les sessions d'Entraînement.</p>
<p><strong>Révision :</strong> Sélectionnez un paquet puis cliquez sur <em>Étudier</em> (ou double-cliquez sur un paquet) pour commencer la révision des cartes dues. Une carte de position affiche la position sur le plateau ; une carte de score annonce le score et laisse le plateau tel qu'il est. Évaluez votre rappel avec les touches <em>1</em> (À revoir), <em>2</em> (Difficile), <em>3</em> (Correct), ou <em>4</em> (Facile). Appuyez sur <em>Esc</em> pour arrêter et revenir à la liste des paquets.</p>
<p>Deux comptes portent des noms distincts : la colonne <strong>Échues</strong> de la liste compte toutes les cartes dont l'échéance est passée, y compris les cartes suspendues ou enterrées ; le chiffre du bouton <em>Étudier</em> ne compte que celles qui sont disponibles maintenant, et peut donc être plus petit.</p>
<p><strong>Les décisions de videau font deux cartes, enchaînées.</strong> Une décision de videau est deux questions — « double ? », puis « prend ? » — et blunderDB les enregistre depuis toujours comme deux positions. Un paquet qui n'en sélectionne qu'une moitié reçoit l'autre : la décision est complétée, pas augmentée. Et quand les deux sont dues, la seconde vient <strong>immédiatement</strong> après la première.</p>
<p>Chacune garde sa propre note et son propre calendrier : ce ne sont pas deux temps d'une même carte, ce sont deux cartes. L'enchaînement n'avance aucune échéance — il ordonne les cartes déjà dues, rien de plus. Les deux naissant ensemble, elles sont dues ensemble la première fois, et c'est là qu'il sert.</p>
<p><strong>Afficher la réponse :</strong> La carte pose une question — quel coup jouer, quelle action de videau, ou quels nombres porte un score. Réfléchissez, puis appuyez sur <em>ESPACE</em> (ou cliquez sur la zone masquée) pour dévoiler la réponse : l'analyse enregistrée de la position, telle que l'onglet Analyse la présente, ou la fiche du score, entière. Sur une fiche il n'y a rien à cocher : Anki planifie une mémoire, il ne mesure pas un calcul — c'est l'Entraînement qui compte les fautes. Elle apparaît sous les boutons d'évaluation, qui restent à leur place et à portée. Cliquer sur un coup de la liste le montre sur le plateau.</p>
<p>Rien ne vous oblige à dévoiler la réponse pour évaluer : si vous êtes sûr de vous, les touches <em>1</em> à <em>4</em> restent actives. La réponse se remasque à la carte suivante, mais pas si vous changez simplement d'onglet — allez consulter le panneau Eval ou le commentaire de la position, elle vous attendra au retour.</p>
<p>Une position dépourvue d'analyse enregistrée l'indique directement, sans zone masquée.</p>
<p><strong>Répondre au damier.</strong> Par défaut, vous vous notez vous-même. Dans les Paramètres d'un paquet de positions, cochez <em>Répondre au damier</em> : pour une carte de pions, vous jouez alors le coup sur le damier, comme dans l'exercice Décision, puis <em>Valider</em>. Le moteur juge le coup contre l'analyse enregistrée, dévoile la réponse et <strong>propose une note</strong> : <em>Facile</em> pour une bonne réponse rapide, <em>Correct</em> pour une bonne réponse plus lente, <em>Difficile</em> pour une erreur sous le seuil du blunder, <em>À revoir</em> pour un blunder ou un coup illégal. La note proposée est en surbrillance ; vous gardez la main et notez ce que vous voulez avec <em>1</em> à <em>4</em>. Un coup légal que l'analyse ne classe pas ne propose rien. Les cartes de videau, les cartes de score et les paquets de fiches de score restent en auto-notation. Dévoiler la réponse sans jouer abandonne le coup.</p>
<p><strong>Limiter la séance.</strong> Par défaut, une séance de révision va jusqu'au bout des cartes dues. Vous pouvez la borner à un nombre de cartes, par paquet, dans les Paramètres : cochez <em>Limiter la séance</em> et indiquez combien de cartes une séance doit servir. Quand la limite est atteinte, la séance s'arrête en le disant — le message distingue « limite atteinte, tant de cartes encore dues » d'une file réellement épuisée. Pour continuer malgré tout, l'entraînement libre est là : il sert d'autres positions sans rien modifier au planning.</p>
<p>Une limite de <strong>0</strong> ne sert aucune carte : c'est un état à part entière, utile pour geler un paquet le temps de préparer un tournoi, et ce n'est pas la même chose que « pas de limite ». Le bouton <em>Étudier</em> est alors inactif.</p>
<p>La limite porte sur la <strong>séance</strong>, pas sur la journée. Un paquet blunderDB est bâti sur une collection ou une recherche : c'est un corpus fini, introduit en quelques séances, dont le volume quotidien est déjà borné par sa taille. Un plafond par jour n'y mordrait jamais, ou bien créerait un retard sur un paquet qui tenait en une séance.</p>
<p><strong>Entraînement libre (cram) :</strong> Le bouton <em>Entraînement</em>, à côté de <em>Étudier</em>, lance une session d'entraînement libre : des positions aléatoires du paquet vous sont présentées sans tenir compte de l'échéancier FSRS. Ce mode <strong>ne modifie jamais le planning de révision espacée</strong> — idéal pour s'échauffer avant un tournoi ou réviser intensément un paquet thématique sans perturber son ordonnancement. Une pastille <em>Libre</em> remplace l'état de la carte et un bouton <em>Suivant</em> (touches <em>1</em> à <em>4</em>) fait défiler les positions. <em>Esc</em> revient à la liste sans enregistrer de session interrompue.</p>
<p><strong>Écarter une carte, sans la noter.</strong> Pendant une révision, un clic droit sur l'en-tête de la carte ouvre trois gestes qui la sortent de la séance sans rien dire au planificateur :</p>
<ul>
<li><strong>Suspendre</strong> — la carte garde son échéancier et ne remonte plus jamais tant qu'elle est suspendue. C'est la manière de mettre de côté une carte fausse, ou pas encore utile, sans perdre l'historique qui y est attaché.</li>
<li><strong>Enterrer</strong> — la carte disparaît jusqu'au lendemain. Contrairement à la suspension, cela ne dit rien de sa valeur : c'est pour celle que l'on vient de voir ailleurs, ou que l'on préfère ne pas croiser deux fois dans la soirée.</li>
<li><strong>Retirer</strong> — la carte quitte le paquet, après confirmation. La position, elle, reste dans la base : un paquet est une liste d'étude sur la bibliothèque, jamais une copie de celle-ci.</li>
</ul>
<p>Aucun de ces trois gestes n'enregistre de note : une carte écartée n'est pas une carte répondue, et elle ne compte pas dans le décompte de la séance.</p>
<p><strong>Journal des révisions.</strong> Dans les Paramètres d'un paquet, le bouton <em>Journal des révisions</em> montre ce que le planificateur a été <strong>dit</strong> — date, sujet (le numéro de la position, ou le score), note, état, intervalle accordé — par opposition à ce qu'il prévoit. C'est le seul endroit où une note entrée par erreur se voit. Elle ne s'y corrige pas : l'échéancier reste hors de portée, et cette règle est précisément ce qui rend le journal utile — on ne peut pas réécrire le passé, mais on peut savoir ce qu'il a été.</p>
<p><strong>Arrêt/Reprise :</strong> Vous pouvez interrompre une session de révision à tout moment avec <em>Esc</em>. Le bouton change en <em>Reprendre</em> et affiche votre progression. Cliquez dessus pour reprendre là où vous vous êtes arrêté.</p>
<p><strong>Gestion des paquets :</strong> Utilisez les boutons d'action pour renommer, synchroniser, réinitialiser ou supprimer des paquets (confirmation demandée pour ces deux dernières actions). Les paramètres FSRS (rétention cible, intervalle maximum, aléa) peuvent être configurés par paquet dans les Paramètres (icône engrenage).</p>
<p><strong>Rétention : la cible et la mesure.</strong> La <em>rétention cible</em> est votre choix sur le compromis entre charge de travail et qualité du rappel : plus elle est haute, plus les intervalles raccourcissent et plus vous révisez. En regard, les Paramètres affichent la <strong>rétention mesurée</strong> sur vos propres révisions — une information, jamais un pilotage : blunderDB ne modifie pas votre cible pour poursuivre votre taux de réussite. Sous une vingtaine de révisions, la mesure n'est pas affichée : elle se lirait comme un fait alors qu'elle n'est que du bruit.</p>
<p>Changer la rétention <strong>n'est pas rétroactif</strong> : chaque carte adopte le nouveau rythme à sa prochaine révision, et les échéances déjà fixées ne bougent pas. L'effet est donc progressif, et invisible le jour même.</p>
<p>L'<em>intervalle maximum</em> borne l'espacement. Un paquet créé récemment démarre à un an : une position que l'algorithme reporterait de plusieurs années a quitté le paquet sans que vous l'ayez décidé, et votre propre jeu change plus vite que cela. Les paquets plus anciens conservent la valeur qu'ils avaient.</p>
<h3>Panneau Entraînement</h3>
<p>Le panneau <strong>Anki</strong> fait réviser ce qui se <strong>retient</strong> ; le panneau <strong>Entraînement</strong> fait travailler ce qui se <strong>calcule</strong>, sous la pendule. Il s'ouvre par <code>CTRL-J</code>, par le bouton de la barre d'outils placé juste après « Position aléatoire », ou par la commande <code>train</code>. La fiche de score relève des deux : elle se calcule ici et se retient dans un paquet de fiches de score.</p>
<p>Au repos, le panneau montre le lanceur et le bilan des sessions passées.</p>
<h4>Le lanceur</h4>
<p>Trois choix, puis « Démarrer » :</p>
<ul>
<li>l'<strong>exercice</strong> — <em>Scores</em>, <em>Comptage des pips</em>, <em>Bearoff</em>, <em>Évaluation</em> ou <em>Décision</em> ;</li>
<li>la <strong>source</strong> de la question, quand l'exercice en a plusieurs — <em>Vivier</em> (des formes canoniques de l'exercice), <em>Plateau</em> (la position telle qu'elle est) ou <em>Base</em> (une position de la liste parcourue) ;</li>
<li>la <strong>limite par question</strong> — aucune, 15, 30 ou 60 secondes.</li>
</ul>
<p>La source choisie est mémorisée pour chaque exercice, d'une session à l'autre.</p>
<p>La liste parcourue peut venir des <em>erreurs récurrentes</em> du panneau Stats : un clic sur « Quiz sur ce groupe » la remplace par les positions du groupe et démarre l'exercice Décision.</p>
<p><code>train scores</code>, <code>train pips</code>, <code>train bearoff</code>, <code>train evaluation</code> et <code>train decision</code> ouvrent le panneau et démarrent directement ; <code>train tp</code> et <code>train takepoint</code> sont des synonymes de <code>train scores</code>, <code>train epc</code> de <code>train bearoff</code>, <code>train quiz</code> de <code>train decision</code>.</p>
<h4>Les cinq exercices</h4>
<p><strong>Scores</strong> tire au sort l'un des 36 scores non ordonnés de 2 à 9 away et affiche une <strong>fiche de score</strong> : deux colonnes — <em>Vous</em> et <em>L'adversaire</em> — et sept lignes — le point de prise au videau 2 puis au videau 4, chacun en course longue et au dernier lancer, puis la valeur du gammon aux videaux 1, 2 et 4.</p>
<p>Une ligne de consigne rappelle le geste — estimer chaque nombre de tête, puis <em>Révéler</em>, puis cliquer ceux qu'on a ratés —, et le plateau montre le score tiré sur une table vide.</p>
<p>Chaque colonne ne porte que les cases que les tables de référence — celles qu'affichent les commandes <code>tp2_live</code>, <code>tp2_last</code>, <code>tp4_live</code>, <code>tp4_last</code>, <code>gv1</code>, <code>gv2</code> et <code>gv4</code> — définissent pour sa face : trois nombres à 2a-2a, quatorze au plus, et une seule colonne à score égal. Une ligne qu'aucune des deux faces ne définit ne figure pas sur la fiche — il n'y a donc aucune case « sans objet » à deviner. Les deux faces sont là parce qu'une décision de videau au score a besoin des deux : le point de prise corrigé combine les valeurs de gammon des deux joueurs, et c'est le point de prise de l'adversaire qui dit si votre double passe.</p>
<p><strong>Comptage des pips</strong> demande le compte de pions des <strong>deux</strong> camps. Le pipcount du plateau est masqué tant que la question est ouverte ; « Révéler » l'affiche — <strong>même si vous aviez masqué le pipcount</strong> avec <code>p</code>, sans quoi la réponse resterait invisible et l'exercice invérifiable. C'est un masque et non un réglage : votre choix n'est pas modifié, et il reprend la main dès la question suivante. La source <em>Plateau</em> pose une question sur la position affichée, et une seule ; la source <em>Base</em> tire une nouvelle position à chaque question et l'amène sur le plateau.</p>
<p><strong>Bearoff</strong> demande l'<strong>EPC des deux camps</strong> — le compte de pions effectif, celui qui ajoute au pipcount le gaspillage des pions qui sortent en trop. C'est le domaine où le moteur est exact, et celui où l'EPC se distingue vraiment du compte de pions.</p>
<p>Chaque question est <strong>engendrée</strong> : le moteur part d'une graine et joue quelques lancers, et c'est l'instantané qui vous est posé. Un placement au hasard n'aurait pas les trous, les piles basses et les asymétries d'un vrai bearoff. La graine vient du <em>Vivier</em> (une position de bear-in, puis zéro à dix plis), du <em>Plateau</em> (la position affichée, puis un à quatre plis — jamais zéro, puisque vous venez de la voir) ou de la <em>Base</em> (une position de la liste parcourue, telle quelle : elle est déjà réelle).</p>
<p>Le domaine de l'exercice : les <strong>deux camps entièrement dans leur jan</strong>, de <strong>4 à 15 pions</strong> par camp et le reste sorti, videau au centre, en partie d'argent. Une graine qui n'y entre pas est <strong>refusée en le disant</strong>, et rien ne démarre — aucune adaptation silencieuse : jouer jusqu'à ce que le contact se rompe vous donnerait une position que vous n'avez pas choisie. Un plateau vide fait exception : la question vient alors du vivier, et le panneau dit pourquoi.</p>
<p>L'exercice a besoin de la table de bearoff à un camp ; tant qu'elle s'engendre en arrière-plan (voir Configuration), il le dit plutôt que de poser une question sans réponse.</p>
<p><strong>Évaluation</strong> demande ce que vaut une position : les <strong>chances de gain du joueur au trait</strong>, en pourcentage, et l'<strong>action de videau</strong> — <em>Pas de double</em>, <em>Double, prend</em> ou <em>Double, passe</em>. Ce sont les deux nombres que montre le panneau Eval, demandés avant d'être montrés. Le domaine est <strong>toute position</strong> : une course comme une position de contact, en <strong>partie d'argent</strong>.</p>
<p>Comme pour Bearoff, la question est <strong>engendrée</strong> : la graine vient du <em>Vivier</em> (une position où le contact vient de se rompre, puis zéro à dix plis joués par le moteur), du <em>Plateau</em> (la position affichée, puis un à quatre plis ; la question se pose en partie d'argent, videau au centre, quel que soit le score de la graine) ou de la <em>Base</em> (une position de la liste parcourue, telle quelle). Une position de la base ne convient que si elle est une décision de videau en partie d'argent — sans dés, le videau au centre ou au joueur au trait — ; le tirage passe sinon à la suivante, et quand aucune ne convient, l'exercice le dit. Un plateau qui n'est pas une position de partie — pas quinze pions par camp, ou une partie finie — est <strong>refusé en le disant</strong> ; un plateau vide fait venir la question du vivier.</p>
<p>La vérité est celle du moteur : la base de bearoff à deux camps quand la position y figure, gammonNet à sa profondeur canonique partout ailleurs, et le panneau dit laquelle a répondu. Rien n'est estimé : une position que le moteur n'évalue pas n'est pas posée.</p>
<p><strong>Décision</strong> pose une décision <strong>déjà analysée</strong> : une position de la liste parcourue — sa seule source —, avec la décision qu'elle porte, coup de pions ou action de videau, et c'est l'analyse enregistrée qui juge. Une position sans analyse ne pose pas de question, et une position posée ne revient pas dans la session ; quand la liste est épuisée, le panneau le dit. Sans base ouverte, ou sans position analysée dans la liste, l'exercice <strong>refuse en le disant</strong> et rien ne démarre.</p>
<p>Tant qu'une question d'<em>Évaluation</em> ou de <em>Décision</em> attend sa réponse, le panneau Analyse est masqué : il porte la réponse.</p>
<h4>Répondre</h4>
<p>Le mode de réponse est une propriété de l'exercice, jamais un réglage : ce qui se <strong>compte ou se récite</strong> se déclare, ce qui s'<strong>estime</strong> se saisit — parce que là, la taille de l'erreur est la leçon.</p>
<p><em>Scores</em> et <em>Comptage des pips</em> se <strong>déclarent</strong> : vous calculez de tête, vous cliquez « Révéler », et la vérité s'affiche. Chaque nombre est alors <strong>juste par défaut</strong> — vous cliquez celui que vous avez raté pour le marquer <strong>faute</strong> (<em>Tab</em> puis <em>Espace</em> fait le même geste au clavier), et un second clic annule la marque. Rien ne se tape : un compte de pions ou une case de table est juste ou faux, et l'écrire n'apprend rien de plus que de le lire.</p>
<p><em>Bearoff</em> se <strong>saisit</strong> : vous écrivez les deux EPC, « Valider » les juge à un demi-pion près — la granularité à laquelle l'EPC change une décision de course — et la vérité s'affiche à côté de ce que vous avez écrit. C'est l'application qui juge, il n'y a rien à cocher. L'écart est enregistré <strong>avec son signe</strong> : surestimer n'est pas sous-estimer, et c'est le bilan qui en fait une moyenne.</p>
<p><em>Évaluation</em> mêle les deux gestes dans une même question. Les chances de gain se <strong>saisissent</strong> et se jugent à cinq points près, écart signé compris ; l'action de videau se <strong>choisit</strong> — le clic retient le bouton sans rien juger, et « Valider » juge les deux en une fois. Le videau n'a pas de tolérance : seul le bouton que le verdict du moteur rend juste est juste, et une position <em>trop bonne pour doubler</em> se répond <em>Pas de double</em>. <em>Entrée</em> dans le champ mène à l'action de videau tant qu'elle n'est pas choisie, puis valide. Après la réponse, le panneau affiche la vérité — le verdict en quatre issues —, sa source, et l'<strong>EPC des deux camps</strong> quand la position en a un exact ; l'EPC n'est jamais demandé ici, il a son propre exercice. Le journal compte les deux nombres à part : on peut bien estimer une position et mal lire son videau.</p>
<p><em>Décision</em> se <strong>choisit</strong>. Sur une décision de pions, <strong>jouez le coup sur le damier</strong> : cliquez le point de départ puis la destination, ou glissez le pion, autant de fois qu'il y a de dés. Le damier n'offre que ce qui est jouable — un clic qu'aucun coup légal n'autorise ne déplace rien. Dans le panneau, « Annuler le pas » revient d'un dé, « Recommencer » remet la position telle que la question la pose (le menu du clic droit propose aussi <em>Recommencer</em>), et « Valider », actif une fois le coup complet, le fait juger. Le champ de notation accepte aussi le coup tapé (<code>13/7 8/7</code>, la notation de la transcription) : les pas se posent sur le damier à chaque frappe, un champ rougi dit qu'un pas n'est pas jouable, et ENTRÉE valide le coup complet. Le panneau ayant le focus, RETOUR ARRIÈRE défait un pas, ÉCHAP recommence le coup et ENTRÉE le valide. Sur une décision de videau, cliquez <em>Pas de double</em>, <em>Double, prend</em> ou <em>Double, passe</em> : le clic est la réponse.</p>
<p>La correction distingue trois issues, et les confondre mentirait. Un <strong>coup illégal</strong> n'est pas un coup mal choisi — c'est une faute de règle. Un <strong>coup légal que le moteur n'a pas classé</strong> n'est pas une erreur de jugement : il n'a simplement pas de prix, et il ne coûte rien. Un coup classé coûte ce que l'analyse dit qu'il coûte, en millipoints. Seul un coup classé et sans coût est juste ; le meilleur coup s'affiche dans tous les cas.</p>
<p>Le chronomètre part à l'affichage de la question et s'arrête à « Révéler », à « Valider » ou au clic d'une action de videau de <em>Décision</em> ; la question suivante se prépare pendant que vous répondez, elle n'est donc jamais chronométrée avec la vôtre. Cocher ses fautes n'est pas chronométré non plus. Avec une limite, une question restée sans réponse à l'échéance se révèle seule et compte <strong>hors délai</strong> : tous ses nombres sont faux, et son temps n'entre pas dans la médiane — on ne mesure pas une réponse qui n'a pas été donnée. Une décision hors délai affiche le meilleur coup, et n'entre pas dans le PR de la session.</p>
<p>« Suivante » enregistre la question et en pose une autre. La session n'a pas de longueur fixée : elle dure jusqu'à « Terminer », qui l'écrit au journal, ou « Quitter », qui la jette. Tous les boutons sont dans le panneau ; le plateau montre la question et sa réponse, il ne porte aucune commande.</p>
<p>Tant qu'une question est posée sur le plateau, révélée ou non, les touches qui parcourent la liste ne la font pas défiler : la question garde le plateau jusqu'à « Suivante », « Terminer » ou « Quitter ».</p>
<h4>Le journal et le bilan</h4>
<p>Les sessions terminées sont conservées dans la base elle-même — elles suivent donc le fichier — et sans plafond. Au repos, le panneau affiche une ligne par exercice : le nombre de sessions, le taux de fautes, le temps médian et, au-delà de dix sessions, la <strong>tendance</strong>, c'est-à-dire l'écart entre le taux de fautes des dix dernières sessions et celui de toutes — négatif, vous progressez.</p>
<p>Pour <em>Décision</em>, la ligne donne aussi le <strong>PR</strong> de la dernière session, calculé par la formule que les statistiques appliquent au jeu réel — 500 × erreur moyenne en équité normalisée, sur les décisions jugées. Un PR de 6 à l'entraînement et un PR de 6 en match mesurent la même chose sur la même échelle.</p>
<p>Cliquer le nom de l'exercice déplie le détail <strong>par type de nombre</strong> : « Point de prise 4 · dernier lancer, 6 / 9 ». C'est ce détail qui fait l'intérêt du journal, et il compte par type et non par face : la même case de la même table, vue d'un côté ou de l'autre, est une seule faiblesse.</p>
<p>Une question de <em>Décision</em> garde au journal sa position, la réponse donnée et son coût en millipoints. Le détail de <em>Décision</em> porte donc trois boutons qui agissent sur toutes les positions ratées, chacune une fois, la plus récemment ratée d'abord — une question hors délai compte comme ratée :</p>
<ul>
<li><strong>Reprendre mes ratés</strong> — les positions ratées deviennent la liste parcourue et une session <em>Décision</em> repart dessus ;</li>
<li><strong>Paquet Anki des ratés</strong> — un paquet Anki de ces positions ;</li>
<li><strong>Collection des ratés</strong> — une collection de ces positions.</li>
</ul>
<p>Le paquet et la collection sont nommés « Ratés à Décision » suivis de la date du jour. En ligne de commande, <code>training missed</code> rend la même liste et en fait un paquet (<code>--deck</code>) ou une collection (<code>--collection</code>), et <code>training sessions</code> relit le journal (voir training — Le journal d'entraînement).</p>
<h3>Panneau Duel</h3>
<p>Le panneau <strong>Duel</strong> fait jouer un match entier contre le Bot, sous l'arbitrage de blunderDB : il lance les dés, impose les règles, tient le score et les horloges. Il s'ouvre par <code>CTRL-H</code>, par le bouton « Jouer » de la barre d'outils, ou par la commande <code>duel</code>. Une base doit être ouverte : le Duel s'y écrit après chaque décision.</p>
<p>Sans Duel ouvert, le panneau montre le formulaire, mémorisé d'un Duel à l'autre, et la liste des Duels en suspens :</p>
<ul>
<li><strong>Match</strong> de 1 à 25 points, ou <strong>session en argent</strong> (Jacoby au choix).</li>
<li><strong>Départ</strong> : la position initiale, la position au plateau, ou la position initiale à un score choisi.</li>
<li><strong>Côté joué</strong> (joueur 1 ou 2) et <strong>niveau du Bot</strong> (<code>instant</code>, <code>normal</code>, <code>thorough</code>, ceux de l'analyse). Le sélecteur dit la profondeur de chaque niveau, par exemple « instant (0 coup) » ou « normal (2 coups, élagué) » ; plus profond, c'est plus fort et plus lent.</li>
<li><strong>Cadence</strong> : sans cadence, ou une cadence nommée (une réserve par joueur et un délai gratuit à chaque tour), et ce que fait le temps écoulé : continuer en le notant, ou perdre le match.</li>
<li><strong>Votre nom</strong> et <strong>Enregistrer le match</strong> : décoché, le Duel terminé est jeté au lieu de devenir un Match.</li>
</ul>
<p>« Reprendre » rouvre un Duel en suspens au même point, avec les mêmes dés à venir ; ses horloges étaient arrêtées.</p>
<p>Un Duel en suspens ne quitte pas sa base : l'export de la base ne l'emporte pas, parce que ses dés à venir ne doivent sortir par aucune voie avant la fin. Un Duel terminé part comme tout Match.</p>
<p>Le Duel se joue au plateau. Le panneau montre la feuille de match en deux colonnes, comme la Transcription, les horloges quand une cadence court, une ligne qui dit ce qui est attendu, et « Abandonner le match », « Mettre en pause », « Annuler le match ». Un match en points ne s'enregistre qu'entier : il n'y a pas d'arrêt qui garde un match inachevé. Le score et le videau sont ceux du plateau ; le score et les horloges restent dans la barre d'état quand l'onglet est replié. L'empreinte SHA-256 du germe des dés, publiée par l'Arbitre dès la création, se lit dans l'infobulle de la ligne d'invite du panneau, puis avec le germe dans l'origine du Match terminé. Le plateau passe en mode <strong>DUEL</strong> : la bibliothèque ne se parcourt plus, l'édition, le panneau Eval et les autres onglets ne s'ouvrent pas, et le moteur se tait — aucune évaluation, aucun candidat. Seules restent la Pile (<code>B</code>), le pipcount (<code>P</code>) et l'aide.</p>
<ul>
<li>Avant le lancer, un clic sur le plateau, sur les dés ou sur un pion les lance ; seul un clic sur le videau propose de doubler, et le plateau demande « Doubler » ou « Annuler ». Quand le videau n'est pas disponible, le lancer est automatique.</li>
<li>Face à un double du Bot, le plateau demande « Prendre » ou « Passer ».</li>
<li>Un clic sur un pion le joue avec le dé de gauche encore libre, ou avec l'autre quand celui-ci ne peut pas le jouer ; un double se joue en quatre clics. Un pion peut aussi se glisser vers sa destination. Un dé joué est grisé. Seuls passent les pas d'un coup légal.</li>
<li>Avant de jouer, un clic sur les dés, ou un clic droit sur le plateau, intervertit leur ordre. Pendant le coup, le clic droit sur le plateau reprend tous les pions joués (<code>RETOUR ARRIÈRE</code> aussi).</li>
<li>Le coup complet se valide par un clic sur les dés, par « Valider » sur le plateau, ou par <code>ENTRÉE</code> ou <code>ESPACE</code>. Rien ne se reprend après.</li>
<li>Le Bot répond aussitôt ; ses coups sont rejoués au plateau, lentement.</li>
<li>Le clic droit hors du plateau, ou sur le plateau hors de son coup, ouvre le menu du Duel : mettre la position sur la Pile ou l'en retirer, abandonner la partie pour un simple, un gammon ou un backgammon (à son tour, après confirmation), abandonner le match, le mettre en pause, l'annuler. Ce menu n'offre ni évaluation ni édition.</li>
<li>« Abandonner le match » cède le match entier, à tout moment et après confirmation : la partie en cours va à l'adversaire pour les points qui le portent à la longueur, et le Match s'écrit gagné par lui. En argent, la partie en cours est perdue en simple, à la valeur du videau (avec ou sans règle Jacoby), et la session se clôt. Sous une cadence qui fait perdre le match, une réserve épuisée vaut l'abandon du match par ce joueur.</li>
<li>« Mettre en pause » met le Duel en suspens, horloges arrêtées ; il se reprend au même point. « Annuler le match » le jette, après confirmation : rien n'en est écrit.</li>
<li>Un double-clic hors du plateau met la position sur la Pile, ou l'en retire, comme <code>B</code> ; seul le marque-page au coin du plateau le montre.</li>
</ul>
<p>Chaque décision porte sa durée, avec ou sans cadence. Un abandon n'a pas de durée enregistrée dans le Match.</p>
<p>À la fin, le Match est écrit, son analyse se lance et l'onglet Matchs s'ouvre sur lui. En ligne de commande, <code>blunderdb duel</code> pilote le même Duel (voir duel — Jouer un Duel).</p>
<h3>Panneau Métadonnées</h3>
<p>Le panneau <strong>Métadonnées</strong> (<em>CTRL-M</em>) affiche les informations générales de la base de données courante : <em>Utilisateur</em>, date de création (<em>Créé</em>), <em>Version</em> du schéma et <em>Description</em>. L'utilisateur, la date et la description se modifient sur place et s'enregistrent en quittant le champ ; la version est en lecture seule. Accessible aussi via la commande <code>meta</code>.</p>
<p>Il affiche également, <strong>lorsqu'elle existe</strong>, l'origine de la base — voir Diffuser une base : origine et mot de passe. Une base ordinaire n'affiche pas cette section.</p>
<h3>Diffuser une base : origine et mot de passe</h3>
<p>Un enseignant qui distribue une base de positions dispose de deux mécanismes, indépendants l'un de l'autre, tous deux facultatifs et choisis <strong>au moment de l'export</strong> : marquer le fichier de son origine, et le protéger par un mot de passe.</p>
<div class="admonition note">
<p>Aucun des deux ne suit ce que devient le fichier. blunderDB <strong>n'enregistre rien du côté de celui qui reçoit la base</strong> : ouvrir une base marquée est exactement comme ouvrir n'importe quelle autre, et rien nulle part ne consigne qui l'a ouverte, quand, ni d'où vient son contenu.</p>
</div>
<h4>Marquer une base de son origine</h4>
<p>La fenêtre d'export tient en un seul écran : le formulaire, puis une progression qui se superpose à lui le temps de l'écriture. Elle se ferme d'elle-même une fois terminée, et le résultat s'affiche dans la barre d'état.</p>
<p>Trois points méritent l'attention :</p>
<ul>
<li><strong>L'export porte sur les positions actuellement affichées</strong>, pas sur la base entière. Après une recherche, seuls les résultats partent — la fenêtre le rappelle en tête.</li>
<li><strong>Une collection dont toutes les positions ne sont pas dans la sélection arrive tronquée.</strong> La liste affiche donc, pour chaque collection, la part couverte (« 12/40 ») et la signale en rouge lorsqu'elle est partielle.</li>
<li><strong>Les tournois ne peuvent être exportés qu'avec les matchs</strong> : sans eux, le lien tournoi–match n'existe pas et le tournoi arriverait vide. La case est désactivée tant que « inclure les matchs » ne l'est pas.</li>
</ul>
<p>Les champs <em>Utilisateur</em>, <em>Description</em> et <em>Date de création</em> décrivent le <strong>fichier produit</strong> ; ils sont préremplis depuis la base source. La case <em>Mes filtres enregistrés</em> est à part des autres : elle n'exporte pas du contenu mais vos propres recherches enregistrées, sans utilité dans la base de quelqu'un d'autre.</p>
<p>Cocher <strong>Marquer ce fichier de son origine</strong> fait apparaître deux champs :</p>
<ul>
<li><strong>Origine</strong> — ce qu'est ce fichier et d'où il vient, dans vos mots : « Cours de Jean Dupont — 12 mars 2026 ». Ce champ est <strong>obligatoire</strong> : tant qu'il est vide, le bouton d'export reste inactif.</li>
<li><strong>Note</strong>, facultative — conditions d'utilisation, adresse de contact, une demande de ne pas rediffuser.</li>
</ul>
<p>La marque est signée avec votre identité d'émetteur. Elle est donc <strong>inaltérable et infalsifiable</strong> : nul ne peut la modifier, ni en fabriquer une à votre nom. Elle n'est en revanche <strong>pas ineffaçable</strong> — le fichier distribué est une base SQLite ordinaire, et blunderDB est un logiciel libre. Elle n'empêche rien : elle dit d'où vient le fichier.</p>
<h4>Identité d'émetteur</h4>
<p>Les marques sont signées avec votre <strong>identité d'émetteur</strong>, créée toute seule la première fois que vous marquez un fichier ; il n'y a rien à configurer. Elle appartient à une personne et non à une base : tous vos fichiers portent la même empreinte publique, de la forme <code>A3F1-9C24-7B05-E1D8</code>.</p>
<p>Vous pouvez communiquer cette empreinte à vos destinataires pour qu'ils vérifient qu'un fichier vient bien de vous. L'identité se transporte d'un poste à l'autre en un seul fichier (extension <code>.bdbid</code>), éventuellement protégé par une phrase secrète. <strong>Ce fichier permet de signer en votre nom : ne le partagez pas.</strong></p>
<p>Dans les préférences (icône engrenage de la barre d'outils), l'onglet <em>Identité d'émetteur</em> affiche votre nom et votre empreinte, et propose <em>Enregistrer l'identité…</em>, <em>Charger une identité…</em> et <em>Régénérer…</em>.</p>
<div class="admonition warning">
<p><strong>Régénérer ne révoque rien.</strong> Un filigrane embarque la clé publique qui l'a signé : il se vérifie donc pour toujours, tout seul. Si votre fichier d'identité a fuité, celui qui le détient pourra continuer à signer sous votre ancienne empreinte, et ces marques resteront valides.</p>
<p>Ce qui vous protège après une fuite n'est pas logiciel : c'est de publier votre nouvelle empreinte et de désavouer l'ancienne auprès de vos destinataires.</p>
<p>La régénération écrase la clé actuelle ; blunderDB propose de l'enregistrer avant de la remplacer.</p>
</div>
<h4>Protéger une base par un mot de passe</h4>
<p>Le mot de passe se saisit masqué, ici comme à l'ouverture d'un fichier protégé ; l'icône en forme d'œil l'affiche <strong>tant qu'on la maintient enfoncée</strong>, et le masque de nouveau dès qu'on relâche.</p>
<p>Cocher <strong>Protéger ce fichier par un mot de passe</strong> produit un fichier d'extension <code>.dbx</code> — y compris si vous aviez choisi un nom en <code>.db</code> dans la fenêtre d'enregistrement, celle-ci s'ouvrant avant que le mot de passe ne soit demandé. Pour l'ouvrir, utilisez l'ouverture de base habituelle : la fenêtre de sélection accepte aussi bien les <code>.db</code> que les <code>.dbx</code>. blunderDB demande alors le mot de passe et installe une base ordinaire à côté ; ensuite plus rien n'est demandé.</p>
<p>La fenêtre propose de <strong>supprimer le fichier protégé une fois ouvert</strong> : sans cela vous conservez le même contenu sous deux noms. La case n'est pas cochée par défaut — le fichier protégé reste le vôtre si vous comptez le transmettre — et la suppression n'a lieu qu'après une ouverture réussie.</p>
<div class="admonition warning">
<p>Le mot de passe protège le <strong>transport</strong> du fichier, pas la base. Il empêche un tiers d'ouvrir un fichier qui traîne dans un dossier de téléchargement ou une pièce jointe transférée par erreur. Il ne protège pas de celui à qui vous avez donné le mot de passe.</p>
</div>
<p>Le mot de passe est vérifié à <strong>chaque</strong> ouverture, y compris lorsque le fichier a déjà été ouvert auparavant sur ce poste.</p>
<p>Techniquement, la base est chiffrée par <strong>AES-256 en mode GCM</strong>, avec une clé dérivée du mot de passe par <strong>Argon2id</strong> (64 Mio de mémoire, 3 passes, 4 fils), et un sel tiré au hasard propre à chaque fichier. Le mode GCM authentifie l'ensemble : un mot de passe erroné est détecté comme tel, et toute altération du fichier chiffré l'est également — on n'obtient jamais une base corrompue en silence.</p>
<p>L'en-tête du fichier protégé reste <strong>en clair</strong> : son origine demeure lisible sans le mot de passe.</p>
<h4>Lire l'origine d'un fichier</h4>
<p>Dans l'application, ouvrez le fichier et affichez le panneau <strong>Métadonnées</strong> (commande <code>meta</code>). Une section <strong>Origine</strong> apparaît en tête du panneau, en lecture seule, indiquant ce qui a été inscrit, par qui, quand, et l'état de la signature :</p>
<ul>
<li>« ✓ marquée par vous » : le fichier porte votre marque, intacte ;</li>
<li>« ✓ signature vérifiée » : la marque est intacte et vient d'une autre clé — comparez son empreinte à celle que le producteur vous a communiquée ;</li>
<li>« ⚠ signature invalide » : le document a été modifié ou contrefait.</li>
</ul>
<p>Cette section n'apparaît pas sur une base ordinaire.</p>
<p>En ligne de commande, <code>blunderdb info --db fichier.db</code> affiche l'origine et l'état de la signature, <strong>sans jamais écrire dans le fichier</strong>. La commande fonctionne aussi sur un fichier protégé, sans le mot de passe. Voir <code>CLI_USAGE.md</code> pour les options <code>--watermark</code> et <code>--password</code> de <code>export</code>, ainsi que pour <code>identity</code> et <code>open</code>.</p>
<h4>Publier une base pour d'autres</h4>
<p>Une base marquée se distribue comme n'importe quel fichier — courriel, site personnel, clé USB. blunderDB <strong>ne fournit aucun service</strong> : ni dépôt, ni catalogue hébergé, ni compte. C'est une conséquence directe de sa conception : rien n'est jamais enregistré du côté de celui qui reçoit un fichier, et il n'y aurait donc rien à faire remonter à un service, même s'il en existait un.</p>
<p>Ce qui rend une base publiée utilisable par quelqu'un d'autre tient à quatre champs, tous déjà là :</p>
<ul>
<li><strong>Utilisateur</strong> — qui l'a constituée, sous le nom que vous voulez voir cité.</li>
<li><strong>Description</strong> — ce que la base contient, en une phrase qui tienne dans une liste : « 240 décisions de videau au score, commentées, niveau intermédiaire ».</li>
<li><strong>Origine</strong> (du filigrane) — ce qu'est ce fichier et pour qui il a été produit. C'est ce que le destinataire lit en premier dans le panneau <em>Métadonnées</em>.</li>
<li><strong>Empreinte d'émetteur</strong> — publiez-la à côté du fichier, pas dedans : c'est en la comparant que le destinataire vérifie que le fichier vient de vous et non de quelqu'un qui a repris votre nom.</li>
</ul>
<p>Une base publiée sans filigrane reste parfaitement utilisable ; elle est simplement anonyme, et le panneau <em>Métadonnées</em> n'affiche alors aucune section <em>Origine</em>.</p>
<p>Pour faire connaître une base, la catégorie <em>Show and tell</em> des <code>discussions du dépôt &lt;https://github.com/kevung/blunderDB/discussions&gt;</code>_ sert d'annuaire : c'est une liste tenue par ceux qui publient, pas un service rendu par blunderDB. Y annoncer une base demande le lien, les quatre champs ci-dessus et l'empreinte.</p>
`,
    shortcuts: `
<p>Les infobulles de la barre d'outils rappellent la touche de chaque bouton, sous le nom qu'elle porte sur le clavier de la langue de l'interface : <em>Gauche</em>, <em>Suppr</em>, <em>Page préc.</em>, <em>Page suiv.</em>, <em>Maj</em>. Les tableaux ci-dessous et la modale d'aide emploient les mêmes noms.</p>
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
<td>En mode édition ou Eval : réinitialiser le board, le cube, le score et les dés.</td>
</tr>
<tr>
<td>CTRL-G</td>
<td>Afficher les métadonnées de la position.</td>
</tr>
<tr>
<td>b</td>
<td>Mettre la position affichée sur la Pile (collection « à revoir plus tard »), ou l'en retirer.</td>
</tr>
<tr>
<td>Double-clic hors du plateau</td>
<td>Mettre la position affichée sur la Pile, ou l'en retirer, dans tous les modes.</td>
</tr>
<tr>
<td>Clic droit hors du plateau (édition, Eval)</td>
<td>Ouvrir le menu du plateau : <em>Effacer la position</em>, comme RETOUR ARRIERE, ou <em>Position de départ</em>.</td>
</tr>
<tr>
<td>Clic droit sur le plateau (coup joué au plateau)</td>
<td>Ouvrir le menu du plateau, qui commence par <em>Recommencer</em> : le coup est remis à zéro, pas la position.</td>
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
<td>Début, h</td>
<td>Première position / Partie précédente (navigation match).</td>
</tr>
<tr>
<td>Page préc.</td>
<td>Recule d'une page (cent positions par défaut, réglable dans les Paramètres &gt; Interface : 10, 50, 100, 500 ou 1 000 positions, ou 10 % de la liste ; au début de la liste, s'y arrête) ; dans un match, partie précédente.</td>
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
<td>Fin, l</td>
<td>Dernière position / Partie suivante (navigation match).</td>
</tr>
<tr>
<td>Page suiv.</td>
<td>Avance d'une page (même pas que <em>Page préc.</em> ; à la fin de la liste, s'y arrête) ; dans un match, partie suivante.</td>
</tr>
<tr>
<td>r</td>
<td>Charger une position aléatoire.</td>
</tr>
<tr>
<td>ÉCHAP</td>
<td>Quitter les résultats d'une recherche <code>ss</code> lancée depuis une collection ou un match : retour à la collection, ou au match sur le coup étudié.</td>
</tr>
</tbody>
</table>
<p>Tant qu'une question du panneau Entraînement est posée sur le plateau, les touches qui parcourent la liste ne la font pas défiler : la question garde le plateau. Sur une décision de pions, le panneau ayant le focus : RETOUR ARRIÈRE défait le dernier pas, ÉCHAP recommence le coup, ENTRÉE le valide quand il est complet ; dans le champ de notation, ENTRÉE valide le coup tapé (<code>13/7 8/7</code>).</p>
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
<tr>
<td>ALT-1 … ALT-9</td>
<td>Lancer le filtre épinglé de ce rang dans la bibliothèque de filtres.</td>
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
<td>CTRL-MAJ-L</td>
<td>Classer les positions voisines de la position courante.</td>
</tr>
<tr>
<td>CTRL-P</td>
<td>Afficher/cacher les commentaires.</td>
</tr>
<tr>
<td>CTRL-MAJ-P</td>
<td>Ouvrir/fermer la palette de commandes : commandes, onglets, filtres et matchs, retrouvés par un nom approché.</td>
</tr>
<tr>
<td>CTRL-J</td>
<td>Afficher/cacher le panneau Entraînement.</td>
</tr>
<tr>
<td>CTRL-H</td>
<td>Afficher/cacher le panneau Duel (un match contre le Bot).</td>
</tr>
<tr>
<td>ENTRÉE (Duel)</td>
<td>Valider le coup arrangé au plateau ; rien ne se reprend après.</td>
</tr>
<tr>
<td>ESPACE (Duel)</td>
<td>Valider le coup arrangé au plateau, une fois tous les dés joués ; sans effet sur un coup partiel ou hors de votre tour.</td>
</tr>
<tr>
<td>RETOUR ARRIÈRE (Duel)</td>
<td>Replacer les pions avant la validation.</td>
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
<td>CTRL-MAJ-T</td>
<td>Afficher/cacher le panneau Transcription (brouillons de matchs).</td>
</tr>
<tr>
<td>J / K</td>
<td>Dans la file des propositions d'un tournoi dirigé : descendre, monter.</td>
</tr>
<tr>
<td>ENTRÉE</td>
<td>Dans la file des propositions : confirmer la proposition choisie.</td>
</tr>
<tr>
<td>TAB (page du tournoi)</td>
<td>Premier arrêt dans l'onglet Direction : « Aller à la file » ; ENTRÉE pose le focus sur la file des propositions.</td>
</tr>
<tr>
<td>GAUCHE / DROITE</td>
<td>Dans la fiche de résultat, hors d'un champ : choisir le joueur de gauche ou celui de droite ; ENTRÉE enregistre sa victoire.</td>
</tr>
<tr>
<td>CTRL-Z</td>
<td>Reprendre la dernière décision d'un tournoi dirigé (hors d'un champ de saisie, où il annule la frappe).</td>
</tr>
<tr>
<td>TAB (focus perdu)</td>
<td>Sous la page d'un tournoi dirigé, quand le focus est tombé sur la page : le ramener sur le premier élément de la page, sans ouvrir la recherche.</td>
</tr>
<tr>
<td>PAGE PRÉC. / PAGE SUIV., DÉBUT / FIN</td>
<td>Sous la page d'un tournoi dirigé : faire défiler la page, sans parcourir le plateau qu'elle cache.</td>
</tr>
<tr>
<td>ÉCHAP</td>
<td>Fermer la fiche de résultat ou la reprise en cours.</td>
</tr>
<tr>
<td>?</td>
<td>Afficher/cacher l'aide.</td>
</tr>
</tbody>
</table>
<h3>Duel au plateau</h3>
<table>
<thead>
<tr>
<th>Geste</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic sur le plateau ou sur les dés (avant le lancer)</td>
<td>Lancer les dés. Le videau garde son sens : il propose de doubler.</td>
</tr>
<tr>
<td>Clic sur le videau (avant le lancer)</td>
<td>Proposer de doubler ; « Doubler » ou « Annuler » confirme sur le plateau.</td>
</tr>
<tr>
<td>Clic sur un pion</td>
<td>Le jouer avec le dé de gauche encore libre, ou avec l'autre si celui-ci ne peut pas le jouer. Un dé joué est grisé.</td>
</tr>
<tr>
<td>Glisser un pion</td>
<td>Le jouer vers le point où il est lâché, si un coup légal le permet.</td>
</tr>
<tr>
<td>Clic sur les dés (coup en cours)</td>
<td>Aucun dé joué : intervertir leur ordre. Coup complet : le valider.</td>
</tr>
<tr>
<td>Clic droit sur le plateau (coup en cours)</td>
<td>Reprendre tous les pions joués ; sans pion joué, intervertir les dés.</td>
</tr>
<tr>
<td>Clic droit hors du plateau, ou hors de son coup</td>
<td>Ouvrir le menu du Duel : Pile, abandon, suspension, arrêt.</td>
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
<td>CTRL-Page préc., MAJ-J</td>
<td>Vue précédente.</td>
</tr>
<tr>
<td>CTRL-Page suiv., MAJ-K</td>
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
<tr>
<td>ÉCHAP</td>
<td>Pendant une recherche : l'interrompre.</td>
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
<tr>
<td>Clic (sur l'étoile)</td>
<td>Épingler ou désépingler le filtre.</td>
</tr>
<tr>
<td>Clic (sur une pastille épinglée)</td>
<td>Exécuter la recherche du filtre épinglé.</td>
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
<td>Ctrl+Clic</td>
<td>Ajouter ou retirer un coup de la sélection à rouler.</td>
</tr>
<tr>
<td>Maj+Clic</td>
<td>Étendre la sélection jusqu'au coup cliqué.</td>
</tr>
<tr>
<td>Clic droit</td>
<td>Menu du coup : rouler les coups sélectionnés avec le réglage choisi, ou annuler le rollout en cours ; copier la position et l'analyse, ou la position et les coups sélectionnés.</td>
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
<td>r</td>
<td>Lancer le rollout des coups sélectionnés (sans sélection, de la position) avec le réglage choisi ; une seconde pression l'arrête.</td>
</tr>
<tr>
<td>Esc</td>
<td>Arrêter le rollout en cours, sinon désélectionner le coup. Si aucun coup sélectionné, fermer le panneau, sauf devant les résultats d'une recherche <code>ss</code> : y revenir à la collection ou au match.</td>
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
<td>Ctrl+Clic</td>
<td>Ajouter ou retirer un coup de la sélection à rouler.</td>
</tr>
<tr>
<td>Maj+Clic</td>
<td>Étendre la sélection jusqu'au coup cliqué.</td>
</tr>
<tr>
<td>Clic droit</td>
<td>Menu du coup : rouler les coups sélectionnés avec le réglage choisi, ou annuler le rollout en cours ; copier la position et l'analyse, ou la position et les coups sélectionnés.</td>
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
<td>r</td>
<td>Lancer le rollout des coups sélectionnés (sans sélection, de la position) avec le réglage choisi ; une seconde pression l'arrête.</td>
</tr>
<tr>
<td>Esc</td>
<td>Arrêter le rollout en cours, sinon désélectionner le coup.</td>
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
<td>v</td>
<td>Voir dans la vidéo la décision étudiée, une seconde avant le jet (match portant une source vidéo et un repère à cette décision).</td>
</tr>
<tr>
<td>[ / ] (vidéo ouverte)</td>
<td>Ralentir ou accélérer la lecture de la vidéo, par pas de 0,25 entre 0,25× et 4×.</td>
</tr>
<tr>
<td>/</td>
<td>Aller au champ de filtre (joueur, événement, tournoi, date). <em>Esc</em> efface le filtre.</td>
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
<td>Double-clic, ENTRÉE (ligne au focus)</td>
<td>Sélectionner un tournoi (afficher son détail). TAB atteint les lignes ; un clic simple ne fait que surligner.</td>
</tr>
<tr>
<td>HAUT, k</td>
<td>Sélectionner le tournoi précédent, quand le panneau a le focus ou qu'aucune direction n'est affichée.</td>
</tr>
<tr>
<td>BAS, j</td>
<td>Sélectionner le tournoi suivant, quand le panneau a le focus ou qu'aucune direction n'est affichée.</td>
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
<h3>Page Direction</h3>
<p>Sous la page Direction, <em>J</em>, <em>K</em>, <em>HAUT</em>, <em>BAS</em> et <em>ENTRÉE</em> vont à la file des propositions, sauf quand le focus est sur une case de la grille des tables, où <em>HAUT</em> et <em>BAS</em> changent de case. Les menus contextuels sont décrits dans le manuel (menus contextuels).</p>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic droit, MENU, MAJ-F10</td>
<td>Ouvrir le menu contextuel de l'objet focalisé : case de table, joueur, place de l'arbre, emplacement, proposition, ligne d'historique. HAUT/BAS parcourent le menu, ENTRÉE choisit, ÉCHAP le ferme.</td>
</tr>
<tr>
<td>GAUCHE, DROITE, HAUT, BAS, DÉBUT, FIN</td>
<td>Passer d'une case de la grille des tables à l'autre (cases libres comprises) ; la grille ne prend qu'un arrêt de TAB.</td>
</tr>
<tr>
<td>1 à 9, puis 0 à 9</td>
<td>Ouvrir la fiche de la table de ce numéro ; deux chiffres, dans les 0,4 s, pour une table au-delà de 9.</td>
</tr>
<tr>
<td>M, X</td>
<td>Sur une case occupée, ouvrir la fiche (ou, si elle est ouverte, le champ de table) ; viser une table occupée échange les deux matchs.</td>
</tr>
<tr>
<td>/</td>
<td>Ouvrir la recherche rapide : joueurs, tables, matchs en cours et épreuves de l'événement, retrouvés par un nom ou un numéro de table (détail). Sans effet dans un champ de saisie.</td>
</tr>
<tr>
<td>F11</td>
<td>Mettre la page Direction en plein écran (barre d'outils, onglets, panneau et barre d'état masqués), ou en sortir. ÉCHAP en sort aussi, après avoir fermé le menu ou la fiche ouverts (détail).</td>
</tr>
<tr>
<td>Glisser une case occupée sur une autre</td>
<td>À la souris : sur une case libre, déplacer le match ; sur une case occupée, échanger les deux matchs après confirmation. ÉCHAP annule le glisser.</td>
</tr>
<tr>
<td>Toutes les tables</td>
<td>Sur la grille <em>Toutes les tables</em> d'un événement, les mêmes touches, menus et glisser-déposer agissent sur la table de n'importe quelle épreuve ; un échange avec une autre épreuve nomme les deux dans la confirmation.</td>
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
<h3>Panneau Transcription</h3>
<p>Le panneau prend ces touches lorsqu'il a le focus. Devant la liste des brouillons il en prend déjà quelques-unes, pour que reprendre le travail de la veille ne demande pas la souris.</p>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>BAS, j / HAUT, k (liste des brouillons)</td>
<td>Parcourir les brouillons. Le premier est surligné à l'ouverture : c'est le plus récemment modifié.</td>
</tr>
<tr>
<td>ENTREE (liste des brouillons)</td>
<td>Ouvrir le brouillon surligné.</td>
</tr>
<tr>
<td>n (liste des brouillons)</td>
<td>Ouvrir le formulaire de création.</td>
</tr>
<tr>
<td>Clic</td>
<td>Ouvrir un brouillon de la liste.</td>
</tr>
<tr>
<td>1 … 6</td>
<td>Saisir un dé. Sur le premier coup d'une partie : dé du joueur 1, puis dé du joueur 2 (le plus fort commence et joue les deux dés).</td>
</tr>
<tr>
<td>1 … 6 (jet saisi)</td>
<td>Valider le coup sélectionné et ouvrir le jet suivant. Sur une action relue, où le curseur est posé sur une action déjà écrite, le chiffre recommence le jet de cette action au lieu de valider.</td>
</tr>
<tr>
<td>1 … 6 (curseur sur le premier coup d'une partie)</td>
<td>Taper un autre jet d'ouverture : dé du joueur 1 puis dé du joueur 2. Le plus fort l'emporte — gros dé d'abord, le coup revient au joueur 1 en bas ; petit dé d'abord, au joueur 2 en haut ; le premier candidat est présélectionné. Le même jet retapé ne change rien ; <em>s</em> donne le coup à l'autre joueur.</td>
</tr>
<tr>
<td>BAS, j</td>
<td>Sélectionner le candidat suivant (les flèches du coup apparaissent sur le plateau).</td>
</tr>
<tr>
<td>HAUT, k</td>
<td>Sélectionner le candidat précédent.</td>
</tr>
<tr>
<td>Molette</td>
<td>Sélectionner le candidat suivant ou précédent, au-dessus de la liste comme au-dessus du damier : le regard reste sur le plateau et les flèches défilent.</td>
</tr>
<tr>
<td>Clic (sur une ligne)</td>
<td>Sélectionner ce candidat.</td>
</tr>
<tr>
<td>Double-clic (sur une ligne)</td>
<td>Valider ce candidat.</td>
</tr>
<tr>
<td>Clic (sur le triangle des jets)</td>
<td>Saisir le jet d'un seul geste : la case porte les deux dés, doubles sur la diagonale. Sur le premier coup d'une partie, le triangle laisse la place à une rangée de six dés, un clic donnant le dé d'un camp.</td>
</tr>
<tr>
<td>Clic, glisser (aucun dé saisi)</td>
<td>Jouer le coup directement sur le damier : le pion va du point cliqué à sa destination, contraint aux coups légaux, et les deux dés se déduisent des pas joués.</td>
</tr>
<tr>
<td>Clic, glisser (jet saisi)</td>
<td>Jouer le coup sur le damier, contraint aux coups légaux de ce jet : chaque pas joué ne garde dans la liste que les candidats qui le contiennent, et un coup légal achevé est enregistré aussitôt. Sur une action relue, il la remplace.</td>
</tr>
<tr>
<td>Glisser hors des règles (jet saisi)</td>
<td>Poser le pion là où il est lâché, même depuis un point d'où aucun coup légal ne part, pour transcrire un coup illégal. La suite du coup se joue librement, au clic comme au glissé, et la liste des candidats cède la place à une ligne qui le rappelle.</td>
</tr>
<tr>
<td>ENTREE (coup hors des règles)</td>
<td>Enregistrer le coup avec les dés saisis et le plateau obtenu, marqué coup illégal si aucun coup légal n'atteint ce plateau. Un coup hors des règles n'est jamais enregistré de lui-même.</td>
</tr>
<tr>
<td>RETOUR ARRIERE (coup en cours au plateau)</td>
<td>Défaire le dernier pas joué au plateau. Les pas restants sont rejoués contraints tant qu'un coup légal les contient : défaire le seul pas hors des règles rend la liste.</td>
</tr>
<tr>
<td>ENTREE</td>
<td>Valider le coup sélectionné (dernier coup d'une partie).</td>
</tr>
<tr>
<td>RETOUR ARRIERE</td>
<td>Effacer les deux dés saisis. C'est par là que passe la reprise d'un jet mal lu, puisqu'un chiffre valide.</td>
</tr>
<tr>
<td>Clic (sur les cases du jet)</td>
<td>Effacer les deux dés saisis, comme RETOUR ARRIERE.</td>
</tr>
<tr>
<td>Esc</td>
<td>Abandonner la saisie en cours.</td>
</tr>
<tr>
<td>d</td>
<td>Doubler ou redoubler : le coup sélectionné est validé au passage, en une seule touche.</td>
</tr>
<tr>
<td>t</td>
<td>Prendre le double proposé : le videau passe au preneur à la valeur doublée et le doubleur rejoue. Le curseur posé sur une cellule, écrit la prise à la place de l'action visée ; une passe devenue prise rouvre sa partie, et la suite s'y tape à la place.</td>
</tr>
<tr>
<td>p</td>
<td>Passer le double proposé : la partie est gagnée à la valeur d'avant le double. Le curseur posé sur une cellule, écrit la passe à la place de l'action visée.</td>
</tr>
<tr>
<td>r puis 1, 2 ou 3</td>
<td>Abandonner la partie pour le camp au trait : simple, gammon ou backgammon. Esc entre les deux touches annule sans rien enregistrer.</td>
</tr>
<tr>
<td>Clic (sur le videau)</td>
<td>Proposer un double pour le camp au trait, comme la touche d. Devant une offre, le videau ne répond pas : la prise et la passe sont dans la rangée de boutons.</td>
</tr>
<tr>
<td>GAUCHE, h</td>
<td>Reculer le curseur d'une action dans le transcript.</td>
</tr>
<tr>
<td>DROITE, l</td>
<td>Avancer le curseur d'une action.</td>
</tr>
<tr>
<td>Clic (sur une cellule)</td>
<td>Placer le curseur sur cette action, ou sur le tour manquant d'un double trait.</td>
</tr>
<tr>
<td>Double-clic (sur une cellule)</td>
<td>Taper le coup de cette action au clavier, dans la cellule : 13/7 8/7*, bar/22, 6/off. Seul le coup se tape, les dés sont ceux de la cellule ; ENTREE l'enregistre, même illégal, et Esc referme la cellule sans rien écrire. Vaut pour un coup, une danse, un coup non consigné, et pour la cellule en pointillés de la saisie en cours dès que ses deux dés sont saisis.</td>
</tr>
<tr>
<td>Double-clic (sur le score d'une partie)</td>
<td>Taper le score auquel cette partie a été jouée, dans l'en-tête : 3-2, 3–2 ou 3 2. ENTREE l'enregistre, un champ vidé revient au score que donnent les parties précédentes, et Esc referme le champ sans rien écrire. Un score qui diffère de celui-ci est marqué comme une incohérence. Pas de score en argent.</td>
</tr>
<tr>
<td>Clic droit (sur une cellule)</td>
<td>Ouvrir les corrections de cette action : insérer avant, insérer après, supprimer, changer de camp. Le curseur est amené sur la cellule au passage.</td>
</tr>
<tr>
<td>CTRL-ENTREE</td>
<td>Terminer le brouillon : écrire le match, ou remplacer celui dont il a été ouvert, et libérer le brouillon.</td>
</tr>
<tr>
<td>i</td>
<td>Insérer une action devant celle du curseur (camp proposé pour que la suite reste cohérente).</td>
</tr>
<tr>
<td>a</td>
<td>Insérer une action derrière celle du curseur.</td>
</tr>
<tr>
<td>x, Del</td>
<td>Supprimer la décision en cours d'édition — l'action du curseur, ou la saisie pas encore écrite — et reculer sur la précédente, prête à être corrigée ; les suivantes gardent leur camp. En bout de document, recule sur la dernière action.</td>
</tr>
<tr>
<td>s</td>
<td>Donner l'action du curseur à l'autre camp.</td>
</tr>
<tr>
<td>CTRL-Z</td>
<td>Annuler le dernier geste sur le brouillon.</td>
</tr>
<tr>
<td>CTRL-MAJ-Z</td>
<td>Rétablir le geste annulé.</td>
</tr>
<tr>
<td>ESPACE (vidéo attachée)</td>
<td>Lancer ou mettre en pause la vidéo. Sans vidéo, ESPACE ouvre la ligne de commande, comme ailleurs.</td>
</tr>
<tr>
<td>MAJ-GAUCHE / MAJ-DROITE (vidéo attachée)</td>
<td>Reculer ou avancer la vidéo de 5 secondes.</td>
</tr>
<tr>
<td>CTRL-MAJ-GAUCHE / CTRL-MAJ-DROITE (vidéo attachée)</td>
<td>Reculer ou avancer la vidéo d'une seconde.</td>
</tr>
<tr>
<td>v (vidéo attachée)</td>
<td>Poser l'instant courant de la vidéo comme instant de l'action du curseur.</td>
</tr>
<tr>
<td>MAJ-V (vidéo attachée)</td>
<td>Poser l'instant courant de la vidéo comme instant du jet de l'action du curseur.</td>
</tr>
<tr>
<td>[ / ] (vidéo attachée)</td>
<td>Ralentir ou accélérer la lecture, par pas de 0,25 entre 0,25× et 4× (jusqu'à 2× pour YouTube).</td>
</tr>
</tbody>
</table>
<p>Un jet qui n'autorise aucun coup enregistre la danse de lui-même, sans touche supplémentaire.</p>
<p>Avec une vidéo attachée, seule une validation explicite — <em>ENTREE</em>, le double-clic sur un candidat, le coup achevé au plateau — pose l'instant de l'action. Le chiffre du jet suivant, ou un geste de videau, valide sans instant ; <em>v</em> le pose après coup. Les flèches se lisent à leur place sur le clavier, quelle que soit sa disposition ; <em>[</em> et <em>]</em> se lisent au caractère tapé, <em>ALTGR</em> compris sur un clavier qui les place ainsi. Sans vidéo, ces touches gardent leur sens habituel.</p>
<p>Le chiffre a un seul sens : <strong>il commence un jet là où le curseur est</strong>. En bout de document il n'y a rien sous le curseur, donc il valide le coup sélectionné avant d'ouvrir le jet suivant — le meilleur coup joué coûte ainsi les deux dés et rien de plus, sa validation étant portée par la première touche du tour d'après. Sur une action déjà écrite, où l'on est revenu pour la corriger, il y a quelque chose sous le curseur : le chiffre recommence le jet de cette action, sur place. La différence se voit à l'écran, la cellule visée étant encadrée dans le transcript. La <strong>dernière</strong> action du document fait exception : une fois son jet retapé, le chiffre suivant la valide et ouvre la décision d'après, comme en bout de document, et ENTREE y mène aussi.</p>
<p>Ce qui est en train d'être tapé se dessine dans le transcript, en pointillés, à la place où il sera écrit : une correction recouvre la cellule qu'elle remplace, une insertion ouvre une cellule entre ses deux voisines, et le camp se lit à la colonne. Rien n'est enregistré avant la validation.</p>
<p>Une insertion au milieu du document continue d'insérer : la validation ouvre une cellule vide à la suite, et l'action suivante s'insère à son tour au lieu d'écraser celle d'après. La fin de la partie ou un déplacement du curseur y met fin.</p>
<p>Un geste qui n'a rien à faire le dit, une fois, dans la barre d'état : « rien à annuler » sur une pile vide, « aucune action sous le curseur » en bout de document. La phrase s'efface d'elle-même et rend la place à l'action attendue.</p>
<p>Reculer le curseur sur une action puis ressaisir la corrige <strong>en place</strong> : la validation remplace l'action et le curseur revient là où il était — ou, sur la dernière action, passe au bout du document, où la transcription continue. Si les dés sont corrigés et que le coup enregistré reste un coup légal du nouveau jet, il est conservé ; sinon le premier candidat du nouveau jet est proposé et le coup est signalé « à revoir » jusqu'à la validation. Avancer ou reculer le curseur après avoir changé quelque chose enregistre la correction au passage.</p>
<p>Rien n'est refusé ni supprimé : insérer une action du même camp que sa voisine crée un double trait, supprimer une action peut en créer un autre, changer un camp peut rendre illégaux les coups qui suivent. Ces incohérences sont marquées dans le transcript, jamais corrigées d'office, et le curseur se place sur la première d'entre elles après chaque geste qui écrit — sauf là où l'on continue : après une suppression, une insertion ou une partie rouverte, il reste où l'on tape. Se déplacer ou taper un dé ne le déplace jamais. Le tour qu'un double trait a perdu est une case du transcript : h et l s'y arrêtent, et un jet tapé là s'insère pour le camp à qui il manquait. La pile d'annulation vit en mémoire : elle est perdue à la fermeture du brouillon.</p>
<p>Le panneau lui-même — la liste des brouillons, la création, la saisie, le transcript et la barre du brouillon — est décrit dans Panneau Transcription.</p>
<h3>Planche-contact</h3>
<table>
<thead>
<tr>
<th>Raccourci</th>
<th>Action</th>
</tr>
</thead>
<tbody>
<tr>
<td>GAUCHE, DROITE, HAUT, BAS</td>
<td>Se déplacer dans la grille.</td>
</tr>
<tr>
<td>j, k</td>
<td>Vignette suivante, vignette précédente.</td>
</tr>
<tr>
<td>Début, Fin</td>
<td>Première, dernière vignette de la page.</td>
</tr>
<tr>
<td>Page préc., Page suiv.</td>
<td>Page précédente, page suivante.</td>
</tr>
<tr>
<td>ENTREE, Clic</td>
<td>Ouvrir la position sur le plateau et refermer la planche.</td>
</tr>
<tr>
<td>Esc</td>
<td>Refermer la planche.</td>
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
<td>Page préc.</td>
<td>Haut du contenu.</td>
</tr>
<tr>
<td>Page suiv.</td>
<td>Bas du contenu.</td>
</tr>
<tr>
<td>/</td>
<td>Chercher dans l'aide : Entrée passe à l'occurrence suivante, MAJ-Entrée à la précédente.</td>
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
<td>eval, epc</td>
<td>Ouvre le panneau Eval (Effective Pip Count, probabilité de gain et verdict de videau en bearoff). <code>epc</code> est l'ancien nom de ce panneau, conservé.</td>
</tr>
<tr>
<td>transcribe, tr</td>
<td>Ouvre le panneau Transcription : les brouillons de matchs en cours de saisie, et de quoi en commencer un.</td>
</tr>
<tr>
<td>direct</td>
<td>Ouvre le panneau Tournois pour diriger un tournoi, et le referme. Quand une direction y est ouverte, la zone principale montre le tournoi à la place du plateau ; tout autre onglet ramène le plateau.</td>
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
<td>log</td>
<td>Ouvre le journal d'activité : les deux cents dernières lignes du fichier de journal, avec de quoi les copier pour les joindre à un rapport, ou ouvrir le dossier qui les contient.</td>
</tr>
<tr>
<td>train</td>
<td>Ouvre le panneau Entraînement. Avec un argument, ouvre et démarre : <code>train scores</code> (la fiche de score d'un score tiré au sort ; <code>train tp</code> et <code>train takepoint</code> sont des synonymes), <code>train pips</code> (le compte de pions des deux camps), <code>train bearoff</code> (l'EPC des deux camps sur une position engendrée ; <code>train epc</code> est un synonyme), <code>train evaluation</code> (les chances de gain et l'action de videau d'une position engendrée, en partie d'argent), <code>train decision</code> (une décision analysée de la liste parcourue : le coup se joue sur le plateau, l'action de videau se choisit dans le panneau ; <code>train quiz</code> est un synonyme).</td>
</tr>
<tr>
<td>duel</td>
<td>Ouvre le panneau Duel : un match contre le Bot, arbitré par blunderDB (voir Panneau Duel).</td>
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
<td>resume</td>
<td>Liste les imports que rien n'a terminé (arrêt de l'application) et reprend celui qu'on choisit, en désignant à nouveau le dossier ou les fichiers.</td>
</tr>
<tr>
<td>[number]</td>
<td>Aller à la position d'indice indiqué.</td>
</tr>
<tr>
<td>[number]%</td>
<td>Aller à ce pourcentage de la liste : <code>0%</code> la première position, <code>50%</code> le milieu, <code>100%</code> la dernière.</td>
</tr>
<tr>
<td>grid, gr</td>
<td>Ouvre la planche-contact : la liste parcourue en grille de mini-plateaux, une page de vingt-quatre à la fois ; choisir une vignette ouvre sa position.</td>
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
<td>rollout, ro [fast|standard]</td>
<td>Lance le rollout de la position courante (réglage choisi dans l'onglet gammonNet de la configuration, ou le préréglage nommé) et ouvre le panneau Analyse. <code>ro search [fast|standard]</code> le lance sur la liste affichée, après confirmation avec le total ; <code>ro stop</code> arrête le rollout en cours.</td>
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
<td>lesson, le [N | edit [N]]</td>
<td>Sans argument, liste les leçons de la base dans la barre d'état ; <code>le N</code> ouvre la leçon N à sa première étape ; <code>le edit</code> ouvre l'éditeur de leçons, <code>le edit N</code> sur la leçon N (voir Leçons).</td>
</tr>
<tr>
<td>study, sq</td>
<td>Ouvre la file d'étude transversale : vos blunders que rien n'a encore traités, du plus coûteux au moins coûteux (voir Les blunders que rien n'a encore traités).</td>
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
<td>Chercher parmi les positions affichées : résultats courants, collection ouverte ou match en cours de revue.</td>
</tr>
</tbody>
</table>
<h3>Filtres de recherche</h3>
<p>Cette table est la référence de la grammaire de recherche : la ligne de commande, la bibliothèque de filtres et le drapeau <code>--query</code> de <code>blunderdb search</code> lisent tous les mêmes jetons. La colonne <em>Équivalent CLI</em> donne, quand il existe, le drapeau de <code>search</code> qui fait la même chose (voir Interface en ligne de commande (CLI)) ; un tiret signale un filtre que seule la grammaire exprime.</p>
<p>Cinq jetons ne portent pas leur valeur : ils la lisent sur le plateau de recherche. <code>cube</code> et <code>score</code> reprennent le videau et le score qui y sont posés, <code>d</code> le type de décision, <code>D</code> et <code>D1</code> les dés, <code>x</code> la structure dessinée dans l'onglet <em>Sauf</em>. Un lancer ne s'écrit donc jamais dans le jeton : <code>D65</code> n'existe pas, seule la forme d'exclusion porte ses chiffres (<code>xD65</code>). En ligne de commande, où il n'y a pas de plateau, ces jetons se comparent à un plateau vide ; ce sont les drapeaux de la troisième colonne qu'il faut y employer.</p>
<p>Un seul jeton <strong>classe</strong> au lieu de restreindre : <code>like</code> ordonne le résultat par distance croissante à une position cible, et tous les autres jetons restreignent l'ensemble ainsi classé.</p>
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
<td>dd</td>
<td>La décision est un videau de type Double / No Double (et non une réponse Take / Pass). Implique une décision de videau.</td>
<td><code>--cube-response double</code></td>
</tr>
<tr>
<td>dr</td>
<td>La décision est une réponse Take / Pass. Implique une décision de videau ; avec <code>dd</code>, <code>dr</code> l'emporte.</td>
<td><code>--cube-response takepass</code></td>
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
<td>gt:holding</td>
<td>La position relève d'un plan de jeu donné, du point de vue du joueur au trait : <code>race</code>, <code>bearin</code> (rentrée sous contact), <code>crunch</code>, <code>backgame</code>, <code>acepoint</code>, <code>blitz</code>, <code>primevprime</code>, <code>mutualholding</code>, <code>holding</code>, <code>contact</code>. Répétable (<code>gt:holding gt:mutualholding</code>). Étiquette dérivée comme la phase : calculée à partir du plateau, jamais modifiable, recalculée par <code>blunderdb repair</code>.</td>
<td><code>--game-type</code></td>
</tr>
<tr>
<td>#prime</td>
<td>La position porte ce <strong>tag</strong> dans l'un de ses commentaires. Un tag est un <code>#mot</code> écrit dans la prose ; rien ne le déclare. La comparaison est délimitée, donc <code>#prime</code> ne trouve pas <code>#priming</code> — c'est toute la différence avec le filtre de texte, qui cherche une sous-chaîne. Répétable, et les tags se <strong>cumulent</strong> (<code>#prime #backgame</code> demande les deux) : une position porte plusieurs tags, donc en nommer deux veut dire « les deux ».</td>
<td>—</td>
</tr>
<tr>
<td>n&gt;x</td>
<td>La position a été rencontrée au moins x fois dans la base — le nombre de décisions prises sur elle, tous matchs et tous joueurs confondus ; avec <code>pl</code>, <code>pl!</code> ou <code>op</code>, seules les occurrences de ce joueur comptent. Formes <code>n&gt;3</code>, <code>n&lt;2</code>, <code>n3,10</code> et <code>n4</code> (exactement quatre).</td>
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
<td>tm&gt;x</td>
<td>Le joueur 1 a mis plus de x secondes à décider (coup ou videau). Un coup dont la durée est inconnue ne répond jamais. Avec <code>E</code>, la durée et l'erreur sont celles du même coup joué.</td>
<td>—</td>
</tr>
<tr>
<td>tm&lt;x</td>
<td>Le joueur 1 a mis moins de x secondes à décider.</td>
<td>—</td>
</tr>
<tr>
<td>tmx,y</td>
<td>Le joueur 1 a mis entre x et y secondes à décider, bornes comprises.</td>
<td>—</td>
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
<td><code>au'Alice'</code></td>
<td>La position porte un commentaire signé de cet auteur (nom entier, sans distinction de casse).</td>
<td><code>--comment-author</code></td>
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
<td>Date d'ajout de la position à la base après x (AAAA/MM/JJ). Ce n'est pas la date du match (<code>md</code>) : l'ajout la fixe et la fusion de deux bases la conserve.</td>
<td>—</td>
</tr>
<tr>
<td>T&lt;x</td>
<td>Date d'ajout de la position à la base avant x (AAAA/MM/JJ). Ce n'est pas la date du match (<code>md</code>).</td>
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
<td>tn'nom'</td>
<td>Rechercher dans les tournois dont le nom est <code>nom</code> : la casse est ignorée et <code>*</code> remplace n'importe quelle suite de caractères (ex: <code>tn'open*'</code>).</td>
<td><code>--tournament-name</code></td>
</tr>
<tr>
<td>rd:x</td>
<td>Rechercher dans les matchs de la ronde x (ex: <code>rd:3</code>, <code>rd:Finale</code>) : le texte de la ronde est comparé sans tenir compte de la casse, <code>*</code> remplace n'importe quelle suite de caractères. Répétable (<code>rd:1 rd:2</code>) : l'une ou l'autre.</td>
<td><code>--round</code></td>
</tr>
<tr>
<td>ml:x</td>
<td>Le match a une longueur de x points. Formes <code>ml:7</code>, <code>ml:5,9</code>, <code>ml&gt;5</code> et <code>ml&lt;9</code> (bornes comprises).</td>
<td><code>--match-lengths</code></td>
</tr>
<tr>
<td>md:x..y</td>
<td><strong>Date du match</strong>, lue dans la colonne <code>match_date</code> de la position (la date du plus ancien match qui l'atteint), et non la date de création de l'analyse (<code>T</code>). Chaque borne est une année, un mois ou un jour (<code>2024</code>, <code>2024-06</code>, <code>2024-06-15</code>) et couvre toute sa durée, bornes comprises : <code>md:2024-01..2024-12</code> va du 1er janvier au 31 décembre 2024. Formes <code>md:2024</code> (toute l'année), <code>md&gt;2024-06</code> et <code>md&lt;2024-06</code>.</td>
<td><code>--match-date</code></td>
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
<td>Rechercher les positions issues d'un match impliquant le joueur indiqué, sur l'un ou l'autre camp (ex: <code>pl'Alice'</code>). La casse est ignorée et <code>*</code> remplace n'importe quelle suite de caractères (<code>pl'Ali*'</code>).</td>
<td><code>--player</code></td>
</tr>
<tr>
<td><code>pl!'nom'</code></td>
<td>Seulement les décisions prises par ce joueur : le joueur au trait est celui qui occupe le camp de ce nom dans le match (joueur 1 ou joueur 2). Même règles de casse et de joker que <code>pl</code>.</td>
<td><code>--player</code> <code>--seat-only</code></td>
</tr>
<tr>
<td><code>op'nom'</code></td>
<td>Seulement les matchs où ce joueur est l'adversaire de celui de <code>pl</code> (<code>pl'Alice' op'Bob'</code> : Alice contre Bob, de l'un ou l'autre côté ; avec <code>pl!</code>, les seules décisions d'Alice). Sans <code>pl</code>, <code>op</code> seul désigne un joueur à l'un ou l'autre camp, comme <code>pl</code>. Mêmes règles de casse et de joker.</td>
<td><code>--opponent</code></td>
</tr>
<tr>
<td>pr&gt;x</td>
<td>Le <strong>PR du match</strong> du joueur qui a pris la décision est d'au moins x, lu dans les statistiques par match au camp de ce joueur (pas le PR de l'adversaire) ; un match sans PR est exclu. Formes <code>pr&gt;8</code>, <code>pr&lt;5</code> et <code>pr4,9</code>, bornes comprises.</td>
<td><code>--pr</code></td>
</tr>
<tr>
<td>ad:xg</td>
<td>Moteur et profondeur de l'analyse enregistrée pour la position. Moteurs : <code>xg</code>, <code>gnubg</code>, <code>bgblitz</code>, <code>hedgehog</code>, <code>gammonnet</code> (début du nom du moteur, casse ignorée). Profondeurs : <code>3ply</code> (exactement 3 plis), <code>3ply+</code> (au moins 3 plis), <code>book</code> (livre d'ouvertures), <code>rollout</code> (rollout, y compris XG Roller et Roller++). Répétable : les moteurs sont des alternatives, les profondeurs aussi, et un moteur et une profondeur doivent tous deux convenir (<code>ad:xg ad:gnubg ad:3ply+</code>).</td>
<td><code>--analysis</code></td>
</tr>
<tr>
<td>like, like42, like&lt;12, like42*</td>
<td>Classe le résultat par distance croissante à une position cible, au lieu de le restreindre : <code>like</code> prend la position courante, <code>like42</code> celle d'indice 42, <code>like&lt;12</code> écarte ce qui est à plus de douze pions-pas, <code>like42*</code> élargit la classe de la cible à tous les types de décision et aux deux régimes, argent et match. Voir Panneau Recherche.</td>
<td>—</td>
</tr>
</tbody>
</table>
<p>Les valeurs <code>pl</code>, <code>pl!</code>, <code>op</code>, <code>tn</code>, <code>m</code> et <code>t</code> s'ouvrent par un guillemet ou une apostrophe et se ferment par l'un des deux. Un jeton qui garde un guillemet sans former une valeur complète est ignoré, et <code>blunderdb search --query</code> le signale. Un tag peut contenir une apostrophe (<code>#l'ouverture</code>), jamais un guillemet. Le point-virgule est le séparateur des listes d'identifiants et de tags : il ne figure dans aucune valeur, et <code>ma1;2</code> n'est pas un jeton (écrire <code>ma1 ma2</code>).</p>
<h3>Transcrire depuis un terminal</h3>
<p>La ligne de commande n'a pas de commande de transcription : les gestes se tapent dans l'onglet <em>Transcription</em>. Hors de l'application, ils passent par <code>blunderdb call</code> (Transcrire par l'API), par exemple <code>blunderdb call transcriptions.apply --db base.db --if-match 3 --json '{"id":1,"gesture":{"Kind":"validate"}}'</code>. <code>--if-match</code> nomme la révision rendue par l'appel précédent ; chaque appel est sa propre session, sans <code>sessionId</code> ni annulation d'un appel à l'autre.</p>
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
        Le réseau de neurones <strong>strehl-prob5-512-512-256-256</strong> est l'œuvre d'<strong>Alexander Strehl</strong> (<em>alexstrehl/backgammon-ai-engine</em>, MIT). La recherche, le modèle de
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
        Les bases de bearoff unilatérale (6 points, 15 pions, pour l'EPC) et bilatérale (6 points, 6 pions, pour les verdicts de videau en course) sont calculées par blunderDB lui-même, par un portage
        de l'outil <em>makebearoff</em> de <strong>GNU Backgammon</strong> (GNUbg) ; le résultat est identique octet pour octet à celui de gnubg, dont l'empreinte SHA-256 sert de référence. GNUbg est
        un logiciel libre sous licence GPL.
    </li>
    <li>Les fichiers de match sont lus par <em>xgparser</em>, <em>gnubgparser</em>, <em>bgfparser</em> et <em>ogxmparser</em> (MIT).</li>
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
