package domain

import "testing"

// --- SearchOrderByClause / MatchOrderByClause -------------------------------

func TestSearchOrderByClause(t *testing.T) {
	cases := []struct {
		sort string
		want string
	}{
		{"error", "a.best_move_equity_error DESC NULLS LAST, p.id"},
		{"winrate", "a.player1_win_rate DESC NULLS LAST, p.id"},
		{"close", "a.is_close_cube DESC NULLS LAST, p.id"},
		{"", "p.id"},
		{"unknown-key", "p.id"},
	}
	for _, c := range cases {
		t.Run(c.sort, func(t *testing.T) {
			if got := SearchOrderByClause(c.sort); got != c.want {
				t.Errorf("SearchOrderByClause(%q) = %q, want %q", c.sort, got, c.want)
			}
		})
	}
}

func TestMatchOrderByClause(t *testing.T) {
	cases := []struct {
		sort string
		want string
	}{
		{"date_asc", "COALESCE(m.match_date, m.import_date) ASC, m.id ASC"},
		{"length_desc", "m.match_length DESC, m.id DESC"},
		{"length_asc", "m.match_length ASC, m.id ASC"},
		{"opponent", "LOWER(m.player1_name) ASC, LOWER(m.player2_name) ASC, m.id ASC"},
		{"", "COALESCE(m.match_date, m.import_date) DESC, m.id DESC"},
		{"date", "COALESCE(m.match_date, m.import_date) DESC, m.id DESC"},
		{"bogus", "COALESCE(m.match_date, m.import_date) DESC, m.id DESC"},
	}
	for _, c := range cases {
		t.Run(c.sort, func(t *testing.T) {
			if got := MatchOrderByClause(c.sort); got != c.want {
				t.Errorf("MatchOrderByClause(%q) = %q, want %q", c.sort, got, c.want)
			}
		})
	}
}

// --- EffectiveIncludeFilter --------------------------------------------------

func TestEffectiveIncludeFilter(t *testing.T) {
	blankBoard := func() Board {
		var b Board
		for i := range b.Points {
			b.Points[i] = Point{Checkers: 0, Color: None}
		}
		return b
	}

	t.Run("no exclude constraint leaves include untouched", func(t *testing.T) {
		include := Position{Board: blankBoard()}
		include.Board.Points[6] = Point{Checkers: 2, Color: Black}
		exclude := Position{Board: blankBoard()}

		got := EffectiveIncludeFilter(include, exclude)
		if got.Board.Points[6] != (Point{Checkers: 2, Color: Black}) {
			t.Errorf("point 6 should be untouched: got %+v", got.Board.Points[6])
		}
	})

	t.Run("compatible: include 2, exclude 3 on same colour keeps the include (doc example)", func(t *testing.T) {
		include := Position{Board: blankBoard()}
		include.Board.Points[6] = Point{Checkers: 2, Color: Black}
		exclude := Position{Board: blankBoard()}
		exclude.Board.Points[6] = Point{Checkers: 3, Color: Black}

		got := EffectiveIncludeFilter(include, exclude)
		if got.Board.Points[6] != (Point{Checkers: 2, Color: Black}) {
			t.Errorf("include should survive when exclude count > include count: got %+v", got.Board.Points[6])
		}
	})

	t.Run("contradiction: include 2, exclude 1 on same colour drops the include (doc example)", func(t *testing.T) {
		include := Position{Board: blankBoard()}
		include.Board.Points[6] = Point{Checkers: 2, Color: Black}
		exclude := Position{Board: blankBoard()}
		exclude.Board.Points[6] = Point{Checkers: 1, Color: Black}

		got := EffectiveIncludeFilter(include, exclude)
		if got.Board.Points[6] != (Point{}) {
			t.Errorf("include should be dropped on contradiction: got %+v", got.Board.Points[6])
		}
	})

	t.Run("contradiction: include N, exclude same N on same colour drops the include (I >= E boundary)", func(t *testing.T) {
		include := Position{Board: blankBoard()}
		include.Board.Points[6] = Point{Checkers: 2, Color: Black}
		exclude := Position{Board: blankBoard()}
		exclude.Board.Points[6] = Point{Checkers: 2, Color: Black}

		got := EffectiveIncludeFilter(include, exclude)
		if got.Board.Points[6] != (Point{}) {
			t.Errorf("I == E must contradict (no position has >=2 and <=1): got %+v", got.Board.Points[6])
		}
	})

	t.Run("different colours never contradict", func(t *testing.T) {
		include := Position{Board: blankBoard()}
		include.Board.Points[6] = Point{Checkers: 2, Color: Black}
		exclude := Position{Board: blankBoard()}
		exclude.Board.Points[6] = Point{Checkers: 1, Color: White}

		got := EffectiveIncludeFilter(include, exclude)
		if got.Board.Points[6] != (Point{Checkers: 2, Color: Black}) {
			t.Errorf("include should survive when exclude names a different colour: got %+v", got.Board.Points[6])
		}
	})

	t.Run("ExcludeEmpty marker always wins regardless of count", func(t *testing.T) {
		include := Position{Board: blankBoard()}
		include.Board.Points[6] = Point{Checkers: 5, Color: Black}
		exclude := Position{Board: blankBoard()}
		exclude.Board.Points[6] = Point{Checkers: 1, Color: ExcludeEmpty}

		got := EffectiveIncludeFilter(include, exclude)
		if got.Board.Points[6] != (Point{}) {
			t.Errorf("a \"must be empty\" marker should always drop the include: got %+v", got.Board.Points[6])
		}
	})

	t.Run("include is a value copy: mutating the result does not affect the input", func(t *testing.T) {
		include := Position{Board: blankBoard()}
		include.Board.Points[6] = Point{Checkers: 2, Color: Black}
		exclude := Position{Board: blankBoard()}
		exclude.Board.Points[6] = Point{Checkers: 1, Color: Black}

		_ = EffectiveIncludeFilter(include, exclude)
		if include.Board.Points[6] != (Point{Checkers: 2, Color: Black}) {
			t.Errorf("EffectiveIncludeFilter must not mutate its include argument: got %+v", include.Board.Points[6])
		}
	})
}

