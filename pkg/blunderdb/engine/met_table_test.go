package engine

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func readMET(t *testing.T, name string) *MET {
	t.Helper()
	data, err := os.ReadFile("testdata/met/" + name)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ParseGnubgMET(data)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return m
}

// gnubg's own Kazaross-XG2.xml is the built-in table under another file
// name: its digest must be the built-in one, or importing it would mark
// every analysis "different".
func TestParseGnubgMET_KazarossXG2IsTheBuiltIn(t *testing.T) {
	m := readMET(t, "Kazaross-XG2.xml")
	if m.Length != 25 {
		t.Fatalf("length %d, want 25", m.Length)
	}
	if got, want := m.Digest(), KazarossXG2Digest(); got != want {
		t.Fatalf("digest %s, want the built-in %s", got, want)
	}
	for s0 := 0; s0 < 11; s0++ {
		for s1 := 0; s1 < 11; s1++ {
			for _, cr := range []bool{false, true} {
				if cr && s0 != 10 && s1 != 10 {
					continue
				}
				for pts := 1; pts <= 4; pts++ {
					got := m.GetME(s0, s1, 11, 0, pts, 1, cr)
					want := GnuBGGetME(s0, s1, 11, 0, pts, 1, cr)
					if got != want {
						t.Fatalf("GetME(%d,%d,11,pts=%d,cr=%v) = %v, built-in %v", s0, s1, pts, cr, got, want)
					}
				}
			}
		}
	}
}

func TestParseGnubgMET_AnotherTableAnswersWithItsValues(t *testing.T) {
	m := readMET(t, "Rockwell-Kazaross.xml")
	if m.Digest() == KazarossXG2Digest() {
		t.Fatal("Rockwell-Kazaross shares the built-in digest")
	}
	if !strings.Contains(m.Name, "Rockwell") {
		t.Errorf("name %q", m.Name)
	}
	// 3-away/5-away pre-Crawford, player 0 needing 3 after winning 2 of 7.
	if got := m.GetME(4, 2, 7, 0, 0, 0, false); got != m.preAt(2, 4) {
		t.Errorf("GetME reads %v, table holds %v", got, m.preAt(2, 4))
	}
	if m.preAt(2, 4) == metPre(2, 4) {
		t.Error("Rockwell-Kazaross and Kazaross-XG2 agree at 3-away/5-away: the test reads nothing")
	}
	// Past the table's length, the built-in table answers.
	if got := m.preAt(40, 40); got != metPre(40, 40) {
		t.Errorf("preAt beyond the length = %v, built-in %v", got, metPre(40, 40))
	}
}

func TestParseGnubgMET_RefusesWhatItCannotHold(t *testing.T) {
	data, err := os.ReadFile("testdata/met/zadeh.xml")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{
		"parametric": data,
		"not xml":    []byte("hello"),
		"bad value":  []byte(`<met><info><name>x</name><length>1</length></info><pre-crawford-table type="explicit"><row><me>1.5</me></row></pre-crawford-table><post-crawford-table type="explicit" player="both"><row><me>0.5</me></row></post-crawford-table></met>`),
		"short row":  []byte(`<met><info><name>x</name><length>2</length></info><pre-crawford-table type="explicit"><row><me>0.5</me></row><row><me>0.5</me></row></pre-crawford-table><post-crawford-table type="explicit" player="both"><row><me>0.5</me><me>0.4</me></row></post-crawford-table></met>`),
	}
	for name, in := range cases {
		if _, err := ParseGnubgMET(in); !errors.Is(err, ErrMETFormat) {
			t.Errorf("%s: err %v, want ErrMETFormat", name, err)
		}
	}
}

func TestMETDigestIgnoresLayoutAndName(t *testing.T) {
	a := `<met><info><name>A</name><length>1</length></info><pre-crawford-table type="explicit"><row><me>0.5</me></row></pre-crawford-table><post-crawford-table type="explicit" player="both"><row><me>0.5</me></row></post-crawford-table></met>`
	b := "<!-- c -->\n<met>\n <info><name>B</name>\n<length> 1 </length></info>\n<pre-crawford-table type=\"explicit\">\n<row> <me>0.50000</me> </row></pre-crawford-table><post-crawford-table type=\"explicit\" player=\"both\"><row><me>.5</me></row></post-crawford-table></met>"
	ma, err := ParseGnubgMET([]byte(a))
	if err != nil {
		t.Fatal(err)
	}
	mb, err := ParseGnubgMET([]byte(b))
	if err != nil {
		t.Fatal(err)
	}
	if ma.Digest() != mb.Digest() {
		t.Error("layout or name changed the digest")
	}
}
