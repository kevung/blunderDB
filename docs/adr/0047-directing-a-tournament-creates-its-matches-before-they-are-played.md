# ADR-0047 — Diriger un tournoi, c'est créer ses Matchs avant qu'ils soient joués

Statut : acceptée.
Voir aussi : ADR-0037, ADR-0039, ADR-0044, ADR-0045, ADR-0005.

## Contexte

Le `Tournament` est une étiquette posée après l'import. Diriger un tournoi est l'inverse :
inscrire, apparier, lancer, saisir, classer, avant qu'un seul fichier existe. Le moteur
Nicomaque (`github.com/PileOfCells/backgammon-tournoi`, bibliothèque Go à journal d'événements
rejoué) fait ce travail mais ne persiste rien et n'a pas d'interface. ADR-0037 refuse un
second produit dans le binaire ; il faut dire pourquoi celui-ci n'en est pas un.

## Décision

Un tournoi dirigé est une **source de Matchs**. Chaque match lancé est un **Slot** (deux
Participants, une longueur, une table, un résultat) que remplit une Transcription ou un import.

1. **Une seule entité.** Un `Tournament` porte optionnellement une **Direction** : ce que le
   directeur a décidé, dans l'ordre, jamais modifié, seulement prolongé.
2. **Le moteur propose, le directeur décide, la Direction enregistre.** Rien n'est refusé sauf
   l'impossible ; l'incohérent est accepté, tracé, signalé sans bloquer (posture d'ADR-0044).
3. **L'état n'est jamais stocké** : classement, arbres, appariements et proposition suivante
   sont rejoués depuis la Direction à chaque ouverture.

Hors périmètre : l'écran des joueurs. blunderDB écrit une page HTML autonome à projeter ou
imprimer ; aucune route de consultation, rien dans le front web (ADR-0039 règle 1).

Les renvois « ADR-0047 §N » du code désignent les sections de la spécification
`tasks/nicomaque/fonctionnel.md`.

## Conséquences

- La Direction est une table **append-only**, une ligne par événement, écrite dans la
  transaction du geste ; logique sur `Database` et le contrat `Storage`, exposée au frontend et
  à une sous-commande CLI non interactive. Le démon ne reçoit rien.
- La zone principale affiche autre chose que le plateau seulement quand l'onglet Tournoi est
  actif et une Direction ouverte.
- Vocabulaire dans `CONTEXT.md` : Direction, Participant, Directory, Slot (*journal* et
  *replay* sont déjà pris).
- Nicomaque émet des codes, jamais des phrases ; son journal est versionné. Le moteur reste à
  son auteur, consommé par tag, jamais bifurqué ; ce qu'on lui demande devient des issues de son
  dépôt.
- Le crédit « Nicomaque, moteur de tournoi créé par Nicolas Harmand », avec les liens vers le
  dépôt et sa documentation, figure dans l'aide intégrée, À propos, la page tournoi du manuel,
  un bouton info de la gestion de tournoi et en pied des pages produites.
- Écartés : un exécutable séparé ou gammonGo (le tournoi produit des Matchs, qui n'ont de
  valeur qu'analysés ici) ; deux entités (deux « Open » à réconcilier) ; une fiche personne
  réutilisable (`CONTEXT.md` refuse l'identité ; l'annuaire est une vue dérivée) ; un résultat
  déduit du Match rattaché (la parole du directeur fait foi ; un désaccord est montré, jamais
  résolu seul).

## Garde

`pkg/blunderdb/direction/direction_test.go`.
