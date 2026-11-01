package domain

import "fmt"

// MaxDerivationMove bounds the move text of a derivation.
const MaxDerivationMove = 100

// Derivation says that a position was reached from another one by playing
// Move: the variant's link to the position it comes from. The variant is an
// ordinary position, analysed like any other; the link only records where it
// came from. It says nothing of anyone else.
type Derivation struct {
	// FromID is the id of the position this one derives from.
	FromID int64 `json:"from_id"`
	// Move is the move played from that position, free text ("13/7 24/23"),
	// empty when unknown.
	Move string `json:"move"`
}

// Validate reports why a derivation of position id cannot be recorded.
func (d Derivation) Validate(id int64) error {
	if d.FromID <= 0 {
		return fmt.Errorf("derivation origin %d: want a position id", d.FromID)
	}
	if d.FromID == id {
		return fmt.Errorf("position %d cannot derive from itself", id)
	}
	if len(d.Move) > MaxDerivationMove {
		return fmt.Errorf("derivation move of %d bytes: at most %d", len(d.Move), MaxDerivationMove)
	}
	return nil
}
