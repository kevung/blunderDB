# Les dés d'un scan ne sont lus que confirmés par le texte voisin ; un videau à face vierge est centré

Statut : acceptée. Précise l'ADR-0117 règle 1 (« la légende et le coup de la solution ne sont
pas lus ») pour le contexte d'un livre scanné.

## Contexte

Le lecteur de structure compte les points des dés dessinés. Ce compte est une seule source :
une tache, un point collé au bord ou un dé mal découpé le faussent sans que rien ne le signale.
Sans recoupement, aucun dé n'était lu hors police XG. Deux voies se présentaient : une
certification « complète » du contexte, analogue aux pions (quinze par camp sans doute,
acceptés sur option), ou un recoupement indépendant tiré du livre lui-même.

Pour le videau, plusieurs livres dessinent le videau centré comme un carré vide, souvent en haut
ou en bas du plateau (501 Problems, Backgammon de Magriel et Roberts). Ce carré était pris pour
un videau possédé dont la valeur n'était pas lue.

## Décision

1. **Les dés comptés sont lus seulement si le texte voisin du diagramme les confirme** : un
   jet nommé derrière un verbe (« to play 41 », « rolls a 65 », « rolled 55 ») ou un coup écrit
   dont chaque pas dépense un dé (« 2/0 2/1 » vaut 2-1, « 8/5(2) 6/3(2) » vaut 3-3). Un coup
   dont un pas dépense deux dés (« 13/6 ») ne dit aucun jet. Un mot plus proche du champ d'un
   autre diagramme de la page appartient à celui-ci. Si le texte ne cite qu'un jet, une
   citation concordante suffit ; s'il en cite plusieurs (la prose cite aussi le jet précédent,
   celui de l'adversaire), le jet compté doit y être cité deux fois. Sinon, un doute nommé.
2. **Pas de certification « complète » du contexte.** Le repli (`--fallback`) la tient déjà :
   la position importée porte les dés comptés, et le commentaire de référence demande de les
   contrôler. Un second état intermédiaire doublerait ce chemin sans recouper davantage.
3. **Un videau dessiné seul, à face vierge, est le videau centré** quand le document le dessine
   ainsi, au même endroit (hauteur et côté du champ), à côté d'au moins trois diagrammes ; une
   seule face vierge peut être un chiffre perdu au scan, elle laisse un doute. Un videau possédé
   montre toujours sa valeur. La face est vierge quand aucun pixel de l'intérieur n'est plus
   sombre que son papier de 30 % du contraste entre ce papier et l'encre de son propre contour :
   un chiffre pâle sur un scan délavé reste une marque. Deux carrés dont un vierge, ou un carré
   plus proche du champ d'un autre diagramme, ne sont pas lus.
4. **Ces règles valent pour tout document** : aucune ne nomme un livre ni n'est calée sur une
   mise en page. Le carré d'un videau est cherché à 2,5 points du champ au plus. Un carré, dé
   ou videau, est jugé carré (0,75 à 1,33) une fois retiré l'étirement que mesurent les pions
   du document (largeur sur hauteur) ; un scan étiré l'est pour tout ce qu'il imprime.

## Conséquences

- Mesure sur les pages de réglage (`tasks/mesure-import-document-2026-11.md`), douze livres
  scannés : videau lu 413 → 678 (501 Problems 0 → 256), dés lus 2 → 49 (Masterclass 1 → 42) ;
  aucun livre ne perd une lecture.
- Le risque résiduel d'une lecture fausse des dés suppose qu'un compte faux égale exactement
  un jet ou un coup cité près du diagramme.
- Rejeté : lire les dés sur leur seul motif de points (une seule source) ; exiger un seul jet
  cité (la prose en cite plusieurs et les dés ne seraient presque jamais lus).
