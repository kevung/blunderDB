package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
// TestConfigRoundTripLoadSave: LoadConfig read BearoffTSPath and EpcChallenge
// from disk into the *returned* Config, but never copied them onto the
// receiver `c` (unlike every other field). Since the Wails binding exposes
// methods on that receiver, GetBearoffTSPath()/GetEpcChallenge() — what the
// frontend actually calls — silently reported the zero value after every
// restart, even though the value was correctly persisted and even correctly
// applied to the race engine once via the separate `config` return value in
// main.go's runGUI(). See config.go LoadConfig.
func TestLoadConfigPropagatesBearoffAndEpcChallenge(t *testing.T) {
	isolateXDGConfig(t)

	saver := NewConfig()
	saver.BearoffTSPath = "/tmp/some/external.bd"
	saver.EpcChallenge = true
	if err := saver.SaveConfig(saver); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loader := &Config{}
	if _, err := loader.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loader.BearoffTSPath != "/tmp/some/external.bd" {
		t.Errorf("receiver BearoffTSPath = %q, want %q (frontend calls GetBearoffTSPath() on this receiver)", loader.BearoffTSPath, "/tmp/some/external.bd")
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

func TestSanitizePanelPosition(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{PanelPositionBottom, PanelPositionBottom},
		{PanelPositionSide, PanelPositionSide},
		{PanelPositionAuto, PanelPositionAuto},
		{"", DefaultPanelPosition},         // missing in old config files → default
		{"sideways", DefaultPanelPosition}, // unrecognised → default
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := sanitizePanelPosition(c.in); got != c.want {
				t.Errorf("sanitizePanelPosition(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestClampPanelHeight(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{0, DefaultPanelHeight}, // missing in old config files, or never resized → default
		{380, 380},
		{MinPanelHeight - 10, MinPanelHeight},
		{MaxPanelHeight + 10, MaxPanelHeight},
	}
	for _, c := range cases {
		if got := clampPanelHeight(c.in); got != c.want {
			t.Errorf("clampPanelHeight(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestClampPanelWidth(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{0, DefaultPanelWidth}, // missing in old config files, or never resized → default
		{520, 520},
		{MinPanelWidth - 10, MinPanelWidth},
		{MaxPanelWidth + 10, MaxPanelWidth},
	}
	for _, c := range cases {
		if got := clampPanelWidth(c.in); got != c.want {
			t.Errorf("clampPanelWidth(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestConfigRoundTripLoadSave writes a fully populated Config to a temporary
// XDG_CONFIG_HOME, reloads it from scratch on a fresh receiver, and checks
// every persisted field survives — including the two (BearoffTSPath,
// EpcChallenge) that TestLoadConfigPropagatesBearoffAndEpcChallenge caught
// missing from the receiver-side propagation.
func TestConfigRoundTripLoadSave(t *testing.T) {
	isolateXDGConfig(t)

	original := &Config{
		WindowWidth:      1600,
		WindowHeight:     900,
		LastDatabasePath: "/home/user/matches.db",
		Language:         "fr",
		BoardColors: BoardColors{
			Background: "#101010",
			Border:     "#202020",
			Point1:     "#303030",
			Point2:     "#404040",
			Checker1:   "#505050",
			Checker2:   "#606060",
			Dice:       "#707070",
			DiceDot:    "#808080",
			Cube:       "#909090",
		},
		UIScale:              150,
		PanelPosition:        PanelPositionSide,
		PanelHeight:          520,
		PanelWidth:           640,
		TourSeen:             true,
		TabOrder:             []string{"search", "matches", "analysis"},
		HiddenTabs:           []string{"tournaments"},
		BearoffTSPath:        "/home/user/gnubg_ts6x11.bd",
		EpcChallenge:         true,
		GammonNetDisplayPly:  intPtr(1),
		GammonNetAnalysisPly: intPtr(3),
		GammonNetPruneK:      20,
		GammonNetCandidates:  15,
		GammonNetAutoAnalyze: true,
		CheckForUpdates:      true,
		StatsFilter: StatsFilterPersisted{
			PlayerName:    "Kévin Unger",
			TournamentIDs: []int64{3, 7, 11},
			DateFrom:      "2026-01-01",
			DateTo:        "2026-12-31",
			DecisionType:  intPtr(1),
			MatchLength:   []int{5, 7},
			Metric:        "mwc",
		},
	}

	if err := original.SaveConfig(original); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded := &Config{}
	if _, err := loaded.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	// Every accessor the frontend actually calls goes through the receiver.
	checks := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"WindowWidth", loaded.WindowWidth, original.WindowWidth},
		{"WindowHeight", loaded.WindowHeight, original.WindowHeight},
		{"LastDatabasePath", loaded.GetLastDatabasePath(), original.LastDatabasePath},
		{"Language", loaded.GetLanguage(), original.Language},
		{"BoardColors", loaded.GetBoardColors(), original.BoardColors},
		{"UIScale", loaded.GetUIScale(), original.UIScale},
		{"PanelPosition", loaded.GetPanelPosition(), original.PanelPosition},
		{"PanelHeight", loaded.GetPanelHeight(), original.PanelHeight},
		{"PanelWidth", loaded.GetPanelWidth(), original.PanelWidth},
		{"TourSeen", loaded.GetTourSeen(), original.TourSeen},
		{"TabOrder", loaded.GetTabOrder(), original.TabOrder},
		{"HiddenTabs", loaded.GetHiddenTabs(), original.HiddenTabs},
		{"BearoffTSPath", loaded.GetBearoffTSPath(), original.BearoffTSPath},
		{"EpcChallenge", loaded.GetEpcChallenge(), original.EpcChallenge},
		{"GammonNetDisplayPly", loaded.GetGammonNetDisplayPly(), *original.GammonNetDisplayPly},
		{"GammonNetAnalysisPly", loaded.GetGammonNetAnalysisPly(), *original.GammonNetAnalysisPly},
		{"GammonNetPruneK", loaded.GetGammonNetPruneK(), original.GammonNetPruneK},
		{"GammonNetCandidates", loaded.GetGammonNetCandidates(), original.GammonNetCandidates},
		{"GammonNetAutoAnalyze", loaded.GetGammonNetAutoAnalyze(), original.GammonNetAutoAnalyze},
		{"CheckForUpdates", loaded.GetCheckForUpdates(), original.CheckForUpdates},
		{"StatsFilter", loaded.GetStatsFilter(), original.StatsFilter},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !reflect.DeepEqual(c.got, c.want) {
				t.Errorf("%s round-trip = %+v, want %+v", c.name, c.got, c.want)
			}
		})
	}
}

// TestConfigRoundTripEmptyFieldsKeepDefaults checks that a config saved with
// only the required fields set (as an older config.yaml, before some field
// existed, would look) still loads with every documented default.
func TestConfigRoundTripEmptyFieldsKeepDefaults(t *testing.T) {
	isolateXDGConfig(t)

	minimal := &Config{WindowWidth: 1024, WindowHeight: 768}
	if err := minimal.SaveConfig(minimal); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded := &Config{}
	if _, err := loaded.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if got := loaded.GetLanguage(); got != "en" {
		t.Errorf("GetLanguage() = %q, want default %q", got, "en")
	}
	if got := loaded.GetBoardColors(); got != DefaultBoardColors() {
		t.Errorf("GetBoardColors() = %+v, want defaults", got)
	}
	if got := loaded.GetUIScale(); got != DefaultUIScale {
		t.Errorf("GetUIScale() = %d, want default %d", got, DefaultUIScale)
	}
	if got := loaded.GetPanelPosition(); got != DefaultPanelPosition {
		t.Errorf("GetPanelPosition() = %q, want default %q", got, DefaultPanelPosition)
	}
	if got := loaded.GetPanelHeight(); got != DefaultPanelHeight {
		t.Errorf("GetPanelHeight() = %d, want default %d", got, DefaultPanelHeight)
	}
	if got := loaded.GetPanelWidth(); got != DefaultPanelWidth {
		t.Errorf("GetPanelWidth() = %d, want default %d", got, DefaultPanelWidth)
	}
	if got := loaded.GetTourSeen(); got != false {
		t.Errorf("GetTourSeen() = %v, want false", got)
	}
	if got := loaded.GetBearoffTSPath(); got != "" {
		t.Errorf("GetBearoffTSPath() = %q, want empty", got)
	}
	if got := loaded.GetEpcChallenge(); got != false {
		t.Errorf("GetEpcChallenge() = %v, want false", got)
	}
	if got := loaded.GetGammonNetDisplayPly(); got != DefaultGammonNetPly {
		t.Errorf("GetGammonNetDisplayPly() = %d, want default %d", got, DefaultGammonNetPly)
	}
	if got := loaded.GetGammonNetAnalysisPly(); got != DefaultGammonNetPly {
		t.Errorf("GetGammonNetAnalysisPly() = %d, want default %d", got, DefaultGammonNetPly)
	}
	if got := loaded.GetGammonNetPruneK(); got != DefaultGammonNetPruneK {
		t.Errorf("GetGammonNetPruneK() = %d, want default %d", got, DefaultGammonNetPruneK)
	}
	if got := loaded.GetGammonNetCandidates(); got != DefaultGammonNetCandidates {
		t.Errorf("GetGammonNetCandidates() = %d, want default %d", got, DefaultGammonNetCandidates)
	}
	if got := loaded.GetGammonNetAutoAnalyze(); got != false {
		t.Errorf("GetGammonNetAutoAnalyze() = %v, want false", got)
	}
	if got := loaded.GetCheckForUpdates(); got != false {
		t.Errorf("GetCheckForUpdates() = %v, want false (opt-in, off by default)", got)
	}
}

func TestClampGammonNetPly(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{0, 0}, // 0-ply is a legitimate explicit depth, not a "missing" sentinel
		{2, 2},
		{4, MaxGammonNetPly},
		{-1, MinGammonNetPly},
		{9, MaxGammonNetPly},
	}
	for _, c := range cases {
		if got := clampGammonNetPly(c.in); got != c.want {
			t.Errorf("clampGammonNetPly(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestClampGammonNetPruneK(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{0, DefaultGammonNetPruneK},
		{12, 12},
		{-3, MinGammonNetPruneK},
		{1000, MaxGammonNetPruneK},
	}
	for _, c := range cases {
		if got := clampGammonNetPruneK(c.in); got != c.want {
			t.Errorf("clampGammonNetPruneK(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestClampGammonNetCandidates(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{0, DefaultGammonNetCandidates},
		{10, 10},
		{-1, MinGammonNetCandidates},
		{1000, MaxGammonNetCandidates},
	}
	for _, c := range cases {
		if got := clampGammonNetCandidates(c.in); got != c.want {
			t.Errorf("clampGammonNetCandidates(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestGammonNetDisplayAndAnalysisPlyAreIndependent guards the point of the
// ticket: lowering the display depth must never move the analysis depth, and
// vice versa — ADR-0013's "conflating them is what turns a comfort knob into
// silent damage to data".
func TestGammonNetDisplayAndAnalysisPlyAreIndependent(t *testing.T) {
	isolateXDGConfig(t)

	c := NewConfig()
	if err := c.SaveConfig(c); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := c.SaveGammonNetDisplayPly(1); err != nil {
		t.Fatalf("SaveGammonNetDisplayPly: %v", err)
	}
	if got := c.GetGammonNetAnalysisPly(); got != DefaultGammonNetPly {
		t.Errorf("changing display ply moved analysis ply: GetGammonNetAnalysisPly() = %d, want %d", got, DefaultGammonNetPly)
	}
}

// TestGammonNetZeroPlyExplicitlySavedSurvivesReload is a regression test for
// a bug caught while writing this ticket: an early version mapped a zero
// value to DefaultGammonNetPly unconditionally, which silently promoted a
// deliberately-chosen 0-ply back to 2-ply on every reload. The *int fields
// (nil = unset, non-nil zero = explicit 0-ply) are what make this
// distinguishable — this test is what a plain int field would fail.
func TestGammonNetZeroPlyExplicitlySavedSurvivesReload(t *testing.T) {
	isolateXDGConfig(t)

	c := NewConfig()
	if err := c.SaveConfig(c); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := c.SaveGammonNetDisplayPly(0); err != nil {
		t.Fatalf("SaveGammonNetDisplayPly: %v", err)
	}

	loaded := &Config{}
	if _, err := loaded.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got := loaded.GetGammonNetDisplayPly(); got != 0 {
		t.Errorf("GetGammonNetDisplayPly() after explicit 0-ply save = %d, want 0", got)
	}
}

func intPtr(v int) *int { return &v }

// TestSaveConfig_WritesCurrentFileName guards #241's rename from
// config.yaml to config.json: SaveConfig must create the file under
// configFilePath (config.json), never the legacy name, and must leave no
// temp file behind (the atomic write-then-rename's tmp file, created
// alongside the real path).
func TestSaveConfig_WritesCurrentFileName(t *testing.T) {
	isolateXDGConfig(t)

	c := NewConfig()
	if err := c.SaveConfig(c); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	path, err := xdg.SearchConfigFile(configFilePath)
	if err != nil {
		t.Fatalf("config.json was not created: %v", err)
	}
	if !strings.HasSuffix(path, "config.json") {
		t.Errorf("config file path = %q, want it to end in config.json", path)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("leftover temp file after SaveConfig: %s", e.Name())
		}
	}
}

// TestLoadConfig_MigratesLegacyFileName guards #241: a machine that only has
// the pre-2026-09 config.yaml (content was always JSON despite the
// extension) must still load correctly, and LoadConfig must migrate it to
// config.json so every later run finds the current name directly — without
// losing the legacy file, in case migration itself cannot write (e.g. a
// read-only config directory).
func TestLoadConfig_MigratesLegacyFileName(t *testing.T) {
	isolateXDGConfig(t)

	legacyPath, err := xdg.ConfigFile(legacyConfigFilePath)
	if err != nil {
		t.Fatalf("xdg.ConfigFile(legacy): %v", err)
	}
	legacy := NewConfig()
	legacy.LastDatabasePath = "/tmp/legacy.db"
	data, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(legacyPath, data, 0o644); err != nil {
		t.Fatalf("WriteFile(legacy): %v", err)
	}

	loaded := &Config{}
	got, err := loaded.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got.LastDatabasePath != "/tmp/legacy.db" {
		t.Errorf("LastDatabasePath = %q, want the legacy file's value", got.LastDatabasePath)
	}

	if _, err := xdg.SearchConfigFile(configFilePath); err != nil {
		t.Errorf("config.json was not created by the migration: %v", err)
	}
	if _, err := os.Stat(legacyPath); err != nil {
		t.Errorf("legacy config.yaml was removed or lost during migration: %v", err)
	}
}

// TestLoadConfig_CorruptFileResetsToDefaultsWithBackup guards #241: a
// config.json that fails to parse as JSON (truncated write, hand edit,
// corruption) used to fail LoadConfig outright, which main.go turned into
// os.Exit(1) — the app refused to start at all until a user found and fixed
// or deleted a file most never knew existed. It must instead back the bad
// file up (so nothing is silently lost) and start from a fresh default
// Config.
func TestLoadConfig_CorruptFileResetsToDefaultsWithBackup(t *testing.T) {
	isolateXDGConfig(t)

	path, err := xdg.ConfigFile(configFilePath)
	if err != nil {
		t.Fatalf("xdg.ConfigFile: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loaded := &Config{}
	got, err := loaded.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig on a corrupt file: %v (want it to recover, not fail)", err)
	}
	if got.WindowWidth == 0 {
		t.Error("LoadConfig did not fall back to a populated default Config")
	}

	backup, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("backup file not created: %v", err)
	}
	if string(backup) != "{not valid json" {
		t.Errorf("backup content = %q, want the original corrupt bytes", backup)
	}

	// The reset must itself have been persisted, so a second load does not
	// hit the same corrupt file again.
	reloaded := &Config{}
	if _, err := reloaded.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig after reset: %v", err)
	}
}

// TestConfig_ConfigVersionStamped guards #241's config_version field: both a
// freshly created Config and one loaded from a pre-existing file that
// predates the field must end up stamped with currentConfigVersion.
func TestConfig_ConfigVersionStamped(t *testing.T) {
	isolateXDGConfig(t)

	if v := NewConfig().ConfigVersion; v != currentConfigVersion {
		t.Errorf("NewConfig().ConfigVersion = %d, want %d", v, currentConfigVersion)
	}

	// Simulate a file written before config_version existed: marshal a
	// config, then strip the field back out by hand.
	old := NewConfig()
	old.ConfigVersion = 0
	data, err := json.Marshal(old)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	path, err := xdg.ConfigFile(configFilePath)
	if err != nil {
		t.Fatalf("xdg.ConfigFile: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loaded := &Config{}
	got, err := loaded.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got.ConfigVersion != currentConfigVersion {
		t.Errorf("ConfigVersion after loading a version-less file = %d, want %d", got.ConfigVersion, currentConfigVersion)
	}
	if loaded.ConfigVersion != currentConfigVersion {
		t.Errorf("receiver ConfigVersion = %d, want %d (mirrors every other field's receiver copy)", loaded.ConfigVersion, currentConfigVersion)
	}
}
