# ADR-0087 — Le Départ d'un Duel se règle avant de jouer : score, dés, videau, une partie, cadence

Statut : acceptée. Révise ADR-0072 règle 12 (double offert au Départ, CLI sans Cadence) et
ADR-0073 (cadences nommées). Voir aussi : ADR-0007, ADR-0085.

## Contexte

Le Duel partait de la position initiale, de la position au plateau, ou de la position
initiale à un score saisi à part. Un joueur qui s'entraîne veut régler ce Départ exactement :
le score de chacun (Crawford compris), le jet posé au plateau ou un jet neuf, la décision de
videau qu'il veut travailler, une seule partie, et une cadence lisible en deux nombres. Un
Départ montrant un double offert était refusé (`start_pending_double`).

## Décision

1. **La création porte les choix, la Position du brouillon en est le résultat.**
   `duel.Settings` gagne `Away` (le score en Away scores, sentinelles Crawford comprises : il
   remplace celui du Départ, et sans Départ donne la position initiale à ce score ; le début
   du match reste « pas de Départ », pour que le Match s'exporte en `.mat`), `Reroll` (le jet
   du Départ est retiré, l'Arbitre le tire), `AfterCube` et `SingleGame`. `resolveStart`
   applique ces choix une fois ; le brouillon ne garde que le Départ obtenu et les Actions
   jouées, sans drapeau à rejouer : la reprise ne relit aucun choix.
2. **Avant la décision de videau est le défaut.** C'est elle qu'on vient travailler. Un
   double offert au plateau (videau au centre retourné, ou tenu par l'adversaire du joueur au
   trait, à une décision de videau) n'est plus refusé : *avant*, le Départ est la position
   d'avant le double, que l'Arbitre rejoue comme première Action (sans durée), et
   l'adversaire répond ; *après*, le double est pris — le videau va à l'adversaire — et le
   joueur au trait lance. *Après* sur un videau disponible : l'Arbitre lance aussitôt, comme
   le ferait « Lancer ».
3. **Une seule partie** (`single_game`, document du brouillon en version 3) : le Duel finit
   avec sa première partie, le Match s'écrit avec elle seule (ou rien, si le Duel est
   jeté) — un match inachevé comme une session en argent gardée, au score où la partie s'est
   jouée. Rien de plus n'est enregistré : l'origine du Match (le Départ en XGID) et son
   unique partie disent ce qu'il fut. Une session en argent entre deux Bots devient permise
   en une seule partie, puisqu'elle finit. Une version antérieure refuse un brouillon v3
   plutôt que de jouer au-delà de la partie.
4. **Une cadence se dit en deux nombres** : minutes de réserve par point du match et délai
   par coup en secondes ; le total d'un match à 0-0 vaut minutes par point × longueur.
   Préréglages : *standard* (2 min + 12 s, les règles USBGF/WBGF) et *speed* (0,4 min,
   soit 2 min pour 5 points, + 10 s). Les réserves fixes `rapid-*` ne sont plus offertes ;
   l'Arbitre les accepte toujours. Une cadence créée par le joueur est un réglage de
   l'utilisateur, gardé avec le formulaire du Duel (`Config.SaveDuelForm`), jamais dans la
   base : ouvrir une base n'écrit rien (ADR-0007). La CLI prend `--minutes-per-point` et
   `--delay` ; la durée de ses décisions reste inconnue d'un appel à l'autre.
5. **Le pipcount masqué pour un Duel est un masque, pas un réglage** :
   `duelPipcountHiddenStore` l'emporte sur la préférence (`p`, bouton) le temps du Duel et
   tombe à sa sortie ; la préférence n'est pas touchée.
6. **Partant du plateau, le plateau s'édite comme en Eval** : le lanceur passe le plateau en
   mode EVAL (`enterEvalOnDisplayed`), le même plateau brouillon et les mêmes gestes ; le
   score du plateau et les champs du formulaire n'en font qu'un, dans les deux sens.

## Conséquences

- `RefusedStartPendingDouble` ne sort plus de `Create` : `checkStart` ne voit qu'un Départ
  déjà résolu.
- Le formulaire ne retient ni `reroll` ni `afterCube` : ils dépendent de la position du
  moment et repartent de leur défaut.
- Une session en argent se joue sans cadence dans le formulaire : une réserve par point
  demande une longueur.
