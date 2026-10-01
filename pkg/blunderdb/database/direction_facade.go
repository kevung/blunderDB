package database

import (
	"context"
	"strconv"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction/service"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The Direction and the Rencontre live in the direction service, on the storage contract
// (ADR-0057 rule 2). This file is the desktop's and the CLI's façade over it: each method runs
// the service on the open database's Storage, private scope. It takes d.mu only to read which
// Storage is open — the service writes through the backend's own transactions, and holding the
// non-reentrant lock across a call would forbid the service nothing but deadlock a caller that
// already holds it.

// The service's views keep their database names for the callers of the façade.
type (
	DirectionSummary    = service.DirectionSummary
	DirectionView       = service.DirectionView
	ClockView           = service.ClockView
	BracketMatch        = service.BracketMatch
	BracketFeed         = service.BracketFeed
	BracketSection      = service.BracketSection
	BracketPhase        = service.BracketPhase
	LivesRow            = service.LivesRow
	ConfigChange        = service.ConfigChange
	PhaseLock           = service.PhaseLock
	ConfigPreview       = service.ConfigPreview
	LastDecision        = service.LastDecision
	TableCell           = service.TableCell
	HallCell            = service.HallCell
	HallEvent           = service.HallEvent
	HallView            = service.HallView
	SlotRow             = service.SlotRow
	SlotSuggestion      = service.SlotSuggestion
	MatchSlot           = service.MatchSlot
	RencontreMember     = service.RencontreMember
	RencontreView       = service.RencontreView
	StandingRow         = service.StandingRow
	StandingsSection    = service.StandingsSection
	StandingsView       = service.StandingsView
	DirectoryEntry      = service.DirectoryEntry
	DirectorySource     = service.DirectorySource
	DirectoryCSVError   = service.DirectoryCSVError
	DirectoryCSVWarning = service.DirectoryCSVWarning
	DirectoryImport     = service.DirectoryImport
	FreeSlot            = service.FreeSlot
	HistoryEntry        = service.HistoryEntry
	PairMember          = service.PairMember
	ParticipantRow      = service.ParticipantRow
	SearchEntry         = service.SearchEntry
	EntrySuggestion     = service.EntrySuggestion
)

// PairLabel is the Participant's name for a pair: "A / B", in the order the pair was entered.
func PairLabel(members []PairMember) string { return service.PairLabel(members) }

// PairRating is the rating a pair enters with.
func PairRating(members []PairMember) float64 { return service.PairRating(members) }

// PageWarning reports a display page a gesture could not rewrite.
type PageWarning = service.PageWarning

// OnDirectionPageWarning installs on d the function told of every page a gesture could not
// rewrite — the desktop's status bar. A function, not a method: Database is bound to the
// frontend, which has no use for a callback.
func OnDirectionPageWarning(d *Database, f func(PageWarning)) {
	d.directionMem.OnPageWarning(f)
}

// directionService binds the service to the open database, private scope.
func (d *Database) directionService() *service.Service {
	d.mu.RLock()
	st := d.store
	d.mu.RUnlock()
	return service.New(st, "", &d.directionMem)
}

// DirectionStore returns the Store the direction package runs on, over the open database.
func (d *Database) DirectionStore() direction.Store {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return storage.BindDirection(d.store.Directions(), "")
}

// StartTranscriptionFromSlot opens a draft with its header already filled from the Slot.
//
// The Slot is reserved FROM THE DRAFT, not only from the save: a director who starts typing a
// match must see the Slot taken, or two people will type the same match.
func (d *Database) StartTranscriptionFromSlot(tournamentID int64, slotID string) (*TranscriptionState, error) {
	h, err := d.directionService().SlotHeader(context.Background(), tournamentID, slotID)
	if err != nil {
		return nil, err
	}
	return d.CreateTranscription(h)
}

// clockAt is Clock at a given instant.
func (d *Database) clockAt(tournamentID int64, now time.Time) (*ClockView, error) {
	return d.directionService().ClockAt(context.Background(), tournamentID, now)
}

// realignRencontre puts every member of a restored Rencontre back on the room's tables.
func (d *Database) realignRencontre(id int64) error {
	return d.directionService().Realign(context.Background(), id)
}

func bracketSkeleton(sim *tournoi.State, ph *tournoi.PhaseState) ([]*tournoi.Section, bool) {
	return service.BracketSkeleton(sim, ph)
}

func confirmAt(ctx context.Context, dir *direction.Direction, a tournoi.Action, now time.Time) error {
	return service.ConfirmAt(ctx, dir, a, now)
}

// ConfirmProposal is service.Service.ConfirmProposal on the open database.
func (d *Database) ConfirmProposal(tournamentID int64, actionJSON string) (*DirectionView, error) {
	return d.directionService().ConfirmProposal(context.Background(), tournamentID, actionJSON)
}

// ConfirmAllProposals is service.Service.ConfirmAllProposals on the open database.
func (d *Database) ConfirmAllProposals(tournamentID int64) (*DirectionView, error) {
	return d.directionService().ConfirmAllProposals(context.Background(), tournamentID)
}

// StartMatchManually is service.Service.StartMatchManually on the open database.
func (d *Database) StartMatchManually(tournamentID int64, a string, b string, length int, table int) (*DirectionView, error) {
	return d.directionService().StartMatchManually(context.Background(), tournamentID, a, b, length, table)
}

// FreeParticipants is service.Service.FreeParticipants on the open database.
func (d *Database) FreeParticipants(tournamentID int64) ([]tournoi.Player, error) {
	return d.directionService().FreeParticipants(context.Background(), tournamentID)
}

// ListDirections is service.Service.ListDirections on the open database.
func (d *Database) ListDirections() ([]DirectionSummary, error) {
	return d.directionService().ListDirections(context.Background())
}

// CreateDirection is service.Service.CreateDirection on the open database.
func (d *Database) CreateDirection(tournamentID int64, configJSON string, seed int64) error {
	return d.directionService().CreateDirection(context.Background(), tournamentID, configJSON, seed)
}

// GetDirection is service.Service.GetDirection on the open database.
func (d *Database) GetDirection(tournamentID int64) (*DirectionView, error) {
	return d.directionService().GetDirection(context.Background(), tournamentID)
}

// HasDirection is service.Service.HasDirection on the open database.
func (d *Database) HasDirection(tournamentID int64) (bool, error) {
	return d.directionService().HasDirection(context.Background(), tournamentID)
}

// SetDirectionConfig is service.Service.SetDirectionConfig on the open database.
func (d *Database) SetDirectionConfig(tournamentID int64, configJSON string) error {
	return d.directionService().SetDirectionConfig(context.Background(), tournamentID, configJSON)
}

// SetDirectionOutputDir is service.Service.SetDirectionOutputDir on the open database.
func (d *Database) SetDirectionOutputDir(tournamentID int64, dir string) error {
	return d.directionService().SetDirectionOutputDir(context.Background(), tournamentID, dir)
}

// EnterParticipants is service.Service.EnterParticipants on the open database.
func (d *Database) EnterParticipants(tournamentID int64, playersJSON string) error {
	return d.directionService().EnterParticipants(context.Background(), tournamentID, playersJSON)
}

// DeleteDirection is service.Service.DeleteDirection on the open database.
func (d *Database) DeleteDirection(tournamentID int64) error {
	return d.directionService().DeleteDirection(context.Background(), tournamentID)
}

// DirectionJournalJSON is service.Service.DirectionJournalJSON on the open database.
func (d *Database) DirectionJournalJSON(tournamentID int64) (string, error) {
	return d.directionService().DirectionJournalJSON(context.Background(), tournamentID)
}

// Brackets is service.Service.Brackets on the open database.
func (d *Database) Brackets(tournamentID int64) ([]BracketPhase, error) {
	return d.directionService().Brackets(context.Background(), tournamentID)
}

// Clock is service.Service.Clock on the open database.
func (d *Database) Clock(tournamentID int64) (*ClockView, error) {
	return d.directionService().Clock(context.Background(), tournamentID)
}

// PreviewDirectionConfig is service.Service.PreviewDirectionConfig on the open database.
func (d *Database) PreviewDirectionConfig(tournamentID int64, configJSON string) (*ConfigPreview, error) {
	return d.directionService().PreviewDirectionConfig(context.Background(), tournamentID, configJSON)
}

// LastDecision is service.Service.LastDecision on the open database.
func (d *Database) LastDecision(tournamentID int64) (*LastDecision, error) {
	return d.directionService().LastDecision(context.Background(), tournamentID)
}

// CorrectResult is service.Service.CorrectResult on the open database.
func (d *Database) CorrectResult(tournamentID int64, matchID string, winner string, scoreA int, scoreB int, note string) (*DirectionView, error) {
	return d.directionService().CorrectResult(context.Background(), tournamentID, matchID, winner, scoreA, scoreB, note)
}

// FinishedMatches is service.Service.FinishedMatches on the open database.
func (d *Database) FinishedMatches(tournamentID int64, limit int) ([]TableCell, error) {
	return d.directionService().FinishedMatches(context.Background(), tournamentID, limit)
}

// Directory is service.Service.Directory on the open database.
func (d *Database) Directory() ([]DirectoryEntry, error) {
	return d.directionService().Directory(context.Background())
}

// DirectorySources is service.Service.DirectorySources on the open database.
func (d *Database) DirectorySources() ([]DirectorySource, error) {
	return d.directionService().DirectorySources(context.Background())
}

// DirectoryEntrants is service.Service.DirectoryEntrants on the open database.
func (d *Database) DirectoryEntrants(tournamentID int64) ([]DirectoryEntry, error) {
	return d.directionService().DirectoryEntrants(context.Background(), tournamentID)
}

// DirectoryCSV is service.Service.DirectoryCSV on the open database.
func (d *Database) DirectoryCSV() (string, error) {
	return d.directionService().DirectoryCSV(context.Background())
}

// ParseDirectoryCSV is service.Service.ParseDirectoryCSV on the open database.
func (d *Database) ParseDirectoryCSV(tournamentID int64, body string) (*DirectoryImport, error) {
	return d.directionService().ParseDirectoryCSV(context.Background(), tournamentID, body)
}

// EntrySuggestions is service.Service.EntrySuggestions on the open database.
func (d *Database) EntrySuggestions() ([]EntrySuggestion, error) {
	return d.directionService().EntrySuggestions(context.Background())
}

// Participants is service.Service.Participants on the open database.
func (d *Database) Participants(tournamentID int64) ([]ParticipantRow, error) {
	return d.directionService().Participants(context.Background(), tournamentID)
}

// AddParticipant is service.Service.AddParticipant on the open database.
func (d *Database) AddParticipant(tournamentID int64, name string, club string, rating float64) (*DirectionView, error) {
	return d.directionService().AddParticipant(context.Background(), tournamentID, name, club, rating)
}

// UpdateParticipant is service.Service.UpdateParticipant on the open database.
func (d *Database) UpdateParticipant(tournamentID int64, id string, name string, club string, rating float64) (*DirectionView, error) {
	return d.directionService().UpdateParticipant(context.Background(), tournamentID, id, name, club, rating)
}

// ReinstateParticipant is service.Service.ReinstateParticipant on the open database.
func (d *Database) ReinstateParticipant(tournamentID int64, id string) (*DirectionView, error) {
	return d.directionService().ReinstateParticipant(context.Background(), tournamentID, id)
}

// WithdrawParticipant is service.Service.WithdrawParticipant on the open database.
func (d *Database) WithdrawParticipant(tournamentID int64, id string, afterCurrent bool) (*DirectionView, error) {
	return d.directionService().WithdrawParticipant(context.Background(), tournamentID, id, afterCurrent)
}

// MakeParticipantAbsent is service.Service.MakeParticipantAbsent on the open database.
func (d *Database) MakeParticipantAbsent(tournamentID int64, id string, until string, round int) (*DirectionView, error) {
	return d.directionService().MakeParticipantAbsent(context.Background(), tournamentID, id, until, round)
}

// MakeParticipantAvailable is service.Service.MakeParticipantAvailable on the open database.
func (d *Database) MakeParticipantAvailable(tournamentID int64, id string) (*DirectionView, error) {
	return d.directionService().MakeParticipantAvailable(context.Background(), tournamentID, id)
}

// History is service.Service.History on the open database.
func (d *Database) History(tournamentID int64, player string, match string) ([]HistoryEntry, error) {
	return d.directionService().History(context.Background(), tournamentID, player, match)
}

// AddDirectionNote is service.Service.AddDirectionNote on the open database.
func (d *Database) AddDirectionNote(tournamentID int64, text string) (*DirectionView, error) {
	return d.directionService().AddDirectionNote(context.Background(), tournamentID, text)
}

// SinceLastGesture is service.Service.SinceLastGesture on the open database.
func (d *Database) SinceLastGesture(tournamentID int64, seq int) ([]HistoryEntry, error) {
	return d.directionService().SinceLastGesture(context.Background(), tournamentID, seq)
}

// DirectionFreeSlots is service.Service.DirectionFreeSlots on the open database.
func (d *Database) DirectionFreeSlots(tournamentID int64) ([]FreeSlot, error) {
	return d.directionService().DirectionFreeSlots(context.Background(), tournamentID)
}

// AddParticipantAtSlot is service.Service.AddParticipantAtSlot on the open database.
func (d *Database) AddParticipantAtSlot(tournamentID int64, name string, club string, rating float64, section string, key string) (*DirectionView, error) {
	return d.directionService().AddParticipantAtSlot(context.Background(), tournamentID, name, club, rating, section, key)
}

// SetDirectionStrings is service.Service.SetDirectionStrings on the open database.
func (d *Database) SetDirectionStrings(lang string, catalogJSON string) error {
	return d.directionService().SetDirectionStrings(context.Background(), lang, catalogJSON)
}

// WriteDirectionPage is service.Service.WriteDirectionPage on the open database.
func (d *Database) WriteDirectionPage(tournamentID int64) (string, error) {
	return d.directionService().WriteDirectionPage(context.Background(), tournamentID)
}

// DirectionPageHTML is service.Service.DirectionPageHTML on the open database.
func (d *Database) DirectionPageHTML(tournamentID int64) (string, error) {
	return d.directionService().DirectionPageHTML(context.Background(), tournamentID)
}

// DirectionPairingSheetHTML is service.Service.DirectionPairingSheetHTML on the open database.
func (d *Database) DirectionPairingSheetHTML(tournamentID int64, round int) (string, error) {
	return d.directionService().DirectionPairingSheetHTML(context.Background(), tournamentID, round)
}

// WriteDirectionPairingSheet is service.Service.WriteDirectionPairingSheet on the open database.
func (d *Database) WriteDirectionPairingSheet(tournamentID int64, round int) (string, error) {
	return d.directionService().WriteDirectionPairingSheet(context.Background(), tournamentID, round)
}

// DirectionUpcomingSheetHTML is service.Service.DirectionUpcomingSheetHTML on the open database.
func (d *Database) DirectionUpcomingSheetHTML(tournamentID int64, announced string) (string, error) {
	return d.directionService().DirectionUpcomingSheetHTML(context.Background(), tournamentID, announced)
}

// WriteDirectionUpcomingSheet is service.Service.WriteDirectionUpcomingSheet on the open database.
func (d *Database) WriteDirectionUpcomingSheet(tournamentID int64, announced string) (string, error) {
	return d.directionService().WriteDirectionUpcomingSheet(context.Background(), tournamentID, announced)
}

// DirectionRounds is service.Service.DirectionRounds on the open database.
func (d *Database) DirectionRounds(tournamentID int64) (int, error) {
	return d.directionService().DirectionRounds(context.Background(), tournamentID)
}

// AddPair is service.Service.AddPair on the open database.
func (d *Database) AddPair(tournamentID int64, membersJSON string, rating float64) (*DirectionView, error) {
	return d.directionService().AddPair(context.Background(), tournamentID, membersJSON, rating)
}

// UpdatePair is service.Service.UpdatePair on the open database.
func (d *Database) UpdatePair(tournamentID int64, id string, membersJSON string, rating float64) (*DirectionView, error) {
	return d.directionService().UpdatePair(context.Background(), tournamentID, id, membersJSON, rating)
}

// Pairs is service.Service.Pairs on the open database.
func (d *Database) Pairs(tournamentID int64) (map[string][]PairMember, error) {
	return d.directionService().Pairs(context.Background(), tournamentID)
}

// TableGrid is service.Service.TableGrid on the open database.
func (d *Database) TableGrid(tournamentID int64) ([]TableCell, error) {
	return d.directionService().TableGrid(context.Background(), tournamentID)
}

// RencontreTableGrid is service.Service.RencontreTableGrid on the open database.
func (d *Database) RencontreTableGrid(rencontreID int64) (*HallView, error) {
	return d.directionService().RencontreTableGrid(context.Background(), rencontreID)
}

// EnterResult is service.Service.EnterResult on the open database.
func (d *Database) EnterResult(tournamentID int64, matchID string, winner string, scoreA int, scoreB int, note string) (*DirectionView, error) {
	return d.directionService().EnterResult(context.Background(), tournamentID, matchID, winner, scoreA, scoreB, note)
}

// EnterForfeit is service.Service.EnterForfeit on the open database.
func (d *Database) EnterForfeit(tournamentID int64, matchID string, winner string, note string) (*DirectionView, error) {
	return d.directionService().EnterForfeit(context.Background(), tournamentID, matchID, winner, note)
}

// MoveMatchToTable is service.Service.MoveMatchToTable on the open database.
func (d *Database) MoveMatchToTable(tournamentID int64, matchID string, table int) (*DirectionView, error) {
	return d.directionService().MoveMatchToTable(context.Background(), tournamentID, matchID, table)
}

// CancelMatch is service.Service.CancelMatch on the open database.
func (d *Database) CancelMatch(tournamentID int64, matchID string) (*DirectionView, error) {
	return d.directionService().CancelMatch(context.Background(), tournamentID, matchID)
}

// Slots is service.Service.Slots on the open database.
func (d *Database) Slots(tournamentID int64) ([]SlotRow, error) {
	return d.directionService().Slots(context.Background(), tournamentID)
}

// AttachMatchToSlot is service.Service.AttachMatchToSlot on the open database.
func (d *Database) AttachMatchToSlot(tournamentID int64, slotID string, matchID int64) error {
	return d.directionService().AttachMatchToSlot(context.Background(), tournamentID, slotID, matchID)
}

// DetachMatchFromSlot is service.Service.DetachMatchFromSlot on the open database.
func (d *Database) DetachMatchFromSlot(tournamentID int64, slotID string) error {
	return d.directionService().DetachMatchFromSlot(context.Background(), tournamentID, slotID)
}

// UnattachedMatches is service.Service.UnattachedMatches on the open database.
func (d *Database) UnattachedMatches(tournamentID int64) ([]SlotSuggestion, error) {
	return d.directionService().UnattachedMatches(context.Background(), tournamentID)
}

// SlotOfMatch is service.Service.SlotOfMatch on the open database.
func (d *Database) SlotOfMatch(matchID int64) (*MatchSlot, error) {
	return d.directionService().SlotOfMatch(context.Background(), matchID)
}

// Standings is service.Service.Standings on the open database.
func (d *Database) Standings(tournamentID int64) (*StandingsView, error) {
	return d.directionService().Standings(context.Background(), tournamentID)
}

// StandingsCSV is service.Service.StandingsCSV on the open database.
func (d *Database) StandingsCSV(tournamentID int64) (string, error) {
	return d.directionService().StandingsCSV(context.Background(), tournamentID)
}

// CloseDirection is service.Service.CloseDirection on the open database.
func (d *Database) CloseDirection(tournamentID int64) (*DirectionView, error) {
	return d.directionService().CloseDirection(context.Background(), tournamentID)
}

// ReopenDirection is service.Service.ReopenDirection on the open database.
func (d *Database) ReopenDirection(tournamentID int64) (*DirectionView, error) {
	return d.directionService().ReopenDirection(context.Background(), tournamentID)
}

// CreateRencontre is service.Service.CreateRencontre on the open database.
func (d *Database) CreateRencontre(name string, startsOn string, endsOn string, tables int) (*RencontreView, error) {
	return d.directionService().CreateRencontre(context.Background(), name, startsOn, endsOn, tables)
}

// ListRencontres is service.Service.ListRencontres on the open database.
func (d *Database) ListRencontres() ([]RencontreView, error) {
	return d.directionService().ListRencontres(context.Background())
}

// GetRencontre is service.Service.GetRencontre on the open database.
func (d *Database) GetRencontre(id int64) (*RencontreView, error) {
	return d.directionService().GetRencontre(context.Background(), id)
}

// RencontreOf is service.Service.RencontreOf on the open database.
func (d *Database) RencontreOf(tournamentID int64) (int64, error) {
	return d.directionService().RencontreOf(context.Background(), tournamentID)
}

// UpdateRencontre is service.Service.UpdateRencontre on the open database.
func (d *Database) UpdateRencontre(id int64, name string, startsOn string, endsOn string, tables int) (*RencontreView, error) {
	return d.directionService().UpdateRencontre(context.Background(), id, name, startsOn, endsOn, tables)
}

// PreviewAttachToRencontre is service.Service.PreviewAttachToRencontre on the open database.
func (d *Database) PreviewAttachToRencontre(tournamentID int64, rencontreID int64) (*ConfigPreview, error) {
	return d.directionService().PreviewAttachToRencontre(context.Background(), tournamentID, rencontreID)
}

// AttachToRencontre is service.Service.AttachToRencontre on the open database.
func (d *Database) AttachToRencontre(tournamentID int64, rencontreID int64) (*RencontreView, error) {
	return d.directionService().AttachToRencontre(context.Background(), tournamentID, rencontreID)
}

// DetachFromRencontre is service.Service.DetachFromRencontre on the open database.
func (d *Database) DetachFromRencontre(tournamentID int64) error {
	return d.directionService().DetachFromRencontre(context.Background(), tournamentID)
}

// TrashRencontre is service.Service.TrashRencontre on the open database.
func (d *Database) TrashRencontre(id int64) (int64, error) {
	return d.directionService().TrashRencontre(context.Background(), id)
}

// SetRencontreTableOutOfService is service.Service.SetRencontreTableOutOfService on the open database.
func (d *Database) SetRencontreTableOutOfService(id int64, table int, out bool) (*RencontreView, error) {
	return d.directionService().SetRencontreTableOutOfService(context.Background(), id, table, out)
}

// SetRencontreBreaks is service.Service.SetRencontreBreaks on the open database.
func (d *Database) SetRencontreBreaks(id int64, breaksJSON string) (*RencontreView, error) {
	return d.directionService().SetRencontreBreaks(context.Background(), id, breaksJSON)
}

// SetRencontreTables is service.Service.SetRencontreTables on the open database.
func (d *Database) SetRencontreTables(id int64, settings []domain.TableSetting) (*RencontreView, error) {
	return d.directionService().SetRencontreTables(context.Background(), id, settings)
}

// SetEventRooms is service.Service.SetEventRooms on the open database.
func (d *Database) SetEventRooms(rencontreID, tournamentID int64, rooms []string) (*RencontreView, error) {
	return d.directionService().SetEventRooms(context.Background(), rencontreID, tournamentID, rooms)
}

// SetDirectionTables is service.Service.SetDirectionTables on the open database.
func (d *Database) SetDirectionTables(tournamentID int64, settings []domain.TableSetting) (*DirectionView, error) {
	return d.directionService().SetDirectionTables(context.Background(), tournamentID, settings)
}

// TablePlan is service.Service.TablePlan on the open database: a Tournament's effective
// table properties and the rooms it may play in.
func (d *Database) TablePlan(tournamentID int64) (direction.TablePlan, error) {
	return d.directionService().TablePlan(context.Background(), tournamentID)
}

// SetRencontreOutputDir is service.Service.SetRencontreOutputDir on the open database.
func (d *Database) SetRencontreOutputDir(id int64, dir string) (*RencontreView, error) {
	return d.directionService().SetRencontreOutputDir(context.Background(), id, dir)
}

// RencontrePageHTML is service.Service.RencontrePageHTML on the open database.
func (d *Database) RencontrePageHTML(id int64) (string, error) {
	return d.directionService().RencontrePageHTML(context.Background(), id)
}

// WriteRencontrePage is service.Service.WriteRencontrePage on the open database.
func (d *Database) WriteRencontrePage(id int64) (string, error) {
	return d.directionService().WriteRencontrePage(context.Background(), id)
}

func itoa(n int) string { return strconv.Itoa(n) }

// RencontreSearchIndex is service.Service.RencontreSearchIndex on the open database.
func (d *Database) RencontreSearchIndex(id int64) ([]SearchEntry, error) {
	return d.directionService().RencontreSearchIndex(context.Background(), id)
}

// DirectionSearchIndex is service.Service.DirectionSearchIndex on the open database.
func (d *Database) DirectionSearchIndex(tournamentID int64) ([]SearchEntry, error) {
	return d.directionService().DirectionSearchIndex(context.Background(), tournamentID)
}
