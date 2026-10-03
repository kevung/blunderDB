# ADR-0065 — Club et coach lisent à travers les tenants et n'écrivent que chez soi

Statut : acceptée.
Voir aussi : ADR-0005 (le démon n'authentifie rien), ADR-0059 (MCP), ADR-0062 (classement de
saison), ADR-0063 (`X-Read-Tenants`, le socle) ; `tasks/plan-2026-10b.md`, lot E.

## Contexte

ADR-0063 donne le socle : une lecture `across.*` porte sur `X-Tenant-ID` et les Tenants que le
proxy liste, chaque résultat dit son Tenant, et une position lue porte son hachage Zobrist. Il
reste à servir les quatre usages du lot E — le coach lit ses élèves, commente leurs positions,
un club partage une bibliothèque et publie un classement — sans créer de relation entre
Tenants dans blunderDB, ni de second modèle de commentaire.

## Décision

1. **Le commentaire du coach est un commentaire ordinaire, écrit chez le coach.** Le coach
   range la position de l'élève sous son propre `X-Tenant-ID` (`positions.save` :
   `SavePosition`, même hachage, id du coach) et la commente (`comments.add`, origine `user`,
   le modèle de commentaire existant). Rien n'est écrit chez l'élève. La lecture inverse est
   `across.commentsByZobrist` (`{"zobrists": [...]}`) : pour chaque
   Tenant lu, deux lectures quel que soit le nombre de hachages — les positions que ces
   hachages désignent chez lui (`Positions().ExistsMany`, une requête : `= ANY($2)` en
   PostgreSQL, `IN` par lots de 900 en SQLite), puis leurs commentaires
   (`Comments().ByPositions`) ; chaque ligne dit le Tenant qui a écrit, l'id de la position
   chez lui, le hachage et l'origine. La jointure passe par le hachage, jamais par l'id. Au
   plus `storage.MaxZobristLookups` = 1000 hachages par requête. Toutes les origines sont rendues ; le tri entre la
   note du coach et la remarque d'un fichier importé chez lui revient à l'appelant, qui voit
   l'origine.
2. **La bibliothèque partagée est un Tenant listé, lu sur place.** `across.collectionsList`
   (toutes les collections de l'ensemble) et `across.collectionPositions` (une collection
   d'un Tenant nommé, positions avec leur hachage, `limit` borné comme les autres flux). Ce
   n'est pas le partage par fichier (`exports.sqlite` puis `imports.db`), qui donne une copie
   au receveur ; une collection vivante est lue comme `collections.positions` la lit, par ses
   lignes rangées.
3. **Le classement de club fusionne des tables de joueurs, jamais des personnes.**
   `across.clubRanking` calcule `PlayerTable` par Tenant sous le filtre de statistiques donné
   (une période : `DateFrom`, `DateTo`) et classe toutes les lignes ensemble avec l'ordre de
   la table (`storage.ClubRanking`) : meilleur PR d'abord, lignes sans décision comptée en
   dernier avec le rang 0 (leur PR ne mesure rien), PR égal → rang partagé. Chaque ligne
   porte son Tenant. Un nom n'identifie une personne que dans son Tenant : deux élèves qui
   ont chacun affronté un « Paul » n'ont pas joué le même Paul, donc aucun nom n'est fusionné
   d'un Tenant à l'autre. Qui est le joueur de quel Tenant, l'appelant le sait : `players`
   (paires Tenant, nom) ne garde que ces lignes, `minDecisions` écarte le bruit.
   `filter.TournamentIDs` est refusé (400 `invalid`) : un id de tournoi ne nomme un tournoi
   que dans son Tenant, et appliqué à tous il en choisirait d'autres ailleurs ; une période
   le remplace. La réponse est une page du classement entier (`limit` 100 par défaut, au plus
   `maxPageSize` = 1000, `offset`) avec `total`. Ce n'est pas le classement de saison
   d'ADR-0062, qui cumule des places de tournois dirigés.
4. **Des lectures mono-tenant, composées.** Les trois lectures appellent des méthodes
   mono-tenant (`Positions().ExistsMany`, seule ajoutée au contrat, `Comments().ByPositions`,
   `Collections().List` / `Positions`, `Stats().PlayerTable`) par `ReadAcross` /
   `StreamAcross` / `StreamOne` (règle 7 d'ADR-0063) : la RLS filtre chaque appel, aucun
   schéma ne change. `across.matchMovePositions` prend à son tour `limit` / `offset` (0 vaut
   `maxPageSize`) : le flux du stockage s'arrête une fois la page pleine.
5. **MCP, lecture seule.** `club_matches`, `club_match_positions`, `club_comments`,
   `club_library`, `club_library_positions`, `club_ranking`. `mcp.Engine` recopie
   `X-Read-Tenants` reçu sur `/mcp` sur ses appels `/v1/across.*` et sur eux seuls ; en stdio
   ou sans l'en-tête, l'outil lit le Tenant seul. Une liste coupée à 200 lignes le dit
   (`truncated`, et `total` pour le classement). Le hachage y voyage en chaîne décimale : un
   entier de 64 bits lu comme un double JSON perd ses derniers chiffres.

## Conséquences

- Le glossaire gagne **Coach comment**, **Shared library**, **Club ranking**.
- Les notes du coach suivent la vie de sa copie de la position, chez lui ; la position de
  l'élève n'en porte jamais. Une note sur un plateau que l'élève n'a plus reste chez le coach,
  et réapparaît dès qu'un Tenant lu range de nouveau ce plateau.
- Rejeté : écrire le commentaire chez l'élève — une écriture hors `X-Tenant-ID`, que la règle 5
  d'ADR-0063 exclut par construction.
- Rejeté : un modèle de commentaire « de coach » avec un auteur Tenant — l'auteur est déjà le
  Tenant où le commentaire vit, que chaque ligne `across` nomme.
- Rejeté : fusionner les joueurs par nom à travers les Tenants — une identité devinée, le refus
  d'ADR-0062 règle 4 qui ne vaut qu'à l'intérieur d'un club dirigé.

## Garde

`storagetest/contract_club.go` (dans `RunReadAcrossTests` : SQLite, PostgreSQL, PostgreSQL sous
RLS), `storage/club_test.go`, `internal/server/handlers_across_club_test.go`,
`TestAcrossClub_Postgres`, `pkg/blunderdb/mcp` `TestAcrossTools` (l'en-tête n'atteint que les
routes `across.*`).
