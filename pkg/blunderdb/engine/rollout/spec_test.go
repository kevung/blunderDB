package rollout

import "testing"

func TestParseSpec(t *testing.T) {
	std := Standard()
	std.Ply = 1
	free := Fast()
	free.MaxGames, free.MinGames, free.Truncation = 648, 108, 0
	cases := map[string]Settings{
		"":                             Fast(),
		"rapide":                       Fast(),
		"standard,ply=1":               std,
		"libre games=648 truncation=0": free,
		"games=648;truncation=0":       free,
	}
	for spec, want := range cases {
		got, err := ParseSpec(spec)
		if err != nil || got != want {
			t.Errorf("ParseSpec(%q) = %+v, %v; want %+v", spec, got, err, want)
		}
	}
	for _, bad := range []string{"turbo", "games=100", "ply=1,standard", "depth=3", "jsd=x"} {
		if _, err := ParseSpec(bad); err == nil {
			t.Errorf("ParseSpec(%q) accepted", bad)
		}
	}
}
