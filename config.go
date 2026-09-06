package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/adrg/xdg"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

// configFilePath is the current config file name. It was named
// "config.yaml" from the first release through 2026-09, even though its
// content was always JSON (encoding/json in, encoding/json out) — a
// misnomer, not a format that ever changed (#241). legacyConfigFilePath is
// read once, on a machine that still only has the old file: LoadConfig
// migrates it to configFilePath and every subsequent run finds the new name
// directly, so this fallback costs nothing once migrated.
const (
	configFilePath       = "blunderDB/config.json"
	legacyConfigFilePath = "blunderDB/config.yaml"
)

// currentConfigVersion is bumped whenever a future change needs a real
// migration step (a field renamed or reshaped, not just added) — the same
// idea as domain.DatabaseVersion, at a much smaller scale: today there is
// only version 1, and every config file this build writes carries it.
const currentConfigVersion = 1

type StatsFilterPersisted struct {
	PlayerName    string  `json:"player_name"`
	TournamentIDs []int64 `json:"tournament_ids"`
	DateFrom      string  `json:"date_from"`
	DateTo        string  `json:"date_to"`
	DecisionType  *int    `json:"decision_type"` // nil = all (-1), 0 = checker only, 1 = cube only
	MatchLength   []int   `json:"match_length"`
	Metric        string  `json:"metric"` // "pr" | "mwc"
}

// BoardColors holds the user-customisable board palette. Empty fields fall back
// to DefaultBoardColors() so older config files (and partial customisations)
// keep rendering correctly.
type BoardColors struct {
	Background string `json:"background"` // board background fill
	Border     string `json:"border"`     // board border / point & piece stroke
	Point1     string `json:"point1"`     // light points (triangle fill 1)
	Point2     string `json:"point2"`     // dark points (triangle fill 2)
	Checker1   string `json:"checker1"`   // player 1 checkers
	Checker2   string `json:"checker2"`   // player 2 checkers
	Dice       string `json:"dice"`       // dice face fill
	DiceDot    string `json:"diceDot"`    // dice pip colour
	Cube       string `json:"cube"`       // doubling cube face fill
}

// DefaultBoardColors returns the historical hard-coded palette from Board.svelte.
func DefaultBoardColors() BoardColors {
	return BoardColors{
		Background: "#f0f0f0",
		Border:     "#333333",
		Point1:     "#d9d9d9",
		Point2:     "#a6a6a6",
		Checker1:   "#333333",
		Checker2:   "#ffffff",
		Dice:       "#ffffff",
		DiceDot:    "#000000",
		Cube:       "#ffffff",
	}
}

// withDefaults fills any empty field with its default, so partial or missing
// persisted colours never render as blank.
func (bc BoardColors) withDefaults() BoardColors {
	d := DefaultBoardColors()
	if bc.Background == "" {
		bc.Background = d.Background
	}
	if bc.Border == "" {
		bc.Border = d.Border
	}
	if bc.Point1 == "" {
		bc.Point1 = d.Point1
	}
	if bc.Point2 == "" {
		bc.Point2 = d.Point2
	}
	if bc.Checker1 == "" {
		bc.Checker1 = d.Checker1
	}
	if bc.Checker2 == "" {
		bc.Checker2 = d.Checker2
	}
	if bc.Dice == "" {
		bc.Dice = d.Dice
	}
	if bc.DiceDot == "" {
		bc.DiceDot = d.DiceDot
	}
	if bc.Cube == "" {
		bc.Cube = d.Cube
	}
	return bc
}

// UI scale bounds (percentage). The interface is rendered at UIScale% of its
// native size; 100 means no scaling.
const (
	MinUIScale     = 50
	MaxUIScale     = 200
	DefaultUIScale = 100
)

