# ADR-0082 — La vidéo d'un match horodate ses décisions, et la durée s'en déduit

Statut : acceptée.
Voir aussi : ADR-0073 (la durée par décision, son filtre et son affichage), ADR-0044 et
ADR-0045 (la Transcription et son brouillon), ADR-0057 (l'API de transcription), ADR-0067 (les
métadonnées qui voyagent avec un match), ADR-0039 (le front web embarqué).

## Contexte

La durée de décision (ADR-0073) n'a qu'une source : l'Arbitre d'un Duel. Un match transcrit
depuis une vidéo — la source la plus courante d'un match joué ailleurs, souvent sur YouTube —
n'en porte aucune, alors que la vidéo contient l'information. Ce qu'un transcripteur peut
relever sur une vidéo n'est pas une durée mais un *instant* : le moment où les dés tombent,
le moment où le coup est fini. L'heure de frappe ne vaut rien : on met en pause, on revient
en arrière, on accélère. Et le lien va dans les deux sens : depuis une décision du match, on
veut retourner dans la vidéo, ce qui intéresse autant la revue au bureau qu'un client externe
comme gammonGo.

Lire une vidéo dans le webview impose trois contraintes : le pipeline média de WebKitGTK ne
fait pas de requêtes partielles à travers un schéma d'URL personnalisé, et l'écrivain Linux de
Wails pousse le corps entier dans un tube ; les codecs sont ceux du webview, que le binaire ne
peut pas embarquer ; le lecteur intégré de YouTube refuse une page sans référent HTTP.

## Décision

1. **Le Repère est un instant du média, en millisecondes, posé sur l'Action et porté par le
   coup.** Un coup de pions en a deux : l'instant du jet (les dés tombent) et l'instant de
   l'action (le coup est fini). Un double, une prise, une passe, un abandon n'en ont qu'un,
   celui de l'action. Une danse et un coup non consigné ont les deux comme un coup de pions.
   Posé sur l'Action du brouillon, le Repère est reporté sur le coup du Match à
   l'enregistrement, et un Match rouvert en brouillon le retrouve. Absent, il se lit
   « inconnu », jamais zéro.
2. **La durée se déduit des Repères par le Replay, jamais tapée ni stockée sur l'Action d'une
   Transcription.** La décision de videau d'un coup de pions va de l'instant de l'action
   précédente du document, quel que soit son camp, à l'instant du jet ; la décision de pions
   va de l'instant du jet à l'instant de l'action ; un double, une réponse, un abandon vont de
   l'instant de l'action précédente au leur. Le premier jet d'une partie n'a pas de décision
   de videau ; une danse et un coup non consigné n'ont pas de décision de pions, comme sous
   l'Arbitre. Un Repère manquant laisse inconnues les durées qui en dépendent, à une
   exception : une décision de pions sans instant d'action s'*estime* du jet au premier
   instant de l'Action suivante — son jet, ou l'instant d'un geste de videau —, un majorant
   qui compte le ramassage et le lancer des dés, marqué comme estimé et affiché à part.
   Aucune décision de videau ne s'estime : la fin du coup précédent ne se distingue pas du
   début de la réflexion au videau, et seul l'instant de ce coup, posé par `v` ou par une
   validation explicite, la mesure. Un Repère à rebours ne sert à aucune estimation ; une
   mesure prime toujours. Le marqueur ne survit pas à l'enregistrement : le coup du Match ne
   porte que la durée. Une durée
   mesurée par l'Arbitre prime toujours sur une durée déduite. À l'enregistrement, la durée
   déduite remplit le champ du coup que l'ADR-0073 a défini : même colonne, même filtre, même
   affichage. Un Repère antérieur à celui qui le précède est une Incohérence, marquée et
   jamais corrigée (ADR-0044) ; insérer, supprimer ou corriger une Action ne déplace aucun
   Repère, et le Replay recalcule.
3. **Les Repères se posent de deux façons, et le transcript ramène à la vidéo.** Sur une
   Action *nouvelle*, la première touche de dé pose l'instant du jet et la validation pose
   l'instant de l'action, à l'instant courant de la vidéo ; une correction en place ne touche
   pas aux Repères. Une validation implicite — par le chiffre du jet suivant ou par un geste de
   videau — ne pose aucun instant : le coup reste sans instant d'action plutôt que d'en recevoir
   un faux. Un geste explicite pose l'instant courant sur l'action du curseur, jet ou
   action. Placer le curseur sur une cellule amène la vidéo à son Repère. En sens inverse,
   pendant la lecture, le curseur suit la vidéo dans la partie horodatée : une Action couvre
   la vidéo de son jet — à défaut de l'instant de l'Action précédente — au début de la
   suivante, et au-delà du dernier Repère le curseur revient une fois en fin de document. Ce
   suivi n'écrit rien et ne s'empile pas sur l'annulation, ne ramène jamais la vidéo, se tait
   pendant une saisie ou une pause, et cède à un placement à la main tant que la lecture
   n'a pas quitté l'Action choisie : le geste explicite horodate celle que l'on a désignée.
   Repères et source
   sont des champs optionnels des gestes d'écriture et de l'état exposé (ADR-0057) : un client
   externe pose les siens par l'API, et la logique vit dans `transcript`, le lecteur n'étant
   qu'un fournisseur d'instants.
