package postgres

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestResolveQueryExecMode(t *testing.T) {
	cases := []struct {
		name, dsn, env string
		mode           pgx.QueryExecMode
		override       bool
		wantErr        bool
	}{
		{"default", "postgres://h/db", "", pgx.QueryExecModeCacheDescribe, true, false},
		{"env", "postgres://h/db", "exec", pgx.QueryExecModeExec, true, false},
		{"env statement", "postgres://h/db", "cache_statement", pgx.QueryExecModeCacheStatement, true, false},
		{"dsn wins url", "postgres://h/db?default_query_exec_mode=exec", "simple_protocol", 0, false, false},
		{"dsn wins keyword", "host=h default_query_exec_mode=exec", "", 0, false, false},
		{"invalid", "postgres://h/db", "bogus", 0, false, true},
	}
	for _, c := range cases {
		mode, ov, err := resolveQueryExecMode(c.dsn, c.env)
		if (err != nil) != c.wantErr || ov != c.override || (ov && mode != c.mode) {
			t.Errorf("%s: got (%v,%v,%v)", c.name, mode, ov, err)
		}
	}
}
