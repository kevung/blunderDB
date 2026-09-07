# P19 — La classe d'une voisine, jugée par un joueur

**Statut** : corpus produit, jugement à porter. **Date** : 2026-09-07.
**Rattaché à** : [ADR-0043](../adr/0043-a-neighbour-is-the-same-problem-nearby-and-like-is-a-ranking-token-of-the-search-grammar.md), issues #293 et #331.
**Complète** : [P7](P7-similarite-knn-go.md), qui a choisi la métrique et l'index.

## Ce que P7 a réglé, et ce qu'il n'a pas réglé

P7 a tranché deux questions et bien : la métrique — le transport optimal en
une dimension sur le vecteur des points, plutôt que l'espace latent d'un
réseau qu'aucune littérature ne valide pour le backgammon — et l'index — un
balayage exhaustif, parce que sous cent mille positions le scan linéaire donne
un rappel parfait et qu'un index approximatif coûterait sa cohérence à chaque
écriture.

Il n'a pas réglé, parce que ce n'était pas sa question, **sur quel ensemble** la
métrique s'applique. La fiche J.3 avait prévu la vérification correspondante :
« cinquante positions, deux métriques, un joueur juge si proche selon la
métrique = proche selon lui ». Elle n'a pas eu lieu, et le code livré a classé
toute la bibliothèque.

## Ce que cela a coûté, mesuré

Sur la base de démo — 757 positions, 3 matchs — les dix voisines de n'importe
quelle position étaient :

| rang | ce que c'était | distance |
|---|---|---|
| 1 | le coup de pions jumeau de la décision de videau, même damier | 0 |
| 2-9 | les plis d'avant et d'après, dans le même match | 8 à 16 |
| 10 | la première position d'une autre partie | 35 et au-delà |

Deux plis font un lancer, soit huit à seize pions-pas, et aucune autre partie
ne descend en dessous. La commande répondait donc « voici la partie que tu
regardes » — sur une bibliothèque de matchs importés, c'est-à-dire celle de
tout le monde.

La métrique n'était pas en cause. L'ADR-0043 a défini une **classe
d'équivalence** : même type de décision, même régime pour une décision de
videau, un autre match que la cible. Après quoi les mêmes cibles rendent des
décisions du même type venues d'autres matchs, à 27-74 pions-pas.

## Ce qui reste à juger, et le seuil

Les règles de classe sont un raisonnement, pas une mesure. Ce que le corpus
demande à un joueur est exactement la question de J.3, portée sur la classe
plutôt que sur la métrique : **les premières voisines sont-elles le même
problème ?**

Le corpus se produit par :

```
go run ./cmd/likecorpus -db bibliotheque.db -targets 30 -k 5 -seed <graine> > corpus.md
```

Une trentaine de cibles tirées d'une vraie bibliothèque, leurs cinq premières
voisines, et une colonne à remplir blunderDB ouvert — taper l'indice d'une
position dans la barre de commande amène son plateau à l'écran, et deux
plateaux ne se comparent pas de tête.

**Le seuil est écrit avant la mesure** : si moins des deux tiers des
*premières* voisines sont jugées « même problème », les règles de classe sont
à revoir avant que la documentation du jeton ne soit tenue pour acquise. Noter
la graine dans le tableau ci-dessous : c'est ce qui rend le tirage rejouable.

## Résultats

| Date | Bibliothèque | Cibles | Graine | Premières voisines « oui » | Verdict |
|---|---|---|---|---|---|
| | | | | | |

## Ce que ce corpus ne mesure pas

Le choix de la métrique, qui reste celui de P7 : personne ici ne compare le
transport optimal à une distance L1 ni à un espace latent. Et la valeur du
plafond de distance, qui dépend de la phase — dix pions-pas ne sont rien en
course et une autre position à l'ouverture — et qui n'a donc pas de valeur par
défaut dans le produit, précisément parce qu'aucune mesure ne la justifierait.
