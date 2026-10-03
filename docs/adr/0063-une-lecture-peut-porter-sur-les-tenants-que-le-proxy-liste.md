# ADR-0063 — Une lecture peut porter sur les tenants que le proxy liste

Statut : acceptée.
Voir aussi : ADR-0005 (le démon n'authentifie rien), ADR-0057 (lectures conditionnelles),
ADR-0059 (MCP) ; `tasks/plan-2026-10b.md`, lot E ; issue J.6 (club / coach).

## Contexte

`serve` est étanche par Tenant : chaque appel porte `X-Tenant-ID`, et rien de ce qu'un Tenant
stocke n'est visible d'un autre. Un club ou un coach veut l'inverse, mais contrôlé : le coach
lit les matchs et les stats de ses élèves, un club partage une bibliothèque en lecture et
publie un classement. La relation (qui coache qui, qui appartient à quel club) existe déjà chez
l'hôte, gammonGo, qui authentifie les comptes. La loger aussi dans blunderDB en ferait un
second système d'identité à tenir synchrone, ce qu'ADR-0005 refuse.

## Décision

1. **La relation vit chez l'appelant.** Le proxy qui authentifie écrit, en plus de
   `X-Tenant-ID`, l'en-tête **`X-Read-Tenants`** : la liste des Tenants que la requête lit en
   plus du sien. blunderDB ne stocke aucune relation et n'autorise rien ; il fait confiance à
   cet en-tête comme à `X-Tenant-ID` (ADR-0005, inchangée). Le proxy retire tout
   `X-Read-Tenants` venu du client, comme il retire `X-Tenant-ID`.
2. **Opt-in, et refusé tant qu'il n'est pas activé.** Le démon n'honore `X-Read-Tenants`
   qu'avec `serve --read-tenants` (`BLUNDERDB_READ_TENANTS=true`) ou, embarqué,
   `Config.TrustReadTenants` de `pkg/blunderdb/server`. Désactivé — le défaut —, tout
   `X-Read-Tenants` non vide est refusé (400 `invalid`), jamais ignoré : un proxy écrit avant
   cet en-tête retire `X-Tenant-ID` et transmet le reste ; s'il était lu par défaut, une simple
   mise à jour du démon laisserait tout client derrière ce proxy lire 63 autres Tenants. Le
   refus fait voir sur-le-champ un proxy qui transmet l'en-tête du client. L'opérateur active
   la fonction une fois son proxy corrigé.
3. **Forme.** Une seule ligne d'en-tête : deux lignes `X-Read-Tenants` refusent la requête (un
   proxy qui ajoute sa valeur au lieu de remplacer celle du client fusionnerait les deux
   listes). Des Tenants séparés par des virgules (`2, 3,7`), chacun au format d'ADR-0005
   (entier décimal positif canonique, au plus int64), espaces autour de chaque élément
   ignorés. Un en-tête absent ou vide est le comportement d'avant : la lecture porte sur
   `X-Tenant-ID` seul. Un élément vide, une virgule finale, un nom, un signe, `0`, `02`
   refusent **toute** la requête (400 `invalid`), quelle que soit la route, écriture comprise :
   une liste cassée est un bogue du proxy qui doit se voir, pas une liste plus courte lue en
   silence.
4. **Ensemble de lecture** (`storage.ReadTenants`) : `X-Tenant-ID` d'abord, toujours inclus,
   puis les Tenants listés dans l'ordre de l'en-tête, chacun une fois. Il compte au plus
   `storage.MaxReadTenants` = 64 Tenants **distincts**, `X-Tenant-ID` compris, comptés après
   dédoublonnage (`2,2,3` en compte deux) : une lecture coûte un appel au stockage par
   Tenant, la borne garde le coût d'une requête prévisible ; un club plus grand se découpe
   chez l'appelant.
