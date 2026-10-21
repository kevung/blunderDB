package remote_test

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/remote"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// daemonStorage is a remote backend over its own in-process daemon; Close
// stops the daemon and its database with it.
type daemonStorage struct {
	*remote.Storage
	stop func()
}

func (d daemonStorage) Close() error { d.stop(); return nil }

// The scope "" stands for the backend's own tenant; 42 keeps it apart from
// the explicit scopes the contract cases use ("1", "2").
func newDaemonStorage(t *testing.T) storage.Storage {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	srv, err := server.New(server.Options{Storage: st, Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	rs, err := remote.Open(remote.Options{BaseURL: ts.URL, Tenant: "42"})
	if err != nil {
		t.Fatalf("remote.Open: %v", err)
	}
	return daemonStorage{Storage: rs, stop: func() { ts.Close(); st.Close() }}
}

func TestContract_Remote(t *testing.T) {
	all := os.Getenv("REMOTE_CONTRACT_ALL") != ""
	storagetest.RunContractTestsOnly(t, func() storage.Storage { return newDaemonStorage(t) }, func(name string) bool {
		return all || covered[name]
	})
}

// covered names the contract cases the remote backend passes: those whose
// every method has a /v1 route. A case leaves this list only with the route
// it needs.
var covered = map[string]bool{
	"ActionLabel/SelectsByBestCubeAction":                    true,
	"Aliases/ReadAsOnePerson":                                true,
	"Aliases/StayFlat":                                       true,
	"Analyses/PlayedActionsComeFromTheMatch":                 true,
	"Analyses/RepairDenormalisedColumns":                     true,
	"Analysis/LoadManyDecodesBatch":                          true,
	"Analysis/SaveAndCompress":                               true,
	"Analysis/SaveIsAnUpsert":                                true,
	"Anki/CubePairsAreChained":                               true,
	"Anki/DeckIDWindows":                                     true,
	"Anki/DrawOrderInterleavesTies":                          true,
	"Anki/FilteredWindowsMatchCounters":                      true,
	"Anki/ReviewUpdatesScheduling":                           true,
	"Collection/CopyPosition":                                true,
	"Collection/EvaluateUnderCeiling":                        true,
	"Collection/FilterQueryRoundTrips":                       true,
	"Collection/IDWindows":                                   true,
	"Collection/MoveBetweenCollections":                      true,
	"Collection/PositionIndexMap":                            true,
	"Collection/RenameAndDelete":                             true,
	"Collection/ReorderPositions":                            true,
	"Collection/UnreadableQueryRefused":                      true,
	"Comment/ByPositionsGroupsOldestFirst":                   true,
	"Comment/CRUD":                                           true,
	"Comment/PositionDeleteCascades":                         true,
	"Comment/SearchAcrossPositions":                          true,
	"Comments/TagVocabulary":                                 true,
	"Errors/DanglingReferenceIsNotFound":                     true,
	"Errors/MergePlayersRejectsEmptyAsInvalid":               true,
	"Filter/Pins":                                            true,
	"Filter/SaveAndList":                                     true,
	"History/SaveLoadClear":                                  true,
	"Lesson/Lifecycle":                                       true,
	"Lesson/Progress":                                        true,
	"LibrarySettings/BlunderCountPromisesTheSearch":          true,
	"LibrarySettings/DefaultsWhenNeverSet":                   true,
	"LibrarySettings/RejectsInvertedNesting":                 true,
	"LibrarySettings/RoundTrip":                              true,
	"Match/CreateGameMoveCascade":                            true,
	"Match/DeleteCascade":                                    true,
	"Matches/LastVisitedFollowsTheVisit":                     true,
	"Matches/LastVisitedFallbackIsTheListHead":               true,
	"Matches/LibraryMoves":                                   true,
	"Match/FindByHash":                                       true,
	"Match/HashesReadBack":                                   true,
	"Match/ListFilterSortPaginate":                           true,
	"Match/ListPlayerAliases":                                true,
	"Match/MatchesByPosition":                                true,
	"Match/MoveDecisionTimeRoundTrip":                        true,
	"Match/MoveLuckRoundTrip":                                true,
	"MatchStats/BestCubeActionInvalidates":                   true,
	"MatchStats/PositionUpdateInvalidates":                   true,
	"MatchStats/SharedPositionsCountOnce":                    true,
	"Metadata/Counts":                                        true,
	"Metadata/EstimatedCounts":                               true,
	"Position/CountAndIndexOf":                               true,
	"Position/DedupByZobrist":                                true,
	"Position/ListIDsAndLoadByIDs":                           true,
	"Position/ProvenanceIsSticky":                            true,
	"Position/RepairCrawfordSentinel":                        true,
	"Position/RepairCrawfordSentinelFromPastedXGID":          true,
	"Position/RepairCrawfordSentinelOnePointMatch":           true,
	"Position/SaveCreatedConcurrentOneWinner":                true,
	"Position/SaveCreatedReportsTheInsert":                   true,
	"Position/Save+Load":                                     true,
	"Position/UpdatePreservesId":                             true,
	"Position/UpdateRefusesDuplicate":                        true,
	"Scope/HistoryAndFilterIsolation":                        true,
	"Search/CommentRestrictionLosesNothing":                  true,
	"Search/DecisionTimeFilter":                              true,
	"Search/DecisionTimeWithMoveError":                       true,
	"Search/EncounterCountsEveryOccurrence":                  true,
	"Search/FilterByAnalysisDecodesCompressedBlob":           true,
	"Search/FilterByCommentPresence":                         true,
	"Search/FilterByCubeResponse":                            true,
	"Search/FilterByDecisionType":                            true,
	"Search/FilterByFlagged":                                 true,
	"Search/FilterByIndividuallyImported":                    true,
	"SearchHistory/SaveListDelete":                           true,
	"Search/MoveErrorFilterMaxOverPlays":                     true,
	"Search/Pagination":                                      true,
	"Search/SortPipAndDate":                                  true,
	"Search/TagFilterIsDelimited":                            true,
	"Search/WindowsAgree":                                    true,
	"Search/WindowsAgree/go_phase,_by_close":                 true,
	"Search/WindowsAgree/go_phase,_by_date_asc":              true,
	"Search/WindowsAgree/go_phase,_by_date_desc":             true,
	"Search/WindowsAgree/go_phase,_by_error":                 true,
	"Search/WindowsAgree/go_phase,_by_id":                    true,
	"Search/WindowsAgree/go_phase,_by_pip_asc":               true,
	"Search/WindowsAgree/go_phase,_by_pip_desc":              true,
	"Search/WindowsAgree/go_phase,_by_winrate":               true,
	"Search/WindowsAgree/go_phase_in_chunks,_by_close":       true,
	"Search/WindowsAgree/go_phase_in_chunks,_by_date_asc":    true,
	"Search/WindowsAgree/go_phase_in_chunks,_by_date_desc":   true,
	"Search/WindowsAgree/go_phase_in_chunks,_by_error":       true,
	"Search/WindowsAgree/go_phase_in_chunks,_by_id":          true,
	"Search/WindowsAgree/go_phase_in_chunks,_by_pip_asc":     true,
	"Search/WindowsAgree/go_phase_in_chunks,_by_pip_desc":    true,
	"Search/WindowsAgree/go_phase_in_chunks,_by_winrate":     true,
	"Search/WindowsAgree/move_error,_by_close":               true,
	"Search/WindowsAgree/move_error,_by_date_asc":            true,
	"Search/WindowsAgree/move_error,_by_date_desc":           true,
	"Search/WindowsAgree/move_error,_by_error":               true,
	"Search/WindowsAgree/move_error,_by_id":                  true,
	"Search/WindowsAgree/move_error,_by_pip_asc":             true,
	"Search/WindowsAgree/move_error,_by_pip_desc":            true,
	"Search/WindowsAgree/move_error,_by_winrate":             true,
	"Search/WindowsAgree/move_error_in_chunks,_by_close":     true,
	"Search/WindowsAgree/move_error_in_chunks,_by_date_asc":  true,
	"Search/WindowsAgree/move_error_in_chunks,_by_date_desc": true,
	"Search/WindowsAgree/move_error_in_chunks,_by_error":     true,
	"Search/WindowsAgree/move_error_in_chunks,_by_id":        true,
	"Search/WindowsAgree/move_error_in_chunks,_by_pip_asc":   true,
	"Search/WindowsAgree/move_error_in_chunks,_by_pip_desc":  true,
	"Search/WindowsAgree/move_error_in_chunks,_by_winrate":   true,
	"Search/WindowsAgree/sql_only,_by_close":                 true,
	"Search/WindowsAgree/sql_only,_by_date_asc":              true,
	"Search/WindowsAgree/sql_only,_by_date_desc":             true,
	"Search/WindowsAgree/sql_only,_by_error":                 true,
	"Search/WindowsAgree/sql_only,_by_id":                    true,
	"Search/WindowsAgree/sql_only,_by_pip_asc":               true,
	"Search/WindowsAgree/sql_only,_by_pip_desc":              true,
	"Search/WindowsAgree/sql_only,_by_winrate":               true,
	"Session/MultiScopeIsolation":                            true,
	"Stats/AggregateCounts":                                  true,
	"Stats/AnalysisEngines":                                  true,
	"Stats/BreakdownPositions":                               true,
	"Stats/BreakdownPositions/decision_type":                 true,
	"Stats/BreakdownPositions/no_filter":                     true,
	"Stats/BreakdownPositions/other_decision":                true,
	"Stats/BreakdownPositions/player":                        true,
	"Stats/BreakdownPositions/player_and_type":               true,
	"Stats/Breakdowns":                                       true,
	"Stats/CubeDirections":                                   true,
	"Stats/DirectionalBiases":                                true,
	"Stats/DirectionalBiasesPerMatch":                        true,
	"Stats/DirectionalBiasesScores":                          true,
	"Stats/HeadToHeadWindowsRanking":                         true,
	"Stats/MatchDecisionLosses":                              true,
	"Stats/MatchDecisionLossesCube":                          true,
	"Stats/MatchDetail":                                      true,
	"Stats/MatchMoveGrades":                                  true,
	"Stats/OpponentFilter":                                   true,
	"Stats/PlayedDecisionFollowsPositionEdit":                true,
	"Stats/PlayerTable":                                      true,
	"Stats/PositionIDsByMatch":                               true,
	"Stats/PositionIDsByTournament":                          true,
	"Stats/RecurringErrors":                                  true,
	"Stats/RollingOrderAcrossGames":                          true,
	"Stats/SnowieDenominatorCountsBothPlayers":               true,
	"Stats/StudyPlan":                                        true,
	"Stats/TimeErrorsScoredByAnalysis":                       true,
	"Stats/TournamentReview":                                 true,
	"Tournament/AddRemoveMatch":                              true,
	"Tournament/EffectiveDate":                               true,
	"Training/DeviationStaysOutOfTheMean":                    true,
	"Training/MissedPositions":                               true,
	"Training/NumberStatsCountByType":                        true,
	"Training/SaveAndRead":                                   true,
}
