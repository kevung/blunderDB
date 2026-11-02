package domain

import "fmt"

// A drawing is the marks a player lays over a position: arrows and squares in
// a closed set of colours. It says nothing of anyone else; it belongs to the
// position it is drawn on.
const (
	DrawingArrow  = "arrow"
	DrawingSquare = "square"

	// MaxDrawingMarks bounds a drawing; a board with more is unreadable.
	MaxDrawingMarks = 40
	// MaxDrawingPoint is the highest point a mark may name: 1-24 are the
	// points, 0 and 25 the bar and the off tray.
	MaxDrawingPoint = 25
)

// DrawingColors is the closed set of colours a mark can take.
var DrawingColors = []string{"red", "green", "blue", "yellow"}

// DrawingMark is one arrow (From to To) or one square (on From; To equals
// From) of a position's drawing.
type DrawingMark struct {
	Kind  string `json:"kind"`
	From  int    `json:"from"`
	To    int    `json:"to"`
	Color string `json:"color"`
}

// Validate reports why a mark cannot be drawn, nil when it can. A square is
// normalised by the store: its To is its From.
func (m DrawingMark) Validate() error {
	switch m.Kind {
	case DrawingArrow, DrawingSquare:
	default:
		return fmt.Errorf("drawing kind %q: want %q or %q", m.Kind, DrawingArrow, DrawingSquare)
	}
	ok := false
	for _, c := range DrawingColors {
		ok = ok || c == m.Color
	}
	if !ok {
		return fmt.Errorf("drawing colour %q: want one of %v", m.Color, DrawingColors)
	}
	for _, p := range []int{m.From, m.To} {
		if p < 0 || p > MaxDrawingPoint {
			return fmt.Errorf("drawing point %d: want 0 to %d", p, MaxDrawingPoint)
		}
	}
	if m.Kind == DrawingArrow && m.From == m.To {
		return fmt.Errorf("drawing arrow from point %d to itself", m.From)
	}
	return nil
}

// ValidateDrawing checks a whole drawing.
func ValidateDrawing(marks []DrawingMark) error {
	if len(marks) > MaxDrawingMarks {
		return fmt.Errorf("drawing of %d marks: at most %d", len(marks), MaxDrawingMarks)
	}
	for i, m := range marks {
		if err := m.Validate(); err != nil {
			return fmt.Errorf("mark %d: %w", i, err)
		}
	}
	return nil
}
