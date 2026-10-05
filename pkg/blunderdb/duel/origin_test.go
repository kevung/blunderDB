package duel

import (
	"context"
	"testing"
)

// TestReadOriginRevealsTheSeedOfTheFingerprint: the seed read back from a
// Match gives the fingerprint published when its Duel was created, so the
// rolls can be checked without trusting the Arbiter.
func TestReadOriginRevealsTheSeedOfTheFingerprint(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	svc := newService(t, st, 5)
	cad, _ := NamedCadence("rapid-3+12")
	s, err := svc.Create(ctx, "", Settings{MatchLength: 5, Cadence: &cad,
		Sides: [2]SideSpec{external("A"), external("B")}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	published := s.Fingerprint
	for range 3 {
		if s, err = svc.Play(ctx, "", s.ID, s.Revision, answer(*s.Awaiting)); err != nil {
			t.Fatalf("Play: %v", err)
		}
	}
	kept, err := svc.Stop(ctx, "", s.ID, s.Revision, true)
	if err != nil || kept.Ended == nil {
		t.Fatalf("Stop: %+v, %v", kept, err)
	}

	o, err := ReadOrigin(ctx, st, "", kept.Ended.MatchID)
	if err != nil || o == nil {
		t.Fatalf("ReadOrigin: %+v, %v", o, err)
	}
	if o.DiceSeed == "" || o.Fingerprint != published {
		t.Errorf("fingerprint read back %q, published %q", o.Fingerprint, published)
	}
	if fp, _ := Fingerprint(o.DiceSeed); fp != published {
		t.Errorf("the seed read back does not give the published fingerprint")
	}
	if !o.StoppedEarly || o.Start != "" || o.BotLevel != "" || o.BotEngine != "" {
		t.Errorf("origin = %+v", o.MatchOrigin)
	}
	if o.CadenceSettings == nil || o.CadenceSettings.Name != "rapid-3+12" || o.CadenceSettings.Reserve != 180 {
		t.Errorf("cadence = %+v", o.CadenceSettings)
	}
}

// TestReadOriginOfAMatchNotPlayedHere: no origin is an answer, not an error.
func TestReadOriginOfAMatchNotPlayedHere(t *testing.T) {
	o, err := ReadOrigin(context.Background(), newStore(t), "", 4242)
	if err != nil || o != nil {
		t.Errorf("ReadOrigin of an imported match = %+v, %v; want nil, nil", o, err)
	}
}
