package direction

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The constant is stored with every Direction: it must name the module version go.mod requires.
func TestEngineVersionMatchesGoMod(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	const module = "github.com/PileOfCells/backgammon-tournoi"
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "require "))
		if len(f) >= 2 && f[0] == module {
			if f[1] != EngineVersion {
				t.Fatalf("EngineVersion = %q, go.mod requires %s", EngineVersion, f[1])
			}
			return
		}
	}
	t.Fatalf("go.mod has no require for %s", module)
}
