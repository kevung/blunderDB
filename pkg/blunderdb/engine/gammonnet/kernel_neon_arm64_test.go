// SPDX-License-Identifier: MIT

//go:build arm64 && !purego

package gammonnet

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The NEON kernel is read back because a fused multiply-add is a different
// answer the gold suites' 1e-6 would not notice (ADR-0024), and the vector
// arithmetic is spelled as raw encodings the assembler cannot check.
func TestNEONKernelHasNoFMA(t *testing.T) {
	raw, err := os.ReadFile("kernel_neon_arm64.s")
	if err != nil {
		t.Fatalf("NEON kernel: %v", err)
	}
	var code strings.Builder
	for _, line := range strings.Split(string(raw), "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		code.WriteString(line)
		code.WriteByte('\n')
	}
	asm := code.String()

	if m := regexp.MustCompile(`(?i)\bV?F[N]?M(LA|LS|ADD|SUB)\w*`).FindString(asm); m != "" {
		t.Errorf("the kernel emits %q: no fused multiply-add, ever", m)
	}
	// FMLA/FMLS (vector) share 0x0E20CC00 under mask 0x9F20FC00, whatever
	// the registers, the Q bit or the precision.
	for _, w := range regexp.MustCompile(`WORD\s+\$\((0x[0-9A-Fa-f]+)`).FindAllStringSubmatch(asm, -1) {
		base, err := strconv.ParseUint(w[1], 0, 32)
		if err != nil {
			t.Fatalf("encoding %s: %v", w[1], err)
		}
		if base&0x9F20FC00 == 0x0E20CC00 {
			t.Errorf("encoding %s is FMLA/FMLS", w[1])
		}
	}
	for _, want := range []string{"0x6E20DC00", "0x4E20D400"} {
		if !strings.Contains(asm, want) {
			t.Errorf("encoding %s (FMUL / FADD 4S) is missing: the kernel no longer multiplies then adds", want)
		}
	}
}

// TestNEONIsOptInUntilProven: the default path on arm64 stays the pure-Go
// one until the identity test has passed on real Apple Silicon; the NEON
// kernel is reachable only by naming it.
func TestNEONIsOptInUntilProven(t *testing.T) {
	acc := acceleratedKernels()
	if len(acc) == 0 {
		t.Skip("no ASIMD on this CPU")
	}
	if k, err := resolveKernel("", acc); err != nil || k.name != goKernelName {
		t.Fatalf("default kernel on arm64: got %q, %v; want %q", k.name, err, goKernelName)
	}
	if k, err := resolveKernel("neon", acc); err != nil || k.name != "neon" {
		t.Fatalf("explicit neon: got %q, %v", k.name, err)
	}
}
