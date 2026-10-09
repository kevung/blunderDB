# ADR-0084 — Une collection-requête s'évalue dans le moteur, sous un plafond déclaré

Statut : acceptée.
Voir aussi : ADR-0005 (le daemon fait confiance au tenant de la requête), ADR-0007 (rien ne
s'enregistre chez le destinataire), ADR-0026 (plafond de séance d'un paquet), ADR-0042 (ce
qu'une carte interroge).

## Contexte

Une collection vivante porte une requête (`collection.filter_query`) au lieu d'une liste
d'ids : sa composition est le résultat de la recherche, réévalué à chaque ouverture. Un client
du daemon (gammonGo) évaluait cette requête de son côté ; or évaluer une requête de positions
appartient au moteur, seul à connaître la grammaire et la base. Il manquait trois choses : une
borne au nombre de positions qu'une évaluation rend, un paquet d'étude qui suive une
collection vivante (il ne lisait que les lignes `collection_position`, qu'une collection
vivante n'a pas), et un refus commun aux trois modes d'une requête illisible.

## Décision

1. **`filter_query` vit sur la collection, l'évaluation dans `storage`.** Pas de changement
   de schéma : la colonne existe. `storage.LivingFilters` lit la requête et la résout par
   `searchquery.Living`, le parseur de la barre de commande. `storage.EvaluateCollection`,
   `storage.SyncDeck`, `storage.CreateCollection` et `storage.SetCollectionFilter` sont les
   seules entrées ; `Database`, la CLI et les routes `serve` les appellent toutes, avec le
   `scope` de la requête. Une collection vivante est donc évaluée par `Search()` dans le
   tenant, sous la RLS comme toute recherche.
2. **Plafond déclaré, jamais de troncature muette.** `storage.LivingCollectionCap` (5 000)
   borne une évaluation ; un `limit` plus petit l'abaisse. La réponse
   (`CollectionEvaluation`) porte toujours les ids, le total, le plafond appliqué et
   `truncated`. `truncated` se décide dans la même lecture que les ids (on demande un id de
   plus que le plafond) : il est exact pour cette lecture. Le total vient d'un comptage
   séparé, hors transaction commune ; sous écriture concurrente il peut différer de ce que
   la lecture a vu, mais il n'est jamais rendu sous le nombre d'ids, ni au plus égal au
   plafond quand `truncated` est vrai. La même forme répond pour une liste faite à la main. Les lectures par fenêtres
   (`positionIds`, `countPositions`) restent bornées par leur propre `limit` et ne sont pas
   plafonnées : elles ne rendent jamais la collection entière.
3. **Le paquet fondé sur une collection vivante.** Un paquet de source `collection` dont la
   collection est vivante se resynchronise par `EvaluateCollection` sous le plafond : les
   premières positions dans l'ordre de la recherche reçoivent une carte, les cartes déjà là
   gardent leur planification, aucune n'est retirée quand une position sort de la requête
   (comme pour toute source). La resynchronisation a lieu à l'ouverture de la séance, là où
   l'interface synchronisait déjà. Le rapport (`DeckSync.Source`) dit si le plafond a coupé ;
   `/v1/anki.sync` le renvoie à côté de l'`ok` d'avant. Une liste faite à la main nourrit
   toujours son paquet entière : elle a été choisie position par position.
4. **Rapport au paquet « recherche ».** Il reste ce qu'il est : une liste d'ids figée au
   moment de la création, que seule l'interface sait rejouer. La collection vivante est la
   forme évaluée par le moteur ; un client qui veut un paquet qui suit une requête crée une
   collection vivante et un paquet de source `collection` dessus.
5. **Une requête illisible est refusée à l'écriture**, dans les trois modes : un jeton
   qu'aucune règle ne réclame ferait d'une collection toute la base. `ErrUnreadableFilter`
   enveloppe `ErrInvalid` (code `invalid` côté daemon) ; une requête refusée n'écrit rien,
   car elle est vérifiée avant toute écriture. Créer une collection vivante fait deux
   écritures (création, puis requête) ; si la seconde échoue, la collection juste créée est
   supprimée, sans garantie transactionnelle.

## Conséquences

- Pas de migration : `DatabaseVersion` ne bouge pas.
- Une requête qui sélectionne plus de 5 000 positions nourrit un paquet de ses 5 000
  premières, dans l'ordre de la recherche ; l'utilisateur le lit dans la barre d'état, la CLI
  sur la sortie d'erreur, un client dans `truncated`.
- Relever le plafond est un changement de constante, mais il engage la taille des paquets :
  il se décide ici, pas dans un appel.
