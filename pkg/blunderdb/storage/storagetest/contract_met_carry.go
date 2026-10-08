package storagetest

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/rollouts"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// clubSource is a small explicit gnubg table: a carried table is named by the
// digest of its source, so the source must parse.
const clubSource = `<met><info><name>Club</name><length>2</length></info>
<pre-crawford-table type="explicit"><row><me>0.5</me><me>0.7</me></row><row><me>0.3</me><me>0.5</me></row></pre-crawford-table>
<post-crawford-table player="both" type="explicit"><row><me>0.5</me><me>0.48</me></row></post-crawford-table></met>`

// clubMET is a table of a club, told apart from the built-in one by its digest.
var clubMET = domain.MatchEquityTable{Name: "Club", Digest: clubDigest(), Source: clubSource}

func clubDigest() string {
	m, err := engine.ParseGnubgMET([]byte(clubSource))
	if err != nil {
		panic(err)
	}
	return m.Digest()
}

// verdictBy is a checker analysis whose verdict comes from engineLabel.
func verdictBy(engineLabel string, created time.Time) *domain.PositionAnalysis {
	e := 0.05
	return &domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		CreationDate: created,
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Index: 0, Move: "13/10 6/5", Equity: 0.1, AnalysisEngine: engineLabel, AnalysisDepth: "2-ply"},
			{Index: 1, Move: "24/21 13/11", Equity: 0.05, EquityError: &e, AnalysisEngine: engineLabel, AnalysisDepth: "2-ply"},
		}},
	}
}

const gammonNetLabel = "gammonNet v1.2.1"

func ofAnalysis(t *testing.T, s storage.Storage, id int64) int64 {
	t.Helper()
	got, err := s.MatchEquityTables().OfAnalysis(context.Background(), "", id)
	if err != nil {
		t.Fatalf("OfAnalysis: %v", err)
	}
	return got
}

// testMETDroppedOnReplacement: an analysis's table describes a gammonNet
// verdict. Replaced by a verdict of another engine, by Save or by Merge, the
// analysis no longer names it; a gammonNet verdict saved with its table names
// it in the same write (ADR-0068).
func testMETDroppedOnReplacement(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	mt := s.MatchEquityTables()
	p := provenancePos(31)
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	club, err := mt.Save(ctx, "", clubMET)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	if err := rollouts.SaveValuedAnalysis(ctx, s, "", id, verdictBy(gammonNetLabel, now), club); err != nil {
		t.Fatalf("SaveValuedAnalysis: %v", err)
	}
	if got := ofAnalysis(t, s, id); got != club {
		t.Fatalf("after a gammonNet save with its table: %d, want %d", got, club)
	}
	if got, err := mt.OfAnalyses(ctx, "", []int64{id, id + 1000}); err != nil || len(got) != 1 || got[id] != club {
		t.Errorf("OfAnalyses = %v, %v; want only %d → %d", got, err, id, club)
	}
	if got, err := mt.Load(ctx, "", club); err != nil || got.Source != clubMET.Source || got.Digest != clubMET.Digest {
		t.Errorf("Load = %+v, %v; want the table with its source", got, err)
	}

	if err := s.Analyses().Save(ctx, "", id, verdictBy(gammonNetLabel, now.Add(time.Hour))); err != nil {
		t.Fatal(err)
	}
	if got := ofAnalysis(t, s, id); got != club {
		t.Errorf("after a gammonNet save without a table: %d, want %d kept until its writer tags it", got, club)
	}

	if err := s.Analyses().Save(ctx, "", id, verdictBy("XG", now)); err != nil {
		t.Fatal(err)
	}
	if got := ofAnalysis(t, s, id); got != 0 {
		t.Errorf("after an XG save: %d, want 0 (Kazaross-XG2)", got)
	}

	if err := rollouts.SaveValuedAnalysis(ctx, s, "", id, verdictBy(gammonNetLabel, now), club); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Analyses().Merge(ctx, "", id, nil, func(*domain.PositionAnalysis) *domain.PositionAnalysis {
		return verdictBy("GNUbg", now)
	}); err != nil {
		t.Fatal(err)
	}
	if got := ofAnalysis(t, s, id); got != 0 {
		t.Errorf("after a GNUbg merge: %d, want 0 (Kazaross-XG2)", got)
	}

	if _, err := mt.Load(ctx, "", club+1000); err == nil {
		t.Error("Load(unknown) succeeded, want ErrNotFound")
	}
}