5. **Seules les routes `/v1/across.*` lisent l'en-tête.** Sur tout l'ensemble :
   `across.searchFind`, `across.matchesList`, `across.statsCompute`, `across.playerTable`. Sur
   un Tenant de l'ensemble, nommé dans la requête : `across.matchesGet`,
   `across.matchMovePositions` (les positions d'un match, coup par coup),
   `across.analysesLoadByIds`. Aucune n'écrit ; aucune autre route ne regarde
   `X-Read-Tenants`. **Une écriture ne va donc que dans `X-Tenant-ID`**, par construction : les
   routes existantes restent mono-tenant et inchangées.
6. **Chaque résultat dit son Tenant d'origine** (`{"tenant": "2", "match": …}`). Un id n'est
   unique que dans son Tenant : une ligne lue à travers les Tenants se nomme (tenant, id), et
   les routes à un Tenant prennent les deux. Un Tenant hors de l'ensemble y est refusé (400)
   avant tout appel au stockage. Une position lue porte en plus son **hachage Zobrist**
   (`zobrist`, celui sous lequel `SavePosition` la range et que `positions.exists` reconnaît) :
   l'id ne nomme la position que dans son Tenant, le hachage nomme le même plateau dans tous.
   Les flux sont bornés : `limit` s'applique à chaque Tenant, 0 vaut `maxPageSize` (1000) et
   davantage est refusé, puisqu'une requête multiplie son flux par jusqu'à 64 ; la réponse met
   les pages bout à bout dans l'ordre de l'ensemble. Une erreur sur un Tenant arrête la
   lecture : une réponse partielle passerait pour l'ensemble. En NDJSON, elle arrive comme
   dernière ligne (`{"error": …}`) **après** les pages des Tenants déjà lus, le statut 200 étant
   parti : le client tient alors le flux entier pour échoué, pas ces pages pour la réponse de
   leurs Tenants. Seule une erreur sur le premier Tenant, avant toute ligne, a son propre
   statut HTTP.
7. **Le stockage ne prend pas plusieurs scopes.** `storage.ReadAcross`, `StreamAcross` et
   `ReadOne` appellent les méthodes mono-scope existantes une fois par Tenant, sous un contexte
   qui porte ce Tenant (`storage.WithTenant`) : la sécurité au niveau des lignes de PostgreSQL
   (`serve --rls`) filtre chaque appel comme n'importe quel autre, et les deux backends
   répondent par le code que la suite de contrat tient déjà.
8. **SQLite n'a qu'un Tenant** : en `serve` sur SQLite, `X-Read-Tenants` ne peut lister que `1`
   (sinon 400, comme `X-Tenant-ID`). L'en-tête n'y élargit rien. Le bureau et `call` n'ont ni
   proxy ni autre Tenant : les routes `across.*` sont propres au serveur (`serverOnlyPaths`).
9. **Cache et idempotence.** Les routes `across.*` ne sont ni conditionnelles (pas d'ETag,
   ADR-0057) ni enveloppées par `Idempotency-Key` : ce sont des lectures, sans effet à rejouer.
   La mémoire d'`Idempotency-Key` reste clée par `X-Tenant-ID` et n'est pas touchée. Un cache
   placé devant le démon doit clé sur `X-Tenant-ID` **et** `X-Read-Tenants`.
10. **Coût et traces.** Une requête `across.*` fait jusqu'à 64 lectures au stockage, mais la
    limite de débit la compte une fois, sur `X-Tenant-ID` : l'opérateur dimensionne la base
    et la limite en conséquence, ou le proxy borne la liste plus court. Le journal d'accès
    d'une route `across.*` porte l'en-tête reçu (`read_tenants`, tronqué comme `tenant`).
    `X-Read-Tenants` n'est pas dans les en-têtes CORS autorisés : seul le proxy l'écrit, jamais
    un navigateur.

## Ce que le socle fournit aux fonctions club / coach

Ces fonctions sont une étape suivante ; le socle leur donne :

- *le coach lit ses élèves* : `across.matchesList`, `across.matchesGet`, `across.statsCompute`
  sous `X-Tenant-ID` = coach, `X-Read-Tenants` = élèves ;
- *bibliothèque partagée* : `across.searchFind` avec le Tenant de la bibliothèque listé ;
- *commentaires du coach*, de bout en bout : le coach lit un match d'un élève par
  `across.matchMovePositions` (chaque position avec son tenant, son id chez l'élève et son
  hachage), ses analyses par `across.analysesLoadByIds` (tenant de l'élève, ids lus à l'étape
  d'avant) ; pour commenter, il range la position chez lui (`positions.save` sous son propre
  `X-Tenant-ID`, donc `SavePosition` : même plateau, même hachage, id du coach) puis écrit le
  commentaire sur cet id (`comments.add`). La jointure à la position de l'élève se fait par le
  hachage, jamais par l'id. Rien n'est écrit chez l'élève ;
- *classement de club* : `across.playerTable`, une table par Tenant, à fusionner chez
  l'appelant ;
- *MCP* : `/mcp` passe par le même middleware que `/v1`, qui valide `X-Read-Tenants` à
  l'entrée ; aujourd'hui `mcp.Engine.send` ne recopie que `X-Tenant-ID` sur ses appels `/v1`
  internes. Les outils de lecture across recopieront de même l'en-tête reçu par `/mcp`
  (`req.Extra.Header`) sur l'appel `/v1/across.*` qu'ils font, et lui seul : l'ensemble lu
  reste celui que le proxy a posé, jamais un choix du modèle.

## Conséquences

- La frontière de confiance s'élargit d'un en-tête. Qui atteint le port lit déjà n'importe
  quel Tenant en le nommant dans `X-Tenant-ID`, mais un proxy conforme à ADR-0005 ne protégeait
  que cet en-tête-là : avec `--read-tenants`, il doit aussi retirer `X-Read-Tenants` venu du
  client, faute de quoi un client authentifié lit les Tenants qu'il veut. D'où l'opt-in et le
  refus par défaut (règle 2), et la note de mise à jour du manuel. Les avertissements
  d'ADR-0005 restent tels quels.
- Le glossaire gagne **Read tenants** ; l'entrée **Tenant** dit l'exception.
- Rejeté : une table de relations dans blunderDB (club, coach, élève) — un second système
  d'identité, le refus d'ADR-0005.
- Rejeté : élargir les routes existantes quand l'en-tête est présent — leurs réponses ne disent
  pas leur Tenant, et un id seul y deviendrait ambigu.
- Ce n'est pas le partage de collection : `exports.sqlite` (`collectionIds`) puis
  `imports.db` **copie** des données d'un Tenant chez un autre, par un fichier que l'appelant
  transporte ; le receveur les possède ensuite. `across.*` ne copie rien et ne fait que lire,
  le temps d'une requête. Les deux chemins ne se recouvrent pas : partager pour garder,
  lire across pour consulter.
- Rejeté : une méthode de stockage à plusieurs scopes — chaque requête SQL et chaque politique
  RLS à réécrire, pour un gain que la boucle par Tenant donne déjà.

## Garde

`internal/server/trust_boundary_test.go` (`TestReadTenants_*` : un Tenant non listé n'est
jamais lu, aucune écriture hors `X-Tenant-ID`, l'en-tête refusé sans `--read-tenants`),
`internal/server/middleware/tenant_read_test.go`, `internal/server/handlers_across_test.go`,
`handlers_across_postgres_test.go`, `pkg/blunderdb/storage/across_test.go`, et
`storagetest.RunReadAcrossTests` sur les deux backends, dont une passe PostgreSQL sous RLS
active avec un rôle non superutilisateur.