4. **La source vidéo est une URL http(s) ou un chemin local, sur le brouillon puis sur le
   Match.** Un chemin local est une indication : les Repères gardent leur sens sans le
   fichier, et un fichier introuvable se relocalise. À l'export (ADR-0067), seule une URL
   http(s) voyage ; un chemin local reste chez son auteur, il révèle une arborescence et ne
   sert à personne d'autre. Les Repères voyagent toujours. Le `.mat` ne porte ni l'un ni
   l'autre.
5. **Le bureau lit la vidéo par un serveur HTTP de bouclage, jamais par le serveur d'assets de
   Wails.** Le processus du bureau sert les fichiers attachés aux volets ouverts, et eux seuls, sous une URL à jeton
   sur `127.0.0.1`, avec les requêtes partielles qu'exige la lecture. Une source YouTube se lit
   par le lecteur intégré de YouTube, dans une page hébergée par ce même serveur pour qu'elle
   ait un référent ; une vidéo qui interdit l'intégration se regarde ailleurs et ses Repères
   se posent par le geste explicite. Depuis la revue d'un Match, une décision ouvre sa vidéo :
   dans le lecteur de l'application pour un fichier (règle 7), dans le navigateur par le lien
   horodaté pour YouTube. Le démon `serve` ne lit rien.
6. **L'application ne décode rien elle-même.** Les codecs sont ceux du webview : natifs sur
   Windows et macOS, ceux de GStreamer sur Linux. Un format que le webview ne lit pas se
   diagnostique dans le volet, par le format du fichier et les paquets à installer, jamais par
   un cadre noir. Le Flatpak déclare l'extension ffmpeg du runtime, le paquet AUR nomme ses
   dépendances optionnelles, les notes d'installation listent les greffons.
7. **La vidéo s'affiche à côté du plateau, ou dans le panneau qui l'a ouverte ; un seul
   lecteur, qui appartient au panneau.** Côte à côte est le défaut, pour qu'une vidéo assez
   grande se lise ; le choix et la largeur sont des préférences du poste. Le panneau garde son
   composant de lecture et le pilote comme avant (instants, sauts, touches) ; la zone du
   plateau ne fait que lui prêter une place dans le DOM. Un fichier change donc de place sans
   se recharger ; un cadre YouTube, qu'un navigateur recharge quand il bouge, reprend à son
   instant. Où qu'elle soit, la vidéo compte comme le panneau pour le clavier : un clic dans
   le lecteur ou sur ses séparateurs rend le focus au panneau, parce qu'un `<video>` focalisé
   lit lui-même les flèches et qu'un cadre focalisé avale toutes les touches. La vitesse de
   lecture se règle par pas de 0,25 de 0,25× à 4×, une seule liste filtrée par ce que la
   source accepte ; elle ne survit pas à un changement de source.
   Le côté (gauche par défaut, ou droite) est une préférence du poste comme la largeur, qui
   reste celle de la vidéo quel que soit le côté. On le change par le bouton ⇄ ou en
   glissant une poignée en haut de la vidéo (événements pointeur, pas le glisser HTML5, peu
   fiable sous WebKitGTK et mêlé au dépôt de fichiers) ; un calque posé pendant le glisser
   empêche le cadre de capter la souris, Échap ou un lâcher hors de la zone annule. Le plateau
   ne se déplace pas : le glisser y appartient aux pions. Poignée et bouton ne prennent pas le
   focus.

## Conséquences

- Le schéma gagne la source sur le match et les deux Repères sur le coup, sur les deux
  backends, avec la base de démonstration. Le format du brouillon ne change pas de version :
  les champs sont optionnels.
- Le filtre de recherche sur la durée, la revue d'un Match et ses statistiques de temps
  (ADR-0073) s'appliquent sans changement à un match transcrit depuis une vidéo.
- Une Transcription sans vidéo attachée ne change pas : aucun Repère, aucune durée, et
  aucune touche de plus.
- Rejeté : stocker la durée sur l'Action d'une Transcription (une insertion la fausserait) ;
  l'heure de frappe comme mesure (fragile aux pauses et aux retours) ; la précision à l'image
  (la durée se lit en secondes) ; un lecteur externe piloté (une dépendance et une plomberie
  par plateforme) ; un décodeur WebAssembly embarqué (trente mégaoctets et un décodage
  logiciel pour le seul cas d'une distribution sans greffons) ; le serveur d'assets de Wails
  comme source média ; un chemin local qui voyage à l'export ; une Transcription qui refuse
  un Repère à rebours ; deux lecteurs pour une vidéo, un dans le panneau et un à côté du
  plateau (deux horloges, et un basculement qui perd l'instant) ; un calque qui intercepte les
  clics sur le lecteur (il cacherait les commandes de YouTube et du webview).
- Remplacer un match par un fichier corrigé garde sa source vidéo et perd ses Repères : le
  fichier n'en porte pas.
- Garde : les tests de dérivation dans `transcript`, la suite de contrat des backends sur les
  nouvelles colonnes, `TestMigrationSteps_ContinuousChain`, `TestDemoDatabaseIsCurrent`.
