# Architecture multi-tenant PostgreSQL avec Row-Level Security : guide technique et sourcé

## TL;DR
- **Fail-closed partout** : convertir `X-Tenant-ID` en `BIGINT` via une table de correspondance nom→id avec cache, valider strictement (rejeter en-tête absent/vide/dupliqué, jamais de `tenant_id = 0`), rendre le 0 impossible par contrainte `CHECK` et par policy RLS qui refuse quand `current_setting('app.tenant_id', true)` renvoie NULL. Le piège Go classique — `id, _ := strconv.ParseInt(...)` qui produit 0 sur erreur — est une faille de fuite inter-tenant.
- **Le RLS bien indexé coûte peu** (quelques % avec un index menant par `tenant_id` et un prédicat d'égalité `leakproof`), mais **utilisez `SET LOCAL`/`set_config(...,true)` en portée transaction** plutôt que `set_config(...,false)` + `RESET` à la libération : c'est plus sûr (pas de fuite entre requêtes) et évite le `DISCARD ALL` qui invalide le cache de prepared statements de pgx (erreurs « prepared statement does not exist »). Depuis pgx **v5.7.6 (8 sept. 2025)**, le hook `PrepareConn` remplace `BeforeAcquire` et permet un aller-retour réseau avec remontée d'erreur.
- **Migrations concurrentes** : sérialisez avec `pg_advisory_xact_lock(clé)` dérivée d'un `hashtext()` du nom d'application ; sortez `CREATE INDEX CONCURRENTLY` de toute transaction (annotation `-- +goose NO TRANSACTION` / magic comment de tern) car il ne peut tourner dans un bloc transactionnel et provoque un deadlock avec le verrou. **FK composites `(tenant_id, id)`** = défense en profondeur, mais attention : PostgreSQL vérifie les FK en contournant RLS (canal auxiliaire documenté). Pour la sauvegarde par tenant, `COPY (SELECT … WHERE tenant_id = …) TO` est l'outil de base ; le PITR ne restaure pas un seul tenant sans instance séparée.

## Key Findings

1. **Identifiant de tenant** : préférez un `BIGINT` séquentiel interne stocké dans une table `tenants(id, code/name)`, résolu depuis le nom d'API via un cache. Évitez le hachage de chaîne vers BIGINT (risque de collision). L'UUIDv7 est valable si vous avez besoin de génération distribuée, mais coûte 16 octets vs 8 et un peu plus d'I/O d'index. Le danger n°1 est la conversion silencieuse en Go donnant `tenant_id = 0`.

2. **Coût RLS** : la littérature converge — surcoût faible (souvent 2–6 %, parfois « bruit ») si (a) index menant par `tenant_id`, (b) prédicat d'égalité (leakproof), (c) `current_setting` marqué STABLE évalué une fois par requête. Dégradations connues : prédicats non-leakproof (`LIKE`, fonctions custom) qui ne peuvent passer avant la qual de sécurité, et sous-requêtes dans les policies non aplaties. `set_config(...,false)` à chaque acquisition = un aller-retour réseau supplémentaire ; `SET LOCAL` dans la transaction applicative évite un round-trip séparé.

3. **Migrations** : `pg_advisory_xact_lock` (portée transaction, auto-release) est le choix le plus robuste. golang-migrate prend un `pg_advisory_lock` (portée session) et a des problèmes documentés de « dirty state » et de verrous laissés en cas d'erreur ; tern (de l'auteur de pgx) enveloppe par défaut chaque migration dans une transaction et supporte le magic comment pour désactiver la transaction.

4. **FK composites** : `(tenant_id, id)` empêche qu'une ligne d'un tenant référence celle d'un autre, mais nécessite `UNIQUE (tenant_id, id)` côté table référencée. Les vérifications de FK **contournent RLS** (doc officielle) → canal auxiliaire par messages d'erreur. Utilisez `FORCE ROW LEVEL SECURITY` pour que le propriétaire soit soumis aux policies.

