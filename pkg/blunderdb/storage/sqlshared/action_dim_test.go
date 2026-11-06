package sqlshared

import (
	"errors"
	"testing"
)

func TestActionDim(t *testing.T) {
	for _, c := range []struct {
		in   any
		want int64
	}{{nil, cellNull}, {"", cellNull}, {"3", 3}, {int64(3), 3}, {int32(2), 2}, {int16(1), 1}, {int(4), 4}} {
		if got, err := actionDim(c.in); err != nil || got != c.want {
			t.Errorf("actionDim(%#v) = %d, %v; want %d", c.in, got, err, c.want)
		}
	}
	for _, in := range []any{"x", []byte("1"), 1.5} {
		if _, err := actionDim(in); !errors.Is(err, ErrUnexpectedActionType) {
			t.Errorf("actionDim(%#v) = %v; want ErrUnexpectedActionType", in, err)
		}
	}
}

func TestSQLTimeRefusesUnknownType(t *testing.T) {
	var st sqlTime
	if err := st.Scan(int64(5)); err == nil {
		t.Error("an integer timestamp column was accepted")
	}
	if err := st.Scan(nil); err != nil {
		t.Errorf("NULL: %v", err)
	}
}
