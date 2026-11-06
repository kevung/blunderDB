package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// LoaderRole is the role of the offline bulk load of one tenant's library. It
// never holds BYPASSRLS: a restrictive policy confines it to the tenant named
// by LoaderTenantEnv on top of tenant_isolation, so setting app.tenant_id by
// hand reaches nothing else.
const LoaderRole = "blunderdb_loader"

// LoaderTenantEnv names the tenant the loader may write, an integer above 0,
// read by the server that installs the policy. Unset, the role gets nothing.
const LoaderTenantEnv = "BLUNDERDB_LOADER_TENANT"

const loaderPolicy = "loader_public_only"

func loaderPredicate(tenant int64) string { return "tenant_id = " + strconv.FormatInt(tenant, 10) }

var digits = regexp.MustCompile(`\d+`)

// loaderTenantFromEnv is 0 when the variable is unset or not a tenant.
func loaderTenantFromEnv() int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(os.Getenv(LoaderTenantEnv)), 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// loaderPrivileges are what an import does to a tenant table: it deletes too,
// the caches it invalidates and the journal it cuts. Never TRUNCATE, which no
// policy filters.
var loaderPrivileges = []string{"SELECT", "INSERT", "UPDATE", "DELETE"}

// loaderReads are infrastructure tables without a tenant that opening a
// database reads.
var loaderReads = []string{"metadata", "schema_migrations"}

