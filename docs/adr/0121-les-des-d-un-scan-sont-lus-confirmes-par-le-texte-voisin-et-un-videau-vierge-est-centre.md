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
   dont un pas dépense deux dés (« 13/6 ») ne dit aucun jet. Il suffit qu'un des jets ou coups
   cités concorde avec le compte : la prose cite aussi d'autres jets (le précédent, celui de
   l'adversaire). S'ils discordent tous, le doute dit que le texte contredit l'image.
2. **Pas de certification « complète » du contexte.** Le repli (`--fallback`) la tient déjà :
   la position importée porte les dés comptés, et le commentaire de référence demande de les
   contrôler. Un second état intermédiaire doublerait ce chemin sans recouper davantage.
3. **Un videau dessiné seul, à face vierge, est le videau centré**, où qu'il soit placé. Un videau
   possédé montre toujours sa valeur. La face est vierge quand aucun pixel de l'intérieur n'atteint
   la moitié de l'encre, et ni le carré ni la couche texte ne portent de valeur. Deux carrés dont
   un vierge laissent un doute.
4. **Ces règles valent pour tout document** : aucune ne nomme un livre ni n'est calée sur une
   mise en page. Le carré d'un videau est cherché à 2,5 points du champ au plus. Il peut être
   plus large que haut (jusqu'à 1,75), parce qu'un scan peut être étiré en largeur. Un dé, lui,
   reste carré.

## Conséquences

- Mesure sur les pages de réglage (`tasks/mesure-import-document-2026-11.md`) : videau lu sur
  501 Problems 0 → 271, sur Magriel et Roberts 0 → 10 ; dés lus sur Masterclass 1 → 46, Endgame
  0 → 6, Opening Concepts 2017 1 → 4 ; aucun livre ne perd une lecture.
- Le risque résiduel d'une lecture fausse des dés suppose qu'un compte faux égale exactement
  un jet ou un coup cité près du diagramme.
- Rejeté : lire les dés sur leur seul motif de points (une seule source) ; exiger un seul jet
  cité (la prose en cite plusieurs et les dés ne seraient presque jamais lus).