// Panel position modes. The tabbed panel is either docked at the bottom (the
// historical default), pinned as a vertical column on the side, or switched
// automatically based on the window aspect ratio (handled frontend-side).
const (
	PanelPositionBottom  = "bottom"
	PanelPositionSide    = "side"
	PanelPositionAuto    = "auto"
	DefaultPanelPosition = PanelPositionBottom
)

// Panel size, in pixels — the tabbed panel's height in bottom mode, its width
// in side mode (the other dimension stretches to fill the window). ADR-0017
// unified the Eval tab's stacked tables into a single flex-wrap row, so a
// cube position fits in ~140px and only its moves list ever scrolls; these
// defaults are the pre-existing ones, not sized around that panel. Once the
// user drags the resize handle, their own size is persisted instead
// (frontend saves on mouseup) and these defaults never apply again.
const (
	MinPanelHeight     = 80
	MaxPanelHeight     = 4000
	DefaultPanelHeight = 250

	MinPanelWidth     = 150
	MaxPanelWidth     = 4000
	DefaultPanelWidth = 420
)

// gammonNet settings (ADR-0011, ADR-0013). Two depths, named and clamped
// separately on purpose: display depth is interactive comfort, analysis depth
// is what the batch (#129) writes to a Position's Analysis row. Conflating
// them would let a comfort adjustment silently degrade what gets persisted.
// Both default to the canonical "normal" level (issue #25) —
// gammonnet.DefaultPly / gammonnet.DefaultPruneK, read from gammonNet's own
// export rather than retyped as literals here.
const (
	MinGammonNetPly = 0
	MaxGammonNetPly = gammonnet.MaxPly

	MinGammonNetPruneK = 1
	MaxGammonNetPruneK = 64

	MinGammonNetCandidates     = 1
	MaxGammonNetCandidates     = 50
	DefaultGammonNetCandidates = 10
)

// DefaultGammonNetPly and DefaultGammonNetPruneK are vars, not consts: they
// are read from gammonnet's embedded canonical export at init time, not
// typed in by hand. See gammonnet.DefaultPly / gammonnet.DefaultPruneK
// (search.go there, issue #25) for the source and its measured quality cost.
var (
	DefaultGammonNetPly    = gammonnet.DefaultPly
	DefaultGammonNetPruneK = gammonnet.DefaultPruneK
)

