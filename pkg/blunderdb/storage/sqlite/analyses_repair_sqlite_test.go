package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestRepairDenormalisedColumns exercises the repair on the very defect that
// made it necessary (kevung/blunderDB#115): analysis.cube_error left holding the
// error of an action that was never played.
//
// The column is corrupted in SQL rather than through the API, because the API
// is precisely what has been fixed — going through it could no longer produce
// the bad value, and the test would prove nothing.
func TestRepairDenormalisedColumns(t *testing.T) {
	ctx := context.Background()
	dsn := filepath.Join(t.TempDir(), "repair.db")
	s, err := sqlite.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	pos := domain.InitializePosition()
	pos.DecisionType = domain.CubeAction
	pos.Score = [2]int{3, 5}
	posID, err := s.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	// Le joueur n'a PAS doublé, et il a eu raison : son erreur est nulle.
	a := domain.PositionAnalysis{
		PlayedCubeActions: []string{"No Double"},
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			BestCubeAction:          "No Double",
			CubefulNoDoubleEquity:   0.40,
			CubefulDoubleTakeEquity: 0.20,
			CubefulDoublePassEquity: 1.00,
			CubefulNoDoubleError:    0,
			CubefulDoubleTakeError:  -0.200,
			CubefulDoublePassError:  -0.600,
		},
	}
	if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}

	// Une base saine ne bouge pas. C'est ce qui rend le compteur lisible :
	// une réparation qui réécrit tout à chaque passage ne distingue plus
	// « quelque chose n'allait pas » de « ça a tourné ».
	if n, err := s.Analyses().RepairDenormalisedColumns(ctx, ""); err != nil || n != 0 {
		t.Fatalf("base saine : %d lignes réparées (err %v), want 0", n, err)
	}

	// On abîme exactement comme le défaut le faisait : la colonne reçoit
	// l'erreur du double qui n'a jamais eu lieu (200 mp au lieu de 0).
	raw, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	if _, err := raw.ExecContext(ctx,
		`UPDATE analysis SET cube_error = 200 WHERE position_id = ?`, posID); err != nil {
		t.Fatalf("corrupt: %v", err)
	}
	raw.Close()

	n, err := s.Analyses().RepairDenormalisedColumns(ctx, "")
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}
	if n != 1 {
		t.Fatalf("réparation : %d lignes, want 1", n)
	}

	// La valeur est revenue à celle que dit le JSON, qui n'a jamais bougé —
	// c'est tout le principe : les colonnes sont une projection réparable.
	raw, _ = sql.Open("sqlite", dsn)
	var got int64
	if err := raw.QueryRowContext(ctx,
		`SELECT cube_error FROM analysis WHERE position_id = ?`, posID).Scan(&got); err != nil {
		t.Fatalf("relecture: %v", err)
	}
	raw.Close()
	if got != 0 {
		t.Errorf("cube_error = %d, want 0 (le joueur a eu raison de ne pas doubler)", got)
	}

	// Et une seconde passe ne trouve plus rien.
	if n, err := s.Analyses().RepairDenormalisedColumns(ctx, ""); err != nil || n != 0 {
		t.Errorf("seconde passe : %d lignes (err %v), want 0", n, err)
	}
}
