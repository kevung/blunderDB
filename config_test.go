package main

import (
	"testing"

	"github.com/adrg/xdg"
)

// isolateXDGConfig points XDG_CONFIG_HOME at a throwaway directory so a test
// run never touches (or reuses) the developer's real config.yaml.
func isolateXDGConfig(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	xdg.Reload()
	t.Cleanup(xdg.Reload)
}

// TestLoadConfigPropagatesBearoffAndEpcChallenge is a regression test for a
// bug found while writing the fuller round-trip test in
// TestConfigRoundTripLoadSave: LoadConfig read BearoffTsPath and EpcChallenge
// from disk into the *returned* Config, but never copied them onto the
// receiver `c` (unlike every other field). Since the Wails binding exposes
// methods on that receiver, GetBearoffTsPath()/GetEpcChallenge() — what the
// frontend actually calls — silently reported the zero value after every
// restart, even though the value was correctly persisted and even correctly
// applied to the race engine once via the separate `config` return value in
// main.go's runGUI(). See config.go LoadConfig.
func TestLoadConfigPropagatesBearoffAndEpcChallenge(t *testing.T) {
	isolateXDGConfig(t)

	saver := NewConfig()
	saver.BearoffTsPath = "/tmp/some/external.bd"
	saver.EpcChallenge = true
	if err := saver.SaveConfig(saver); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loader := &Config{}
	if _, err := loader.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loader.BearoffTsPath != "/tmp/some/external.bd" {
		t.Errorf("receiver BearoffTsPath = %q, want %q (frontend calls GetBearoffTsPath() on this receiver)", loader.BearoffTsPath, "/tmp/some/external.bd")
	}
	if !loader.EpcChallenge {
		t.Error("receiver EpcChallenge = false, want true (frontend calls GetEpcChallenge() on this receiver)")
	}
}

func TestDefaultBoardColors(t *testing.T) {
	d := DefaultBoardColors()
	if d.Background != "#f0f0f0" || d.Checker2 != "#ffffff" {
		t.Fatalf("unexpected defaults: %+v", d)
	}
}

func TestBoardColorsWithDefaults(t *testing.T) {
	// A zero-value struct (e.g. an old config without board_colors) must be
	// fully populated with defaults.
	got := (BoardColors{}).withDefaults()
	if got != DefaultBoardColors() {
		t.Errorf("zero value not defaulted: %+v", got)
	}

	// A partial customisation keeps the set fields and defaults the blanks.
	partial := BoardColors{Background: "#000000", Checker1: "#ff0000"}.withDefaults()
	if partial.Background != "#000000" {
		t.Errorf("Background overwritten: %q", partial.Background)
	}
	if partial.Checker1 != "#ff0000" {
		t.Errorf("Checker1 overwritten: %q", partial.Checker1)
	}
	if partial.Border != DefaultBoardColors().Border {
		t.Errorf("blank Border not defaulted: %q", partial.Border)
	}
}

func TestNewConfigHasBoardColors(t *testing.T) {
	c := NewConfig()
	if c.BoardColors != DefaultBoardColors() {
		t.Errorf("NewConfig missing default board colors: %+v", c.BoardColors)
	}
}

func TestGetBoardColorsDefaultsEmpty(t *testing.T) {
	c := &Config{}
	if c.GetBoardColors() != DefaultBoardColors() {
		t.Errorf("GetBoardColors did not default empty config")
	}
}

func TestClampUIScale(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{0, DefaultUIScale}, // missing in old config files → default
		{100, 100},          // in range
		{50, 50},            // lower bound
		{200, 200},          // upper bound
		{10, MinUIScale},    // below range → clamped up
		{1000, MaxUIScale},  // above range → clamped down
		{-5, MinUIScale},    // negative → clamped up
	}
	for _, c := range cases {
		if got := clampUIScale(c.in); got != c.want {
			t.Errorf("clampUIScale(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestNewConfigHasDefaultUIScale(t *testing.T) {
	if c := NewConfig(); c.UIScale != DefaultUIScale {
		t.Errorf("NewConfig UIScale = %d, want %d", c.UIScale, DefaultUIScale)
	}
}

func TestGetUIScaleDefaultsEmpty(t *testing.T) {
	// An empty config (e.g. an old file with no ui_scale) must report the default.
	if got := (&Config{}).GetUIScale(); got != DefaultUIScale {
		t.Errorf("GetUIScale on empty config = %d, want %d", got, DefaultUIScale)
	}
}
