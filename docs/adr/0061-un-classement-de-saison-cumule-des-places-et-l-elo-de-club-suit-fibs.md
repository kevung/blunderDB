# ADR-0061 — Un classement de saison cumule des places, et l'Elo de club suit la formule FIBS

Statut : acceptée.
Voir aussi : ADR-0047, ADR-0056.

## Contexte

`Ranking` ne couvre qu'un tournoi. Un club classe une saison : plusieurs tournois d'une
Rencontre ou d'une période, des points par épreuve, parfois un Elo de club. Aucun format
fédéral n'est identifié : seul un export générique (CSV, JSON) est livré.

## Décision

1. **Dérivé, jamais stocké.** `service.SeasonRanking` rejoue les journaux à chaque appel, comme
   le classement d'un tournoi ; pas de table, pas de migration. Un tournoi non clos est listé
   et ne rapporte rien : ses places ne sont pas encore des faits.
2. **Des places, pas des scores.** Une place vaut `Points[place-1]` (barème fourni, défaut
   25-18-15-12-10-8-6-4-2-1), plus `Participation` par tournoi clos. Des ex æquo à la place `r`
   se partagent la moyenne des places `r..r+k-1` : une égalité ne crée ni ne détruit de points.
   Le classement est par total, puis meilleure place, puis nom — un ordre total ; égaux en
   total et en meilleure place partagent le rang.
3. **Elo de club : FIBS.** Départ 1500 ; pour un match en `n` points gagné par `W` contre `L`,
   `P = 1 / (1 + 10^(-(W-L)·√n / 2000))`, `Δ = 4·√n·(1-P)`, ajouté au vainqueur, retiré au
   perdant (somme nulle). Les matchs sont rejoués par date de tournoi puis heure de fin ; un
   forfait ne compte pas. Pas de facteur d'expérience : une saison de club est trop courte pour
   qu'il serve. Toute autre mesure de niveau du dépôt (Elo d'un corpus) suit cette formule.
4. **Une personne est un nom.** L'identifiant d'un joueur est celui de son tournoi ; d'un
   tournoi à l'autre, c'est le nom (casse et espaces repliés) qui la reconnaît, comme pour les
   joueurs occupés d'une Rencontre.

## Conséquences

Exposé partout : `blunderdb tournament ranking --season`, `/v1/rencontres.ranking`,
`Database.SeasonRanking`. Un format fédéral s'ajoutera comme un rendu de plus de `SeasonView`,
une fois son cahier des charges connu.
