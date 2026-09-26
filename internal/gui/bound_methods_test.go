package gui

import (
	"reflect"
	"sort"
	"testing"
)

// boundAppMethods is the explicit, sorted list of *App methods bound to the
// frontend, so a newly exported method cannot become reachable from the
// webview without a deliberate update here.
var boundAppMethods = []string{
	"BearoffStatus",
	"CancelCubeMatrix",
	"CancelEvaluationAtRest",
	"CancelGammonNetBatch",
	"CheckForUpdate",
	"CollectImportableFiles",
	"ComputeCubeMatrix",
	"CopyImageToClipboard",
	"DeleteFile",
	"EnsureBearoffTables",
	"GenerateBearoffQuestion",
	"GenerateEvaluationQuestion",
	"GenerateBearoffTable",
	"CancelBearoffGeneration",
	"PauseBearoffGeneration",
	"DiscardBearoffCheckpoint",
	"BearoffPlan",
	"DeleteBearoffTable",
	"EvaluatePositionImmediate",
	"ExportIssuerIdentity",
	"FolderWatchStatus",
	"GetIssuerIdentity",
	"ImportIssuerIdentity",
	"IsDirectory",
	"LegalMoves",
	"LooksLikeOGID",
	"ReadLogTail",
	"OpenBearoffFileDialog",
	"OpenDatabaseDialog",
	"OpenDirectionOutputDialog",
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
	"SaveBoardImageDialog",
	"SaveBoardPNG",
	"SaveBoardSVG",
	"SaveCSV",
	"SaveDatabaseDialog",
	"SetIssuerName",
	"ShowAlert",
	"ShowQuestionDialog",
	"StartFolderWatch",
	"StopFolderWatch",
	"SuggestWatchFolder",
	"StartEvaluationAtRest",
	"StartGammonNetBatch",
	"StartGammonNetMatchBatch",
	"StartGammonNetStaleBatch",
	"StartupFilePath",
}

// TestBoundAppMethodsMatchExpected fails on any difference between *App's
// exported methods and boundAppMethods.
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
