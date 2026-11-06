package postgres

import (
	"context"
	"testing"
)

// Only a loader's transaction stages by COPY: the pool, which has CopyFrom too,
// must keep the arrays, or a server's import would scatter a stage over
// several connections.
func TestBulkStatementStagesOnlyInACopyTx(t *testing.T) {
	var db execer // nil: any query on it would panic
	q, args, err := bulkStatement(context.Background(), db, positionBulkInsertSQL, positionBulkInsertTypes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if q != positionBulkInsertSQL || len(args) != len(positionBulkInsertTypes) {
		t.Errorf("a plain execer must read the arrays: %d args", len(args))
	}
}
