package domain

import (
	"reflect"
	"testing"
)

func TestSameDicePairings(t *testing.T) {
	alias := func(a, c string) PlayerAlias { return PlayerAlias{Alias: a, Canonical: c} }
	pairing := func(as ...PlayerAlias) AliasPairing { return AliasPairing{Aliases: as} }
	cases := []struct {
		name           string
		p1, p2, o1, o2 string
		want           []AliasPairing
	}{
		{"one name shared, same seat", "A. Fairweather", "Bram", "Ada Fairweather", "Bram",
			[]AliasPairing{pairing(alias("A. Fairweather", "Ada Fairweather"))}},
		{"one name shared, other seat", "bram ", "A. F", "Ada", "Bram",
			[]AliasPairing{pairing(alias("A. F", "Ada"))}},
		{"no name shared: both readings, seat order first", "AF", "BV", "Ada", "Bram",
			[]AliasPairing{
				pairing(alias("AF", "Ada"), alias("BV", "Bram")),
				pairing(alias("AF", "Bram"), alias("BV", "Ada")),
			}},
		{"an empty name gives no alias", "", "BV", "Ada", "Bram",
			[]AliasPairing{pairing(alias("BV", "Bram")), pairing(alias("BV", "Ada"))}},
	}
	for _, c := range cases {
		if got := SameDicePairings(c.p1, c.p2, c.o1, c.o2); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}
