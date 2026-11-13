package searchquery

import (
	"errors"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/reach"
)

// The GUI words the refusals of `like` from the code heading their text, not
// from the English sentence: the codes are a contract with the front end
// (frontend/src/utils/refusal.js).
func TestLikeRefusalCodes(t *testing.T) {
	_, diags := Parse("s like42<1*")
	starred := Refusal(diags)
	if !errors.Is(starred, ErrLikeStarred) || !errors.Is(starred, ErrRefused) {
		t.Fatalf("starred refusal = %v, want ErrLikeStarred and ErrRefused", starred)
	}
	if !strings.HasPrefix(starred.Error(), "refused query: like-starred: like42<1*: ") {
		t.Errorf("starred refusal text = %q", starred.Error())
	}
	if !strings.HasPrefix(ErrLikeUnranked.Error(), "refused query: like-unranked: ") {
		t.Errorf("unranked refusal text = %q", ErrLikeUnranked.Error())
	}
	if err := reach.CheckOrder(reach.MaxOrder + 1); !strings.HasPrefix(err.Error(), "like-order: ") {
		t.Errorf("order refusal text = %q", err.Error())
	}
}
