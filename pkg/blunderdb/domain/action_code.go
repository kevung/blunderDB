package domain

// Action labels are stored as integer codes (ADR-0071): move.move_type,
// move.cube_action and analysis.best_cube_action hold a handful of distinct
// labels repeated over millions of rows. The labels stay free text in the
// domain — the importers write them verbatim and filters match them verbatim —
// so a code is a lossless stand-in for one exact string, never a
// normalisation: "No Double" and "NoDouble" keep two codes.
//
// actionLabels is the fixed part of the code space: the code of a label is
// its index. The list is append-only — a code written into a library must
// keep its label forever. A label missing from it is not refused: the storage
// registers it in the library's action_label table under a code from
// FirstRegisteredActionCode up.
var actionLabels = [...]string{
	"",
	// move.move_type
	"checker", "cube",
	// Cube actions as the importers and the engine write them.
	"No Double", "Double, Take", "Double, Pass",
	"Too good to double, pass", "Too good to double, take",
	"Double", "Take", "Pass", "Drop", "Beaver",
	"Double/Take", "Double/Pass", "Double/Beaver",
	"NoDouble", "No double", "Double, take", "Double, pass",
	"Too good to double", "Too good", "TG",
	"Redouble", "No Redouble", "Redouble, Take", "Redouble, Pass",
	"Double / Take", "Double / Pass", "Double / Prendre", "Double / Refuser", "Double / Reject",
	"No redouble", "Redouble, take", "Redouble, pass",
	"Too good to redouble, pass", "Too good to redouble, take",
	"Double, Beaver", "Double, beaver", "Too Good", "No Double, Take", "No Double, Pass",
}

// FirstRegisteredActionCode is the first code the storage hands out to a
// label actionLabels does not hold. Codes below it are fixed by actionLabels.
const FirstRegisteredActionCode = 1000

var actionCodes = func() map[string]int64 {
	m := make(map[string]int64, len(actionLabels))
	for i, l := range actionLabels {
		if _, dup := m[l]; dup {
			panic("domain: duplicate action label " + l)
		}
		m[l] = int64(i)
	}
	return m
}()

// ActionCode returns the fixed code of label, and false when label has none
// (the storage then registers it).
func ActionCode(label string) (int64, bool) {
	c, ok := actionCodes[label]
	return c, ok
}

// ActionLabels returns the fixed labels, the code of each being its index.
func ActionLabels() []string { return actionLabels[:] }