type Config struct {
	// ConfigVersion names the shape of this file, so a future field rename
	// or reshape has something to branch a migration step on (#241). Always
	// currentConfigVersion once loaded/saved by this build; 0 on a file
	// written before this field existed.
	ConfigVersion    int                  `json:"config_version"`
	WindowWidth      int                  `json:"window_width"`
	WindowHeight     int                  `json:"window_height"`
	LastDatabasePath string               `json:"last_database_path"`
	StatsFilter      StatsFilterPersisted `json:"stats_filter,omitempty"`
	Language         string               `json:"language,omitempty"`
	BoardColors      BoardColors          `json:"board_colors,omitempty"`
	UIScale          int                  `json:"ui_scale,omitempty"`
	PanelPosition    string               `json:"panel_position,omitempty"`
	PanelHeight      int                  `json:"panel_height,omitempty"`
	PanelWidth       int                  `json:"panel_width,omitempty"`
	TourSeen         bool                 `json:"tour_seen,omitempty"`
	// TabOrder is the tabbed panel's tab ids (TabbedPanel.svelte's `tabs`
	// array), in the order the user last dragged them to. Empty means "use
	// the built-in order" — the frontend owns the canonical id list and
	// labels, this only ever reorders/filters it (#215).
	TabOrder []string `json:"tab_order,omitempty"`
	// HiddenTabs is the subset of tab ids the user chose to hide from the
	// tabbed panel's tab bar (#215). A tab hidden here can still be reached
	// by its own keyboard shortcut; hiding only removes the tab button.
	HiddenTabs []string `json:"hidden_tabs,omitempty"`
	// BearoffTSPath is an optional user-supplied two-sided bearoff database
	// (.bd) widening the generated TS-06-06 (ADR-0009). Empty = none.
	BearoffTSPath string `json:"bearoff_ts_path,omitempty"`
	// BearoffRate is what one core of THIS machine measured on a finished
	// two-sided run: seconds per n³ (bearoffgen's cost model). It is what
	// turns "about 20 minutes" from a claim about the developer's laptop into
	// one about the user's. 0 until a run wide enough to be representative
	// has finished here.
	BearoffRate float64 `json:"bearoff_rate,omitempty"`
	// BearoffCores is the core count the user last chose for a generation.
	// 0 = the default, every core but one.
	BearoffCores int `json:"bearoff_cores,omitempty"`
	// EpcChallenge persists the EPC panel's training mode ("défi"): results
	// are masked after each edit until the user clicks a zone to reveal it.
	EpcChallenge bool `json:"epc_challenge,omitempty"`
	// gammonNet settings (ADR-0011, ADR-0013). See the Min/Max/Default
	// constants above for the meaning of each field. The two depths are
	// pointers, like StatsFilterPersisted.DecisionType above: 0-ply is a
	// legitimate, user-selectable value, so a plain int could not tell an
	// explicit 0-ply apart from an old config file that predates this field —
	// nil is "unset, use the canonical default", a non-nil zero is "0-ply,
	// chosen".
	GammonNetDisplayPly  *int `json:"gammonnet_display_ply,omitempty"`
	GammonNetAnalysisPly *int `json:"gammonnet_analysis_ply,omitempty"`
	GammonNetPruneK      int  `json:"gammonnet_prune_k,omitempty"`
	GammonNetCandidates  int  `json:"gammonnet_candidates,omitempty"`
	GammonNetAutoAnalyze bool `json:"gammonnet_auto_analyze,omitempty"`
	// CheckForUpdates opts into gui.App.CheckForUpdate querying the GitHub
	// Releases API at startup (#241) — off by default (a network call an
	// offline-first tool must never make unasked) and forced off regardless
	// of this setting on a package-managed install (see
	// gui.isPackageManaged): the package manager is that channel's own
	// update mechanism, and blunderDB pointing at a GitHub release the
	// distro hasn't packaged yet would just confuse the user.
	CheckForUpdates bool `json:"check_for_updates,omitempty"`
	// Watched folder (#258, fiche I.2): a directory blunderDB looks at while
	// it is running, importing each match file that APPEARS in it. Empty
	// means no watch, and that is the default — a tool does not start
	// reading a folder of yours because it guessed where your matches are.
	//
	// WatchFolderIntervalSeconds is what the user chose; 0 means the
	// package's own default (watch.DefaultInterval). The clamping lives in
	// watch.ClampInterval, so the floor is stated once.
	// Theme is the named interface theme (#286): "system" (follow the
	// desktop), "light", "dark", "contrast" or "print". Empty means system —
	// a tool does not impose its light or its dark on a desktop that has
	// already decided. The theme's VALUES live in the frontend
	// (utils/themes.js): they are design tokens, and Go has no business
	// holding a second copy of a palette it never reads.
	Theme string `json:"theme,omitempty"`

	WatchFolder                bool   `json:"watch_folder,omitempty"`
	WatchFolderPath            string `json:"watch_folder_path,omitempty"`
	WatchFolderIntervalSeconds int    `json:"watch_folder_interval_seconds,omitempty"`
}

// clampUIScale coerces a persisted/incoming scale into the supported range,
// mapping the zero value (missing in older config files) to the default.
func clampUIScale(scale int) int {
	if scale == 0 {
		return DefaultUIScale
	}
	if scale < MinUIScale {
		return MinUIScale
	}
	if scale > MaxUIScale {
		return MaxUIScale
	}
	return scale
}

// sanitizePanelPosition coerces a persisted/incoming panel position into a
// known value, mapping the zero value (missing in older config files) and any
// unrecognised string to the default (bottom).
func sanitizePanelPosition(pos string) string {
	switch pos {
	case PanelPositionBottom, PanelPositionSide, PanelPositionAuto:
		return pos
	default:
		return DefaultPanelPosition
	}
}