5. **Sauvegarde/restauration par tenant** : pas d'outil natif de filtrage par ligne dans pg_dump (le patch `--include-table-data-where` a été « returned with feedback » en 2019). Solutions : `COPY (SELECT ... WHERE tenant_id=$1) TO`, schéma par tenant + pg_dump `--schema`, ou réplication logique. PITR = cluster entier. Tests d'isolation automatisés indispensables (accès croisé, fuzzing d'en-têtes, audit `pg_class.relrowsecurity`/`pg_policies`, vérification `NOT rolbypassrls`/`NOT rolsuper`).

## Details

### 1. Identifiants de tenant : de la chaîne d'API au BIGINT

**Comparaison des stratégies**

| Approche | Taille | Avantages | Inconvénients |
|---|---|---|---|
| **Séquentiel BIGINT** (`GENERATED ALWAYS AS IDENTITY`) | 8 o | plus petit et plus rapide en index/FK ; simple à déboguer | expose le volume ; conflits en systèmes distribués |
| **Table lookup nom→id + cache** | 8 o (BIGINT) | découple le nom public du BIGINT interne compact ; métadonnées tenant centralisées | nécessite un cache pour éviter un aller-retour par requête |
| **Hash déterministe chaîne→BIGINT** | 8 o | pas de lookup | **collisions** (anniversaire sur 64 bits) ; changement de fonction = rupture ; à proscrire comme clé d'isolation |
| **UUIDv4** | 16 o | génération distribuée, pas de divulgation de volume | aléatoire → fragmentation B-tree, page splits massifs, index plus gros |
| **UUIDv7** | 16 o | ordonné dans le temps → localité d'index proche d'un séquentiel ; PG 18 a `uuidv7()` natif | 16 o vs 8 ; encode un timestamp décodable ; contention sur la feuille droite en écriture très concurrente |

**Recommandation** : une table `tenants` avec un `BIGINT` interne comme clé d'isolation, et le nom/`code` d'API résolu via cette table avec un cache applicatif (LRU + TTL court, invalidation sur changement). PlanetScale (« Approaches to tenancy in Postgres ») le résume précisément : « A dedicated tenants table gives you a place to store this and lets the tenant_id column across your schema remain a compact, performant BIGINT foreign key. » Sur UUIDv7 vs v4 : Better Stack Community (« UUID v7 in PostgreSQL 18 ») chiffre la fragmentation — « Sequential IDs cause 10-20 page splits per million records. UUID v4 causes 5,000-10,000+ splits—that's 500 times more. » Le benchmark de référence est celui d'**Andrey Borodin (auteur du patch `uuidv7`), publié sur pgsql-hackers le 28 novembre 2024** (MacBook Air M2, insertion de 30M lignes) : UUIDv4 = 2 003 918 ms (33:23) contre UUIDv7 = 337 001 ms (05:37), soit « almost an order of magnitude better ». Mais BIGINT reste le plus rapide et le plus compact quand une seule base possède la séquence.

**Le piège Go des conversions silencieuses (critique pour la sécurité)**

Le motif dangereux :
```go
// ☠️ DANGER : sur erreur, tenantID vaut 0 et l'erreur est ignorée
tenantID, _ := strconv.ParseInt(r.Header.Get("X-Tenant-ID"), 10, 64)
```
Si l'en-tête est absent, vide, ou non numérique, `ParseInt` renvoie `0, err` — et le `_` jette l'erreur. Le serveur poursuit avec `tenant_id = 0`. Comme le note la communauté Go, « If the input is "abc", port becomes 0 » avec `strconv.Atoi(s)` + `_`. Un `tenant_id = 0` ou NULL est catastrophique : selon l'écriture de la policy, il peut soit tout bloquer (bien) soit matcher des lignes 0 réelles ou, pire, si la policy compare à un `current_setting` mal typé, provoquer un comportement inattendu.

Le motif fail-closed correct :
```go
func tenantFromHeader(r *http.Request) (int64, error) {
    vals := r.Header.Values("X-Tenant-ID")   // gère les en-têtes dupliqués
    if len(vals) != 1 {
        return 0, fmt.Errorf("X-Tenant-ID absent ou dupliqué")
    }
    raw := strings.TrimSpace(vals[0])
    if raw == "" {
        return 0, fmt.Errorf("X-Tenant-ID vide")
    }
    id, err := strconv.ParseInt(raw, 10, 64) // bitSize=64 explicite
    if err != nil {
        return 0, fmt.Errorf("X-Tenant-ID invalide: %w", err)
    }
    if id <= 0 {                              // interdit 0 ET négatifs
        return 0, fmt.Errorf("X-Tenant-ID doit être > 0")
    }
    // Puis : résolution/validation via la table tenants (cache), fail-closed si inconnu
    return id, nil
}
```
Notes : (1) utilisez `ParseInt(..., 64)` avec bitSize explicite plutôt que `Atoi` (dont la largeur dépend de la plateforme — problème réel documenté sur ARM 32 bits) ; (2) ne renvoyez jamais `err.Error()` brut au client (`strconv` inclut la chaîne d'entrée dans le message — vecteur XSS documenté, issue golang/go #13127).

**Rendre le tenant_id 0/NULL impossible côté base (défense en profondeur)**
```sql
-- 1) Contrainte CHECK sur chaque table tenant
ALTER TABLE documents ADD CONSTRAINT documents_tenant_positive
  CHECK (tenant_id > 0);

-- 2) Policy RLS qui refuse quand le GUC n'est pas défini.
--    current_setting('app.tenant_id', true) renvoie NULL si non défini (2e arg = missing_ok).
--    NULL::bigint <> tenant_id => NULL => la ligne n'est PAS visible (fail-closed).
CREATE POLICY tenant_isolation ON documents
  FOR ALL
  USING  (tenant_id = current_setting('app.tenant_id', true)::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::bigint);
```
Le second argument `true` (`missing_ok`) de `current_setting` fait renvoyer NULL si le paramètre n'est pas défini au lieu de lever une erreur ; comme `tenant_id = NULL` s'évalue à NULL (donc non-vrai), aucune ligne n'est visible tant que le GUC n'est pas positionné — c'est le comportement fail-closed souhaité.

### 2. Coût de performance du RLS et sémantique des hooks pgxpool

**Surcoût mesuré (littérature)**
- pgDash (« Exploring Row Level Security In PostgreSQL ») est le repère le plus honnête : « Enabling RLS and setting policies has a measurable overhead. Exactly how much though, is impossible to state generically », et ajoute : « If you are doing a non-trivial RLS implementation, plan for measuring and quantifying the impact of the overhead in your production deployment. »
- Plusieurs retours d'expérience : avec index composite menant par `tenant_id` et policy en égalité, le surcoût est « noise. 2-4% on typical queries » (retour Ktor/PG16, 10M lignes/500 tenants) ; un autre affirme « virtually zero measurable overhead (0.02ms) » quand les policies sont enveloppées et indexées.
- Cas de dégradation : sans index, la policy devient un filtre séquentiel par ligne. Les policies contenant une fonction volatile non enveloppée sont ré-évaluées par ligne ; l'astuce `(SELECT auth.uid())` force une évaluation unique (InitPlan).

**Le rôle des fonctions `leakproof`** (fondamental). Doc officielle (§5.9) : l'expression de policy « will be evaluated for each row prior to any conditions or functions coming from the user's query. (The only exceptions to this rule are `leakproof` functions … the optimizer may choose to apply such functions ahead of the row-security check.) » Conséquence pratique : un prédicat utilisateur non-leakproof (ex. `LIKE`, opérateur custom) ne peut pas être poussé avant la qual RLS ; le plan devient plus lent qu'une table sans RLS. Le commit `215b43c` (« Improve RLS planning by marking individual quals with security levels ») a remplacé la vieille implémentation par sous-requête security-barrier (« results in very inefficient plans ») par un marquage de `security_level` par qual, les quals leakproof pouvant passer devant. `current_setting()` étant STABLE, sa valeur est fixée pour la durée de l'instruction et le planificateur peut l'utiliser comme clé d'index scan.

**`set_config(...,false)` à l'acquisition vs `SET LOCAL` en transaction**
- `SELECT set_config('app.tenant_id', $1, false)` à chaque acquisition de connexion = **un aller-retour réseau supplémentaire** dédié, et le paramètre persiste au niveau *session* (3e argument `false`). Il faut donc le `RESET` à la libération — sinon fuite du contexte vers la requête suivante sur la même connexion.
- `SET LOCAL app.tenant_id = …` (ou `set_config('app.tenant_id', $1, true)`) est **transaction-scoped** : PostgreSQL le rejette à `COMMIT`/`ROLLBACK`, donc il ne peut pas fuir. myDBA.dev le documente précisément : « SET LOCAL values are discarded at COMMIT or ROLLBACK, so they can't leak into the next transaction on a shared connection. » C'est le motif de sécurité recommandé, surtout derrière un pooler transaction-mode (PgBouncer).

**Sémantique EXACTE des hooks pgxpool v5** (struct `pgxpool.Config`, source `pgxpool/pool.go`)
- `BeforeConnect func(context.Context, *pgx.ConnConfig) error` — avant création de connexion.
- `AfterConnect func(context.Context, *pgx.Conn) error` — après établissement, avant ajout au pool (bon endroit pour `LoadType`, préparation d'objets stables par connexion).
- `BeforeAcquire func(context.Context, *pgx.Conn) bool` — **DÉPRÉCIÉ**. Commentaire verbatim : « It must return true to allow the acquisition or false to indicate that the connection should be destroyed and a different connection should be acquired. Deprecated: Use PrepareConn instead. If both PrepareConn and BeforeAcquire are set, PrepareConn will take precedence, ignoring BeforeAcquire. »
- `PrepareConn func(context.Context, *pgx.Conn) (bool, error)` — **introduit en pgx v5.7.6, release du 8 septembre 2025** (confirmé par la page Tags officielle jackc/pgx : « Release v5.7.6 · Sep 8, 2025 · a2fca03 »), PR #2329 (Jonathan Hall/flimzy), mergée le 31 mai 2025 (commit 4015a0c). C'est le remplaçant de `BeforeAcquire`. Il reçoit un `context.Context` ET le `*pgx.Conn`, **peut donc faire un aller-retour réseau** (`conn.Exec(ctx, "SELECT set_config('app.tenant_id',$1,false)")`) **et remonter une erreur** au caller — ce que `BeforeAcquire` ne pouvait pas. Les 4 cas de retour (commentaire verbatim) :
  - `true, nil` → la requête procède normalement ;
  - `true, err` → la connexion retourne au pool, la requête déclenchante échoue avec l'erreur ;
  - `false, err` → la connexion est détruite, la requête échoue avec l'erreur ;
  - `false, nil` → la connexion est détruite, la requête est réessayée sur une nouvelle connexion.
- `AfterRelease func(*pgx.Conn) bool` — après release, avant retour au pool ; **ne reçoit PAS de context** (« It must return true to return the connection to the pool or false to destroy the connection. »). Les appels réseau (ex. `conn.Exec(context.Background(), "RESET ALL")`) utilisent donc `context.Background()`. Il s'exécute de façon asynchrone dans une goroutine. Dans la discussion #1666, jackc explique : « AfterRelease doesn't take a context because it is called asynchronously after Release which itself cannot block. »
- `BeforeClose func(*pgx.Conn)` — juste avant fermeture/retrait du pool ; pas de context non plus (le destructeur du pool appelle `conn.Close` avec un `context.WithTimeout(context.Background(), 15*time.Second)`).

**Recommandation (combinaison la moins coûteuse et la plus sûre)**
Ne pas positionner le tenant dans `BeforeAcquire`/`PrepareConn` avec un `set_config(...,false)` + `RESET ALL` à la release. Préférez : ouvrir une transaction par requête et poser `SET LOCAL` juste après `BEGIN`. Cela (a) évite un round-trip séparé — le `SET LOCAL` peut être pipeliné/regroupé avec la première requête ; (b) est intrinsèquement fail-safe (auto-reset à la fin de la transaction) ; (c) n'exige aucun `DISCARD ALL`/`RESET ALL` qui casserait le cache de prepared statements.

```go
func WithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID int64,
    fn func(pgx.Tx) error) error {
    tx, err := pool.Begin(ctx)
    if err != nil { return err }
    defer tx.Rollback(ctx) // no-op après Commit
    // set_config(...,true) == SET LOCAL, mais paramétrable proprement
    if _, err = tx.Exec(ctx,
        "SELECT set_config('app.tenant_id', $1, true)",
        strconv.FormatInt(tenantID, 10)); err != nil {
        return err
    }
    if err = fn(tx); err != nil { return err }
    return tx.Commit(ctx)
}
```

**Impact de `RESET`/`DISCARD ALL` sur le cache de prepared statements de pgx**
Par défaut, pgx v5 utilise `QueryExecModeCacheStatement` : protocole étendu, prepare + cache automatique, exécution en un aller-retour une fois le statement caché. `DISCARD ALL` exécute (doc officielle) la séquence `CLOSE ALL; SET SESSION AUTHORIZATION DEFAULT; RESET ALL; DEALLOCATE ALL; UNLISTEN *; SELECT pg_advisory_unlock_all(); DISCARD PLANS; DISCARD TEMP; DISCARD SEQUENCES;`. Le `DEALLOCATE ALL` **détruit les prepared statements côté serveur** alors que pgx croit encore les avoir en cache → erreurs « prepared statement does not exist (SQLSTATE 26000) » (issue pgx #2442 : « pgx assumed it was still prepared because of the cache … the query is forever tainted »). **Différence clé** : `RESET ALL` ne réinitialise que les paramètres GUC de session (ce qui suffit pour effacer un `app.tenant_id` posé en session), tandis que `DISCARD ALL` fait tout cela **plus** le `DEALLOCATE ALL` destructeur. Donc, si vous devez nettoyer une connexion à la release, **utilisez `RESET ALL` (ou juste `RESET app.tenant_id`), jamais `DISCARD ALL`**, pour préserver le cache. Mieux encore : n'ayez rien à nettoyer en utilisant `SET LOCAL`. Note pooler : derrière PgBouncer en transaction-mode, `QueryExecModeCacheStatement` casse (« prepared statement already exists ») sauf PgBouncer ≥ 1.21 avec `max_prepared_statements` ; sinon utilisez `QueryExecModeSimpleProtocol` (0 prepared statement, mais 1 round-trip) ou `QueryExecModeCacheDescribe` (le plus sûr par défaut derrière pooler d'après les analyses).

**Pool par tenant vs pool partagé**
- *Pool partagé* (recommandé par défaut) : un seul pool, tenant posé par `SET LOCAL` par transaction. Mémoire et nombre de connexions maîtrisés, mais isolation logique seulement (dépend de la correction des policies).
- *Pool par tenant* : isolation forte (une connexion ne sert qu'un tenant), diagnostic simple, mais **coût mémoire et risque d'épuisement** : chaque pool réserve des connexions (défaut pgx `MaxConns = max(4, NumCPU)`), donc N tenants × M connexions peut dépasser `max_connections` de PostgreSQL. Réservez le pool-par-tenant à un petit nombre de très gros tenants (« noisy neighbor »), sinon pool partagé + RLS.

### 3. Migrations concurrentes

**`pg_advisory_lock` : choix de la clé et de la portée**
Les fonctions acceptent soit une clé `bigint` unique, soit deux `integer` (espaces disjoints). Doc officielle : « pg_advisory_lock locks an application-defined resource, which can be identified either by a single 64-bit key value or two 32-bit key values ». Pour dériver une clé stable, `hashtext('mon-app-migrations')` est le motif courant (« The advisory lock function requires an int … the use of hashtext is often useful »). Utilisez la **forme deux-entiers** avec un namespace fixe pour éviter les collisions entre sous-systèmes (ex. `pg_advisory_xact_lock(hashtext('myapp'), 1)`).

Portée :
- `pg_advisory_lock` = **session** : doit être libéré explicitement (`pg_advisory_unlock`) ou survit jusqu'à déconnexion → **risque de verrou orphelin** en cas de crash.
- `pg_advisory_xact_lock` = **transaction** : libéré automatiquement à `COMMIT`/`ROLLBACK`, **impossible à libérer manuellement** — « a crash or an unhandled error never leaks a lock ». **C'est le choix recommandé pour les migrations.**
- `pg_try_advisory_lock` / `pg_try_advisory_xact_lock` renvoient un booléen immédiatement (pas d'attente) → combinez avec `SET lock_timeout` ou une boucle de retry avec back-off.

Motif recommandé :
```sql
BEGIN;
SELECT pg_advisory_xact_lock(hashtext('myapp_schema_migrations'));
-- appliquer les migrations en attente ici
COMMIT; -- le verrou est relâché automatiquement
```

**Comparaison des outils**

| Outil | Verrou pris | Type | Notes / problèmes connus |
|---|---|---|---|
| **golang-migrate** | oui | `pg_advisory_lock` (session) | « dirty flag » posé avant chaque migration ; en cas d'échec, l'état reste « dirty » et exige un `force` manuel. Issue #581 : une erreur dans un `BEGIN/COMMIT` explicite laisse la transaction avortée → l'ordre de libération du verrou échoue. Wrapping transactionnel par migration non natif (issue #641, `TransactionsEnabled` proposé). Deadlock documenté avec `CREATE INDEX CONCURRENTLY` en parallèle (issue #960). |
| **tern** (jackc, auteur de pgx) | oui | verrouillage pour protéger contre migrateurs concurrents | Migrations enveloppées dans une transaction **par défaut** ; magic comment pour désactiver la transaction (nécessaire pour `create index concurrently`). |
| **goose** | oui (récent) / verrou applicatif | selon version | `-- +goose NO TRANSACTION` pour sortir de la transaction ; erreur classique #292 « CREATE INDEX CONCURRENTLY cannot run inside a transaction block ». Extensions tierces (`gooseplus`) ajoutent une table de verrou globale pour déploiements parallèles. |
| **atlas** | oui | verrou pour garantir une seule migration à la fois | « Atlas acquires a lock ensuring that only one migration happens at a time » ; supporte l'exécution en transaction, linting (`migrate lint`) des opérations destructrices. |
| **dbmate** | oui | verrou | Léger, agnostique ; gère explicitement le cas de N instances démarrant simultanément (une acquiert, les autres attendent puis constatent « no migration to apply »). |

**DDL transactionnel et `CREATE INDEX CONCURRENTLY`**
PostgreSQL supporte le DDL transactionnel : envelopper chaque migration dans une transaction garantit un rollback propre en cas d'échec (pas d'état partiel). **Exception majeure** : `CREATE INDEX CONCURRENTLY` (et `DROP INDEX CONCURRENTLY`) **ne peut pas s'exécuter dans un bloc transactionnel** (`ERROR: CREATE INDEX CONCURRENTLY cannot run inside a transaction block`). Pire, en migrations concurrentes il y a un **deadlock documenté** (flyway #1654, golang-migrate #960, postgres-migrations #36) : l'instance A détient le verrou et lance le `CREATE INDEX CONCURRENTLY`, qui doit attendre les transactions en cours ; l'instance B attend le verrou → interblocage. Solutions : (1) annoter la migration `--no-transaction` / `-- +goose NO TRANSACTION` / magic comment tern ; (2) idéalement, exécuter ces migrations en dehors de la fenêtre de démarrage concurrent, ou s'assurer qu'une seule instance les lance ; (3) prévoir un retry avec `DROP INDEX CONCURRENTLY IF EXISTS` pour nettoyer un index laissé `INVALID` par un échec.

**Version de schéma : une fois en fin de chaîne vs après chaque migration**
Posez la version **après chaque migration**, dans la même transaction que la migration (quand c'est transactionnel), pour que l'échec d'une migration ne fasse pas mentir la version. Poser la version une seule fois en fin de chaîne fait qu'un échec au milieu laisse la base dans un état où la version enregistrée ne reflète pas les migrations réellement appliquées — impossible de reprendre proprement. Pour les migrations non transactionnelles (`CONCURRENTLY`), le bookkeeping de version doit être fait avec soin (idempotence, vérification d'index `INVALID`).

### 4. Clés étrangères composites `(tenant_id, id)`

**Principe et coût**
La FK composite force la ligne enfant à référencer une ligne parent **du même tenant** :
```sql
CREATE TABLE projects (
  tenant_id BIGINT NOT NULL,
  id        BIGINT GENERATED ALWAYS AS IDENTITY,
  PRIMARY KEY (tenant_id, id)         -- ou UNIQUE (tenant_id, id)
);
CREATE TABLE tasks (
  tenant_id  BIGINT NOT NULL,
  id         BIGINT GENERATED ALWAYS AS IDENTITY,
  project_id BIGINT NOT NULL,
  PRIMARY KEY (tenant_id, id),
  FOREIGN KEY (tenant_id, project_id)
    REFERENCES projects (tenant_id, id)  -- exige UNIQUE(tenant_id,id) côté projects
);
```
La contrainte référencée **doit être UNIQUE** : le standard SQL exige qu'une FK référence une contrainte unique (rappelé par Laurenz Albe sur pgsql-hackers). Coût : une FK composite implique un index côté référencé (déjà là si `PK (tenant_id, id)`) et l'index de support côté référençant ; les clés plus larges augmentent légèrement la taille des index et le coût d'écriture. Bénéfice : **défense en profondeur** — même en cas de bug applicatif ou de policy RLS mal écrite, la base refuse une référence inter-tenant. Bonus : tous les index menant par `tenant_id` aident les requêtes tenant-local et facilitent un futur sharding (thread pgsql-hackers, Paul Martinez). Piège documenté : `ON DELETE SET NULL` ne fonctionne pas proprement avec une clé composite partagée (il tenterait de mettre NULL aussi le `tenant_id`). Alternative/complément : un `CHECK` de cohérence, ou garder `id` globalement unique en plus.

**FK et RLS : le canal auxiliaire (point critique)**
Documentation officielle (§5.9) : « Referential integrity checks, such as unique or primary key constraints and foreign key references, always bypass row security to ensure that data integrity is maintained. Care must be taken when developing schemas and row level policies to avoid "covert channel" leaks of information through such referential integrity checks. » Concrètement : une violation d'unicité ou de FK peut **révéler l'existence d'une ligne invisible** au tenant courant, via le message d'erreur. La doc ajoute que `TRUNCATE` et `REFERENCES` ne sont pas soumis à RLS. Mitigations : (1) rendre les valeurs uniques non devinables ; (2) messages d'erreur applicatifs génériques (ne pas propager le détail PostgreSQL au client) ; (3) considérer que RLS filtre les *lectures*, pas les *vérifications de contraintes*.

**`FORCE ROW LEVEL SECURITY`**
Par défaut, **le propriétaire de la table contourne RLS** (doc §5.9 : « the table's owner is typically not subject to row security policies »). Si votre application se connecte comme propriétaire, « you've enabled RLS that does nothing ». Deux règles : (a) l'application se connecte avec un rôle **non-propriétaire, sans `BYPASSRLS`, non-superuser** ; (b) activez `ALTER TABLE t FORCE ROW LEVEL SECURITY` pour que même le propriétaire soit soumis aux policies. Le motif de rôles recommandé (QueryPlane) : `app_owner` (migrations, DDL) distinct de `app_user` (requêtes, aucun DDL, soumis à RLS).

### 5. Sauvegarde et restauration par tenant

**Extraction d'un tenant**
- **pg_dump ne filtre pas par ligne**. Il n'a que `--table`, `--exclude-table`, `--exclude-table-data`. Le patch `--include-table-data-where=table:filter` (Carter Thaxton, 2018) a été « Returned with feedback » en 2019 et n'a pas été intégré. La réponse récurrente de la communauté (David G. Johnston, Jeremy Finzel sur pgsql-hackers) : utiliser `\copy (SELECT … WHERE …) TO`.
- **`COPY (SELECT … WHERE tenant_id = $1) TO`** est l'outil de base, par table :
```sql
\copy (SELECT * FROM documents WHERE tenant_id = 42) TO 'documents_t42.csv' CSV HEADER
```
  À noter (myDBA.dev) : `COPY TO` applique les policies SELECT ; assurez-vous que le rôle qui exporte voit bien toutes les lignes du tenant (ou faites l'export avec un rôle `BYPASSRLS` dédié + `WHERE tenant_id`). `COPY FROM` n'est pas supporté sur tables protégées par RLS → réimport via `INSERT`.
- **pg_dump defaults `row_security = off`** et échoue si le rôle ne peut pas contourner RLS ; `--enable-row-security` donne un dump *filtré/partiel*. Pour un dump complet, le rôle doit avoir `BYPASSRLS`.
- **Schéma par tenant** : si vous isolez par schéma, `pg_dump --schema=tenant_42` extrait proprement un tenant (dépendances résolues). C'est le seul cas où pg_dump fait le travail nativement.
- **Réplication logique / logical decoding** : pour extraire/reconstruire un tenant sans figer tout le cluster (ClickHouse : « logical decoding/WAL-based reconstruction »).

**Restauration d'un tenant**
Réimport via `INSERT`/`COPY FROM` avec **remapping des identifiants** si les `id` entrent en conflit (surtout avec des séquentiels globaux). Deux stratégies : (a) restaurer dans une base de staging puis `INSERT … SELECT` avec réécriture des clés ; (b) si les PK sont `(tenant_id, id)`, un tenant peut être réinséré tel quel dans une base vide de ce tenant. Gérez l'ordre des FK (parents avant enfants) ou différez les contraintes.

**PITR et pourquoi il ne restaure pas un tenant**
Le PITR (restauration WAL à un instant) opère au niveau **cluster physique** : il restaure toute l'instance à un point dans le temps, pas une ligne ni un tenant. Pour « restaurer un seul tenant à hier », il faut : restaurer un **cluster séparé** au point voulu, puis **extraire** le tenant avec `COPY (SELECT … WHERE tenant_id=…)`, puis réimporter dans la base de production. ClickHouse le confirme : « physical PITR restores the whole cluster … You typically need logical techniques … to replay or extract a single tenant. »

**Tester l'isolation (indispensable)**
1. **Tests d'accès croisé automatisés** — le tenant A ne doit jamais voir B. En pgTAP dans la CI :
```sql
BEGIN;
SELECT plan(2);
SET LOCAL app.tenant_id = '1';
SELECT is((SELECT count(*) FROM documents WHERE tenant_id = 2), 0::bigint,
          'tenant 1 ne voit pas tenant 2');
SET LOCAL app.tenant_id = '2';
SELECT is((SELECT count(*) FROM documents), 5::bigint, 'tenant 2 voit ses lignes');
SELECT finish();
ROLLBACK;
```
   Exécuté via `pg_prove` dans GitHub Actions ; un échec de policy casse le build avant la prod.
2. **Fuzzing de l'en-tête `X-Tenant-ID`** — cas à couvrir : absent, vide, non numérique, `0`, négatif, très grand (> `int64`, débordement), espaces, tentative d'injection SQL (`1; DROP TABLE`), en-têtes dupliqués (`r.Header.Values` doit renvoyer exactement 1), Unicode/`+0`/`-0`. Chaque cas doit produire un **rejet fail-closed** (HTTP 400), jamais un `tenant_id = 0` ni un accès.
3. **Tests de propriété** : pour tout couple (A, B) et toute opération CRUD, A n'affecte jamais les lignes de B.
4. **Audit des tables sans RLS** (à faire tourner en CI) :
```sql
-- Tables du schéma applicatif sans RLS activé
SELECT n.nspname, c.relname, c.relrowsecurity, c.relforcerowsecurity
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind = 'r'
  AND (c.relrowsecurity = false OR c.relforcerowsecurity = false);
-- Tables avec RLS activé mais AUCUNE policy (deny-all silencieux, ou oubli)
SELECT c.relname
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname='public' AND c.relkind='r' AND c.relrowsecurity
  AND NOT EXISTS (SELECT 1 FROM pg_policies p
                  WHERE p.schemaname='public' AND p.tablename=c.relname);
```
   Toute table applicative retournée est un bloqueur de release. `relrowsecurity` = RLS activé, `relforcerowsecurity` = FORCE actif ; `pg_policies` liste les policies.
5. **Vérifier le rôle applicatif** (ni superuser ni BYPASSRLS) :
```sql
SELECT rolname, rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user;
-- rolsuper et rolbypassrls DOIVENT être false pour le rôle applicatif
```

### Protocole de mesure du surcoût RLS

**Principe** : comparer strictement *avec* et *sans* RLS, toutes autres variables égales, sur données réalistes, en isolant la variable RLS.

1. **Isoler la variable** : deux configurations identiques (même schéma, mêmes index menant par `tenant_id`, mêmes volumes, même `shared_buffers`/`work_mem`) — l'une avec `ENABLE ROW LEVEL SECURITY` + policies + `FORCE`, l'autre sans. Testez en tant que **rôle applicatif réel** : « The plan you get as a superuser bypasses RLS entirely and tells you nothing. »
2. **Volumes réalistes** : ex. 10–50M lignes réparties sur des centaines/milliers de tenants, avec skew réaliste (gros et petits tenants).
3. **EXPLAIN (ANALYZE, BUFFERS)** connecté comme rôle applicatif, `SET LOCAL app.tenant_id`, sur les requêtes chaudes. Cherchez : le filtre RLS pousse-t-il un `Index Scan` sur `(tenant_id, …)` ou tombe-t-il en `Seq Scan` ? Le prédicat RLS est-il appliqué tôt (bien) ou tard/après un JOIN (mauvais — souvent un prédicat non-leakproof) ?
4. **pgbench avec scripts personnalisés** :
```sql
-- rls.sql
\set tid random(1, 1000)
BEGIN;
SELECT set_config('app.tenant_id', :tid::text, true);
SELECT abalance FROM accounts WHERE aid = (:tid * 1000 + random(1,1000));
COMMIT;
```
   Lancez `pgbench -n -c <clients> -j <threads> -T 60 -f rls.sql -P 5 --report-per-command`, avec les mêmes paramètres sur la version sans RLS. `-P 5` donne tps + latence moyenne + écart-type par tranche ; `--report-per-command` isole la latence par commande.
5. **Répétitions et percentiles** : au moins 3–5 runs, comparez les **médianes** (pas le meilleur cas) ; capturez p50/p95/p99. pgbench fournit moyenne + écart-type ; pour de vrais percentiles, utilisez `--log` (log par transaction) puis calculez p50/p95/p99 hors ligne, ou `pg_stat_monitor` (histogrammes). `pg_stat_statements` ne fournit que moyenne + stddev (les percentiles s'*approximent* via mean/stddev — méthode « statistically-dodgy-but-practically-useful » de pgMustard, à réserver au dégrossissage).
6. **Métriques à collecter** : tps ; latence p50/p95/p99 ; `EXPLAIN ANALYZE` (temps de planification vs exécution, buffers hit/read) ; type de scan (index vs seq) ; latence d'acquisition de connexion (`pgxpool.Stat().AcquireDuration()` / temps `Acquire`) — pour chiffrer le round-trip d'un éventuel `set_config` à l'acquisition ; CPU serveur.
7. **Interprétation** : un surcoût de quelques % avec plans en `Index Scan` = coût attendu et acceptable. Un `Seq Scan` inattendu ou un prédicat RLS appliqué après un JOIN = index manquant ou fonction non-leakproof → corrigez (index menant par `tenant_id`, marquer les helpers `LEAKPROOF` si sûr, envelopper les fonctions volatiles). Comparez toujours la même requête sur les deux configurations, pas des requêtes différentes.

## Recommendations

**Étape 1 — Fondations d'isolation (avant tout trafic multi-tenant)**
- Table `tenants(id BIGINT IDENTITY PK, code TEXT UNIQUE, …)` ; `tenant_id BIGINT NOT NULL` + `CHECK (tenant_id > 0)` sur chaque table applicative ; index menant par `tenant_id`.
- Middleware Go fail-closed pour `X-Tenant-ID` (validation stricte, rejet 400 sur absent/vide/dupliqué/≤0/non numérique ; `ParseInt(...,64)` ; jamais `_` sur l'erreur ; ne pas renvoyer le message brut).
- Rôle applicatif dédié **sans** `BYPASSRLS`, **non**-superuser, **non**-propriétaire ; `ENABLE` + `FORCE ROW LEVEL SECURITY` sur toutes les tables ; policy `USING/WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::bigint)`.

**Étape 2 — Contexte tenant par transaction (le motif sûr)**
- `BEGIN; SELECT set_config('app.tenant_id',$1,true); … ; COMMIT` (helper `WithTenant`). N'utilisez PAS `set_config(...,false)` + `RESET`/`DISCARD ALL` à la release (round-trip inutile + invalidation du cache de prepared statements).
- Si vous devez absolument poser le tenant à l'acquisition, utilisez `PrepareConn` (pgx ≥ v5.7.6) et **ne nettoyez qu'avec `RESET app.tenant_id`**, jamais `DISCARD ALL`.
- Derrière PgBouncer transaction-mode : `QueryExecModeCacheDescribe` par défaut, ou `CacheStatement` seulement avec PgBouncer ≥ 1.21 + `max_prepared_statements`.

**Étape 3 — Migrations robustes**
- Adoptez tern (cohérent avec pgx) ou atlas ; sinon golang-migrate en connaissant ses limites (dirty state, verrou session). Enveloppez le passage de migrations dans `pg_advisory_xact_lock(hashtext('myapp'), 1)`. Sortez `CREATE INDEX CONCURRENTLY` de la transaction (annotation dédiée) et évitez de le lancer depuis plusieurs instances simultanément. Posez la version après chaque migration, dans sa transaction.

**Étape 4 — Défense en profondeur & données**
- FK composites `(tenant_id, id)` avec `UNIQUE (tenant_id, id)` référencé, là où l'intégrité inter-entités importe. Messages d'erreur applicatifs génériques (canal auxiliaire FK/unicité).
- Sauvegarde par tenant via `COPY (SELECT … WHERE tenant_id=$1) TO` (rôle `BYPASSRLS` dédié) ou schéma-par-tenant + `pg_dump --schema`. Documentez que le PITR = cluster entier ; procédure de restauration tenant = cluster de staging + extraction.

**Étape 5 — Tests et CI (bloquants)**
- pgTAP d'accès croisé + fuzzing d'en-têtes + audit `pg_class`/`pg_policies` (tables sans RLS/sans policy) + vérification `NOT rolsuper AND NOT rolbypassrls` — tous exécutés en CI, échec = release bloquée.
- Protocole de mesure RLS (EXPLAIN ANALYZE en rôle app, pgbench custom, p50/p95/p99, médianes sur 3–5 runs) avant chaque changement majeur de schéma/policy.

**Seuils qui changent les décisions**
- Surcoût RLS > ~10–15 % après indexation → chercher un `Seq Scan`/prédicat non-leakproof ; envisager fonction helper `LEAKPROOF` ou réécriture de policy.
- Épuisement de connexions avec pool-par-tenant → repasser au pool partagé + RLS.
- Gros tenant « noisy neighbor » dégradant les autres → isoler ce tenant (pool/instance/schéma dédié) tout en gardant RLS pour le reste.
- Besoin de génération d'ID distribuée / fusion multi-source → passer de BIGINT à UUIDv7 (PG ≥ 18 pour `uuidv7()` natif).

## Caveats
- **Chiffres de surcoût RLS très dépendants du contexte.** Les valeurs « 2–4 % », « 0,02 ms », « 0,3 ms de policy », « SET LOCAL < 0,1 ms » proviennent de billets d'ingénierie et de benchmarks tiers (retours Ktor/PG16, DevriQ/Supabase, DEV) avec des configurations spécifiques ; traitez-les comme des ordres de grandeur, pas des garanties. pgDash le dit explicitement : « impossible to state generically ». Mesurez dans votre environnement.
- **Sources primaires vs secondaires.** La sémantique des hooks pgxpool, `DISCARD ALL`/`RESET ALL`, le contournement RLS par les FK, `leakproof`, les advisory locks et pg_dump sont issus de la documentation PostgreSQL officielle, du code source jackc/pgx et des listes pgsql-hackers (sources primaires). Les comparaisons d'outils de migration et certains chiffres de benchmark viennent de blogs (secondaires) ; recoupez avant de vous engager.
- **Versions.** `uuidv7()` natif = **PostgreSQL 18, publié le 25 septembre 2025** (notes de version officielles E.6 : « Add UUID version 7 generation function uuidv7() (Andrey Borodin) … This UUID value is temporally sortable. ») ; `security_invoker` sur vues = PG ≥ 15 ; `PrepareConn` = pgx v5.7.6 (8 sept. 2025). Vérifiez vos versions exactes.
- **CVE et canaux auxiliaires.** RLS n'est pas à l'épreuve des canaux auxiliaires (statistiques du planificateur — CVE-2019-10130 ; messages d'erreur FK/unicité ; timing des fonctions non-leakproof). Gardez PostgreSQL patché et considérez RLS comme une **couche de défense en profondeur**, pas comme l'unique barrière : filtrez aussi côté application.
- **Le patch pg_dump `--include-table-data-where` n'existe pas en natif** ; ne comptez pas dessus. Vérifiez le comportement `row_security`/`--enable-row-security` de votre version de pg_dump avant d'automatiser des dumps par tenant.