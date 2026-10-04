// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math/rand"
	"os"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

func loadTestMET(t *testing.T, name string) *engine.MET {
	t.Helper()
	data, err := os.ReadFile("../testdata/met/" + name)
	if err != nil {
		t.Fatal(err)
	}
	m, err := engine.ParseGnubgMET(data)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// The library's table reaches the verdict at a match score, never at money,
// and a table equal to the built-in one answers exactly what nil does.
func TestEvaluatePositionWithMET(t *testing.T) {
	rockwell := loadTestMET(t, "Rockwell-Kazaross.xml")
	builtin := loadTestMET(t, "Kazaross-XG2.xml")
	searcher, err := NewBatchSearcher(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(20261004))

	differed := false
	for attempt := 0; attempt < 60; attempt++ {
		pos := randomBoard(rng, domain.White)
		pos.Cube = domain.Cube{Owner: domain.None, Value: 0}
		pos.Dice = [2]int{0, 0}
		pos.Score = [2]int{4, 6}
		if attempt%5 == 0 {
			pos.Score = [2]int{-1, -1}
		}

		base, err := EvaluatePositionWith(searcher, pos, 0, 0, 0)
		if err != nil || base.Cube == nil {
			continue
		}
		same, err := EvaluatePositionWithMET(searcher, pos, builtin, 0, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(base, same) {
			t.Fatalf("attempt %d: an imported Kazaross-XG2 changed the verdict", attempt)
		}
		other, err := EvaluatePositionWithMET(searcher, pos, rockwell, 0, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		if pos.IsMoney() {
			if !reflect.DeepEqual(base, other) {
				t.Fatalf("attempt %d: the MET changed a money verdict", attempt)
			}
			continue
		}
		if !reflect.DeepEqual(base, other) {
			differed = true
		}
	}
	if !differed {
		t.Fatal("Rockwell-Kazaross never changed a 4-away/6-away cube verdict")
	}
}
