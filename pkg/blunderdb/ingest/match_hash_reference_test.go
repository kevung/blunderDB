package ingest

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestMatchHashReference freezes both content hashes of every real match under
// testdata/. Existing databases deduplicate on these values: a change to a
// hashing function, or to a parser upstream of it, would let a re-import of an
// already stored match slip in as a new one.
//
// Fixtures of one match in several formats share the same canonical hash; the
// format-specific hash differs on purpose.
func TestMatchHashReference(t *testing.T) {
	t.Parallel()
	const (
		canonTest    = "42f3a000f750128ad7989e693527f2042ad55113c32a9af69bec158a6864a8af"
		canonCharlot = "ea2ee0743ef35e3aabf5e76586ce614bb8d39316e9a26c71d5e4956921967391"
		aachen       = "2024-08-10-Aachen-1x11pt-1x7pt-2x7ptDoubleConsultation"
	)
	cases := []struct{ file, matchHash, canonical string }{
		{"test.xg", "04bea47408f39652f9e148f6ac37d7483ccfd441c87ae9c5cde284c53b37f3bd", canonTest},
		{"test.sgf", "a0699549243906e9683ee419f2c9545da0242a9b623238017896945e24a6337b", canonTest},
		{"test.mat", "664609786127f6eec9f50d925bca270b790736d7a2caaf75444b78cbf8e60a8b", canonTest},
		{"charlot1-charlot2_7p_2025-11-08-2305.xg", "0b50c2ca7cfa8a5dbd5eaffa5c2104aa09c6cf8974aa56a82be8e28b9bd72782", canonCharlot},
		{"charlot1-charlot2_7p_2025-11-08-2305.sgf", "4fc0f5b0af8b8be712c12de01206b2884a6e2b712135105d0d1160791640921b", canonCharlot},
		{"charlot1-charlot2_7p_2025-11-08-2305.mat", "4a579baff11e39c46e6d3b74d30d7be3977c866e192b5fdcba5e1eed4512666f", canonCharlot},
		{"HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg", "009b6ceaafe4e2bef4dc87d07b6085bdb8179bbf50bde3336ec22b2bf79e7b58", "3064337bbff36a5c54af84480bc1018822aa66d99838f1e1cbb37304fbec9d6b"},
		{"match_with_comment.xg", "452a25f0d5c5de6d9ddc15af4827b4411a8f92adf649a7ddf371f53d35236b8a", "3d868434dcd53918f7a2cb5bf711ad75afc75433f927878cda66a0592d14eda2"},
		{aachen + "/double/Lux Heuler-Franzosen 7 point match 12.08.2024.xg", "0945e99f89ee324a5d36da4c084668e912d49e44c235b5dd853962d3ea046d87", "92831bad91acdd9c2d563a5eae9938f67d0240d33166f4d554374202baf04dce"},
		{aachen + "/double/Squire-Jørgensen-Harmand-Unger 7 point match 18-08-2024.xg", "9ca72259232102c1fb47872b91ef3b3b9d3110ea20b175c4df97054f57093e3b", "586cf01527aadd279144c87084085f708536a84dd6fae4f31d4cfeefa322890a"},
		{aachen + "/double/ronde1-UngerHarmand-FriebeJacobi-7pt.xg", "49a60f751e5bb4a9c003e130c6a13d1266d77b28ea8e61d443e57fcdea79ec10", "a08252ce060ce42122739c3e27c3edf2e28723626900f4dfff343cbdbd05429b"},
		{aachen + "/double/ronde3-UngerHarmand-HuyckLarsen.xg", "ecd10f9b11669bd83e90d13215b7c495accc3a21ff425521c84c8703a9969793", "fd94159d77cc04be2f8577228d1b3a63e376fa98b1757a5343b8277096597ad0"},
		{aachen + "/main-intermediate/1-Intermediate-Round1-Unger-Kozoman-11pt.xg", "350941484f737e97c273dbde4c8b71e3c28f574e73fb3e1cf2422a8938c72a0c", "4f1968fdabb2f9f0ee6b33def86eeba8698e29b504a1c946688ed50a0749a068"},
		{aachen + "/main-intermediate/2-Intermediate-Round2-Unger-Lahme-7pt.xg", "1c2f9cf7dea435bf98b4ca2a7465be40825800bc019ca8a16b622197111ec950", "02a0f6a5d10daea86e35cd7ef438295afe196789ff679dec8d43f2e249033062"},
		{"gnubg_roll_then_resign_1p.mat", "1d9695b222212ee08ed91fe1dae1b22e23266a15eb650c71b72afd08bfe88b46", "f17ac150a8b0558a1c287c5ca36b645b40f6d0a3edc2eac2430ef74ad9151b1f"},
		{"gnubg_selfplay_drops_7p.mat", "184d9382d0346e2b0ec0057784b4faba52fbd51c4b241d48a80598cd1af5fe6f", "22ad3adb887f9c0032c6e8cf2000f32ccad32ddfdb58a2e880fda34ef89d6571"},
		{"TachiAI_V_player_Nov_2__2025__16_55.bgf", "b20af907ac51f33250257cfc10fb4be97efd93fb871844324f07507f517cb758", "99551fb744f765e4801604858412a02c80cfa79b36433cf8eb6446c78077d942"},
	}

	covered := map[string]bool{}
	for _, c := range cases {
		path := filepath.Join("..", "..", "..", "testdata", filepath.FromSlash(c.file))
		covered[path] = true
		t.Run(c.file, func(t *testing.T) {
			t.Parallel()
			var g *MatchGraph
			var err error
			switch strings.ToLower(filepath.Ext(path)) {
			case ".xg":
				g, err = MapXG(path)
			case ".bgf":
				g, err = MapBGF(path)
			default:
				g, err = MapGnuBG(path)
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.Match.MatchHash != c.matchHash {
				t.Errorf("match_hash = %s, want %s", g.Match.MatchHash, c.matchHash)
			}
			if g.Match.CanonicalHash != c.canonical {
				t.Errorf("canonical_hash = %s, want %s", g.Match.CanonicalHash, c.canonical)
			}
		})
	}

	// A match fixture added to testdata/ must get its reference here.
	root := filepath.Join("..", "..", "..", "testdata")
	for _, pat := range []string{"*.xg", "*/*/*.xg", "*.sgf", "*.mat", "*.bgf"} {
		found, err := filepath.Glob(filepath.Join(root, pat))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range found {
			if !covered[f] {
				t.Errorf("%s has no reference hash", f)
			}
		}
	}
}
