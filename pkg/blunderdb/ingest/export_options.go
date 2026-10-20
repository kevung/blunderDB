package ingest

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// WholeLibraryOptions is the export options' everything-on preset: the whole
// library, every family and what travels with a position. The daemon reads a
// request body over it, so that a field left out keeps its family.
func WholeLibraryOptions() domain.ExportOptions {
	return domain.ExportOptions{
		WholeLibrary:          true,
		IncludeAnalysis:       true,
		IncludeComments:       true,
		IncludeFilterLibrary:  true,
		IncludePlayedMoves:    true,
		IncludeMatches:        true,
		IncludeTranscriptions: true,
		IncludeAliases:        true,
		IncludeCollections:    true,
		IncludeTournaments:    true,
		IncludeAnkiDecks:      true,
		IncludeLessons:        true,
		IncludeAnkiHistory:    true,
	}
}

// OptionsFor maps the export options of the desktop, the CLI and the daemon onto
// what ExportSQLite and PlanExport read. It is the only reading of them, so that
// the three entry points cannot disagree on what one set of options exports.
// Metadata, Watermark and Password are left to the caller, which seals the
// watermark with its own identity.
//
// The whole library is WholeTenant; a family the options turn off or narrow to
// some ids is then taken out of it. Anything else is a list of positions, and the
// matches it carries are, unless named, those its positions were played in.
func OptionsFor(ctx context.Context, src storage.Storage, scope string, opts domain.ExportOptions) (ExportOptions, error) {
	if opts.Incremental {
		return incrementalOptions(ctx, src, scope, opts)
	}
	if opts.WholeLibrary {
		return wholeLibraryOptions(opts), nil
	}
	plan := ExportOptions{
		Format:         FormatSQLite,
		Analysis:       opts.IncludeAnalysis,
		Comments:       opts.IncludeComments,
		PlayedMoves:    opts.IncludePlayedMoves,
		FilterLibrary:  opts.IncludeFilterLibrary,
		Transcriptions: opts.IncludeTranscriptions,
		Aliases:        opts.IncludeAliases,
		AnkiHistory:    opts.IncludeAnkiHistory,
	}
	sel := Selection{
		AllPositions:  opts.AllPositions,
		PositionIDs:   opts.PositionIDs,
		TournamentIDs: opts.TournamentIDs,
		// A match leaves with the positions its moves were played in: without them its
		// moves would point at nothing in the file.
		MatchPositions: true,
	}
	// Positions wins over PositionIDs when set: callers holding positions
	// that are not in the database keep working.
	if len(opts.Positions) > 0 {
		sel.AllPositions, sel.PositionIDs = false, nil
		sel.Positions = make([]*domain.Position, len(opts.Positions))
		for i := range opts.Positions {
			sel.Positions[i] = &opts.Positions[i]
		}
	}
	if opts.IncludeCollections {
		sel.CollectionIDs = opts.CollectionIDs
		sel.CollectionPositions = opts.IncludeCollectionPositions
	}
	if opts.IncludeMatches {
		switch {
		case len(opts.MatchIDs) > 0:
			sel.MatchIDs = opts.MatchIDs
		case sel.AllPositions:
			sel.AllMatches = true
		default:
			ids := sel.PositionIDs
			for _, p := range sel.Positions {
				ids = append(ids, p.ID)
			}
			reached, err := src.Matches().MatchIDsReachedBy(ctx, scope, ids)
			if err != nil {
				return plan, err
			}
			sel.MatchIDs = reached
		}
	}
	// A tournament chosen carries its matches, whatever the matches family says.
	sel.TournamentMatches = len(sel.TournamentIDs) > 0
	sel.DeckIDs, sel.DeckPositions = opts.DeckIDs, len(opts.DeckIDs) > 0
	if opts.IncludeLessons {
		sel.LessonIDs = opts.LessonIDs
		sel.AllLessons = len(opts.LessonIDs) == 0
		sel.LessonContents = true
	}
	plan.Selection = sel
	return plan, nil
}

// wholeLibraryOptions is WholeTenant less what opts leaves out. Each family follows the
// same rule: off drops it, a list of ids narrows it to them, on with no ids keeps all of it.
func wholeLibraryOptions(opts domain.ExportOptions) ExportOptions {
	plan := WholeTenant(FormatSQLite)
	plan.Analysis = opts.IncludeAnalysis
	plan.Comments = opts.IncludeComments
	plan.PlayedMoves = opts.IncludePlayedMoves
	plan.FilterLibrary = opts.IncludeFilterLibrary
	plan.Transcriptions = opts.IncludeTranscriptions
	plan.Aliases = opts.IncludeAliases
	plan.AnkiHistory = opts.IncludeAnkiHistory
	sel := &plan.Selection
	sel.AllMatches, sel.MatchIDs = narrow(opts.IncludeMatches, opts.MatchIDs)
	sel.AllCollections, sel.CollectionIDs = narrow(opts.IncludeCollections, opts.CollectionIDs)
	sel.AllTournaments, sel.TournamentIDs = narrow(opts.IncludeTournaments, opts.TournamentIDs)
	// A tournament chosen carries its matches, whatever the matches family says.
	sel.TournamentMatches = sel.AllTournaments || len(sel.TournamentIDs) > 0
	sel.AllLessons, sel.LessonIDs = narrow(opts.IncludeLessons, opts.LessonIDs)
	sel.LessonContents = opts.IncludeLessons
	plan.AnkiDecks, sel.DeckIDs = narrow(opts.IncludeAnkiDecks, opts.DeckIDs)
	return plan
}

// narrow reads one family of a whole-library export: all of it, the listed ids, or none.
func narrow(include bool, ids []int64) (all bool, picked []int64) {
	if !include {
		return false, nil
	}
	if len(ids) > 0 {
		return false, ids
	}
	return true, nil
}
