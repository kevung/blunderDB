package gui

import (
	"reflect"
	"sort"
	"testing"
)

// boundAppMethods is the exhaustive, explicit list of *App methods that are
// meant to be bound to the Wails frontend (run.go's Bind: []interface{}{app,
// ...}). This is a drift guard, not a completeness contract in the other
// direction (#241): it does not check that the frontend calls every one of
// these — internal/cli/parity_test.go's databaseParity map is the place
// that kind of cross-referencing already happens for *database.Database —
// but it does mean a newly exported App method cannot silently start being
// bound (and thus reachable from arbitrary frontend/webview code) without a
// deliberate update here, since TestBoundAppMethodsMatchExpected fails the
// moment reflection sees a name this list does not.
//
// Kept sorted so a diff against a future mismatch is easy to read.
var boundAppMethods = []string{
	"BearoffStatus",
	"CancelEvaluationAtRest",
	"CancelGammonNetBatch",
	"CheckForUpdate",
	"CollectImportableFiles",
	"CopyImageToClipboard",
	"DeleteFile",
	"EnsureBearoffTables",
	"GenerateBearoffTable",
	"CancelBearoffGeneration",
	"DeleteBearoffTable",
	"EvaluatePositionImmediate",
	"ExportIssuerIdentity",
	"GetIssuerIdentity",
	"ImportIssuerIdentity",
	"IsDirectory",
	"OpenBearoffFileDialog",
	"OpenDatabaseDialog",
	"OpenExportDatabaseDialog",
	"OpenExportMatDialog",
	"OpenImportDatabaseDialog",
	"OpenLogsFolder",
	"OpenPositionFilesDialog",
	"OpenPositionFolderDialog",
	"PathExists",
	"PickIdentityFile",
	"PrepareDemoDatabase",
	"ReadFileContent",
	"RegenerateIssuerIdentity",
	"SaveDatabaseDialog",
	"SetIssuerName",
	"ShowAlert",
	"ShowQuestionDialog",
	"StartEvaluationAtRest",
	"StartGammonNetBatch",
	"StartGammonNetStaleBatch",
	"StartupFilePath",
}

// TestBoundAppMethodsMatchExpected fails on ANY difference (added or
// removed) between *App's actual exported method set and boundAppMethods
// above (#241): the two "removed" cases from this ticket
// (OpenPositionDialog, OpenXGFileDialog — confirmed zero references anywhere
// in frontend/src, yet still exported and thus still bound) are exactly
// what this guard would have caught earlier, and it is what catches the
// next one.
func TestBoundAppMethodsMatchExpected(t *testing.T) {
	typ := reflect.TypeOf(&App{})
	var got []string
	for i := 0; i < typ.NumMethod(); i++ {
		got = append(got, typ.Method(i).Name)
	}
	sort.Strings(got)

	want := append([]string(nil), boundAppMethods...)
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("*App has %d exported methods, boundAppMethods lists %d — see the diff below", len(got), len(want))
	}
	gotSet := make(map[string]bool, len(got))
	for _, m := range got {
		gotSet[m] = true
	}
	wantSet := make(map[string]bool, len(want))
	for _, m := range want {
		wantSet[m] = true
	}
	for _, m := range got {
		if !wantSet[m] {
			t.Errorf("*App exports %s, which is not in boundAppMethods — add it there deliberately (or stop exporting it)", m)
		}
	}
	for _, m := range want {
		if !gotSet[m] {
			t.Errorf("boundAppMethods lists %s, but *App no longer exports it — remove it from the list", m)
		}
	}
}
