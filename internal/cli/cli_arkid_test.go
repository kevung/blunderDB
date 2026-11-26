package cli

import (
	"strings"
	"testing"
)

func runArkID(t *testing.T, args ...string) string {
	t.Helper()
	cli := &CLI{db: NewDatabase()}
	t.Cleanup(func() { cli.db.Close() })
	var err error
	out := captureOut(t, cli, func() { err = cli.runArkID(args) })
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestArkIDCommand(t *testing.T) {
	out := runArkID(t, "XGID=-b----E-C---eE---c-e----B-:0:0:1:65:0:0:0:0:10")
	if got := strings.TrimSpace(out); got != "[$]65:(ySQmLhfA)S3Qm3L3hf3" {
		t.Errorf("arkid of the opening = %q", got)
	}
	out = runArkID(t, "--expanded", "[34]65:x2(ySQmLhfA)S3Qm3L3hf3")
	if got := strings.TrimSpace(out); got != "[34]65:x2(ymhf)m3hf3(SQLA)S3QL3" {
		t.Errorf("expanded = %q", got)
	}
}
