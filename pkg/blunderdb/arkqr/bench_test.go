package arkqr

import "testing"

func benchMatches(b *testing.B) []Match {
	var out []Match
	for _, cm := range loadCorpus(b) {
		out = append(out, Match{Doc: cm.doc})
	}
	return out
}

func BenchmarkEncode(b *testing.B) {
	ms := benchMatches(b)
	b.ReportAllocs()
	for b.Loop() {
		for _, m := range ms {
			if _, err := Encode(m); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkDecode(b *testing.B) {
	var streams [][]byte
	for _, m := range benchMatches(b) {
		data, err := Encode(m)
		if err != nil {
			b.Fatal(err)
		}
		streams = append(streams, data)
	}
	b.ReportAllocs()
	for b.Loop() {
		for _, data := range streams {
			if _, err := Decode(data); err != nil {
				b.Fatal(err)
			}
		}
	}
}
