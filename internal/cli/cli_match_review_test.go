package cli

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The summary's review gives each player's L₇ over the match, with its
// interval over the games, as the GUI's review does.
func TestWriteMatchReviewShowsL7(t *testing.T) {
	var r storage.MatchReview
	r.Players[0].MWC7 = domain.MatchMWC7(0.1, 7, []float64{0.01, 0.06, 0.03})
	var sb strings.Builder
	writeMatchReview(&sb, [2]string{"Ann", "Abe"}, r)
	out := sb.String()
	if want := "MWC loss, 7-point scale: " + formatMWC7(r.Players[0].MWC7); !strings.Contains(out, want) {
		t.Errorf("review lacks %q:\n%s", want, out)
	}
	if strings.Count(out, "7-point scale") != 1 {
		t.Errorf("a seat without L₇ shows one:\n%s", out)
	}
}

// A review that cannot be read fails the summary instead of vanishing from it.
func TestMatchSummaryReportsAReviewError(t *testing.T) {
	cli, _ := setupCLIWithDB(t)
	if err := cli.db.Close(); err != nil {
		t.Fatal(err)
	}
	m := &Match{ID: 1, Player1Name: "Ann", Player2Name: "Abe", MatchLength: 7}
	if _, err := cli.formatMatchSummary(m, nil, nil); err == nil || !strings.Contains(err.Error(), "match review") {
		t.Errorf("summary over an unreadable review: err %v", err)
	}
}
