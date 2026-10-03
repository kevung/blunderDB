package domain

import "testing"

func TestFillSourceMetadataKeepsWhatIsStated(t *testing.T) {
	a, b := 1500.0, 1700.0
	yes, no := true, false
	dst := Match{Player1Elo: &a, Transcriber: "kept", HasJacoby: &no}
	src := Match{Player1Elo: &b, Player2Elo: &b, Transcriber: "other", HasJacoby: &yes, HasBeaver: &yes, EngineVersion: "v"}
	if !FillSourceMetadata(&dst, &src) {
		t.Fatal("reported no change")
	}
	if *dst.Player1Elo != a || dst.Transcriber != "kept" || *dst.HasJacoby {
		t.Errorf("stated field overwritten: %+v", dst)
	}
	if dst.Player2Elo == nil || dst.HasBeaver == nil || dst.EngineVersion != "v" {
		t.Errorf("unknown field not filled: %+v", dst)
	}
	if FillSourceMetadata(&dst, &src) {
		t.Error("second fill reported a change")
	}
}
