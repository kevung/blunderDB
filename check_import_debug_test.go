package main

import (
	"fmt"
	"testing"
)

func TestCheckErrorColumns(t *testing.T) {
	db := newTestDB(t)
	if _, err := db.ImportGnuBGMatch("testdata/test.sgf"); err != nil {
		t.Fatal("import:", err)
	}

	// Check which positions have cube analysis AND checker analysis
	rows, err := db.db.Query(`
		SELECT a.id, a.cube_error, a.best_move_equity_error, p.decision_type,
		       a.best_cube_action
		FROM analysis a
		JOIN position p ON p.id = a.position_id
		WHERE p.decision_type = 0
		LIMIT 30
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	nonZeroMove := 0
	zeroWithCubeAction := 0
	total := 0
	for rows.Next() {
		var id, cubeErr, moveErr int64
		var dt *int
		var cubeAction *string
		if err := rows.Scan(&id, &cubeErr, &moveErr, &dt, &cubeAction); err != nil {
			t.Fatal(err)
		}
		ca := ""
		if cubeAction != nil {
			ca = *cubeAction
		}
		fmt.Printf("id=%d dt=%v cube_error=%d move_error=%d best_cube_action=%q\n",
			id, dt, cubeErr, moveErr, ca)
		total++
		if moveErr != 0 {
			nonZeroMove++
		}
		if moveErr == 0 && ca != "" {
			zeroWithCubeAction++
		}
	}
	t.Logf("total=%d nonZeroMove=%d zeroMoveButHasCubeAction=%d", total, nonZeroMove, zeroWithCubeAction)
}
