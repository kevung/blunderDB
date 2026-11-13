# Une voisine est à N jets légaux, et `like` ne rend que la relation exacte

Statut : acceptée. Remplace l'ADR-0043 (règles 1, 2 pour `<n` et `*`, 3 pour la classe, 4).

## Contexte

La distance de transport en pions-pas de l'ADR-0043 ignore le sens de marche, l'alternance des
joueurs et la légalité des coups. Mesurée sur une base réelle de 105 000 positions (25 cibles de
milieu de partie), aucune des 250 premières voisines n'était atteignable en un ou deux jets, et la
position qui suit réellement la cible dans sa partie sortait à une distance de 29 à 53. La classe
d'équivalence et l'exclusion du match de la cible ne corrigeaient pas une distance qui ne parle pas
de jeu.

## Décision

1. **Deux positions sont à distance N** (N ≥ 1) si N jets légaux successifs mènent de la
   structure de pions de l'une à celle de l'autre. Les jets alternent les camps ; le premier est
   joué par l'un ou l'autre ; le sens est libre (A → B ou B → A), si bien que la relation est
   symétrique. La distance d'une paire est le plus petit tel N ; la même structure est à 0.
2. **Structure de pions seulement** : videau, score, dés, type de décision et camp au trait n'en
   font pas partie. Une structure est la paire non ordonnée des deux camps, chacun dans sa propre
   numérotation — ce que garde une position normalisée au joueur au trait.
3. **Légalité de `domain.LegalMoves`** : entrée depuis la barre d'abord, usage maximal des dés, dé
   le plus fort quand un seul passe, sortie exacte ou depuis le point le plus haut, frappe. Un jet
   injouable compte pour un jet. `engine/reach` porte son propre générateur, sans notation ni
   étapes ; les tests le tiennent à `domain.LegalMoves` jet par jet.
4. **Dans toute la base** : ni classe ni exclusion du match de la cible. La position suivante de
   la partie est à un jet, et c'est une réponse.
5. **`like<n>` rend les positions à distance ≤ n**, triées par distance puis par identifiant, sans
   jamais compléter par plus loin : la liste peut être vide. Sans `<n`, n = 2
   (`reach.DefaultOrder`). La distance de chaque résultat est rendue (API, CLI, interface).
6. **Exacte ou refusée** : un ordre au-delà de `reach.MaxOrder` (2) est refusé (`ErrInvalid`),
   jamais approché. Les filtres bon marché — pions sortis monotones, fenêtre de pips, comptes
   cumulés qui ne reculent que d'un pion par frappe, au plus quatre pions déplacés par jet,
   frappes d'après le dernier jet d'un camp encore sur la barre, dés compatibles avec les pips
   joués — n'élaguent qu'avant le test exact par génération des coups.
7. **La cible « plateau dessiné »** (mode édition) suit la même relation.
8. **Transfert sans provenance** : `like<id>` nomme une ligne de la base et reste
   `ReadsProvenance` ; seul `like` sur un plateau dessiné voyage.

## Mesures

Base réelle de 105 538 positions, SQLite, une recherche sans autre filtre (balayage compris,
≈ 0,45 s) :

| cible | N = 1 | N = 2 |
|---|---|---|
| ouverture (3 cibles) | 0,44–0,66 s, 361–1 701 résultats | 1,75–2,05 s, 1 976–2 500 résultats |
| milieu de partie (3) | 0,43–0,45 s, 2–8 résultats | 0,51–2,66 s, 4–8 résultats |
| fin de sortie (3) | 0,44–0,45 s, 2–9 résultats | 0,46–0,48 s, 7–60 résultats |

N = 3 n'est pas servi : à trois jets, les filtres ne bornent plus les frappes du premier camp et
la génération par candidate devient la norme.

## Conséquences

- `engine/reach` (structure, générateur, distance) ; `SearchStore.Rank` y délègue pour les deux
  stockages ; `storage.ClassOf`, `SimilarOptions`, `AppendClassSQL`, `MatchesOfPosition` et la
  forme `like*` disparaissent, avec `cmd/likecorpus` (un corpus jugé à l'œil n'a plus d'objet
  pour une relation exacte).
- Le réglage « distance maximale » se compte en jets ; une valeur que la recherche ne sert pas
  se lit comme zéro.
- « Ailleurs » (ADR-0103) classe ses voisines par la même relation.
- `engine.SimilarityDistance` ne sert plus que les positions de référence (ADR-0080), qui gardent
  leur rayon en pions-pas.
- Rejeté : garder l'ancienne distance sous un autre nom — le propriétaire a choisi le
  remplacement.
- Rejeté : un ensemble précalculé des structures à deux jets de la cible — un demi-million
  d'entrées en milieu de partie, pour un gain que les filtres donnent déjà.

## Garde

`pkg/blunderdb/engine/reach/reach_test.go` (générateur contre `domain.LegalMoves`, distance contre
l'oracle `reachtest`), `pkg/blunderdb/database/db_similar_test.go` (un match réel),
`pkg/blunderdb/storage/storagetest/contract_similar.go` (les deux stockages).