// clampPanelHeight coerces a persisted/incoming bottom-mode panel height into
// the supported range, mapping the zero value (missing in older config files,
// or never resized by the user) to the default.
func clampPanelHeight(height int) int {
	if height == 0 {
		return DefaultPanelHeight
	}
	if height < MinPanelHeight {
		return MinPanelHeight
	}
	if height > MaxPanelHeight {
		return MaxPanelHeight
	}
	return height
}

// clampPanelWidth coerces a persisted/incoming side-mode panel width the same
// way clampPanelHeight does for the height.
func clampPanelWidth(width int) int {
	if width == 0 {
		return DefaultPanelWidth
	}
	if width < MinPanelWidth {
		return MinPanelWidth
	}
	if width > MaxPanelWidth {
		return MaxPanelWidth
	}
	return width
}

// clampGammonNetPly coerces an explicit search depth into the supported
// range. It does not special-case 0: 0-ply is a legitimate depth, not a
// missing-setting sentinel — see the *int fields on Config.
func clampGammonNetPly(ply int) int {
	if ply < MinGammonNetPly {
		return MinGammonNetPly
	}
	if ply > MaxGammonNetPly {
		return MaxGammonNetPly
	}
	return ply
}

// clampGammonNetPruneK coerces a persisted/incoming pruning width into the
// supported range, mapping the zero value to the canonical default (k=12).
func clampGammonNetPruneK(k int) int {
	if k == 0 {
		return DefaultGammonNetPruneK
	}
	if k < MinGammonNetPruneK {
		return MinGammonNetPruneK
	}
	if k > MaxGammonNetPruneK {
		return MaxGammonNetPruneK
	}
	return k
}

// clampGammonNetCandidates coerces a persisted/incoming candidate-move count
// into the supported range, mapping the zero value to the default (10).
func clampGammonNetCandidates(n int) int {
	if n == 0 {
		return DefaultGammonNetCandidates
	}
	if n < MinGammonNetCandidates {
		return MinGammonNetCandidates
	}
	if n > MaxGammonNetCandidates {
		return MaxGammonNetCandidates
	}
	return n
}

func NewConfig() *Config {
	initialWidth, initialHeight := calculateInitialDimensions()
	return &Config{
		ConfigVersion: currentConfigVersion,
		WindowWidth:   initialWidth,
		WindowHeight:  initialHeight,
		Language:      "en",
		BoardColors:   DefaultBoardColors(),
		UIScale:       DefaultUIScale,
		PanelPosition: DefaultPanelPosition,
		PanelHeight:   DefaultPanelHeight,
		PanelWidth:    DefaultPanelWidth,
		// GammonNetDisplayPly/GammonNetAnalysisPly stay nil: the Get
		// accessors report DefaultGammonNetPly for a nil pointer.
		GammonNetPruneK:     DefaultGammonNetPruneK,
		GammonNetCandidates: DefaultGammonNetCandidates,
	}
}

func calculateInitialDimensions() (int, int) {
	initialWidth := 1024 // Adjusted width for better compatibility
	var aspectFactor float64
	if runtime.GOOS == "windows" {
		aspectFactor = 0.814 // Adjusted aspect factor for Windows
	} else {
		aspectFactor = 0.7815 // Original aspect factor for Linux
	}
	initialHeight := int(float64(initialWidth) * aspectFactor) // Adjust to have equal space above and below
	return initialWidth, initialHeight
}