// ApplyLoaderRole grants the loader role what an import writes and installs
// its policy for tenant on every tenant table, when an exploitant created the
// role and tenant is above 0; else it does nothing, the role reaching no
// table. A policy for another tenant is replaced. Idempotent, and silent once
// in place: a start takes no lock. Run by the tables' owner, after Migrate and
// ApplyRLS.
func (s *Storage) ApplyLoaderRole(ctx context.Context, tenant int64) error {
	if tenant <= 0 {
		return nil
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, LoaderRole).Scan(&exists); err != nil {
		return fmt.Errorf("postgres: loader role: %w", err)
	}
	if !exists {
		return nil
	}
	var stmts []string
	for _, t := range rlsTables {
		qual, found, err := s.loaderPolicyQual(ctx, t)
		if err != nil {
			return err
		}
		if !found || policyTenant(qual) != tenant {
			if found {
				stmts = append(stmts, `DROP POLICY `+loaderPolicy+` ON `+t)
			}
			stmts = append(stmts, `CREATE POLICY `+loaderPolicy+` ON `+t+` AS RESTRICTIVE FOR ALL TO `+LoaderRole+
				` USING (`+loaderPredicate(tenant)+`) WITH CHECK (`+loaderPredicate(tenant)+`)`)
		}
		// has_table_privilege with a list is true when one is held: ask for each.
		for _, priv := range loaderPrivileges {
			var granted bool
			if err := s.pool.QueryRow(ctx, `SELECT has_table_privilege($1, $2, $3)`, LoaderRole, t, priv).Scan(&granted); err != nil {
				return fmt.Errorf("postgres: loader role on %s: %w", t, err)
			}
			if !granted {
				stmts = append(stmts, `GRANT `+priv+` ON `+t+` TO `+LoaderRole)
			}
		}
	}
	for _, t := range loaderReads {
		var granted bool
		if err := s.pool.QueryRow(ctx, `SELECT has_table_privilege($1, $2, 'SELECT')`, LoaderRole, t).Scan(&granted); err != nil {
			return fmt.Errorf("postgres: loader role on %s: %w", t, err)
		}
		if !granted {
			stmts = append(stmts, `GRANT SELECT ON `+t+` TO `+LoaderRole)
		}
	}
	if len(stmts) == 0 {
		return nil
	}
	stmts = append(stmts, `GRANT USAGE ON ALL SEQUENCES IN SCHEMA `+pgx.Identifier{"public"}.Sanitize()+` TO `+LoaderRole)
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SET LOCAL lock_timeout = '`+rlsLockTimeout+`'`); err != nil {
			return err
		}
		for _, stmt := range stmts {
			if _, err := tx.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("postgres: loader role: %w", err)
			}
		}
		return nil
	})
}

// loaderPolicyQual reads the loader policy's USING expression on table.
func (s *Storage) loaderPolicyQual(ctx context.Context, table string) (qual string, found bool, err error) {
	var q *string
	err = s.pool.QueryRow(ctx, `SELECT qual FROM pg_policies WHERE schemaname = current_schema() AND tablename = $1 AND policyname = $2`,
		table, loaderPolicy).Scan(&q)
	if err == pgx.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("postgres: loader role on %s: %w", table, err)
	}
	if q == nil {
		return "", true, nil
	}
	return *q, true, nil
}

// policyTenant is the number a policy's expression compares tenant_id to, 0 if
// none.
func policyTenant(qual string) int64 {
	n, _ := strconv.ParseInt(digits.FindString(qual), 10, 64)
	return n
}

// ActsAsLoader says whether this connection's role is the loader or a member
// of it (a superuser is a member of every role, and is not counted): Open
// refuses to serve so.
func (s *Storage) ActsAsLoader(ctx context.Context) (bool, error) {
	var loader bool
	err := s.pool.QueryRow(ctx, `SELECT CASE WHEN NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1) THEN false
		WHEN current_user = $1 THEN true
		WHEN (SELECT rolsuper FROM pg_roles WHERE rolname = current_user) THEN false
		ELSE pg_has_role(current_user, $1, 'MEMBER') END`, LoaderRole).Scan(&loader)
	if err != nil {
		return false, fmt.Errorf("postgres: loader role check: %w", err)
	}
	return loader, nil
}

// OpenLoader opens dsn as the loader of tenant: under row-level security,
// without migrating (the server, owner of the schema, does that), and refuses a
// database whose schema is not this binary's, a role that is not the loader or
// whose policy is not for tenant. Writes go by COPY.
func OpenLoader(ctx context.Context, dsn string, tenant int64) (*Storage, error) {
	if tenant <= 0 {
		return nil, fmt.Errorf("postgres: the loader's tenant must be above 0, got %d", tenant)
	}
	s, err := openPool(ctx, dsn, &storage.Options{EnableRLS: true})
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*Storage, error) {
		s.pool.Close()
		return nil, err
	}
	loader, err := s.ActsAsLoader(ctx)
	if err != nil {
		return fail(err)
	}
	var super, bypass bool
	if err := s.pool.QueryRow(ctx, `SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&super, &bypass); err != nil {
		return fail(fmt.Errorf("postgres: loader role check: %w", err))
	}
	if !loader || super || bypass {
		return fail(fmt.Errorf("postgres: the connection's role is not %s, or escapes row-level security", LoaderRole))
	}
	qual, found, err := s.loaderPolicyQual(ctx, "position")
	if err != nil {
		return fail(err)
	}
	if !found || policyTenant(qual) != tenant {
		return fail(fmt.Errorf("postgres: the loader policy is not installed for tenant %d: restart the server with %s=%d", tenant, LoaderTenantEnv, tenant))
	}
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fail(fmt.Errorf("postgres: read migrations dir: %w", err))
	}
	var want []string
	for _, e := range entries {
		if n := e.Name(); strings.HasSuffix(n, ".sql") && !strings.HasPrefix(n, "001_") {
			want = append(want, strings.TrimSuffix(n, ".sql"))
		}
	}
	var missing int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM unnest($1::text[]) v
		WHERE NOT EXISTS (SELECT 1 FROM schema_migrations m WHERE m.version = v)`, want).Scan(&missing); err != nil {
		return fail(fmt.Errorf("postgres: loader schema check: %w", err))
	}
	if missing > 0 {
		return fail(fmt.Errorf("postgres: %d migrations missing: start the server once on this database first", missing))
	}
	s.copyWrites = true
	return s, nil
}
