# ADR-0079 — La vidéo d'un match horodate ses décisions, et la durée s'en déduit

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
   l'Arbitre. Un Repère manquant laisse inconnues les durées qui en dépendent. Une durée
   mesurée par l'Arbitre prime toujours sur une durée déduite. À l'enregistrement, la durée
   déduite remplit le champ du coup que l'ADR-0073 a défini : même colonne, même filtre, même
   affichage. Un Repère antérieur à celui qui le précède est une Incohérence, marquée et
   jamais corrigée (ADR-0044) ; insérer, supprimer ou corriger une Action ne déplace aucun
   Repère, et le Replay recalcule.
3. **Les Repères se posent de deux façons, et le transcript ramène à la vidéo.** Sur une
   Action *nouvelle*, la première touche de dé pose l'instant du jet et la validation pose
   l'instant de l'action, à l'instant courant de la vidéo ; une correction en place ne touche
   pas aux Repères. Un geste explicite pose l'instant courant sur l'action du curseur, jet ou
   action. Placer le curseur sur une cellule amène la vidéo à son Repère. Repères et source
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
   Wails.** Le processus du bureau sert le fichier attaché, et lui seul, sous une URL à jeton
   sur `127.0.0.1`, avec les requêtes partielles qu'exige la lecture. Une source YouTube se lit
   par le lecteur intégré de YouTube, dans une page hébergée par ce même serveur pour qu'elle
   ait un référent ; une vidéo qui interdit l'intégration se regarde ailleurs et ses Repères
   se posent par le geste explicite. Depuis la revue d'un Match, une décision ouvre sa vidéo :
   dans le volet pour un fichier, dans le navigateur par le lien horodaté pour YouTube. Le
   démon `serve` ne lit rien.
6. **L'application ne décode rien elle-même.** Les codecs sont ceux du webview : natifs sur
   Windows et macOS, ceux de GStreamer sur Linux. Un format que le webview ne lit pas se
   diagnostique dans le volet, par le format du fichier et les paquets à installer, jamais par
   un cadre noir. Le Flatpak déclare l'extension ffmpeg du runtime, le paquet AUR nomme ses
   dépendances optionnelles, les notes d'installation listent les greffons.

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
  un Repère à rebours.
- Garde : les tests de dérivation dans `transcript`, la suite de contrat des backends sur les
  nouvelles colonnes, `TestMigrationSteps_ContinuousChain`, `TestDemoDatabaseIsCurrent`.
