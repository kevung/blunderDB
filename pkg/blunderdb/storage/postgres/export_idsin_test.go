package postgres

// IDsIn exposes the dialect helper to the external, build-tagged PG tests.
func IDsIn(expr string, ids []string) (string, []any) { return shared{}.IDsIn(expr, ids) }