// LoadConfig reads the persisted config, tolerating three situations that
// used to either crash the GUI at startup or silently misbehave (#241):
//
//   - No file at all (first run, or a fresh XDG_CONFIG_HOME): a fresh
//     default Config is created and saved under the current name.
//   - Only the pre-2026-09 legacy name (config.yaml, holding the same JSON):
//     read once, then immediately re-saved under the current name — every
//     later run finds it directly and this branch never runs again.
//   - A file that exists under the current name but fails to parse as JSON
//     (truncated by a crash mid-write, hand-edited into invalid JSON, disk
//     corruption): rather than propagating the error up to main.go, which
//     used to os.Exit(1) and leave the user with an app that will not start
//     until they find and fix or delete a file they likely do not know
//     exists, the unreadable file is backed up next to itself
//     (config.json.bak) and a fresh default Config takes its place. The
//     backup means nothing is silently destroyed — a user who cares can
//     recover their old settings from it — but the app starts either way.
func (c *Config) LoadConfig() (*Config, error) {
	configPath, err := xdg.SearchConfigFile(configFilePath)
	migrating := false
	if err != nil {
		if legacyPath, legacyErr := xdg.SearchConfigFile(legacyConfigFilePath); legacyErr == nil {
			configPath = legacyPath
			migrating = true
			slog.Info("config file found under its legacy name, migrating", "path", configPath)
		}
	}
	if configPath == "" {
		slog.Info("config file not found, creating a new one")
		config := NewConfig()
		if err := c.SaveConfig(config); err != nil {
			return nil, err
		}
		return config, nil
	}
	slog.Info("config file found", "path", configPath)

	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(raw, &config); err != nil {
		backupPath := configPath + ".bak"
		if werr := os.WriteFile(backupPath, raw, 0o600); werr != nil {
			slog.Warn("config file is corrupt and could not be backed up; resetting to defaults", "path", configPath, "parse_err", err, "backup_err", werr)
		} else {
			slog.Warn("config file is corrupt, backed up and reset to defaults", "path", configPath, "backup", backupPath, "parse_err", err)
		}
		config = *NewConfig()
		if err := c.SaveConfig(&config); err != nil {
			return nil, err
		}
		return &config, nil
	}
	if config.ConfigVersion == 0 {
		// A file written before this field existed: there is nothing to
		// actually migrate yet (every field so far still reads the same
		// way), so this only stamps the version going forward.
		config.ConfigVersion = currentConfigVersion
	}

	// Update the receiver so the Wails-bound instance has the loaded values
	c.WindowWidth = config.WindowWidth
	c.WindowHeight = config.WindowHeight
	c.LastDatabasePath = config.LastDatabasePath
	c.StatsFilter = config.StatsFilter
	c.Language = config.Language
	if c.Language == "" {
		c.Language = "en"
	}
	c.BoardColors = config.BoardColors.withDefaults()
	config.BoardColors = c.BoardColors
	c.UIScale = clampUIScale(config.UIScale)
	config.UIScale = c.UIScale
	c.PanelPosition = sanitizePanelPosition(config.PanelPosition)
	config.PanelPosition = c.PanelPosition
	c.PanelHeight = clampPanelHeight(config.PanelHeight)
	config.PanelHeight = c.PanelHeight
	c.PanelWidth = clampPanelWidth(config.PanelWidth)
	config.PanelWidth = c.PanelWidth
	c.TourSeen = config.TourSeen
	c.TabOrder = config.TabOrder
	c.HiddenTabs = config.HiddenTabs
	c.BearoffTSPath = config.BearoffTSPath
	c.BearoffRate = config.BearoffRate
	c.BearoffCores = config.BearoffCores
	c.EpcChallenge = config.EpcChallenge
	c.GammonNetDisplayPly = config.GammonNetDisplayPly
	c.GammonNetAnalysisPly = config.GammonNetAnalysisPly
	c.GammonNetPruneK = clampGammonNetPruneK(config.GammonNetPruneK)
	config.GammonNetPruneK = c.GammonNetPruneK
	c.GammonNetCandidates = clampGammonNetCandidates(config.GammonNetCandidates)
	config.GammonNetCandidates = c.GammonNetCandidates
	c.GammonNetAutoAnalyze = config.GammonNetAutoAnalyze
	c.CheckForUpdates = config.CheckForUpdates
	c.ConfigVersion = config.ConfigVersion

	if migrating {
		// Best-effort: failing to write the new name must not fail startup —
		// the legacy file is still there and will be found again next run.
		if err := c.SaveConfig(&config); err != nil {
			slog.Warn("could not migrate config to its current file name", "from", configPath, "to", configFilePath, "err", err)
		} else {
			slog.Info("config migrated to its current file name", "from", configPath)
		}
	}

	return &config, nil
}

