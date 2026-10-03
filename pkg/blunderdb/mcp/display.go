package mcp

import (
	"context"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Display is a screen the tools can drive: the GUI that hosts the server. Its
// methods only ask; the frontend acts on its own state (a view of its tabs, a
// search through the command bar), so a display tool reads nothing the other
// tools could not and writes nothing to the database.
type Display interface {
	// OpenView opens a new view tab named name, runs query (the
	// search_positions grammar) in it and shows it.
	OpenView(name, query string) error
	// ShowPosition shows one position on the board.
	ShowPosition(id int64) error
}

// maxViewName bounds a view's tab label: a sentence names it, a paragraph
// would not fit the tab bar.
const maxViewName = 40

// ViewName shortens a sentence to a view tab's label, on a word boundary.
func ViewName(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= maxViewName {
		return s
	}
	cut := string(r[:maxViewName])
	if i := strings.LastIndex(cut, " "); i > maxViewName/2 {
		cut = cut[:i]
	}
	return strings.TrimSpace(cut) + "…"
}

// DisplayTools registers open_view and show_position over d. Both are
// read-only for the database — they change what is on screen, not what is
// stored — so a read-only server offers them.
func DisplayTools(d Display) Extension {
	return func(tb *Toolbox) {
		type viewIn struct {
			Name  string `json:"name,omitempty" jsonschema:"the tab label, a few words; default: the query"`
			Query string `json:"query" jsonschema:"the search, in the search_positions grammar"`
		}
		Add(tb, Reads, &sdk.Tool{Name: "open_view", Title: "Open a view on screen",
			Description: "Open a new view tab in the blunderDB window, run a search in it (search_positions grammar) and switch to it, so the user sees the positions and the search's tokens in the Search panel. Check the query with search_positions first."},
			func(ctx context.Context, req *sdk.CallToolRequest, a viewIn) (any, error) {
				q := strings.TrimSpace(a.Query)
				if q == "" {
					return nil, errors.New("query is empty")
				}
				var parsed struct {
					Diags     []obj  `json:"diags"`
					Canonical string `json:"canonical"`
				}
				if err := tb.Engine.Call(ctx, req, "search.parse", obj{"query": q}, &parsed); err != nil {
					return nil, err
				}
				name := ViewName(a.Name)
				if name == "" {
					name = ViewName(q)
				}
				if err := d.OpenView(name, q); err != nil {
					return nil, err
				}
				return obj{"opened": name, "canonical": parsed.Canonical, "notes": parsed.Diags}, nil
			})

		type showIn struct {
			PositionID int64 `json:"positionId" jsonschema:"the position id"`
		}
		Add(tb, Reads, &sdk.Tool{Name: "show_position", Title: "Show a position on screen",
			Description: "Show one position of the database on the blunderDB board, by its id."},
			func(ctx context.Context, req *sdk.CallToolRequest, a showIn) (any, error) {
				// Loaded first: an id the database does not hold is refused here,
				// not left for the screen to fail on silently.
				if _, err := tb.position(ctx, req, positionRef{PositionID: a.PositionID}); err != nil {
					return nil, err
				}
				if err := d.ShowPosition(a.PositionID); err != nil {
					return nil, err
				}
				return obj{"shown": a.PositionID}, nil
			})
	}
}
