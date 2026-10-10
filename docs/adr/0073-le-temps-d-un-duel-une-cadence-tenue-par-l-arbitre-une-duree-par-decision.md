# ADR-0073 — Le temps d'un Duel : une Cadence tenue par l'Arbitre, une durée par décision

Statut : acceptée. Cadences nommées révisées par ADR-0089 (minutes par point + délai, standard et speed).
Voir aussi : ADR-0072 (le Duel, ses Côtés, l'Arbitre), ADR-0046 (les seuils d'erreur, que la
revue croise avec la durée).

## Contexte

Un Duel (ADR-0072) se joue sous l'arbitrage de blunderDB. Deux temps s'y mesurent, qu'il ne
faut pas confondre : celui qu'une horloge retire à un joueur, qui est une règle du jeu et peut
ne pas exister, et celui qu'un joueur a mis à décider, qui est une donnée d'analyse et existe
toujours. Aucun import ne fournit le second ; ni gnubg ni gammonNet n'offrent de modèle du
premier.

## Décision

1. **Une Cadence est une réserve et un délai, tenus par l'Arbitre.** Une réserve par Côté
   pour le match, un délai gratuit à chaque tour ; pas d'incrément. L'Arbitre horodate à la
   réception de l'Action, jamais chez le client. Sans Cadence est le défaut. Un Duel
   suspendu arrête les horloges ; un Duel ouvert les fait courir jusqu'à ce qu'on le
   suspende, quel que soit le processus qui le tient, arrêt ou plantage de ce processus
   compris : ce temps est connu et compté. Un client qui veut arrêter les horloges suspend
   le Duel. La durée « inconnue » d'une décision reprise ne vaut plus que pour un brouillon
   que sa ligne dit en suspens alors que son horloge courait : une ligne antérieure au
   schéma 2.36.0, où l'état ouvert vivait en mémoire, ou un Duel ouvert porté par une
   migration, qui arrive en suspens (ADR-0072 règle 10). Le temps écoulé est un fait ; sa conséquence est un réglage
   du Duel à deux valeurs — *continuer* (défaut du bureau : le dépassement est noté, le match
   d'entraînement reste entier) ou *perdre le match*. En points, perdre au temps vaut
   l'abandon du match par le joueur dont la réserve s'est épuisée (ADR-0074 règle 2) : le
   Match s'écrit gagné par l'autre, avec son score final, puisqu'un Duel n'écrit qu'un match
   entier (ADR-0072 règle 10) ; son origine porte le fait, et la fin du Duel ne nomme aucun
   abandon. En argent, la session s'arrête où elle en est, sans partie ajoutée.
2. **La durée se mesure par décision et devient une propriété du coup.** Elle court dès que
   le Côté a le trait, délai compris, avec ou sans Cadence : du trait au lancer pour la
   décision de videau, du lancer au coup validé pour la décision de pions, et une durée
   propre pour un double, une réponse, un abandon. Quand le videau n'est pas disponible, le
   lancer est automatique et il n'y a pas de durée avant lui. Les deux Côtés sont mesurés.
   Écrite sur l'Action du brouillon, elle est reportée sur le coup du Match : c'est un
   changement de schéma, et la valeur absente — tout Match venu d'ailleurs — se lit
   « inconnue », jamais zéro.
3. **La durée se cherche comme l'erreur.** La recherche reçoit un filtre sur la durée de
   décision, dans la même grammaire et dans les trois modes que le filtre sur l'erreur
   jouée, pour qu'ils se combinent — longtemps réfléchi et faux quand même, vite joué et
   gaffe. La revue d'un Match montre la durée à côté de chaque décision, la trie, la résume
   par joueur et la trace au fil du match ; position par position, elle se lit discrètement
   près de la décision jouée. Partout, une durée inconnue s'affiche vide.

## Conséquences

- La durée ne dépend pas de l'horloge : une Cadence qui change, ou qui manque, ne change pas
  ce que la revue montre.
- Le schéma gagne la durée sur le coup ; l'import des temps qu'un fichier source porterait
  (le format d'eXtreme Gammon lit un réglage d'horloge) remplirait le même champ, et reste à
  établir.
- Rejeté : une durée par tour (elle confond « réfléchi longtemps sur le coup » et « lancé
  sans penser à doubler ») ; l'incrément à la Fischer (le backgammon ne s'en sert pas) ; un
  Match perdu au temps laissé à son score, sans vainqueur (il ne serait pas un match entier,
  et le panneau Matchs n'en montre pas d'autre) ; une durée comptée après le délai (elle dépendrait
  de la Cadence).
