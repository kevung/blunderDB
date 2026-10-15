package database

import "context"

// migrate_2_41_0_to_2_42_0 lets a review be undone: anki_review_log gains
// the card's scheduling state just before each review (prev_*). Every review
// already logged keeps NULL there — its prior state was never kept, so it
// cannot be undone. A library still lacking the table gets it with the
// columns from EnsureSchema afterwards.
func (d *Database) migrate_2_41_0_to_2_42_0(context.Context) error {
	present, err := d.columnExists("anki_review_log", "id")
	if err != nil || !present {
		return err
	}
	for _, col := range []string{
		"prev_due DATETIME",
		"prev_stability REAL",
		"prev_difficulty REAL",
		"prev_elapsed_days INTEGER",
		"prev_scheduled_days INTEGER",
		"prev_reps INTEGER",
		"prev_lapses INTEGER",
		"prev_state INTEGER",
		"prev_last_review DATETIME",
	} {
		if err := d.addColumn("anki_review_log", col); err != nil {
			return err
		}
	}
	return nil
}