// SaveConfig writes config as indented JSON to the current config file,
// atomically: it writes to a sibling temp file, fsyncs it, then renames it
// over the real path (the same write-then-rename shape
// resumableDownload/bearoff_download.go already uses for the bearoff
// database download). A crash or power loss mid-write can therefore never
// leave config.json truncated or half-written — the rename either lands
// completely or the old file is untouched (#241).
func (c *Config) SaveConfig(config *Config) error {
	configPath, err := xdg.ConfigFile(configFilePath)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(configPath)
	tmp, err := os.CreateTemp(dir, ".config-*.json.tmp")
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	tmpPath := tmp.Name()
	// Any early return below must not leave the temp file behind.
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	success = true
	return nil
}

func (c *Config) SaveWindowDimensions(width, height int) error {
	c.WindowWidth = width
	c.WindowHeight = height
	return c.SaveConfig(c)
}

func (c *Config) SaveLastDatabasePath(path string) error {
	c.LastDatabasePath = path
	return c.SaveConfig(c)
}

func (c *Config) GetLastDatabasePath() string {
	return c.LastDatabasePath
}

// GetLanguage returns the persisted UI language code (defaults to "en").
func (c *Config) GetLanguage() string {
	if c.Language == "" {
		return "en"
	}
	return c.Language
}

// SaveLanguage persists the given UI language code to disk.
func (c *Config) SaveLanguage(lang string) error {
	c.Language = lang
	return c.SaveConfig(c)
}

// GetBoardColors returns the persisted board palette (empty fields defaulted).
func (c *Config) GetBoardColors() BoardColors {
	return c.BoardColors.withDefaults()
}

// SaveBoardColors persists the given board palette to disk.
func (c *Config) SaveBoardColors(colors BoardColors) error {
	c.BoardColors = colors.withDefaults()
	return c.SaveConfig(c)
}

// GetUIScale returns the persisted interface scale as a percentage (clamped to
// the supported range; defaults to 100).
func (c *Config) GetUIScale() int {
	return clampUIScale(c.UIScale)
}

// SaveUIScale persists the given interface scale (percentage) to disk, clamped
// to the supported range.
func (c *Config) SaveUIScale(scale int) error {
	c.UIScale = clampUIScale(scale)
	return c.SaveConfig(c)
}

// GetPanelPosition returns the persisted panel position mode (sanitised;
// defaults to "bottom").
func (c *Config) GetPanelPosition() string {
	return sanitizePanelPosition(c.PanelPosition)
}

// SavePanelPosition persists the given panel position mode to disk, coerced to
// a known value.
func (c *Config) SavePanelPosition(pos string) error {
	c.PanelPosition = sanitizePanelPosition(pos)
	return c.SaveConfig(c)
}

// GetPanelHeight returns the persisted bottom-mode panel height in pixels
// (clamped; defaults to 380).
func (c *Config) GetPanelHeight() int {
	return clampPanelHeight(c.PanelHeight)
}

// SavePanelHeight persists the bottom-mode panel height, clamped to the
// supported range.
func (c *Config) SavePanelHeight(height int) error {
	c.PanelHeight = clampPanelHeight(height)
	return c.SaveConfig(c)
}

// GetPanelWidth returns the persisted side-mode panel width in pixels
// (clamped; defaults to 520).
func (c *Config) GetPanelWidth() int {
	return clampPanelWidth(c.PanelWidth)
}

