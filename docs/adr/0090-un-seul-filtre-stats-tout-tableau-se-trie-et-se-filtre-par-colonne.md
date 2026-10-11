# ADR-0090 — Un seul filtre Stats ; tout tableau se trie et se filtre par colonne

Statut : acceptée.
Prolonge : ADR-0085 (grammaire des panneaux).

## Contexte

Les tableaux du panneau Stats étaient écrits à la main, onglet par onglet : l'un triable (Joueurs), les autres figés dans
l'ordre du backend, aucun filtrable. D'où une section *Classement* dans Corpus, avec son
propre seuil de décisions, qui recalculait à la demande ce que l'onglet Joueurs montre déjà.

## Décision

1. **Tout tableau se trie et se filtre par colonne.** `panels/PanelTable.svelte` reçoit un
   `viewId` : il possède alors le tri et les filtres et les applique lui-même
   (`utils/columnView.js`, pur). Une colonne déclare `value(row)` — `null` pour une valeur
   jamais mesurée, toujours en dernier, qui ne passe aucune borne — et `filter: 'number'` ou
   `'text'`. Une seconde ligne d'en-tête porte un champ compact par colonne filtrable, sans
   prose : `≥ …` / `…` en indication, le libellé de la colonne pour l'accessibilité.
2. **Syntaxe numérique unique** : `>=500`, `>500`, `<2.5`, `<=2`, `3-7` (bornes incluses),
   `5`. Un texte illisible est signalé au champ et ne filtre rien. Un filtre texte est une
   sous-chaîne insensible à la casse ; les égalités se départagent par la première colonne
   texte.
3. **Le tri se retient, les filtres non.** Le tri de chaque `viewId` va dans Config.yaml
   (`table_sorts`) ; une colonne disparue est ignorée. Les filtres de colonne durent la
   session : retenus d'un lancement à l'autre, ils cacheraient des lignes sans rien à
   l'écran pour dire pourquoi.
4. **Sans `viewId`, rien ne change** : les panneaux Matchs, Tournois, Collections, Anki et
   Transcription trient par `bind:sort`, avec leurs propres filtres.
5. **Un compte est un `CountLink`** (ADR-0085, G7) dans les tableaux Stats aussi ; plus de
   CSS de lien maison.

## Conséquences

- La section *Classement* de Corpus disparaît : l'onglet Joueurs trié par PR, colonne
  *Décisions* filtrée par `>=N`, la remplace. `blunderdb stats ranking` et
  `/v1/stats.ranking` restent, la ligne de commande n'ayant pas de tableau à trier.
- Un nouveau tableau Stats se déclare par ses colonnes et un `viewId` ; il n'écrit ni tri ni
  filtre.
- Le tableau de direction de l'onglet Erreurs et le bloc de comparaison de deux joueurs
  restent écrits à la main : ce sont des grilles fixes, pas des listes.