// testMETTravelsWithImport: importing a database whose analyses cite a club
// table brings the table, merged by digest with one the receiver holds and
// not made current, and the analyses name the receiver's id for it
// (ADR-0068, rule 5).
func testMETTravelsWithImport(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "club.db")
	src, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	srcClub, err := src.MatchEquityTables().Save(ctx, "", clubMET)
	if err != nil {
		t.Fatal(err)
	}
	if err := src.MatchEquityTables().SetCurrent(ctx, "", srcClub); err != nil {
		t.Fatal(err)
	}
	positions := [3]domain.Position{provenancePos(41), provenancePos(42), provenancePos(43)}
	var srcIDs [3]int64
	for i := range positions {
		if srcIDs[i], err = src.Positions().Save(ctx, "", &positions[i]); err != nil {
			t.Fatal(err)
		}
	}
	// 41: tagged with the club table; 42: an untagged gammonNet verdict;
	// 43: tagged, merged into a receiver's rollout-only analysis.
	for i, met := range []int64{srcClub, 0, srcClub} {
		if err := rollouts.SaveValuedAnalysis(ctx, src, "", srcIDs[i], verdictBy(gammonNetLabel, now), met); err != nil {
			t.Fatal(err)
		}
	}
	_ = src.Close()

	// The receiver already holds the same table under its own name.
	mine := clubMET
	mine.Name = "Mine"
	held, err := s.MatchEquityTables().Save(ctx, "", mine)
	if err != nil {
		t.Fatal(err)
	}
	p43 := positions[2]
	target43, err := s.Positions().Save(ctx, "", &p43)
	if err != nil {
		t.Fatal(err)
	}
	if err := rollouts.Store(ctx, s, "", target43, movesRollout(rollout.Fast(), "13/10 6/5", 0.6)); err != nil {
		t.Fatal(err)
	}

	if _, err := (ingest.DBImporter{S: s}).Import(ctx, "", ingest.Source{Format: ingest.FormatNativeDB, Path: path}, nil); err != nil {
		t.Fatalf("Import: %v", err)
	}

	list, err := s.MatchEquityTables().List(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != held || list[0].Name != "Mine" || list[0].Current {
		t.Errorf("receiver's tables %+v, want only %q (id %d), not current", list, "Mine", held)
	}
	if cur, err := s.MatchEquityTables().Current(ctx, ""); err != nil || cur != nil {
		t.Errorf("receiver's current table %+v, %v; want the built-in one", cur, err)
	}
	for i, want := range []int64{held, 0, held} {
		p := positions[i]
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatal(err)
		}
		if got := ofAnalysis(t, s, id); got != want {
			t.Errorf("position %d: analysis names table %d, want %d", i+41, got, want)
		}
	}
}

// testMETTravelsWithExport: an exported file carries the tables its analyses
// cite, without the current flag, and nothing else (ADR-0068, rule 5).
func testMETTravelsWithExport(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	mt := s.MatchEquityTables()
	club, err := mt.Save(ctx, "", clubMET)
	if err != nil {
		t.Fatal(err)
	}
	unused := domain.MatchEquityTable{Name: "Unused", Digest: "unused-digest", Source: "<met>unused</met>"}
	if _, err := mt.Save(ctx, "", unused); err != nil {
		t.Fatal(err)
	}
	if err := mt.SetCurrent(ctx, "", club); err != nil {
		t.Fatal(err)
	}
	p := provenancePos(51)
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	if err := rollouts.SaveValuedAnalysis(ctx, s, "", id, verdictBy(gammonNetLabel, now), club); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "export.db")
	if _, err := ingest.ExportSQLite(ctx, s, "", path, ingest.WholeTenant(ingest.FormatSQLite)); err != nil {
		t.Fatalf("ExportSQLite: %v", err)
	}
	out, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	list, err := out.MatchEquityTables().List(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Digest != clubMET.Digest || list[0].Current {
		t.Fatalf("exported tables %+v, want only the cited club table, not current", list)
	}
	q := provenancePos(51)
	outID, err := out.Positions().Save(ctx, "", &q)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := out.MatchEquityTables().OfAnalysis(ctx, "", outID); err != nil || got != list[0].ID {
		t.Errorf("exported analysis names table %d, %v; want %d", got, err, list[0].ID)
	}
}