// SavePanelWidth persists the side-mode panel width, clamped to the
// supported range.
func (c *Config) SavePanelWidth(width int) error {
	c.PanelWidth = clampPanelWidth(width)
	return c.SaveConfig(c)
}

// GetTourSeen reports whether the first-run guided-tour catalog has been shown.
func (c *Config) GetTourSeen() bool {
	return c.TourSeen
}

// SaveTourSeen persists whether the first-run guided-tour catalog has been shown.
func (c *Config) SaveTourSeen(seen bool) error {
	c.TourSeen = seen
	return c.SaveConfig(c)
}

// GetTabOrder returns the persisted tab order for the tabbed panel. Empty
// means "no custom order yet" — the frontend falls back to its built-in order.
func (c *Config) GetTabOrder() []string {
	return c.TabOrder
}

// SaveTabOrder persists the tab order reached after a drag-to-reorder.
func (c *Config) SaveTabOrder(order []string) error {
	c.TabOrder = order
	return c.SaveConfig(c)
}

// GetHiddenTabs returns the ids of tabs the user chose to hide from the
// tabbed panel's tab bar.
func (c *Config) GetHiddenTabs() []string {
	return c.HiddenTabs
}

// SaveHiddenTabs persists the set of hidden tab ids.
func (c *Config) SaveHiddenTabs(hidden []string) error {
	c.HiddenTabs = hidden
	return c.SaveConfig(c)
}

// GetBearoffTSPath returns the persisted external two-sided bearoff path.
func (c *Config) GetBearoffTSPath() string {
	return c.BearoffTSPath
}

// SaveBearoffTSPath persists the external two-sided bearoff path ("" clears
// it) and applies it to the running engine immediately.
func (c *Config) SaveBearoffTSPath(path string) error {
	c.BearoffTSPath = path
	race.SetExternalPath(path)
	return c.SaveConfig(c)
}

// GetBearoffRate returns the sweep rate measured on this machine, 0 when no
// representative run has finished here yet.
func (c *Config) GetBearoffRate() float64 {
	return c.BearoffRate
}

// SaveBearoffRate persists the sweep rate measured on this machine.
func (c *Config) SaveBearoffRate(rate float64) error {
	c.BearoffRate = rate
	return c.SaveConfig(c)
}

// GetBearoffCores returns the core count the user last generated with, 0 for
// the default.
func (c *Config) GetBearoffCores() int {
	return c.BearoffCores
}

// SaveBearoffCores persists the core count for the next generation.
func (c *Config) SaveBearoffCores(cores int) error {
	c.BearoffCores = cores
	return c.SaveConfig(c)
}

// GetEpcChallenge returns the persisted EPC training-mode flag.
func (c *Config) GetEpcChallenge() bool {
	return c.EpcChallenge
}

// SaveEpcChallenge persists the EPC training-mode flag.
func (c *Config) SaveEpcChallenge(on bool) error {
	c.EpcChallenge = on
	return c.SaveConfig(c)
}

// GetGammonNetDisplayPly returns the persisted interactive-display search
// depth (clamped; defaults to 2-ply when unset). Comfort only — never written
// to a Position's Analysis row.
func (c *Config) GetGammonNetDisplayPly() int {
	if c.GammonNetDisplayPly == nil {
		return DefaultGammonNetPly
	}
	return clampGammonNetPly(*c.GammonNetDisplayPly)
}

// SaveGammonNetDisplayPly persists the interactive-display search depth.
func (c *Config) SaveGammonNetDisplayPly(ply int) error {
	v := clampGammonNetPly(ply)
	c.GammonNetDisplayPly = &v
	return c.SaveConfig(c)
}

// GetGammonNetAnalysisPly returns the persisted batch-analysis search depth
// (clamped; defaults to 2-ply when unset) — what the batch (#129) writes to
// Analysis.
func (c *Config) GetGammonNetAnalysisPly() int {
	if c.GammonNetAnalysisPly == nil {
		return DefaultGammonNetPly
	}
	return clampGammonNetPly(*c.GammonNetAnalysisPly)
}

