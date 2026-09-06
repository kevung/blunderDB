package database

import (
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// trashDB opens a fresh database for a trash test.
func trashDB(t *testing.T) *Database {
	t.Helper()
	db := NewDatabase()
	if err := db.SetupDatabase(filepath.Join(t.TempDir(), "trash.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	closeOnCleanup(t, db)
	return db
}

// La corbeille rend une position AVEC ce qui a cascadé d'elle : la rendre nue
// serait une restauration de nom seulement.
func TestTrashPosition_RendLAnalyseEtLesCommentaires(t *testing.T) {
	t.Parallel()
	db := trashDB(t)

	pos := InitializePosition()
	pos.Dice = [2]int{6, 5}
	id, err := db.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	if err := db.SaveAnalysis(id, PositionAnalysis{AnalysisType: "CheckerMove"}); err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}
	if err := db.SaveComment(id, "à revoir"); err != nil {
		t.Fatalf("SaveComment: %v", err)
	}

	trashID, err := db.TrashPosition(id)
	if err != nil {
		t.Fatalf("TrashPosition: %v", err)
	}
	if _, err := db.LoadPosition(int(id)); err == nil {
		t.Fatal("la position est toujours là après sa mise à la corbeille")
	}

	restored, err := db.RestoreFromTrash(trashID)
	if err != nil {
		t.Fatalf("RestoreFromTrash: %v", err)
	}
	if _, err := db.LoadPosition(int(restored)); err != nil {
		t.Fatalf("la position restaurée est introuvable : %v", err)
	}
	a, err := db.LoadAnalysis(restored)
	if err != nil || a == nil {
		t.Errorf("l'analyse n'est pas revenue avec la position (%v)", err)
	}
	comments, err := db.GetCommentsByPosition(restored)
	if err != nil {
		t.Fatalf("GetCommentsByPosition: %v", err)
	}
	if len(comments) != 1 || comments[0].Text != "à revoir" {
		t.Errorf("les commentaires ne sont pas revenus : %+v", comments)
	}

	// L'entrée disparaît de la corbeille une fois restaurée : la garder
	// permettrait de restaurer deux fois la même chose.
	if n, err := db.CountTrash(); err != nil || n != 0 {
		t.Errorf("après restauration : %d entrées (%v), attendu 0", n, err)
	}
}

// Restaurer passe par SavePosition, donc par la déduplication Zobrist : si la
// même position est revenue entre-temps, la restauration atterrit dessus au
// lieu d'en créer une seconde. C'est l'invariant qui fait le travail, pas une
// concession.
func TestRestore_LaDeduplicationDecideOuLaPositionAtterrit(t *testing.T) {
	t.Parallel()
	db := trashDB(t)

	pos := InitializePosition()
	pos.Dice = [2]int{3, 1}
	id, err := db.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	trashID, err := db.TrashPosition(id)
	if err != nil {
		t.Fatalf("TrashPosition: %v", err)
	}

	// La même position rentre par un autre chemin avant la restauration.
	again := InitializePosition()
	again.Dice = [2]int{3, 1}
	reimported, err := db.SavePosition(&again)
	if err != nil {
		t.Fatalf("SavePosition (réimport): %v", err)
	}

	restored, err := db.RestoreFromTrash(trashID)
	if err != nil {
		t.Fatalf("RestoreFromTrash: %v", err)
	}
	if restored != reimported {
		t.Errorf("la restauration a créé une seconde ligne (%d) au lieu de retrouver %d",
			restored, reimported)
	}
}

// Une collection revient avec sa liste ; les positions, elles, n'ont jamais
// bougé — une collection est une vue sur elles.
func TestTrashCollection_RendLaListe(t *testing.T) {
	t.Parallel()
	db := trashDB(t)

	var ids []int64
	for i := 0; i < 3; i++ {
		p := InitializePosition()
		p.Dice = [2]int{6, i + 1}
		id, err := db.SavePosition(&p)
		if err != nil {
			t.Fatalf("SavePosition: %v", err)
		}
		ids = append(ids, id)
	}
	colID, err := db.CreateCollection("À revoir", "")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	for _, id := range ids {
		if err := db.AddPositionToCollection(colID, id); err != nil {
			t.Fatalf("AddPositionToCollection: %v", err)
		}
	}

	trashID, err := db.TrashCollection(colID)
	if err != nil {
		t.Fatalf("TrashCollection: %v", err)
	}
	// Les positions survivent à la suppression de la collection.
	for _, id := range ids {
		if _, err := db.LoadPosition(int(id)); err != nil {
			t.Fatalf("supprimer une collection a emporté la position %d", id)
		}
	}

	restored, err := db.RestoreFromTrash(trashID)
	if err != nil {
		t.Fatalf("RestoreFromTrash: %v", err)
	}
	got, err := db.GetCollectionPositions(restored)
	if err != nil {
		t.Fatalf("GetCollectionPositions: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("la collection restaurée porte %d positions, attendu 3", len(got))
	}
}

// La corbeille se vide par `vacuum`, avec la rétention du domaine — jamais à
// l'ouverture.
func TestVacuum_PurgeLaCorbeilleSelonSonAge(t *testing.T) {
	t.Parallel()
	db := trashDB(t)

	pos := InitializePosition()
	id, err := db.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	if _, err := db.TrashPosition(id); err != nil {
		t.Fatalf("TrashPosition: %v", err)
	}

	res, err := db.Vacuum()
	if err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	// Supprimée à l'instant : la rétention la protège.
	if res.TrashPurged != 0 {
		t.Errorf("vacuum a purgé %d entrées supprimées à l'instant", res.TrashPurged)
	}
	if n, _ := db.CountTrash(); n != 1 {
		t.Errorf("après vacuum : %d entrées, attendu 1", n)
	}

	if _, err := db.EmptyTrash(0); err != nil {
		t.Fatalf("EmptyTrash: %v", err)
	}
	if n, _ := db.CountTrash(); n != 0 {
		t.Errorf("après vidage : %d entrées, attendu 0", n)
	}
	_ = domain.TrashRetentionDays
}