// testMETTravelsWithNDJSON: the NDJSON interchange carries the table each
// analysis cites, once, and the receiver stores it, not current, and tags
// the analysis with it — in both directions through s (ADR-0068, rule 5).
func testMETTravelsWithNDJSON(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	src, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "src.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	srcClub, err := src.MatchEquityTables().Save(ctx, "", clubMET)
	if err != nil {
		t.Fatal(err)
	}
	if err := src.MatchEquityTables().SetCurrent(ctx, "", srcClub); err != nil {
		t.Fatal(err)
	}
	positions := [3]domain.Position{provenancePos(61), provenancePos(62), provenancePos(63)}
	for i, met := range []int64{srcClub, 0, srcClub} {
		id, err := src.Positions().Save(ctx, "", &positions[i])
		if err != nil {
			t.Fatal(err)
		}
		if err := rollouts.SaveValuedAnalysis(ctx, src, "", id, verdictBy(gammonNetLabel, now), met); err != nil {
			t.Fatal(err)
		}
	}

	var stream bytes.Buffer
	if err := (ingest.JSONExporter{S: src}).Export(ctx, "", &stream, ingest.ExportOptions{}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	if n := strings.Count(stream.String(), `"matchEquityTable"`); n != 1 {
		t.Errorf("stream carries the table %d times, want once", n)
	}
	if _, err := (ingest.JSONImporter{S: s}).Import(ctx, "", ingest.Source{Reader: &stream}, nil); err != nil {
		t.Fatalf("Import: %v", err)
	}
	checkCarried := func(dst storage.Storage, where string) {
		t.Helper()
		list, err := dst.MatchEquityTables().List(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 || list[0].Digest != clubMET.Digest || list[0].Current {
			t.Fatalf("%s: tables %+v, want the club table, not current", where, list)
		}
		for i, want := range []int64{list[0].ID, 0, list[0].ID} {
			p := positions[i]
			id, err := dst.Positions().Save(ctx, "", &p)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := dst.MatchEquityTables().OfAnalysis(ctx, "", id); err != nil || got != want {
				t.Errorf("%s: position %d names table %d, %v; want %d", where, i+61, got, err, want)
			}
		}
	}
	checkCarried(s, "receiver")

	var back bytes.Buffer
	if err := (ingest.JSONExporter{S: s}).Export(ctx, "", &back, ingest.ExportOptions{}); err != nil {
		t.Fatalf("Export from s: %v", err)
	}
	out, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "out.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if _, err := (ingest.JSONImporter{S: out}).Import(ctx, "", ingest.Source{Reader: &back}, nil); err != nil {
		t.Fatalf("Import from s: %v", err)
	}
	checkCarried(out, "re-export")

	// A record citing a table the stream never carried is refused.
	orphan := `{"position":` + mustJSON(t, provenancePos(64)) + `,"analysis":` + mustJSON(t, verdictBy(gammonNetLabel, now)) + `,"met":99}` + "\n"
	if _, err := (ingest.JSONImporter{S: out}).Import(ctx, "", ingest.Source{Reader: strings.NewReader(orphan)}, nil); err == nil {
		t.Error("Import of a record citing an absent table succeeded, want an error")
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// testMETKeptByTrash: restoring a deleted position gives its analysis back
// with the table it was valued with.
func testMETKeptByTrash(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	club, err := s.MatchEquityTables().Save(ctx, "", clubMET)
	if err != nil {
		t.Fatal(err)
	}
	p := provenancePos(71)
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	if err := rollouts.SaveValuedAnalysis(ctx, s, "", id, verdictBy(gammonNetLabel, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)), club); err != nil {
		t.Fatal(err)
	}
	entry, err := trash.Position(ctx, s, "", id)
	if err != nil {
		t.Fatalf("trash.Position: %v", err)
	}
	res, err := trash.Restore(ctx, s, "", entry)
	restored := res.ID
	if err != nil {
		t.Fatalf("trash.Restore: %v", err)
	}
	if got := ofAnalysis(t, s, restored); got != club {
		t.Errorf("restored analysis names table %d, want %d", got, club)
	}
}
