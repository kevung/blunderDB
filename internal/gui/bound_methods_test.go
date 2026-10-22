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
	"ActivateLibraryGroup",
	"CloseLibraryGroup",
	"ElsewhereInGroups",
	"LibraryGroupHolding",
	"ListLibraryGroups",
	"OpenLibraryGroup",
	"SetLibraryGroupLocked",
	"TransferToGroup",
	"CopyPositionToGroup",
	"UndoPositionCopy",
	"ViewerCatalogOf",
	"ViewerShow",
	"ViewerSteps",
	"AssistantAsk",
	"AssistantCancel",
	"AssistantConfirm",
	"AssistantHasKey",
	"AssistantPresets",
	"AssistantReset",
	"AssistantSetKey",
	"BearoffStatus",
	"CancelCubeMatrix",
	"CancelEvaluationAtRest",
	"CancelGammonNetBatch",
	"CancelLibraryImport",
	"CancelMaintenance",
	"CancelRollout",
	"RolloutPresets",
	"RolloutStatus",
	"StartRollout",
	"StartRolloutFiltered",
	"StartRolloutIDs",
	"CheckForUpdate",
	"CollectImportableFiles",
	"ImportFiles",
	"ImportFilesIn",
	"ConfigureMCPHost",
	"GetMCPHostStatus",
	"HelpBundle",
	"StartTournamentDisplay",
	"StopTournamentDisplay",
	"GetTournamentDisplayStatus",
	"ComputeCubeMatrix",
	"CopyImageToClipboard",
	"CountRolloutIDs",
	"DeleteFile",
	"EnsureBearoffTables",
	"GenerateBearoffQuestion",
	"GenerateEvaluationQuestion",
	"OpeningBookStatus",
	"OpeningBookQuestion",
	"OpeningBookGrade",
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
	"GnubgID",
	"LooksLikeOGID",
	"ReadLogTail",
	"OpenBearoffFileDialog",
	"OpenDatabaseDialog",
	"OpenDirectionOutputDialog",
	"OpenExportDatabaseDialog",
	"OpenExportApkgDialog",
	"OpenExportMatDialog",
	"OpenImportDatabaseDialog",
	"OpenMETDialog",
	"OpenLocalPage",
	"OpenLogsFolder",
	"OpenPositionFilesDialog",
	"OpenPositionFolderDialog",
	"OpenSyncFolderDialog",
	"MediaURL",
	"CheckVideoMux",
	"OpenVideoExternally",
	"PathExists",
	"PickTranscriptionVideo",
	"VideoSourceKind",
	"YouTubeEmbedURL",
	"YouTubeWatchURL",
	"PickIdentityFile",
	"ReleaseMedia",
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
	"QueueGammonNetBatch",
	"QueueGammonNetBatchIn",
	"StartGammonNetBatch",
	"StartGammonNetMatchBatch",
	"StartGammonNetStaleBatch",
	"StartMaintenance",
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