// --- ContainsAnyCheckerOf -----------------------------------------------------

func TestContainsAnyCheckerOf(t *testing.T) {
	blankBoard := func() Board {
		var b Board
		for i := range b.Points {
			b.Points[i] = Point{Checkers: 0, Color: None}
		}
		return b
	}

	t.Run("no filter points set: nothing excluded", func(t *testing.T) {
		p := Position{Board: blankBoard()}
		p.Board.Points[6] = Point{Checkers: 5, Color: Black}
		filter := Position{Board: blankBoard()}

		if p.ContainsAnyCheckerOf(filter) {
			t.Error("an empty filter should never match")
		}
	})

	t.Run("position has at least as many checkers of the filtered colour: matches", func(t *testing.T) {
		p := Position{Board: blankBoard()}
		p.Board.Points[6] = Point{Checkers: 3, Color: Black}
		filter := Position{Board: blankBoard()}
		filter.Board.Points[6] = Point{Checkers: 2, Color: Black}

		if !p.ContainsAnyCheckerOf(filter) {
			t.Error("position has >= the excluded count: should match (be rejected by the caller)")
		}
	})

	t.Run("position has fewer checkers than excluded: does not match", func(t *testing.T) {
		p := Position{Board: blankBoard()}
		p.Board.Points[6] = Point{Checkers: 1, Color: Black}
		filter := Position{Board: blankBoard()}
		filter.Board.Points[6] = Point{Checkers: 2, Color: Black}

		if p.ContainsAnyCheckerOf(filter) {
			t.Error("position has fewer than the excluded count: should not match")
		}
	})

	t.Run("wrong colour: does not match", func(t *testing.T) {
		p := Position{Board: blankBoard()}
		p.Board.Points[6] = Point{Checkers: 5, Color: White}
		filter := Position{Board: blankBoard()}
		filter.Board.Points[6] = Point{Checkers: 1, Color: Black}

		if p.ContainsAnyCheckerOf(filter) {
			t.Error("different colour on the point should not match")
		}
	})

	t.Run("OR semantics: any one matching point is enough", func(t *testing.T) {
		p := Position{Board: blankBoard()}
		p.Board.Points[10] = Point{Checkers: 1, Color: White}
		filter := Position{Board: blankBoard()}
		filter.Board.Points[6] = Point{Checkers: 1, Color: Black}  // does not match
		filter.Board.Points[10] = Point{Checkers: 1, Color: White} // matches

		if !p.ContainsAnyCheckerOf(filter) {
			t.Error("expected OR semantics: one matching point is enough")
		}
	})

	t.Run("ExcludeEmpty marker: matches when the point holds any checker", func(t *testing.T) {
		p := Position{Board: blankBoard()}
		p.Board.Points[6] = Point{Checkers: 1, Color: Black}
		filter := Position{Board: blankBoard()}
		filter.Board.Points[6] = Point{Checkers: 1, Color: ExcludeEmpty}

		if !p.ContainsAnyCheckerOf(filter) {
			t.Error("a \"must be empty\" marker should match when the point is occupied")
		}
	})

	t.Run("ExcludeEmpty marker: does not match when the point is empty", func(t *testing.T) {
		p := Position{Board: blankBoard()}
		filter := Position{Board: blankBoard()}
		filter.Board.Points[6] = Point{Checkers: 1, Color: ExcludeEmpty}

		if p.ContainsAnyCheckerOf(filter) {
			t.Error("a \"must be empty\" marker should not match an actually-empty point")
		}
	})
}