// SaveGammonNetAnalysisPly persists the batch-analysis search depth.
func (c *Config) SaveGammonNetAnalysisPly(ply int) error {
	v := clampGammonNetPly(ply)
	c.GammonNetAnalysisPly = &v
	return c.SaveConfig(c)
}

// GetGammonNetPruneK returns the persisted pruning width (clamped; defaults
// to 12, the canonical value).
func (c *Config) GetGammonNetPruneK() int {
	return clampGammonNetPruneK(c.GammonNetPruneK)
}

// SaveGammonNetPruneK persists the pruning width.
func (c *Config) SaveGammonNetPruneK(k int) error {
	c.GammonNetPruneK = clampGammonNetPruneK(k)
	return c.SaveConfig(c)
}

// GetGammonNetCandidates returns the persisted number of candidate moves
// shown (clamped; defaults to 10).
func (c *Config) GetGammonNetCandidates() int {
	return clampGammonNetCandidates(c.GammonNetCandidates)
}

// SaveGammonNetCandidates persists the number of candidate moves shown.
func (c *Config) SaveGammonNetCandidates(n int) error {
	c.GammonNetCandidates = clampGammonNetCandidates(n)
	return c.SaveConfig(c)
}

// GetGammonNetAutoAnalyze returns whether an import that brought no analysis
// triggers the batch job automatically (#129).
func (c *Config) GetGammonNetAutoAnalyze() bool {
	return c.GammonNetAutoAnalyze
}

// SaveGammonNetAutoAnalyze persists the auto-analyze-after-import flag.
func (c *Config) SaveGammonNetAutoAnalyze(on bool) error {
	c.GammonNetAutoAnalyze = on
	return c.SaveConfig(c)
}

// GetTheme returns the persisted interface theme ("" = follow the desktop).
func (c *Config) GetTheme() string {
	return c.Theme
}

// SaveTheme persists the chosen interface theme.
func (c *Config) SaveTheme(name string) error {
	c.Theme = name
	return c.SaveConfig(c)
}

// GetWatchFolder returns whether the watched folder is on, its path, and the
// interval in seconds (0 = the default). Three values rather than three
// getters: the frontend reads them together, and a half-read setting — on
// with no path — is the shape that produces a watch nobody asked for.
func (c *Config) GetWatchFolder() (bool, string, int) {
	return c.WatchFolder, c.WatchFolderPath, c.WatchFolderIntervalSeconds
}

// SaveWatchFolder persists the watched folder. An empty path turns the watch
// off whatever `on` says: "watch, but nowhere" is not a state worth storing.
func (c *Config) SaveWatchFolder(on bool, path string, intervalSeconds int) error {
	if path == "" {
		on = false
	}
	if intervalSeconds < 0 {
		intervalSeconds = 0
	}
	c.WatchFolder = on
	c.WatchFolderPath = path
	c.WatchFolderIntervalSeconds = intervalSeconds
	return c.SaveConfig(c)
}

// GetStatsFilter returns the persisted stats filter (called from the frontend).
func (c *Config) GetStatsFilter() StatsFilterPersisted {
	return c.StatsFilter
}

// SaveStatsFilter persists the given stats filter to disk.
func (c *Config) SaveStatsFilter(filter StatsFilterPersisted) error {
	c.StatsFilter = filter
	return c.SaveConfig(c)
}

// GetCheckForUpdates returns whether gui.App.CheckForUpdate is allowed to
// query the GitHub Releases API. Off by default (#241).
func (c *Config) GetCheckForUpdates() bool {
	return c.CheckForUpdates
}

// SaveCheckForUpdates persists the update-check opt-in.
func (c *Config) SaveCheckForUpdates(on bool) error {
	c.CheckForUpdates = on
	return c.SaveConfig(c)
}
