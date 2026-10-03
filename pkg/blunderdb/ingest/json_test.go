package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func TestJSONRoundTrip(t *testing.T) {
	ctx := context.Background()

	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	p := domain.InitializePosition()
	id, err := src.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.Comments().Add(ctx, "", id, "nice position"); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := (JSONExporter{S: src}).Export(ctx, "", &buf, ExportOptions{}); err != nil {
		t.Fatalf("export: %v", err)
	}
	if !strings.Contains(buf.String(), "nice position") {
		t.Fatalf("export missing comment:\n%s", buf.String())
	}

	dst, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()

	sum, err := (JSONImporter{S: dst}).Import(ctx, "", Source{Reader: &buf}, nil)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if sum.SavedPositions != 1 {
		t.Fatalf("SavedPositions = %d, want 1", sum.SavedPositions)
	}

	counts, err := dst.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if counts.Positions != 1 {
		t.Fatalf("dst positions = %d, want 1", counts.Positions)
	}

	text, err := dst.Comments().Text(ctx, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "nice position") {
		t.Fatalf("imported comment = %q, want to contain 'nice position'", text)
	}
}

func TestJSONImportCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	dst, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()

	body := `{"position":{"decisionType":0}}` + "\n"
	_, err = (JSONImporter{S: dst}).Import(ctx, "", Source{Reader: strings.NewReader(body)}, nil)
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
	// Nothing should have been committed.
	counts, _ := dst.Metadata().Counts(context.Background(), "")
	if counts.Positions != 0 {
		t.Fatalf("positions after cancelled import = %d, want 0", counts.Positions)
	}
}

// An NDJSON record lands on a position the target already holds: its
// rollouts stay, and its analysis is filled, not replaced.
func TestJSONImportKeepsRollouts(t *testing.T) {
	ctx := context.Background()
	dst, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	p := domain.InitializePosition()
	id, err := dst.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	held := &domain.PositionAnalysis{}
	held.AttachRollout(domain.RolloutAnalysis{Signature: "held", Games: 216})
	if err := dst.Analyses().Save(ctx, "", id, held); err != nil {
		t.Fatal(err)
	}

	pos := domain.InitializePosition()
	rec, _ := json.Marshal(positionBundle{Position: &pos, Analysis: &domain.PositionAnalysis{AnalysisType: "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Move: "8/5 6/5"}}}}})
	body := string(rec) + "\n"
	if _, err := (JSONImporter{S: dst}).Import(ctx, "", Source{Reader: strings.NewReader(body)}, nil); err != nil {
		t.Fatal(err)
	}
	got, err := dst.Analyses().Load(ctx, "", id)
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasRollout("held") || !got.HasPrimary() {
		t.Errorf("rollouts %+v, primary %v — want the held rollout beside the imported analysis", got.Rollouts, got.HasPrimary())
	}
}
