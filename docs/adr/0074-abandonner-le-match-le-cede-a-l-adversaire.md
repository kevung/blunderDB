# ADR-0074 — Abandonner le match le cède à l'adversaire

Statut : acceptée.
Voir aussi : ADR-0072 (le Duel ; règle 10, l'arrêt et le suspens), ADR-0045 §6 (l'abandon d'une
partie est un fait de la Partie, pas un coup).

## Contexte

Un Duel offrait trois manières de quitter la table : le suspendre (horloges arrêtées, reprise au
même point), l'« arrêter et garder » (un Match inachevé, sans vainqueur) et l'« arrêter et jeter »
(rien n'est écrit). Aucune ne permettait ce que fait un joueur qui quitte un tournoi : céder le
match. L'abandon existant cède une *partie*, à 1, 2 ou 3 fois le videau, et le match continue.
« Arrêter et garder » prêtait à confusion avec un abandon, alors que la règle 10 d'ADR-0072 dit
précisément le contraire : arrêter n'est jamais céder.

## Décision

1. **Abandonner le match est une Action du jeu**, `forfeit` (`transcript.KindForfeit`), jouée par
   un Côté à tout moment — la décision attendue ou non, entre deux parties aussi : elle ne répond
   à aucune Décision. Elle termine la partie en cours (une nouvelle quand aucune ne court, son
   lancer d'ouverture laissé de côté), gagnée par l'autre Côté, et rien ne la suit.
2. **Ce qu'elle vaut.** À un score de match, les points qui portent l'adversaire à la longueur :
   le match est *gagné*, avec un score final cohérent, et non arrêté avant la fin. En money, il
   n'y a pas de match à donner : l'abandon clôt la session, et la partie en cours est perdue en
   simple à la valeur du videau, avec ou sans règle Jacoby. C'est un choix de l'utilisateur :
   abandonner est un geste de concession, qui vaut ce que vaut une partie cédée au niveau le plus
   bas, et non la pire issue possible. Un abandon peut donc coûter moins que jouer la partie
   jusqu'au bout.
3. **Le Duel se termine comme un match gagné** : son Match s'écrit (ou le brouillon est jeté s'il
   a été créé sans enregistrement), le germe est révélé, et l'origine ne le marque pas « arrêté
   avant la fin ». La fin du Duel nomme le Côté qui a abandonné (`Ending.Forfeited`) ; le Match,
   lui, ne porte que des parties et leurs vainqueurs, comme tout Match : aucun champ de schéma
   n'est ajouté.
4. **L'interface graphique offre trois gestes** : *Abandonner le match* (confirmé), *Mettre en
   pause* (le suspens d'ADR-0072, horloges arrêtées) et *Annuler le match* (confirmé ; rien
   n'est écrit). « Arrêter et garder » n'y figure plus : la pause garde le match.
5. **L'API et la CLI gardent l'arrêt gardé.** `/v1/duels.stop {keep}` et `blunderdb duel stop`
   sont un contrat publié (clients générés, gammonGo) ; ils restent, et la règle 10 d'ADR-0072
   vaut toujours pour eux. L'abandon s'y ajoute : `/v1/duels.forfeit {id, side}`,
   `blunderdb duel forfeit --side`, `Database.ForfeitDuel` — le Côté y est requis, puisque
   l'abandon ne répond à aucune Décision qui le nommerait.

## Conséquences

- Un match abandonné compte comme un match perdu par celui qui l'a cédé ; ses décisions comptent
  comme celles de tout Match (ADR-0072 règle 1).
- La dernière partie d'un Match abandonné n'a pas de coup final : comme une partie cédée, elle se
  lit à son vainqueur et à ses points.
- Rejeté : l'écrire dans l'origine du Match (`abandoned_by`) en laissant le score tel quel — un
  changement de schéma pour un Match dont le score ne dirait pas qui l'a gagné ; un abandon de
  partie au niveau qui atteint la longueur — trois fois le videau n'y suffit pas toujours . En
  money, le barème du backgammon au videau (ou d'une simple sous Jacoby), retenu d'abord pour que
  l'abandon ne coûte jamais moins que jouer, a été remplacé par la simple au videau.
