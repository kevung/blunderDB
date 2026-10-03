# ADR-0061 — Une lecture peut porter sur les tenants que le proxy liste

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
2. **Forme.** Des Tenants séparés par des virgules (`2, 3,7`), chacun au format d'ADR-0005
   (entier décimal positif canonique), espaces autour de chaque élément ignorés. Un en-tête
   absent ou vide est le comportement d'avant : la lecture porte sur `X-Tenant-ID` seul. Un
   élément vide, un nom, `0`, `02` refusent **toute** la requête (400 `invalid`), quelle que
   soit la route, écriture comprise : une liste cassée est un bogue du proxy qui doit se voir,
   pas une liste plus courte lue en silence.
3. **Ensemble de lecture** (`storage.ReadTenants`) : `X-Tenant-ID` d'abord, toujours inclus,
   puis les Tenants listés dans l'ordre de l'en-tête, chacun une fois. Il compte au plus
   `storage.MaxReadTenants` = 64 Tenants, `X-Tenant-ID` compris : une lecture coûte un appel au
   stockage par Tenant, la borne garde le coût d'une requête prévisible ; un club plus grand se
   découpe chez l'appelant.
4. **Seules les routes `/v1/across.*` lisent l'en-tête** : `across.searchFind`,
   `across.matchesList`, `across.matchesGet`, `across.statsCompute`, `across.playerTable`.
   Aucune n'écrit ; aucune autre route ne regarde `X-Read-Tenants`. **Une écriture ne va donc
   que dans `X-Tenant-ID`**, par construction : les routes existantes restent mono-tenant et
   inchangées.
5. **Chaque résultat dit son Tenant d'origine** (`{"tenant": "2", "match": …}`). Un id n'est
   unique que dans son Tenant : une ligne lue à travers les Tenants se nomme (tenant, id), et
   `across.matchesGet` prend les deux. Un Tenant hors de l'ensemble y est refusé (400) avant
   tout appel au stockage. Les bornes (`limit`, `offset`) s'appliquent à chaque Tenant ; la
   réponse met les pages bout à bout dans l'ordre de l'ensemble. Une erreur sur un Tenant
   arrête la lecture : une réponse partielle passerait pour l'ensemble.
6. **Le stockage ne prend pas plusieurs scopes.** `storage.ReadAcross`, `StreamAcross` et
   `ReadOne` appellent les méthodes mono-scope existantes une fois par Tenant, sous un contexte
   qui porte ce Tenant (`storage.WithTenant`) : la sécurité au niveau des lignes de PostgreSQL
   (`serve --rls`) filtre chaque appel comme n'importe quel autre, et les deux backends
   répondent par le code que la suite de contrat tient déjà.
7. **SQLite n'a qu'un Tenant** : en `serve` sur SQLite, `X-Read-Tenants` ne peut lister que `1`
   (sinon 400, comme `X-Tenant-ID`). L'en-tête n'y élargit rien. Le bureau et `call` n'ont ni
   proxy ni autre Tenant : les routes `across.*` sont propres au serveur (`serverOnlyPaths`).
8. **Cache et idempotence.** Les routes `across.*` ne sont ni conditionnelles (pas d'ETag,
   ADR-0057) ni enveloppées par `Idempotency-Key` : ce sont des lectures, sans effet à rejouer.
   La mémoire d'`Idempotency-Key` reste clée par `X-Tenant-ID` et n'est pas touchée. Un cache
   placé devant le démon doit clé sur `X-Tenant-ID` **et** `X-Read-Tenants`. La limite de débit
   reste comptée par `X-Tenant-ID`.

## Ce que le socle fournit aux fonctions club / coach

Ces fonctions sont une étape suivante ; le socle leur donne :

- *le coach lit ses élèves* : `across.matchesList`, `across.matchesGet`, `across.statsCompute`
  sous `X-Tenant-ID` = coach, `X-Read-Tenants` = élèves ;
- *bibliothèque partagée* : `across.searchFind` avec le Tenant de la bibliothèque listé ;
- *commentaires du coach* : écrits chez le coach (écriture dans `X-Tenant-ID`), joints à la
  position de l'élève par son hachage Zobrist, que porte chaque position lue à travers les
  Tenants ;
- *classement de club* : `across.playerTable`, une table par Tenant, à fusionner chez
  l'appelant ;
- *MCP* : les mêmes lectures, à exposer par le serveur `/mcp` avec le même ensemble.

## Conséquences

- La frontière de confiance ne bouge pas : qui atteint le port lit déjà n'importe quel Tenant
  en le nommant dans `X-Tenant-ID` ; `X-Read-Tenants` n'ajoute pas de pouvoir, il en économise
  les allers-retours. Les avertissements d'ADR-0005 restent tels quels.
- Le glossaire gagne **Read tenants** ; l'entrée **Tenant** dit l'exception.
- Rejeté : une table de relations dans blunderDB (club, coach, élève) — un second système
  d'identité, le refus d'ADR-0005.
- Rejeté : élargir les routes existantes quand l'en-tête est présent — leurs réponses ne disent
  pas leur Tenant, et un id seul y deviendrait ambigu.
- Rejeté : une méthode de stockage à plusieurs scopes — chaque requête SQL et chaque politique
  RLS à réécrire, pour un gain que la boucle par Tenant donne déjà.

## Garde

`internal/server/trust_boundary_test.go` (`TestReadTenants_*` : un Tenant non listé n'est
jamais lu, aucune écriture hors `X-Tenant-ID`), `internal/server/handlers_across_test.go`,
`handlers_across_postgres_test.go`, `pkg/blunderdb/storage/across_test.go`, et
`storagetest.RunReadAcrossTests` sur les deux backends.
