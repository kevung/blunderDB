package gammonnet

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// fusedOp matches the instruction column of a fused multiply-add in the
// compiler's -S listing: VFMADD231SD and kin on amd64, FMADDD/FMSUBD/
// FNMADDD/FNMSUBD on arm64 (and the same mnemonics on riscv64, ppc64, s390x).
var fusedOp = regexp.MustCompile(`\t(V)?F(N)?M(ADD|SUB)\w*\t`)

// fmaTargets are the targets where the Go compiler fuses a*b + c: arm64
// always, amd64 from GOAMD64=v3. amd64 at the default level never fuses, so
// it is the reference the other two must reproduce bit for bit.
var fmaTargets = []struct{ name, goarch, goamd64 string }{
	{"arm64", "arm64", ""},
	{"amd64-v3", "amd64", "v3"},
}

// TestEngineEmitsNoFusedMultiplyAdd holds the engine's cross-platform bit
// parity at the instruction level: no package under engine/ may compile to
// a fused multiply-add on any target, since a fused a*b + c rounds once
// where amd64 rounds twice. Every product summed on the evaluation path is
// written float64(a*b), a conversion the compiler must round and so cannot
// fuse. The canary package, built alongside, proves the detector still
// recognises a fused instruction.
func TestEngineEmitsNoFusedMultiplyAdd(t *testing.T) {
	if testing.Short() {
		t.Skip("cross-compiles the engine twice with -S; run without -short")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("no go toolchain: %v", err)
	}
	const engine = "github.com/kevung/blunderdb/pkg/blunderdb/engine/..."
	const canary = "fmacanary/canary.go:"
	for _, tg := range fmaTargets {
		t.Run(tg.name, func(t *testing.T) {
			cmd := exec.Command(goBin, "build", "-gcflags=-S", "-o", os.DevNull, engine, "./testdata/fmacanary")
			cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH="+tg.goarch, "GOAMD64="+tg.goamd64)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("go build: %v\n%s", err, tail(stderr.String()))
			}
			sawCanary := false
			var fused []string
			sc := bufio.NewScanner(&stderr)
			sc.Buffer(make([]byte, 1<<20), 1<<20)
			for sc.Scan() {
				line := sc.Text()
				if !fusedOp.MatchString(line) {
					continue
				}
				if strings.Contains(line, canary) {
					sawCanary = true
					continue
				}
				fused = append(fused, strings.TrimSpace(line))
			}
			if err := sc.Err(); err != nil {
				t.Fatalf("reading the listing: %v", err)
			}
			if !sawCanary {
				t.Fatalf("the %s canary shows no fused instruction: the detector no longer recognises one", canary)
			}
			if len(fused) > 0 {
				if len(fused) > 20 {
					fused = fused[:20]
				}
				t.Errorf("%d fused multiply-add(s) on %s; round each product with float64(a*b):\n%s",
					len(fused), tg.name, strings.Join(fused, "\n"))
			}
		})
	}
}

func tail(s string) string {
	if len(s) > 2000 {
		return s[len(s)-2000:]
	}
	return s
}
