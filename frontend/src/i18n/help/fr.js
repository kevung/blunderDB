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
<p>Les positions sont stockées dans une base de données représentée par un fichier <em>.db</em>. L'application de bureau ouvre ce fichier directement, jamais une adresse réseau : le mode serveur (Mode headless (serveur)) est un autre mode du même binaire, et l'on passe de l'un à l'autre en exportant ou en migrant la base, pas en pointant l'application vers une URL.</p>
<h3>Interactions principales</h3>
<p>Les principales interactions possibles avec blunderDB sont:</p>
<ul>
<li>ajouter une nouvelle position,</li>
<li>modifier une position existante,</li>
<li>copier l'image du board dans le presse-papier (PNG) via <strong>CTRL-X</strong>, ou avec l'analyse complète via <strong>CTRL-X CTRL-X</strong>,</li>
<li>supprimer une position existante,</li>
<li>rechercher une ou plusieurs positions,</li>
<li>importer des matchs depuis différentes sources (XG, GNUbg, BGBlitz, Jellyfish), y compris les commentaires depuis les fichiers XG,</li>
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
<li>étudier les positions par répétition espacée (panneau Anki),</li>
<li>afficher les métadonnées de la base de données (panneau métadonnées).</li>
</ul>
<p>Des fenêtres modales peuvent s'afficher pour:</p>
<ul>
<li>afficher l'aide de blunderDB,</li>
<li>afficher le catalogue des visites guidées (voir Visites guidées et base d'exemple),</li>
<li>paramétrer l'export de la base de données,</li>
<li>configurer blunderDB, notamment la langue de l'interface (voir Configuration).</li>
</ul>
<p>La zone d'affichage principale met à disposition à l'utilisateur:</p>
<ul>
<li>un board afin d'afficher ou d'éditer une position de backgammon,</li>
<li>le niveau et le propriétaire du cube,</li>
<li>le compte de course de chaque joueur,</li>
<li>le score de chaque joueur,</li>
<li>les dés à jouer. Si aucune valeur n'est affichée sur les dés, la position des dés indique quel joueur a le trait et que la position est une décision de cube. Lorsque la décision de cube est une réponse à un doublement (prise/passe), le videau proposé est affiché au centre du plateau, à la valeur offerte.</li>
</ul>
<p>Un clic droit sur le plateau ouvre un menu contextuel proposant : évaluer la position affichée dans le panneau Eval, évaluer son miroir, copier l'image du plateau avec son analyse dans le presse-papier (l'équivalent de <em>CTRL-X CTRL-X</em>, moins facile à découvrir), <strong>enregistrer l'image dans un fichier</strong> en SVG ou en PNG, ouvrir une nouvelle vue sur cette position, et — si la position vient déjà de la base — l'ajouter à un paquet Anki (répétition espacée).</p>
<p>Le presse-papier est le geste courant ; enregistrer est l'autre besoin — l'illustration d'un article, d'un message de forum, d'une leçon. Le <strong>SVG</strong> y est proposé parce que le plateau en est un : c'est la forme qui survit à un agrandissement, celle qu'on met dans un document sans la flouter. Le PNG en dérive, comme la copie dans le presse-papier : un seul rendu, trois destinations, donc aucune ne peut diverger des autres. Ce menu n'apparaît pas dans le panneau Eval ni dans le panneau Recherche, où le bouton droit sert déjà à poser les pions de l'autre couleur. Voir Amener une position dans le panneau Eval pour amener une position dans le panneau Eval.</p>
<p>La barre d'état est structurée de gauche à droite par les informations suivantes:</p>
<ul>
<li>la ligne de commande, accessible en appuyant sur la touche <em>ESPACE</em>,</li>
<li>un message d'information lié à une opération réalisée par l'utilisateur,</li>
<li>l'index de la position courante, suivi du nombre de positions dans la bibliothèque courante (ou les informations de coup/partie lors de la navigation dans un match),</li>
<li>le <strong>compteur de bibliothèque</strong> — « 412 positions · 38 blunders · 5 matchs » — où chaque nombre <strong>ouvre ce qu'il compte</strong> : les positions, la recherche <code>E&gt;100</code> préparée dans la ligne de commande, ou la liste des matchs. Un chiffre qu'on ne peut pas suivre est une décoration. Le seuil des blunders est celui des statistiques, cent millipoints : deux seuils feraient dire deux choses au même mot.</li>
</ul>
<div class="admonition note">
<p>Dans le cas de positions issues d'une recherche par l'utilisateur, le nombre de positions indiqué dans la barre d'état correspond au nombre de positions filtrées.</p>
</div>
<p>L'onglet <strong>Anki</strong> porte un <strong>badge</strong> quand des cartes sont à réviser, tous paquets confondus. Ce chiffre est la raison d'ouvrir l'onglet ; il n'a donc rien à faire derrière lui. Zéro n'affiche rien : un badge qui dit « 0 » est du bruit.</p>
<p>La commande <code>log</code> ouvre le <strong>journal d'activité</strong> : les deux cents dernières lignes du fichier de journal, un bouton pour les copier — de quoi joindre un rapport à un signalement — et un autre pour ouvrir le dossier qui les contient. Le journal n'est ni filtré ni reformaté : un journal qu'on embellit est un journal qu'on ne peut plus citer.</p>
<p>Dans l'<strong>historique de recherche</strong> du panneau Recherche, chaque jeton d'une commande enregistrée s'affiche en pastille nommée — <em>Sans contact</em>, <em>Erreur de coup</em> — plutôt qu'en jeton nu. La commande exacte reste en infobulle, car c'est elle qu'on relance ; et un jeton que blunderDB ne reconnaît pas s'affiche <strong>tel quel</strong> plutôt que traduit au plus proche.</p>
<h3>Onglets de vues</h3>
<p>Sous la barre d'outils, une barre d'onglets permet de travailler avec plusieurs <strong>vues</strong> en parallèle. Chaque vue est un espace de travail indépendant qui conserve sa propre liste de positions, l'index de la position courante, la position affichée, l'analyse et le coup sélectionné, le panneau actif, le commentaire en cours ainsi que le contexte de navigation dans un match. Il est ainsi possible, par exemple, de garder une recherche ouverte dans une vue tout en parcourant un match dans une autre.</p>
<ul>
<li><strong>Créer une vue</strong> : cliquer sur le bouton <em>+</em> de la barre d'onglets ou appuyer sur <em>CTRL-T</em>. La nouvelle vue démarre comme une copie de la vue courante.</li>
<li><strong>Fermer une vue</strong> : cliquer sur la croix de l'onglet ou appuyer sur <em>CTRL-W</em>. La dernière vue ne peut pas être fermée.</li>
<li><strong>Changer de vue</strong> : cliquer sur un onglet, appuyer sur <em>CTRL-PageUp</em> / <em>CTRL-PageDown</em> (ou <em>MAJ-J</em> / <em>MAJ-K</em>) pour passer à la vue précédente / suivante, ou <em>CTRL-1</em> à <em>CTRL-9</em> pour atteindre directement la n-ième vue.</li>
<li><strong>Renommer une vue</strong> : double-cliquer sur l'onglet, saisir le nouveau nom et valider avec <em>ENTREE</em>.</li>
</ul>
<p>Les vues sont enregistrées avec l'état de session de la base de données et restaurées à sa réouverture.</p>
<h3>Configuration</h3>
<p>Le bouton de configuration (icône en forme de rouage) situé dans la barre d'outils, à gauche du bouton d'aide, ouvre la fenêtre de configuration de blunderDB. Elle est organisée en six onglets :</p>
<ul>
<li><strong>Interface</strong> — langue, échelle d'affichage, position du panneau ;</li>
<li><strong>Couleurs</strong> — les couleurs du plateau ;</li>
<li><strong>Bearoff</strong> — les tables de sortie utilisées par le panneau Eval ;</li>
<li><strong>gammonNet</strong> — les réglages de l'évaluateur embarqué, décrits ci-dessous ;</li>
<li><strong>Dossier surveillé</strong> — l'import automatique des matchs qui arrivent dans un dossier, décrit ci-dessous ;</li>
<li><strong>Identité d'émetteur</strong> — la clé qui signe vos filigranes, décrite à la section Diffuser une base : origine et mot de passe.</li>
</ul>
<p>L'onglet <em>Interface</em> propose d'abord un <strong>thème</strong> : <em>suivre le système</em>, <em>clair</em>, <em>sombre</em>, <em>contraste élevé</em> ou <em>imprimable</em>. Le thème règle les couleurs de l'interface et <strong>propose une palette de plateau</strong> — une interface sombre autour d'un plateau clair n'est pas un thème sombre, c'est la moitié d'un, puisque le plateau occupe l'essentiel de la fenêtre.</p>
<p>Vous gardez le dernier mot, et le mécanisme le garantit plutôt que de le promettre : l'onglet <em>Couleurs</em> continue de régler le plateau directement, et une couleur choisie après le thème est la vôtre. Au démarrage, seuls les jetons de l'interface sont appliqués, jamais la palette du plateau — celle que vous avez réglée est déjà chargée, et la réécrire à chaque lancement effacerait votre travail une session à la fois. Voir <code>ADR-0038 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0038-a-named-theme-carries-the-board-palette-and-the-user-still-has-the-last-word.md&gt;</code>__.</p>
<p><em>Suivre le système</em> est le réglage par défaut : il obéit à la préférence clair/sombre du bureau, y compris lorsqu'elle change en cours de session. Un outil n'impose pas son clair ou son sombre à un bureau qui a déjà tranché.</p>
<p>L'onglet <em>Interface</em> permet aussi de choisir la langue parmi l'anglais, le français, l'allemand, l'italien, l'espagnol, le finnois, le japonais, le grec et le russe. L'ensemble de l'interface (barre d'outils, panneaux, messages, aide) est traduit dans la langue sélectionnée. Le choix de la langue est enregistré et conservé d'une session à l'autre.</p>
<p>Le même onglet propose aussi le bouton <strong>Compacter la base</strong>, qui récupère l'espace disque laissé par les suppressions (matchs, tournois, purges) : la base de données ne rétrécit jamais toute seule quand on supprime des données, il faut demander explicitement ce compactage. L'opération peut prendre du temps sur une grosse base et nécessite, temporairement, environ deux fois sa taille en espace disque libre (blunderDB refuse de démarrer plutôt que de risquer un compactage interrompu) ; une confirmation est donc demandée avant de lancer l'opération. Le résultat — l'espace gagné, en mégaoctets — s'affiche ensuite dans la barre d'état. La même opération est disponible en ligne de commande via <code>blunderdb vacuum</code> (voir Interface en ligne de commande (CLI)).</p>
<p>Le bouton <strong>Ouvrir le dossier des journaux</strong>, juste en dessous, ouvre le dossier contenant le journal de l'application — utile pour joindre des détails à un signalement de problème, en particulier quand blunderDB est lancé depuis un raccourci ou un double-clic, sans terminal attaché pour afficher quoi que ce soit.</p>
<p>La case <strong>Vérifier les mises à jour au démarrage</strong>, désactivée par défaut, interroge une fois la page des dernières versions du dépôt GitHub à chaque lancement et affiche, dans la barre d'état, un message si une version plus récente est disponible — jamais une fenêtre qui bloque l'utilisation. Cette vérification reste désactivée automatiquement sur une installation passée par un gestionnaire de paquets (Flatpak, Homebrew, un paquet de distribution…) : c'est ce canal-là qui gère alors les mises à jour, pas blunderDB lui-même.</p>
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
<p>L'onglet <em>Général</em> porte enfin <strong>Réparer les analyses</strong> : les colonnes d'analyse que la recherche et les statistiques interrogent sont une projection des analyses stockées, lesquelles restent intactes. Un défaut de projection se répare donc sans rien réimporter. C'est explicite et jamais automatique — réécrire les colonnes d'analyse de quelqu'un au seul motif qu'il ouvre sa base n'est pas une chose qu'un outil doit faire dans son dos. Le même <code>blunderdb repair</code> est disponible en ligne de commande.</p>
<p>L'onglet <strong>gammonNet</strong> règle l'évaluateur embarqué (voir <code>ADR-0011 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0011-gammonnet-is-ported-to-go-and-the-representation-boundary-sits-at-the-evaluator-s-edge.md&gt;</code>__). Deux profondeurs de recherche y sont réglables, nommées et conservées séparément — abaisser l'une ne modifie jamais l'autre :</p>
<ul>
<li><strong>Profondeur d'affichage</strong> — le confort interactif pendant l'édition du plateau ; jamais écrite en base.</li>
<li><strong>Profondeur d'analyse</strong> — ce que le lot d'analyse après import écrit dans l'Analyse d'une position.</li>
</ul>
<p>Les deux valent par défaut <strong>2-ply</strong>, la configuration canonique. L'onglet propose aussi l'<strong>élagage</strong> (par défaut <code>k=12</code>) et le <strong>nombre de coups candidats affichés</strong> (par défaut 10), ainsi qu'une case <strong>analyser automatiquement après import</strong> qui, une fois activée, vérifie après chaque import s'il reste des positions <strong>sans aucune analyse</strong> (ni gammonNet, ni XG, ni GNUbg, ni BGBlitz — la règle est « une évaluation ne comble qu'un trou », jamais un remplacement) et, le cas échéant, lance en tâche de fond une analyse gammonNet à la profondeur d'analyse configurée. Un bouton <strong>Analyser maintenant</strong> relance manuellement le même rattrapage, utile pour une bibliothèque constituée avant l'existence de cette fonctionnalité.</p>
<p>Un second bouton, <strong>Ré-analyser les positions périmées</strong>, couvre le cas inverse : une position déjà analysée par gammonNet, mais dont l'analyse stockée a été écrite par une version de moteur plus ancienne que celle en cours d'exécution, ou à une profondeur différente de la profondeur d'analyse configurée ci-dessus, y est signalée comme périmée et réévaluée. Une position portant en plus une analyse XG, GNUbg ou BGBlitz n'est jamais touchée par ce bouton, quel que soit son contenu gammonNet — la protection d'<code>ADR-0013 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md&gt;</code>__ reste inconditionnelle. Le nombre affiché à côté de chaque bouton (positions sans analyse, positions périmées) est purement informatif ; le lot recalcule sa propre liste au moment de démarrer.</p>
<p>Les deux lots sont <strong>bornés, visibles et annulables, jamais un démon silencieux</strong> : leur progression (<code>positions analysées / total</code>) et un bouton d'annulation apparaissent dans la barre de statut pendant toute leur durée, et disparaissent une fois terminés au profit d'un message résumant le résultat — combien de positions ont été <strong>analysées</strong>, combien ont été <strong>refusées</strong> (une position que gammonNet décline d'évaluer, comme un score de match hors de portée de sa table de match, ce qui n'est jamais une panne) et combien ont <strong>échoué</strong> (retentées, inchangées, au prochain lancement). Fermer l'application pendant l'un ou l'autre ne perd rien : chaque position analysée est écrite au fil de l'eau, et un prochain lancement reprend exactement là où l'analyse s'était arrêtée, sans aucun journal à tenir.</p>
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
<li><strong>L'import est silencieux.</strong> Vous étiez en train d'étudier une position quand vos matchs sont arrivés : vous reprendre l'écran serait le pire moment. L'import se fait sans fenêtre, et la barre d'état affiche un bandeau donnant le compte des matchs importés, ignorés (doublons) et en échec, avec un bouton qui ouvre le compte rendu complet si vous le souhaitez. Tout le reste est identique à un import manuel : mêmes doublons détectés, même lot d'import, même analyse automatique si elle est activée.</li>
</ul>
<p>L'intervalle par défaut est de dix secondes ; le plancher est de deux. Le dossier n'est pas parcouru récursivement : un dossier surveillé est l'endroit où un outil dépose ses matchs, pas une arborescence à explorer. Un partage réseau démonté n'arrête pas la surveillance et ne fait pas non plus passer son contenu pour nouveau à son retour.</p>
<p>La même surveillance existe en ligne de commande, avec <code>blunderdb import --type batch --dir &lt;dossier&gt; --watch</code> (voir Interface en ligne de commande (CLI)) : c'est la forme qu'un serveur, une tâche planifiée ou un script peuvent utiliser.</p>
<p>La fenêtre de configuration regroupe également des réglages d'affichage de l'interface. Un curseur d'<strong>échelle de l'interface</strong> permet d'agrandir ou de réduire l'ensemble des éléments, ce qui est utile sur les écrans à haute densité ou pour améliorer la lisibilité. Un menu <strong>position des panneaux</strong> détermine l'emplacement des panneaux (recherche, matchs, analyse) par rapport au plateau : <em>en bas</em>, <em>sur le côté</em> ou <em>automatique</em> (le côté est alors choisi sur les écrans larges afin de mieux exploiter l'espace disponible). Comme les autres réglages, ces choix sont conservés d'une session à l'autre.</p>
<h3>Visites guidées et base d'exemple</h3>
<p>Pour faciliter la prise en main, blunderDB propose des <strong>visites guidées</strong> de l'interface. Le catalogue des visites s'ouvre depuis la barre d'outils ou avec la commande <code>tour</code> (alias <code>tutorial</code>). Sept visites sont disponibles : un tour général de l'interface, et des visites dédiées à la recherche de positions, à la revue des matchs, à la revue des tournois, au panneau Eval, à la révision Anki et aux statistiques. Chaque visite met en évidence les éléments concernés de l'interface, étape par étape, ouvre au passage le panneau dont elle parle, et peut être rejouée à tout moment. Au premier démarrage, le tour général est proposé automatiquement.</p>
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
<h3>Panneau Analyse</h3>
<p>Le panneau <strong>Analyse</strong> (<em>CTRL-L</em>) affiche les données d'analyse de la position courante importées depuis eXtreme Gammon (XG), GNUbg ou BGBlitz. Il présente les meilleures alternatives (coups de pions ou décisions de videau) avec leurs valeurs d'équité et les erreurs correspondantes. La touche <em>d</em> bascule entre l'analyse des coups de pions et l'analyse du cube. Lors de la navigation dans un match, le coup effectivement joué est mis en évidence dans la liste des alternatives. Appuyer sur <em>CTRL-L</em> ou exécuter la commande <code>list</code> pour afficher ou masquer le panneau.</p>
<p>Sous les tableaux, une <strong>phrase</strong> dit parfois ce que la décision jouée a coûté et pourquoi : « Vous perdez 120 mMWC : le coup joué laisse trois blots là où 13/7 8/7 n'en laisse qu'un. » Elle est produite par six règles mesurables — l'exposition, un point du jan fait ou manqué, les chances de gammon abandonnées, une sécurité qui coûte plus qu'elle ne rapporte, et les deux sens d'une erreur de videau (doubler trop tard ou trop tôt, prendre trop large ou passer trop serré).</p>
<p>La règle qui compte est celle du <strong>silence</strong> : la phrase n'apparaît que si une règle s'applique de façon confiante, et sur une erreur qui dépasse le seuil à partir duquel les moteurs s'accordent à dire qu'elle en est une. Le reste du temps, il n'y a pas de phrase — ni cadre vide, ni « nous ne savons pas ». Une explication fausse coûte plus cher que pas d'explication : elle apprend quelque chose d'inexact.</p>
<p>Lorsqu'une position a été jugée par <strong>plusieurs moteurs</strong>, une bande en tête du panneau les met côte à côte : une ligne par moteur, avec sa profondeur et sa réponse — le verdict de videau, ou son propre meilleur coup. Elle dit d'abord s'ils sont d'accord, et c'est le désaccord qui la justifie : « XG dit double, prend ; gammonNet dit pas de double » se lit d'un coup d'œil, là où il fallait comparer deux tableaux en diagonale.</p>
<p>Le meilleur coup d'un moteur est le meilleur <strong>de ce moteur</strong> : la liste des coups candidats est triée par équité, tous moteurs confondus, et son premier élément n'est donc le meilleur coup d'aucun d'eux en particulier.</p>
<p>La bande n'apparaît que s'il y a effectivement plusieurs moteurs, et elle n'existe que dans ce panneau : le panneau Eval présente <strong>une</strong> décision, celle du moteur embarqué (<code>ADR-0017 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0017-the-panel-shows-position-facts-plus-the-one-decision-the-board-asks.md&gt;</code>__), et une comparaison n'y aurait pas sa place.</p>
<p>Les coups sont écrits comme on les lit sur le plateau, ici comme dans le panneau Eval : le pion le moins avancé bouge d'abord, et <strong>un pion qui enchaîne plusieurs dés ne s'écrit qu'une fois</strong> — un 64 joué avec le même pion se lit <code>24/14</code>, et <code>24/14*</code> s'il frappe en arrivant. Le détail de l'enchaînement ne réapparaît que lorsqu'il dit quelque chose de plus : une frappe <em>en cours de route</em> conserve son point de passage, <code>24/18* 18/14</code>, sans quoi la frappe en 18 disparaîtrait de la notation.</p>
<p>L'équité d'une analyse importée suit la même règle que le panneau Eval : la colonne annonce son référentiel, « Équité (money) » ou « Équité (match) » selon le score de la position analysée, jamais un simple « Équité » muet sur l'échelle. Les règles <strong>Jacoby</strong> et <strong>Beaver</strong> actives sur une position en money game s'affichent, elles aussi, en badges sous le tableau de décision de videau.</p>
<h3>Panneau Commentaires</h3>
<p>Le panneau <strong>Commentaires</strong> (<em>CTRL-P</em>) affiche, ajoute et modifie les commentaires associés à la position courante. Une position peut en porter plusieurs : ils sont tous affichés, du plus récent au plus ancien. Les commentaires importés depuis les fichiers XG sont automatiquement associés aux positions correspondantes. Appuyer sur <em>CTRL-P</em> ou exécuter la commande <code>comment</code> pour afficher ou masquer le panneau.</p>
<p>Chaque commentaire venu d'un fichier porte une <strong>étiquette de provenance</strong> (<code>XG</code>, <code>GNU BG</code>, <code>BGF</code>, ou <em>importé</em> lorsque la provenance n'a pas été enregistrée). Les commentaires que vous avez écrits n'en portent pas : c'est le cas courant, et le signaler à chaque ligne serait du bruit. Modifier un commentaire importé vous l'attribue : après la modification, la phrase est la vôtre.</p>
<p>Cette distinction a une conséquence visible ailleurs : supprimer un match n'efface plus une position sur laquelle <strong>vous</strong> aviez écrit. Une note reprise du fichier source, elle, disparaît avec le match qui l'a apportée.</p>
<h4>Les tags</h4>
<p>Un <strong>tag</strong> est un <code>#mot</code> écrit dans un commentaire. Rien ne le déclare, aucune table ne le porte, et c'est voulu : le vocabulaire est votre prose, et exiger une déclaration avant de pouvoir taguer transformerait une habitude en paperasse.</p>
<p>Ce qui manquait, c'était l'autre moitié : <strong>voir</strong> le vocabulaire qu'on s'est construit, et cliquer un tag plutôt que se rappeler comment on l'écrivait. La commande <code>tags</code>, ou le bouton <code>#</code> de la zone de saisie, ouvre la fenêtre du vocabulaire : les tags de cette base, chacun avec le <strong>nombre de positions</strong> qui le portent, cliquables pour lancer la recherche correspondante. Sous la liste figurent les tags recommandés que la base n'utilise pas encore — un vocabulaire tiré de la littérature du backgammon (<code>#blitz</code>, <code>#prime</code>, <code>#holding</code>, <code>#backgame</code>, <code>#containment</code>, <code>#crunch</code>, <code>#ace-point</code>, <code>#timing</code>…), suggéré et jamais imposé : un tag absent de cette liste vaut exactement autant qu'un tag qui y figure.</p>
<p>Pendant la frappe, taper <code>#</code> propose les tags que <strong>cette base</strong> utilise déjà, puis les recommandés. C'est ce qui évite d'écrire <code>#back-game</code> un jour et <code>#backgame</code> le lendemain, ce que rien d'autre ne rattraperait.</p>
<p>La recherche par tag s'écrit <code>#prime</code> dans la ligne de commande. Elle est <strong>délimitée</strong> : <code>#prime</code> ne trouve pas <code>#priming</code>, là où une recherche de texte ordinaire, qui cherche une sous-chaîne, ne sait pas les distinguer. Plusieurs tags se <strong>cumulent</strong> — <code>s #prime #backgame</code> demande les positions qui portent les deux — parce qu'une position porte plusieurs tags : en nommer deux ne peut vouloir dire que « les deux ». C'est l'inverse du filtre de phase ou de provenance, où une position n'a qu'une valeur et où nommer deux valeurs ne peut vouloir dire que « l'une ou l'autre ».</p>
<p>La même liste s'obtient hors de l'interface avec <code>blunderdb list --type tags</code> (voir Interface en ligne de commande (CLI)).</p>
<h3>La corbeille</h3>
<p>Supprimer une position, une collection ou un commentaire passe désormais par une <strong>corbeille</strong> : la suppression a bien lieu, mais une copie de ce qui disparaît est gardée trente jours. La commande <code>trash</code> ouvre la fenêtre qui les liste, avec pour chacune <em>Restaurer</em> et <em>Supprimer</em>.</p>
<p>Une position restaurée revient avec <strong>son analyse et ses commentaires</strong> — la rendre nue serait une restauration de nom seulement. Elle ne revient pas sous son ancien numéro : la ligne d'origine n'existe plus, et blunderDB la réenregistre par son empreinte, ce qui garantit qu'elle ne crée jamais de doublon mais lui donne un nouvel identifiant. Une collection revient avec sa liste ; les positions qu'elle contenait, elles, n'avaient jamais été supprimées — une collection est une vue sur elles.</p>
<p>Ce qui a plus de trente jours est supprimé par la commande <code>vacuum</code>, jamais à l'ouverture d'une base : ne pas faire de <code>vacuum</code>, c'est tout garder.</p>
<div class="admonition note">
<p>La corbeille ne voyage pas. Un export ne l'emporte pas, et supprimer un match n'y met rien : la purge des positions orphelines qui suit une suppression de match est un nettoyage automatique, pas un geste de l'utilisateur — voir la règle de rétention dans Panneau Matchs.</p>
</div>
<h3>Panneau Recherche</h3>
<p>Le panneau <strong>Recherche</strong> (<em>CTRL-F</em> ou <em>TAB</em>) permet de filtrer les positions selon des critères combinables librement : structure de pions, type de décision de videau, magnitude d'erreur, dates, tags, etc. La touche <em>TAB</em> ouvre simultanément le panneau de recherche et l'éditeur de position, permettant de définir une structure de pions à rechercher sur le plateau.</p>
<p>Pour affiner une recherche parmi les positions actuellement filtrées, utiliser la commande <code>ss</code> suivie de filtres (ex: <code>ss nc</code>, <code>ss E&gt;40</code>). Le panneau de recherche propose également une case à cocher <em>Rechercher dans les résultats actuels</em> pour la même fonctionnalité.</p>
<p>Le panneau propose un contrôle explicite du <strong>type de décision</strong> recherché : <em>Indifférent</em> (aucun filtre), <em>Pions</em> (décisions de coup) ou <em>Videau</em> (décisions de cube). Lorsque <em>Videau</em> est sélectionné, une seconde liste précise le sous-type : <em>Tous</em>, <em>Double / Pas de double</em> (le joueur au trait doit décider de doubler) ou <em>Prise / Passe</em> (réponse à un doublement adverse). Le contrôle est synchronisé avec le plateau : modifier les dés ou le videau sur le plateau met à jour le type de décision, et inversement. En mode <em>Prise / Passe</em>, le videau est affiché au centre du plateau à la valeur offerte ; cette valeur reste éditable.</p>
<p>La <strong>phase de partie</strong> — ouverture, milieu de partie, course, sortie des pions — est une étiquette calculée par blunderDB à partir du plateau seul, jamais modifiable, et disponible en recherche par le jeton <code>ph:</code> de la ligne de commande (<code>ph:race</code>, répétable : <code>ph:race ph:bearoff</code>). Trois de ses quatre frontières sont celles que GNU Backgammon emploie pour aiguiller ses réseaux ; la quatrième, où s'arrête l'ouverture, est une convention de blunderDB : une position en est encore à l'ouverture tant qu'aucun des deux camps n'a déplacé plus de quatre pions de leurs points de départ, qu'aucun pion n'est sorti et qu'aucun n'est sur la barre.</p>
<div class="admonition note">
<p>L'étiquette est recalculée par la commande <code>blunderdb repair</code>. Sur une base ouverte pour la première fois avec cette version, le calcul est fait une fois, à l'ouverture. Une base dont les phases n'ont jamais été calculées ne renvoie rien pour <code>ph:</code> — rien, plutôt qu'une réponse fausse.</p>
</div>
<p>La commande <code>like</code> répond à une autre question que les jetons : elle remplace la liste parcourue par les positions les plus <strong>proches</strong> de la position courante, de la plus proche à la plus lointaine. La proximité est une distance de transport, exprimée en pions-pas — la quantité de mouvement de pions qui sépare les deux positions — et le point de vue est toujours celui du joueur au trait. Ce n'est pas un filtre : la similarité <strong>classe</strong> toute la bibliothèque au lieu de la restreindre, et ne se combine donc pas avec les jetons.</p>
<p>Le jeton <code>n</code> compte les <strong>rencontres</strong> : <code>n&gt;3</code> retient les positions auxquelles plus de trois coups aboutissent, tous matchs confondus. C'est une autre question que « qu'ai-je raté » — une position rencontrée vingt fois et bien jouée dix-neuf reste celle qu'il faut savoir par cœur. Le compte porte sur les coups, pas sur les matchs : la même position deux fois dans un match compte pour deux, parce que c'étaient deux décisions.</p>
<p>Une phrase en toutes lettres peut remplacer les jetons, avec la commande <code>ask</code> : <code>ask mes blunders de videau au score</code>. La phrase est <strong>traduite en jetons</strong>, écrits dans la barre de commande — on les relit, puis on lance. Rien n'est deviné et rien ne part sur le réseau : le vocabulaire est fixe, la même phrase rend toujours la même requête, et ce qui n'a pas été compris est <strong>dit</strong> plutôt que passé sous silence. Une traduction fausse se voit ainsi avant de renvoyer des résultats faux, et les jetons s'apprennent en les lisant.</p>
<p>Deux intentions ne sont pas des jetons et se posent sur le plateau de recherche plutôt que dans la ligne : « de videau » ou « de pions » (le type de décision) et « au score » ou « en argent ». <code>ask</code> les y pose.</p>
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
<p>Le panneau de recherche comporte trois onglets sur son bord gauche : <em>Recherche</em> (les filtres), <em>Historique</em> et <em>Enregistrés</em>. L'onglet <strong>Historique</strong> liste les recherches passées avec leur date et leur commande : un clic sélectionne une recherche et affiche la position associée sur le plateau, un double-clic la ré-exécute. Chaque entrée peut être enregistrée dans la bibliothèque de filtres (icône signet, en donnant un nom au filtre) ou supprimée. L'onglet <strong>Enregistrés</strong> contient la <strong>bibliothèque de filtres</strong> : double-cliquer sur un filtre enregistré pour relancer la recherche correspondante (voir Annexe: Utilisation avancée des filtres). La commande <code>history</code> (alias <code>hi</code>) ouvre le panneau de recherche.</p>
<div class="admonition tip">
<p>Se référer à la liste des commandes pour la liste des filtres disponibles.</p>
</div>
<h3>Panneau Collections</h3>
<p>Le panneau <strong>Collections</strong> (<em>CTRL-B</em>) permet de gérer des collections de positions. Les collections peuvent être créées, renommées et supprimées. Des positions peuvent y être ajoutées ou retirées (touche <em>Suppr</em>, confirmation demandée). Double-cliquer sur une collection pour parcourir ses positions avec les touches <em>GAUCHE</em> et <em>DROITE</em>. L'ordre des collections et des positions au sein des collections peut être modifié par glisser-déposer. Appuyer sur <em>CTRL-B</em> ou exécuter la commande <code>collection</code> pour afficher ou masquer le panneau.</p>
<h3>Import : ce qui est écrit, ce qui ne l'est jamais</h3>
<p>Importer un match, une position ou une autre base ajoute ce qui manque ; cela ne remplace pas ce qui est déjà là.</p>
<ul>
<li><strong>Une position n'est jamais dupliquée.</strong> C'est son identité — pions, videau, dés, score — qui la reconnaît, jamais le fichier d'où elle vient : la même position rencontrée dans deux matchs reste une seule ligne.</li>
<li><strong>Une analyse par moteur.</strong> eXtreme Gammon, GNUbg, BGBlitz et l'évaluateur embarqué cohabitent sur une même position, et le panneau Analyse indique l'origine de chacune. Importer l'une n'efface pas l'autre.</li>
<li><strong>Une analyse importée n'est jamais recalculée.</strong> blunderDB la range telle quelle, avec son étiquette de niveau (« 3-ply », « XG Roller++ », « Book »), ses équités, ses erreurs, ses probabilités et la chance du lancer. La règle est « une évaluation ne comble qu'un trou » : l'analyse automatique après import ne visite que les positions sans <strong>aucune</strong> analyse, et <em>Ré-analyser les positions périmées</em> laisse intacte toute position portant une analyse importée (voir Configuration).</li>
<li><strong>Réimporter le même fichier ne réécrit rien.</strong> Le match est reconnu comme déjà présent ; seules les marques posées dans le logiciel d'origine sont ajoutées, sans toucher aux commentaires ni aux analyses.</li>
<li><strong>Ce que blunderDB n'écrit jamais</strong> : une chance recalculée — elle est lue dans le fichier source, ou reste inconnue — et un rollout, dont il n'ouvre pas les données dans un fichier <code>.xg</code> et qu'il ne sait pas produire.</li>
</ul>
<p>Une collection peut être <strong>vivante</strong> : sa composition n'est plus une liste faite à la main mais le résultat d'une <strong>recherche</strong>, réévalué chaque fois qu'on l'ouvre. Le bouton ◇ en tête de la collection la rend vivante avec la dernière recherche lancée ; ◈ signale qu'elle l'est déjà, et le même bouton la rend à sa liste. Rien n'est détruit en la rendant vivante : les positions qu'elle contenait sont toujours là quand on revient en arrière.</p>
<p>Une collection vivante dont la requête porte un jeton que cette version ne connaît plus <strong>refuse de s'ouvrir</strong> en le disant, plutôt que de renvoyer toute la base. C'est la seule panne qu'un filtre enregistré ne doit pas avoir : s'élargir en silence.</p>
<h3>Panneau Matchs</h3>
<p>Le panneau <strong>Matchs</strong> (<em>CTRL-Tab</em>) liste les matchs importés. Double-cliquer sur un match (ou appuyer sur <em>ENTREE</em>) pour naviguer dans ses coups. La commande <code>m</code> reprend la navigation dans le dernier match visité.</p>
<p>L'utilisateur peut:</p>
<ul>
<li>parcourir les coups d'un match en utilisant les touches <em>GAUCHE</em> et <em>DROITE</em>,</li>
<li>passer d'une partie à l'autre à l'aide des touches <em>PageUp</em> et <em>PageDown</em>,</li>
<li>afficher l'analyse des coups (pions et cube) en appuyant sur <em>CTRL-L</em>,</li>
<li>basculer entre l'analyse des coups de pions et du cube avec la touche <em>d</em>,</li>
<li>voir le coup effectivement joué mis en évidence dans l'analyse.</li>
</ul>
<p>La dernière position visitée dans chaque match est mémorisée et restaurée automatiquement. Appuyer sur <em>CTRL-Tab</em> ou exécuter la commande <code>match</code> pour afficher ou masquer le panneau.</p>
<p>Le bouton <strong>⊕</strong> d'une ligne enrichit ce match depuis un fichier. Il n'y a rien de nouveau derrière : réimporter le même match dans un autre format l'enrichit déjà en place — l'empreinte canonique reconnaît qu'il s'agit du même match, et les analyses et commentaires du second fichier viennent compléter le premier. Ce que le bouton apporte, c'est qu'on le trouve : personne ne devine qu'un import est aussi un enrichissement. Le compte rendu qui suit dit lequel des deux a eu lieu — « enrichis : 1 » plutôt que « importés : 1 ».</p>
<p>Chaque match peut être exporté en transcription Jellyfish <code>.mat</code> via le bouton ⬇ de la liste des matchs ou le bouton <em>.mat</em> de la fiche du match.</p>
<p>Le bouton <strong>Fusionner les joueurs</strong> de la barre d'outils du panneau ouvre une fenêtre listant tous les noms de joueurs de la base avec leur nombre de matchs : sélectionner les variantes d'orthographe d'un même joueur, choisir le nom canonique à conserver, puis fusionner. Utile pour unifier les statistiques par joueur lorsqu'un même joueur apparaît sous plusieurs noms.</p>
<p>Lorsqu'un match est ouvert, une <strong>barre d'informations</strong> apparaît au-dessus du plateau : elle rappelle les joueurs en présence (<em>joueur 1</em> contre <em>joueur 2</em>) ainsi que le contexte du match (événement, lieu, ronde, date et longueur du match, lorsque ces informations sont disponibles). Cette barre s'affiche aussi en dehors du mode match : lorsqu'une position étudiée (issue d'une recherche, d'une collection ou d'un accès direct) provient d'un ou de plusieurs matchs, elle en indique la <strong>provenance</strong> — le premier match concerné et, le cas échéant, un badge « +N » listant les autres au survol. Une position importée seule, qu'aucun match ne référence, n'affiche rien.</p>
<p>À l'ouverture d'une base contenant des matchs, le panneau <strong>Matchs</strong> est affiché d'emblée et la revue débute directement sur la première position, afin de commencer immédiatement la navigation.</p>
<div class="admonition note">
<p>Une base de données ne peut être ouverte en écriture que par une seule fenêtre à la fois. Si vous ouvrez une base déjà ouverte dans une autre fenêtre de blunderDB, elle s'ouvre en <strong>lecture seule</strong> : la navigation, la recherche et l'analyse restent possibles, mais toute modification est désactivée et la barre de titre affiche « [lecture seule] ».</p>
</div>
<div class="admonition tip">
<p>Se référer à Raccourcis clavier pour les raccourcis disponibles.</p>
</div>
<h3>Panneau Tournois</h3>
<p>Le panneau <strong>Tournois</strong> (<em>CTRL-Y</em>) permet de regrouper des matchs en tournois pour un suivi organisé et une analyse statistique par événement. Les tournois peuvent être créés, renommés et supprimés ; les matchs peuvent leur être assignés. Les statistiques du panneau Stats peuvent être filtrées par tournoi. Appuyer sur <em>CTRL-Y</em> pour afficher ou masquer le panneau.</p>
<p>Les tournois se remplissent d'eux-mêmes à l'import. Les fichiers XG, GnuBG et BGF nomment leur événement ; à l'import d'un match nouveau, blunderDB le classe dans le tournoi de ce nom et crée celui-ci s'il n'existe pas encore. La date et le lieu du tournoi restent vides — c'est ici qu'on les renseigne. Un match déjà présent dans la base n'est jamais reclassé : réimporter son fichier ne défait pas le rangement fait à la main.</p>
<p>La colonne <strong>PR</strong> de chaque tournoi affiche le PR du <strong>joueur de référence</strong> — c'est-à-dire le joueur présent dans le plus grand nombre de matchs du tournoi (en cas d'égalité, celui ayant pris le plus de décisions). Le PR ne mélange donc pas votre jeu avec celui de vos adversaires : pour vos propres tournois, il reflète votre performance seule. Le nom du joueur de référence apparaît en infobulle au survol de la valeur.</p>
<h3>Panneau Stats</h3>
<h4>Introduction</h4>
<p>Le panneau <strong>Stats</strong> permet d'analyser son niveau de jeu et de suivre sa progression dans le temps à partir des positions importées dans la base de données. Il calcule et affiche les indicateurs <strong>PR</strong> (<em>Performance Rating</em>) et <strong>MWC cost</strong> (Match Winning Chance cost) pour l'ensemble des positions ou un sous-ensemble filtré.</p>
<p>Le panneau Stats est particulièrement utile pour :</p>
<ul>
<li><strong>situer son niveau</strong> par rapport aux bandes de niveau (<em>Classe mondiale</em>, <em>Expert</em>, *Avancé*…) grâce au PR global ;</li>
<li><strong>suivre sa progression</strong> tournoi après tournoi ou match après match grâce aux graphiques de l'onglet Progression ;</li>
<li><strong>identifier ses points faibles</strong> : onglet Erreurs pour voir la répartition entre coups joués et décisions de videau, et la distribution des magnitudes d'erreur ;</li>
<li><strong>comparer les joueurs de la base</strong> entre eux, une ligne par joueur, grâce à l'onglet Joueurs — utile pour suivre une compétition entière ;</li>
<li><strong>accéder directement aux positions concernées</strong> en cliquant sur n'importe quel indicateur (drill-down).</li>
</ul>
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
</ul>
<p>Un bouton <strong>Reset</strong> remet tous les filtres à zéro (sauf le joueur auto-détecté).</p>
<div class="admonition note">
<p>Les filtres sont persistés dans la configuration de blunderDB (<code>config.yaml</code>) et sont restaurés à la prochaine ouverture.</p>
</div>
<h4>Toggle PR / MWC</h4>
<p>Le bouton <strong>PR / MWC</strong> en haut du panneau bascule la métrique affichée dans tous les onglets.</p>
<p><strong>PR (Performance Rating)</strong></p>
<blockquote>
<p>L'erreur moyenne d'équité par décision comptée, multipliée par 500 comme le font eXtreme Gammon et GNUbg : un PR de 5,0 vaut 0,010 d'équité perdue par décision, soit 10 millipoints (mpt). La règle de comptage exacte — quelles décisions entrent au dénominateur, comment le score est converti — est celle de Annexe : Modèle de statistiques — alignement XG / gnuBG / blunderDB.</p>
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
<p>Probabilité cumulée de victoire de match perdue à cause des erreurs, sur l'ensemble du jeu de données filtré. Calculé à partir de la MET Kazaross-XG2 embarquée dans blunderDB.</p>
<div class="admonition caution">
<p>Le MWC cost <strong>n'est pas applicable</strong> aux positions <em>money-game</em> (sans enjeu de match). Ces positions sont exclues du calcul MWC. Les valeurs MWC dépendent de la MET utilisée ; elles ne sont pas directement comparables entre logiciels utilisant des METs différentes.</p>
</div>
</blockquote>
<p>Le basculement PR ↔ MWC est instantané : aucun recalcul backend n'est effectué.</p>
<h4>Le rapport HTML</h4>
<p>Le bouton <strong>Rapport HTML</strong> de l'en-tête du panneau produit un document <strong>autonome</strong> : un seul fichier, sans image externe, sans feuille de style distante, sans script. Les diagrammes y sont des SVG en ligne, dessinés par le même rendu que le plateau à l'écran, avec votre palette. Il s'ouvre dans n'importe quel navigateur, s'envoie par courriel, et <strong>s'imprime en PDF par le navigateur lui-même</strong> — ce qui évite d'embarquer un générateur de PDF pour produire ce que tout le monde a déjà.</p>
<p>Il contient les indicateurs du périmètre courant (positions, matchs, décisions comptées, PR global, pions et videau), puis les <strong>dix décisions les plus coûteuses</strong>, chacune avec son diagramme, son coût, le match d'où elle vient et le meilleur coup lorsqu'une analyse le donne.</p>
<p>Le rapport porte le <strong>filtre courant</strong> du panneau Stats. Un rapport qui ne dit pas son périmètre est un rapport dont les chiffres ne veulent rien dire : réglez le filtre — un tournoi, une plage de dates, un joueur — avant de le produire.</p>
<h4>Onglet Dashboard</h4>
<p>L'onglet <strong>Dashboard</strong> donne une vue synthétique des indicateurs clés.</p>
<h5>Cartes de niveau</h5>
<p>Trois cartes affichent le PR (ou MWC) pour :</p>
<ul>
<li><strong>PR Global</strong> — toutes les décisions (coups + videau) ;</li>
<li><strong>PR Coup</strong> — coups joués seulement ;</li>
<li><strong>PR Cube</strong> — décisions de videau seulement.</li>
</ul>
<p>Cliquer sur une carte charge dans le panneau d'analyse les positions du sous-ensemble correspondant (drill-down).</p>
<div class="admonition note">
<p>Le nombre total de décisions est affiché en bas de chaque carte au survol.</p>
</div>
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
<h4>Onglet Ventilations</h4>
<p>L'onglet <strong>Ventilations</strong> découpe les mêmes décisions que les chiffres globaux selon quatre axes. Aucun d'eux ne redéfinit ce qui compte comme une décision : ce serait un second PR sous le même nom.</p>
<ul>
<li><strong>Par phase de partie</strong> — ouverture, milieu de partie, course, sortie des pions. C'est ce qui répond à « mon PR en course contre mon PR en contact ». L'étiquette est calculée depuis le plateau (voir Panneau Recherche) ; une base dont les phases n'ont jamais été calculées range tout sous <em>Non classée</em>, et <code>blunderdb repair</code> la remplit.</li>
<li><strong>Par plan de jeu</strong> — course, blitz, tenue, backgame, amorce contre amorce… C'est la ventilation pour laquelle le classificateur existe : « où est-ce que je perds le plus ? », plan par plan. Même étiquette dérivée que la phase, mêmes réserves, et <code>blunderdb repair</code> la remplit de la même façon.</li>
<li><strong>Par étiquette</strong> — les <code>#mot</code> écrits dans les commentaires. Une position peut en porter plusieurs : <strong>ces lignes ne s'additionnent pas au total</strong>, et le panneau le dit sous le tableau. Une étiquette qualifie, elle ne partitionne pas.</li>
<li><strong>Par score</strong> — l'écart au but des deux camps, lu du côté du joueur au trait, donc du côté de celui qui décide. La ligne <em>Money</em> est la partie d'argent. Une cellule de moins de dix décisions est <strong>grisée avec son effectif visible</strong> plutôt que cachée : trop peu pour être lue, mais l'omission reste vérifiable.</li>
</ul>
<div class="admonition note">
<p>La partie Crawford n'est pas distinguée : blunderDB n'enregistre pas cet indicateur sur une position. L'effet pratique est faible — une partie Crawford n'a aucune décision de videau — mais l'omission est réelle et vaut mieux d'être écrite que laissée à deviner.</p>
</div>
<h4>Étude et jeu réel</h4>
<p>La commande <code>blunderdb list --type study --days 30</code> met côte à côte, plan de jeu par plan de jeu, trois nombres : combien de <strong>positions distinctes</strong> ont été révisées sur la période, quel était le PR <strong>avant</strong> elle, quel est le PR <strong>depuis</strong>.</p>
<p>Trois nombres, et pas un quatrième. Il n'y a <strong>ni colonne de gain ni flèche</strong>, parce que rien ici ne contrôle quoi que ce soit : le joueur a pu rencontrer des adversaires plus forts, changer de format, ou simplement jouer plus de courses ce mois-ci. Le rapprochement est celui du lecteur ; une colonne qui annoncerait un effet affirmerait une causalité que ces données ne portent pas. Les nombres, eux, sont exacts.</p>
<p>Les révisions sont comptées en <strong>positions distinctes</strong> : une carte revue quatre fois dans le mois est une position étudiée, et compter les répétitions ferait passer un mois de bachotage pour un mois de couverture. Les décisions du PR, elles, sont toutes comptées — chacune a été prise une fois. Un PR appuyé sur moins de dix décisions s'affiche <code>—</code>, avec son effectif visible à côté.</p>
<h4>Onglet Joueurs</h4>
<p>Les quatre onglets précédents décrivent <strong>un</strong> joueur ; l'onglet <strong>Joueurs</strong> les compare tous. Il affiche une ligne par joueur de la base, ce qui répond au besoin d'un organisateur suivant une compétition entière plutôt qu'un joueur en particulier.</p>
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
<td>Le nom <strong>tel qu'il figure dans les matchs</strong>. Un joueur enregistré sous deux orthographes apparaît donc sur deux lignes ; utilisez la fusion de joueurs pour les réunir.</td>
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
<td>Nombre d'erreurs graves (au moins 0,100 EMG).</td>
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
<li><strong>Ouvrir le détail d'un joueur</strong> — cliquez sur une ligne. Le joueur est sélectionné dans la barre de filtres et l'affichage bascule sur l'onglet Dashboard.</li>
<li><strong>Restreindre la période</strong> — les filtres de dates, de tournois et de longueur de match s'appliquent normalement, ce qui permet de borner le tableau aux dates d'une compétition.</li>
</ul>
<div class="admonition note">
<p>Dans cet onglet, la liste <strong>Joueur</strong> et le choix du <strong>type de décision</strong> sont désactivés : le tableau montre tous les joueurs, et il ventile déjà les décisions de pions et de videau en colonnes distinctes.</p>
</div>
<div class="admonition important">
<p>Un tiret (« — ») signale une valeur <strong>jamais mesurée</strong>, à ne pas confondre avec zéro. C'est notamment le cas de la colonne Chance pour tout match importé avant la version 2.15.0 du schéma : la chance n'était alors pas conservée, et rien ne permet de la reconstituer après coup — il faut réimporter les fichiers source. Les formats qui ne la transportent pas (BGF, Jellyfish <code>.mat</code>) n'en fourniront jamais.</p>
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
<li>Le MWC cost est calculé à partir de la <strong>MET Kazaross-XG2</strong>, table de référence de facto dans le backgammon compétitif. Les résultats ne sont pas directement comparables avec des logiciels utilisant d'autres METs. C'est la même table, lue par le même point d'entrée, que celle dont l'évaluateur embarqué se sert pour ses décisions de videau au score : les statistiques et le moteur ne peuvent pas diverger là-dessus. Elle donne ses valeurs propres jusqu'à 25 points à faire de chaque côté ; au-delà, elle est prolongée par une table de Zadeh calculée comme celle de GNUbg, jusqu'à 64.</li>
<li>Les positions <em>money-game</em> (sans score de match) sont <strong>exclues</strong> du calcul MWC. Si votre base de données contient beaucoup de positions money-game, le MWC cost peut être sous-estimé ou indisponible.</li>
<li>Le MWC cost est cumulatif sur l'ensemble du jeu de données filtré — pas un indicateur par décision. Il mesure l'impact total de vos erreurs sur vos chances de victoire.</li>
</ul>
<h3>Panneau Eval</h3>
<p>Le panneau <strong>Eval</strong> (<em>CTRL-E</em>) évalue en direct la position posée sur le plateau, quelle qu'elle soit ; sur une position de bearoff il se spécialise et calcule en plus l'EPC (Effective Pip Count). Il est activé en appuyant sur <em>CTRL-E</em>, en cliquant sur l'onglet Eval dans le panneau inférieur, ou en exécutant la commande <code>epc</code>. Cette commande garde son nom d'origine : le panneau s'est appelé <em>EPC</em>, puis <em>Bearoff</em>, avant de devenir <em>Eval</em> — c'est donc ici qu'il faut chercher ce qu'une version antérieure appelait le panneau Bearoff, le nom ne désignant plus que l'onglet de configuration des tables de sortie.</p>
<p>Le panneau montre toujours la <strong>seule décision</strong> que la position posée sur le plateau appelle — jamais deux à la fois — et les faits qui vont avec. Chaque quantité se lit dans l'axe qui lui convient plutôt que dans un axe unique imposé : la probabilité de gain, de gammon, de backgammon et l'équité cubeless de chaque joueur, calculées <em>avant le jet</em>, se lisent <strong>par joueur</strong> (bas, haut, puis Δ), à gauche de la décision de videau, quand aucun dé n'est posé. Les faits et la décision restent côte à côte : la décision de videau ne passe jamais sous les chiffres qui la justifient, quelles que soient la langue de l'interface et la position sur le plateau. Dès que des dés sont posés, ces mêmes valeurs <em>avant le jet</em> changent d'axe : elles se lisent <strong>au trait</strong>, en tête de la liste des coups candidats, sous forme d'une ligne italique <em>avant le jet</em> — pas un coup candidat de plus, un repère contre lequel lire chaque coup. L'écart entre cette ligne et un coup contient la chance du jet, jamais le mérite du coup, et elle ne porte donc aucune colonne d'erreur. Sur une position de bearoff pur, un second tableau, toujours <strong>par joueur</strong> et toujours présent, dés posés ou non, porte l'EPC, le pip count, le wastage, le nombre moyen de lancers et l'écart type ; ces cinq colonnes ne migrent jamais. Les deux tableaux sont empilés et partagent la même grille de colonnes : mêmes bords, mêmes repères de colonne, une seule colonne de pastilles — ils se lisent comme un seul objet à deux étages. Le badge de régime, l'attribution du moteur (la profondeur de la dernière évaluation y figure aussi) et la case <em>Défi</em> forment une bande à part, alignée à droite au-dessus des tableaux.</p>
<p>Seule la liste des coups candidats défile — la ligne <em>avant le jet</em>, elle aussi, reste épinglée au-dessus d'elle ; le reste du panneau (faits, badge, décision de videau) reste toujours visible, sans réglage particulier de la taille du panneau.</p>
<p>Le tableau de faits et la décision sont calculés par gammonNet, embarqué, sans XG ni gnubg. Le calcul suit la position sans jamais figer l'interface : une profondeur 0-ply s'affiche immédiatement à chaque geste, puis, après une demi-seconde d'immobilité, une évaluation plus profonde (2 plis par défaut, réglable dans l'onglet <em>gammonNet</em> de la configuration) la remplace en arrière-plan — tout nouveau geste annule ce calcul de fond. La profondeur affichée dans la bande de badges, ou au sein du badge de régime sur une position de course, est toujours celle qui a effectivement produit le chiffre montré, jamais celle demandée ; elle ne se répète pas sur chaque ligne, puisqu'une évaluation en direct partage la même profondeur pour tous les coups. L'équité des coups candidats et de la décision de videau suit le score de la position : en money game elle est exprimée en points, à un score de match en <strong>équité normalisée</strong> — la même échelle que XG et GNU Backgammon, où gagner la valeur du videau courant vaut +1 et la perdre −1 — jamais mélangées dans un même tableau. L'en-tête de la colonne le dit explicitement plutôt que de laisser deviner l'échelle : « Équité (money) » en money game, « Équité (match) » à un score de match. Elle tient compte du <strong>videau vivant</strong> : la recherche valorise chaque position finale par le modèle de videau (Janowski, efficacité mesurée) dans l'état du videau de la position, comme le font XG et GNU Backgammon en évaluation <em>cubeful</em>. C'est ce qui rend visibles au score les effets gammon-go et gammon-save — à 4-away/2-away, le joueur mené joue 8/2 6/2 sur un 6-4 d'ouverture parce que son double précoce donnera au gammon la valeur du match, ce qu'une évaluation sans videau ne peut pas voir. La ligne <em>avant le jet</em>, elle, reste une équité <strong>cubeless</strong> : c'est un fait de la position, pas une décision. Ce panneau ne modifie jamais la base : c'est un calcul, pas une analyse enregistrée. Cliquer un coup candidat l'affiche sur le plateau sous forme de flèches, exactement comme dans le panneau Analyse. Le bouton <strong>?</strong> discret, dans la bande de badges, mène au dépôt du moteur <code>gammonNet &lt;https://github.com/kevung/gammonNet&gt;</code>_ ; l'attribution complète (réseau Strehl, configuration gammonNet) figure dans les Remerciements de l'aide.</p>
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
<p>Un troisième badge, <strong>Videau max</strong>, apparaît lorsque l'identifiant d'origine plafonne le videau — au score comme en money game. Celui-là ne décrit pas le calcul affiché au-dessus : l'évaluateur intégré ne modélise pas de plafond, et le verdict est donc celui d'un videau libre. C'est justement pourquoi le badge est là : un videau plafonné est la seule raison visible pour laquelle blunderDB et eXtreme Gammon peuvent annoncer deux verdicts différents sur la même position.</p>
<p>Le badge de régime, la profondeur d'évaluation, le lien vers le moteur et la case <em>Défi</em> forment une bande à part, alignée à droite au-dessus des tableaux.</p>
<p>Le <strong>joueur au trait</strong> et la <strong>position du videau</strong> s'éditent directement sur le plateau, comme en mode édition : cliquer le rectangle bearoff/score d'un joueur lui donne le trait ; cliquer le videau fait tourner centré → possédé bas → possédé haut (clic droit en sens inverse). La valeur du videau reste épinglée — en money game les équités sont exprimées en unités du videau courant, seul son propriétaire compte. L'analyse est recalculée aussitôt. En régime estimé, le badge lui-même est cliquable et ouvre directement l'onglet <em>Bearoff</em> de la configuration ; son infobulle explique pourquoi (verdict de videau non estimable, <code>ADR-0009 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md&gt;</code>__) et comment étendre le domaine exact.</p>
<p>Le <strong>score</strong> s'édite lui aussi directement sur le plateau, comme en mode édition : clic gauche sur le rectangle score d'un joueur décrémente son nombre de points à faire, clic droit l'incrémente. Sortir du score <em>money</em> (-1, -1) en éditant un seul camp aligne automatiquement l'autre camp sur la même valeur plutôt que de laisser un score incohérent. Sur une position de bearoff en régime <em>exact</em>, passer d'un score money à un score de match laisse la probabilité de gain telle quelle (une lecture en base, valable quel que soit le référentiel) mais bascule l'équité et le verdict de videau affichés vers ceux du régime <em>évalué</em> — la table exacte étant money par construction, elle ne sait pas répondre à la question posée au score. Le badge devient alors composite (« exact (gain) · évalué (videau) ») pour le dire explicitement.</p>
<p>Les <strong>dés</strong> s'éditent enfin de la même façon, et ce sont eux qui décident de la question posée : des dés posés font une décision de pions (la liste des coups candidats), pas de dés une décision de videau. Clic gauche sur un dé fait monter sa valeur (6 revient à 1), clic droit la fait descendre (1 revient à 6) ; cliquer un dé sur un plateau qui n'en a pas en pose deux d'un coup — un seul dé ne serait ni une décision de pions ni une décision de videau. Cliquer le rectangle d'un joueur retire les dés pour poser une question de videau, et le clic suivant sur un dé les remet tels qu'ils étaient.</p>
<p><em>RETOUR ARRIERE</em>, ou un double-clic en dehors du plateau, efface la position : plateau vide, score money (-1, -1), pas de dés posés — des valeurs propres au panneau Eval, différentes de celles utilisées en mode édition (7 partout, dés 3-1), pour rester cohérentes avec ce que le panneau affiche par défaut.</p>
<h4>Matrice du videau</h4>
<p>Une décision de videau n'est pas une propriété du damier. Les mêmes pions, le même compte de pips, se doublent à 2-away/4-away et ne se doublent pas à 4-away/2-away ; un joueur qui a appris la réponse money n'a appris qu'une case d'une grille. Le panneau Eval montre la case que la position porte ; la <strong>matrice du videau</strong> montre la grille entière.</p>
<p>La commande <code>cm</code> l'ouvre sur la position affichée. Chaque case donne le verdict à un score : la ligne est le nombre de points qu'il reste à faire au joueur au trait, la colonne celui qu'il reste à faire à son adversaire. Les quatre verdicts s'écrivent <em>PD</em> (pas de double), <em>DP</em> (double, prend), <em>DR</em> (double, refuse) et <em>TB</em> (trop bon) ; une case que le moteur refuse porte un point d'interrogation et dit pourquoi au survol, qui donne aussi les trois équités de la case. Trois longueurs de match sont proposées : 5, 7 et 9 points.</p>
<p>Le score de la position est remplacé par celui de chaque case ; son <strong>videau</strong>, lui, est conservé. La grille répond à « à quel score retournerais-je <em>ce</em> videau », pas à ce que ferait une position centrée. Elle est post-Crawford d'un bout à l'autre : pendant la partie Crawford le videau n'est pas en jeu, et une colonne de « vous ne pouvez pas doubler » ne dirait rien de la position.</p>
<p>Chaque case est une recherche à part entière. Le moteur tient compte du score — il ne joue pas la même partie à 2-away qu'à 7-away —, donc une seule recherche relue à travers des équités de match différentes serait fausse exactement là où le score compte. La grille arrive d'abord en 0-ply, puis se recalcule à la profondeur d'affichage configurée une fois la fenêtre au repos : la même escalade que le reste du panneau, pour une grille de 9 points qui coûte environ une seconde et demie.</p>
<p>La même grille se calcule hors de l'interface, avec la commande cubematrix de la ligne de commande.</p>
<h4>Amener une position dans le panneau Eval</h4>
<p>Le panneau s'ouvre par défaut sur une position de bearoff, mais l'étude part le plus souvent d'une position déjà en main. Deux gestes l'y amènent :</p>
<ul>
<li><strong>Clic droit sur le plateau</strong>, dans un panneau d'analyse ou pendant la navigation d'un match, puis <em>Évaluer cette position</em> : le panneau Eval s'ouvre directement sur cette position, telle qu'elle est affichée. Le menu contextuel n'apparaît pas dans le panneau Eval ni dans le panneau Recherche, où le bouton droit sert déjà à poser les pions de l'autre couleur.</li>
<li><strong>CTRL-C puis CTRL-V</strong> : copier la position depuis le panneau d'analyse, puis la coller une fois dans le panneau Eval. Le collage accepte aussi un identifiant venu d'ailleurs — un XGID (eXtreme Gammon, GNU Backgammon, une autre instance de blunderDB) ou un OGID (OpenGammon) : il suffit qu'il soit dans le presse-papier.</li>
<li><strong>La commande</strong> <code>import XGID=…</code> (ou <code>import OGID=…</code>) pour le cas où l'identifiant n'est pas dans le presse-papier mais dans un message, sur un forum lu dans un terminal, ou produit par un script. C'est le même verbe qu'<code>import</code> tout court : sans argument il ouvre un sélecteur de fichiers, avec un argument il lit l'identifiant. Le chemin est ensuite identique à celui du collage — même lecture, même déduplication, même ouverture de la position importée.</li>
</ul>
<p>Un OGID ne porte qu'une position : ni évaluation, ni commentaire. La position arrive donc sans analyse, exactement comme un XGID nu, et l'évaluateur intégré peut la combler ensuite.</p>
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
<p><strong>Probabilité de gain et verdict, régime évalué.</strong> Hors du domaine exact, la probabilité de gain provient de la sortie brute de gammonNet (recherche 0- ou 2-plis selon le geste, jamais lue dans une table), et le verdict d'un « Decide » Janowski appliqué à cette sortie — la recherche <em>joue</em> la trajectoire au lieu d'en résumer un instantané, ce qui est précisément ce que le régime estimé ne pouvait pas faire (voir plus bas) et permet, seul des trois régimes avec l'exact, un verdict <strong>au score de match</strong>.</p>
<p>Ce régime a été mesuré, pas seulement supposé, contre la table two-sided intégrée (<code>TestEvalMeasure</code>, 4000 décisions money échantillonnées, paramètres canoniques 2-plis k=12) : accord de verdict money <strong>93,4 %</strong> (3735/4000), ventilé par distance au point de prise de gammonNet — 61,1 % à moins de 1 % du point de prise (la zone la plus sensible à un pile ou face), 88,3 % entre 1 et 5 %, 91,5 % entre 5 et 10 %, 94,0 % entre 10 et 20 %, 94,4 % au-delà. Écart de probabilité de gain : moyenne 0,85 %, médiane 0,44 %, 95e centile 3,21 %, maximum 8,30 %. Écart d'équité cubeful : moyenne 0,039, médiane 0,018, 95e centile 0,151, maximum 0,406. La forme est celle attendue : l'essentiel du désaccord se concentre exactement au point de prise, où deux méthodes légitimement différentes divergent le plus sur une décision serrée — pas une erreur diffuse qui coûterait de l'équité partout.</p>
<p>Cette mesure porte sur des décisions <strong>money</strong>, en course. Le verdict au score de match — que ce régime est seul à savoir rendre — et les positions de contact n'ont pas de mesure publiée : ce qui précède ne se transporte pas à ces cas.</p>
<p><strong>Pourquoi pas plus profond que 2 plis ?</strong> Parce que la mesure dit que cela ne rapporte rien. Une décision de pions coûte 99 ms à 2 plis et 8,4 s à 3 plis sur la même machine — <strong>85 fois plus</strong>. Sur quarante décisions réelles rejouées aux deux profondeurs, la recherche plus profonde a changé d'avis <strong>deux fois</strong>, et les deux fois le gain qu'elle s'attribuait à elle-même valait au plus 0,0005 d'équité normalisée : deux ordres de grandeur sous 0,020, le seuil à partir duquel eXtreme Gammon parle d'erreur. Par décision, tous cas confondus, le gain est de 0,0000.</p>
<p>Le réglage n'est donc pas proposé. Il ne s'agit pas de dire que 3 plis ne vaut rien en général, mais que sur <em>ce</em> réseau, avec le filtre canonique, il ne paie pas l'attente de quelqu'un devant un panneau. La mesure est reproductible (<code>TestThreePlyMeasure</code>) et la conclusion se rejugera si le réseau change.</p>
<p><strong>Pourquoi le verdict estimé n'existe-t-il pas ?</strong> Ce qui suit vise spécifiquement la méthode par <em>convolution</em> (régime estimé), pas le régime évalué ci-dessus : l'équité cubeful est un problème de <em>trajectoire</em> (quand doubler), qu'aucun résumé statistique de la position ne capture — le meilleur modèle statique mesuré laisse une erreur résiduelle (écart type 0,016 d'équité, maximum 0,20) qui suffit à inverser toutes les décisions serrées. De même, la conversion du verdict au score de match via une table d'équités de match a été mesurée insuffisante (12 % de désaccords avec l'analyse 2-ply de GNUbg, avec de vraies bourdes). Un verdict faux affiché avec aplomb étant pire que pas de verdict, la convolution n'a jamais eu le droit d'afficher de verdict — c'est une recherche qui joue la trajectoire, pas un résumé statistique, qui comble ce trou.</p>
<div class="admonition note">
<p>Les bases de bearoff sont des tables mathématiques immuables. blunderDB les calcule lui-même, à l'identique de l'outil <code>makebearoff</code> de GNUbg — octet pour octet — dans l'onglet <em>Bearoff</em> de la configuration ou avec <code>blunderdb bearoff generate</code>.</p>
</div>
<h3>Panneau Anki</h3>
<p>Le panneau <strong>Anki</strong> (<em>CTRL-K</em>) permet d'étudier des positions par répétition espacée en utilisant l'algorithme FSRS. L'utilisateur peut créer des paquets à partir de collections ou de résultats de recherche.</p>
<p><strong>Création de paquets :</strong> Cliquez sur <em>New Deck</em> pour créer un paquet à partir d'une collection ou des résultats de recherche courants. Les paquets basés sur une recherche se synchronisent automatiquement à l'activation de l'onglet Anki.</p>
<p><strong>Révision :</strong> Sélectionnez un paquet puis cliquez sur <em>Study</em> (ou double-cliquez sur un paquet) pour commencer la révision des cartes dues. Chaque carte affiche la position correspondante sur le plateau. Évaluez votre rappel avec les touches <em>1</em> (À revoir), <em>2</em> (Difficile), <em>3</em> (Bien), ou <em>4</em> (Facile). Appuyez sur <em>Esc</em> pour arrêter et revenir à la liste des paquets.</p>
<p><strong>Les décisions de videau font deux cartes, enchaînées.</strong> Une décision de videau est deux questions — « double ? », puis « prend ? » — et blunderDB les enregistre depuis toujours comme deux positions. Un paquet qui n'en sélectionne qu'une moitié reçoit l'autre : la décision est complétée, pas augmentée. Et quand les deux sont dues, la seconde vient <strong>immédiatement</strong> après la première.</p>
<p>Chacune garde sa propre note et son propre calendrier : ce ne sont pas deux temps d'une même carte, ce sont deux cartes. L'enchaînement n'avance aucune échéance — il ordonne les cartes déjà dues, rien de plus. Les deux naissant ensemble, elles sont dues ensemble la première fois, et c'est là qu'il sert.</p>
<p><strong>Afficher la réponse :</strong> La carte pose une question — quel coup jouer, ou quelle action de videau. Réfléchissez, puis appuyez sur <em>ESPACE</em> (ou cliquez sur la zone masquée) pour dévoiler la réponse : l'analyse enregistrée de la position, telle que l'onglet Analyse la présente. Elle apparaît sous les boutons d'évaluation, qui restent à leur place et à portée. Cliquer sur un coup de la liste le montre sur le plateau.</p>
<p>Rien ne vous oblige à dévoiler la réponse pour évaluer : si vous êtes sûr de vous, les touches <em>1</em> à <em>4</em> restent actives. La réponse se remasque à la carte suivante, mais pas si vous changez simplement d'onglet — allez consulter le panneau Éval ou le commentaire de la position, elle vous attendra au retour.</p>
<p>Une position dépourvue d'analyse enregistrée l'indique directement, sans zone masquée.</p>
<p><strong>Limiter la séance.</strong> Par défaut, une séance de révision va jusqu'au bout des cartes dues. Vous pouvez la borner à un nombre de cartes, par paquet, dans les Paramètres : cochez <em>Limiter la séance</em> et indiquez combien de cartes une séance doit servir. Quand la limite est atteinte, la séance s'arrête en le disant — le message distingue « limite atteinte, tant de cartes encore dues » d'une file réellement épuisée. Pour continuer malgré tout, l'entraînement libre est là : il sert d'autres positions sans rien modifier au planning.</p>
<p>Une limite de <strong>0</strong> ne sert aucune carte : c'est un état à part entière, utile pour geler un paquet le temps de préparer un tournoi, et ce n'est pas la même chose que « pas de limite ». Le bouton <em>Study</em> est alors inactif.</p>
<p>La limite porte sur la <strong>séance</strong>, pas sur la journée. Un paquet blunderDB est bâti sur une collection ou une recherche : c'est un corpus fini, introduit en quelques séances, dont le volume quotidien est déjà borné par sa taille. Un plafond par jour n'y mordrait jamais, ou bien créerait un retard sur un paquet qui tenait en une séance.</p>
<p><strong>Entraînement libre (cram) :</strong> Le bouton <em>Cram</em>, à côté de <em>Study</em>, lance une session d'entraînement libre : des positions aléatoires du paquet vous sont présentées sans tenir compte de l'échéancier FSRS. Ce mode <strong>ne modifie jamais le planning de révision espacée</strong> — idéal pour s'échauffer avant un tournoi ou réviser intensément un paquet thématique sans perturber son ordonnancement. Une pastille <em>Cram</em> remplace l'état de la carte et un bouton <em>Suivant</em> (touches <em>1</em> à <em>4</em>) fait défiler les positions. <em>Esc</em> revient à la liste sans enregistrer de session interrompue.</p>
<p><strong>Écarter une carte, sans la noter.</strong> Pendant une révision, un clic droit sur l'en-tête de la carte ouvre trois gestes qui la sortent de la séance sans rien dire au planificateur :</p>
<ul>
<li><strong>Suspendre</strong> — la carte garde son échéancier et ne remonte plus jamais tant qu'elle est suspendue. C'est la manière de mettre de côté une carte fausse, ou pas encore utile, sans perdre l'historique qui y est attaché.</li>
<li><strong>Enterrer</strong> — la carte disparaît jusqu'au lendemain. Contrairement à la suspension, cela ne dit rien de sa valeur : c'est pour celle que l'on vient de voir ailleurs, ou que l'on préfère ne pas croiser deux fois dans la soirée.</li>
<li><strong>Retirer</strong> — la carte quitte le paquet, après confirmation. La position, elle, reste dans la base : un paquet est une liste d'étude sur la bibliothèque, jamais une copie de celle-ci.</li>
</ul>
<p>Aucun de ces trois gestes n'enregistre de note : une carte écartée n'est pas une carte répondue, et elle ne compte pas dans le décompte de la séance.</p>
<p><strong>Journal des révisions.</strong> Dans les Paramètres d'un paquet, le bouton <em>Journal des révisions</em> montre ce que le planificateur a été <strong>dit</strong> — date, position, note, état, intervalle accordé — par opposition à ce qu'il prévoit. C'est le seul endroit où une note entrée par erreur se voit. Elle ne s'y corrige pas : l'échéancier reste hors de portée, et cette règle est précisément ce qui rend le journal utile — on ne peut pas réécrire le passé, mais on peut savoir ce qu'il a été.</p>
<p><strong>Arrêt/Reprise :</strong> Vous pouvez interrompre une session de révision à tout moment avec <em>Esc</em>. Le bouton change en <em>Resume</em> et affiche votre progression. Cliquez dessus pour reprendre là où vous vous êtes arrêté.</p>
<p><strong>Gestion des paquets :</strong> Utilisez les boutons d'action pour renommer, synchroniser, réinitialiser ou supprimer des paquets (confirmation demandée pour ces deux dernières actions). Les paramètres FSRS (rétention cible, intervalle maximum, aléa) peuvent être configurés par paquet dans les Paramètres (icône engrenage).</p>
<p><strong>Rétention : la cible et la mesure.</strong> La <em>rétention cible</em> est votre choix sur le compromis entre charge de travail et qualité du rappel : plus elle est haute, plus les intervalles raccourcissent et plus vous révisez. En regard, les Paramètres affichent la <strong>rétention mesurée</strong> sur vos propres révisions — une information, jamais un pilotage : blunderDB ne modifie pas votre cible pour poursuivre votre taux de réussite. Sous une vingtaine de révisions, la mesure n'est pas affichée : elle se lirait comme un fait alors qu'elle n'est que du bruit.</p>
<p>Changer la rétention <strong>n'est pas rétroactif</strong> : chaque carte adopte le nouveau rythme à sa prochaine révision, et les échéances déjà fixées ne bougent pas. L'effet est donc progressif, et invisible le jour même.</p>
<p>L'<em>intervalle maximum</em> borne l'espacement. Un paquet créé récemment démarre à un an : une position que l'algorithme reporterait de plusieurs années a quitté le paquet sans que vous l'ayez décidé, et votre propre jeu change plus vite que cela. Les paquets plus anciens conservent la valeur qu'ils avaient.</p>
<h3>Micro-entraînements</h3>
<p>Le panneau Anki fait réviser un <strong>jugement</strong> ; les micro-entraînements font travailler les trois <strong>calculs</strong> qui se font en partie, sous la pendule, et qu'aucune révision espacée ne muscle. La commande <code>train</code> en lance une session de cinq questions :</p>
<ul>
<li><code>train pips</code> — compter les pions du joueur au trait, sur la position affichée.</li>
<li><code>train epc</code> — estimer l'EPC de ce même joueur, sur une position de course que le moteur sait évaluer.</li>
<li><code>train tp</code> — retrouver le point de prise d'une course longue à un score tiré au hasard, celui de la table <code>tp2_live</code>.</li>
</ul>
<p>La question EST la position affichée : le plateau est celui de l'application, et la barre au-dessus ne porte que la question, la saisie et la correction. La réponse se tape et se valide au clavier (<em>Entrée</em> vérifie, puis passe à la suivante ; <em>Échap</em> quitte la session).</p>
<p>La tolérance dépend de l'exercice, et elle est dite plutôt que devinée : le comptage de pions n'en a <strong>aucune</strong> — une addition juste à un pion près est une addition fausse — l'EPC accepte un demi-pion, le point de prise deux points de pourcentage. À la fin, la session affiche le nombre de bonnes réponses et le temps <strong>médian</strong> par question.</p>
<p>Seul ce résumé est conservé, dans les métadonnées de la base : la session ne garde pas la trace question par question, et rien n'est écrit tant qu'elle n'est pas terminée. Quitter en cours de route n'enregistre donc rien.</p>
<h4>Quiz : le PR d'entraînement</h4>
<p><code>train quiz</code> pose un quatrième exercice, d'une autre nature. Le panneau Anki fait mémoriser ; le quiz <strong>teste</strong>. Cinq positions déjà analysées sont tirées de la liste parcourue, et il faut décider :</p>
<ul>
<li>sur une décision de pions, écrire le coup au clavier, en notation (<code>13/7 8/7</code>) ;</li>
<li>sur une décision de videau, cliquer <em>Pas de double</em>, <em>Double, prend</em> ou <em>Double, passe</em>.</li>
</ul>
<p>Le panneau Analyse est masqué tant que la question n'a pas reçu de réponse : il porte la réponse, et une question dont la réponse est affichée à côté n'est pas une question.</p>
<p>La correction distingue trois issues, et les confondre mentirait. Un <strong>coup illégal</strong> n'est pas un coup mal choisi — c'est une faute de règle. Un <strong>coup légal que le moteur n'a pas classé</strong> n'est pas une faute du tout : il n'a simplement pas de prix, et il ne coûte donc rien à la session. Un coup classé coûte ce que l'analyse dit qu'il coûte, en millipoints.</p>
<p>À la fin, la session affiche un <strong>PR de quiz</strong> calculé par la formule que les statistiques appliquent au jeu réel — 500 × erreur moyenne en équité normalisée. C'est ce qui rend les deux nombres comparables : un PR de quiz de 6 et un PR de match de 6 mesurent la même chose sur la même échelle.</p>
<h3>Panneau Métadonnées</h3>
<p>Le panneau <strong>Métadonnées</strong> affiche les informations générales de la base de données courante : nom, description, nombre de positions, nombre de matchs et de parties, version du schéma. Accessible via la commande <code>meta</code>.</p>
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
<p>Les champs <em>Utilisateur</em>, <em>Description</em> et <em>Date</em> décrivent le <strong>fichier produit</strong> ; ils sont préremplis depuis la base source. La case <em>Mes filtres enregistrés</em> est à part des autres : elle n'exporte pas du contenu mais vos propres recherches enregistrées, sans utilité dans la base de quelqu'un d'autre.</p>
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
<li>« ✓ signature vérifiée — marquée par vous » : le fichier porte votre marque, intacte ;</li>
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
<td>log</td>
<td>Ouvre le journal d'activité : les deux cents dernières lignes du fichier de journal, avec de quoi les copier pour les joindre à un rapport, ou ouvrir le dossier qui les contient.</td>
</tr>
<tr>
<td>ask</td>
<td>Traduit une phrase en toutes lettres — français ou anglais — en jetons de recherche : <code>ask mes blunders de videau au score</code>. Les jetons sont écrits dans la barre de commande, pas lancés : on les relit, puis Entrée. Ce qui n'a pas été compris est dit, jamais deviné.</td>
</tr>
<tr>
<td>like</td>
<td>Remplace la liste parcourue par les positions les plus proches de la position courante — ou de celle dont l'indice est donné (<code>like 42</code>). La proximité est une distance de transport en pions-pas : ce n'est pas un filtre, elle classe toute la base plutôt que de la restreindre, et ne se combine donc pas avec les jetons de recherche.</td>
</tr>
<tr>
<td>train</td>
<td>Lance une session de micro-entraînement. Prend un argument : <code>train pips</code> (compte de pions), <code>train epc</code>, <code>train tp</code> (point de prise au score), <code>train quiz</code> (le coup ou l'action de videau, notés contre l'analyse enregistrée). Cinq questions, chronométrées, corrigées sur-le-champ.</td>
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
<td>La position a été rencontrée plus de x fois dans la base — le nombre de coups qui y aboutissent, tous matchs confondus. Formes <code>n&gt;3</code>, <code>n&lt;2</code>, <code>n3,10</code> et <code>n4</code> (exactement quatre).</td>
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
