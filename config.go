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

// The config is JSON under both names; legacyConfigFilePath is read only
// until LoadConfig has migrated it to configFilePath.
const (
	configFilePath       = "blunderDB/config.json"
	legacyConfigFilePath = "blunderDB/config.yaml"
)

// currentConfigVersion is bumped only when a field is renamed or reshaped
// (an added field needs no migration).
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

// BoardColors holds the user-customisable board palette; empty fields fall
// back to DefaultBoardColors().
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

// DefaultBoardColors returns Board.svelte's built-in palette.
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

// Bounds for the `like` ranking preferences (ADR-0043).
const (
	// DefaultLikeLimit: a ranked list is browsed, not read at a glance.
	DefaultLikeLimit = 30
	// MaxLikeLimit: a ranking is a reading list, not an export.
	MaxLikeLimit = 500
)

// Panel position modes; "auto" follows the window aspect ratio (frontend-side).
const (
	PanelPositionBottom  = "bottom"
	PanelPositionSide    = "side"
	PanelPositionAuto    = "auto"
	DefaultPanelPosition = PanelPositionBottom
)

// Panel size in pixels: height in bottom mode, width in side mode. The
// defaults apply only until the user first drags the resize handle.
const (
	MinPanelHeight     = 80
	MaxPanelHeight     = 4000
	DefaultPanelHeight = 250

	MinPanelWidth     = 150
	MaxPanelWidth     = 4000
	DefaultPanelWidth = 420
)

// gammonNet settings (ADR-0011, ADR-0013). Display and analysis depths stay
// separate so a comfort setting never degrades what the batch persists.
const (
	MinGammonNetPly = 0
	MaxGammonNetPly = gammonnet.MaxPly

	MinGammonNetPruneK = 1
	MaxGammonNetPruneK = 64

	MinGammonNetCandidates     = 1
	MaxGammonNetCandidates     = 50
	DefaultGammonNetCandidates = 10
)

// DefaultGammonNetPly and DefaultGammonNetPruneK are vars: they come from
// gammonnet's embedded canonical export (gammonnet.DefaultPly/DefaultPruneK).
var (
	DefaultGammonNetPly    = gammonnet.DefaultPly
	DefaultGammonNetPruneK = gammonnet.DefaultPruneK
)

type Config struct {
	// ConfigVersion is currentConfigVersion once loaded or saved; 0 on a
	// file that predates the field.
	ConfigVersion    int                  `json:"config_version"`
	WindowWidth      int                  `json:"window_width"`
	WindowHeight     int                  `json:"window_height"`
	LastDatabasePath string               `json:"last_database_path"`
	StatsFilter      StatsFilterPersisted `json:"stats_filter,omitempty"`
	Language         string               `json:"language,omitempty"`
	BoardColors      BoardColors          `json:"board_colors,omitempty"`
	UIScale          int                  `json:"ui_scale,omitempty"`
	// LikeLimit is how many neighbours a `like` ranking returns;
	// LikeMaxDistance the checker-pip ceiling, 0 = none. No default ceiling:
	// its scale depends on the phase and is unmeasured (ADR-0043 rule 4).
	LikeLimit       int    `json:"like_limit,omitempty"`
	LikeMaxDistance int    `json:"like_max_distance,omitempty"`
	PanelPosition   string `json:"panel_position,omitempty"`
	PanelHeight     int    `json:"panel_height,omitempty"`
	PanelWidth      int    `json:"panel_width,omitempty"`
	TourSeen        bool   `json:"tour_seen,omitempty"`
	// TabOrder is the user's order of TabbedPanel.svelte's tab ids; empty
	// means the built-in order, which the frontend owns.
	TabOrder []string `json:"tab_order,omitempty"`
	// HiddenTabs removes tab buttons only; their shortcuts still work.
	HiddenTabs []string `json:"hidden_tabs,omitempty"`
	// BearoffTSPath is an optional user-supplied two-sided bearoff database
	// (.bd) widening the generated TS-06-06 (ADR-0009). Empty = none.
	BearoffTSPath string `json:"bearoff_ts_path,omitempty"`
	// BearoffRate is seconds per n³ (bearoffgen's cost model) measured on
	// one core of this machine, so time estimates are the user's own. 0
	// until a representative run has finished here.
	BearoffRate float64 `json:"bearoff_rate,omitempty"`
	// BearoffCores is the core count the user last chose for a generation.
	// 0 = the default, every core but one.
	BearoffCores int `json:"bearoff_cores,omitempty"`
	// EpcChallenge persists the EPC panel's training mode ("défi"): results
	// are masked after each edit until the user clicks a zone to reveal it.
	EpcChallenge bool `json:"epc_challenge,omitempty"`
	// TrainingSeedSources is each Training exercise's last seed source
	// ("pool", "board", "library"; ADR-0041 rule 2). Per exercise because
	// they offer different sources; here, not in library metadata, because
	// it is a habit of the user, not a property of the file.
	TrainingSeedSources map[string]string `json:"training_seed_sources,omitempty"`
	// The depths are pointers because 0-ply is a valid choice: nil means
	// unset (canonical default), a non-nil zero means 0-ply chosen.
	GammonNetDisplayPly  *int `json:"gammonnet_display_ply,omitempty"`
	GammonNetAnalysisPly *int `json:"gammonnet_analysis_ply,omitempty"`
	GammonNetPruneK      int  `json:"gammonnet_prune_k,omitempty"`
	GammonNetCandidates  int  `json:"gammonnet_candidates,omitempty"`
	GammonNetAutoAnalyze bool `json:"gammonnet_auto_analyze,omitempty"`
	// CheckForUpdates opts into gui.App.CheckForUpdate at startup. Off by
	// default (no unasked network call); forced off on a package-managed
	// install (gui.isPackageManaged), whose package manager owns updates.
	CheckForUpdates bool `json:"check_for_updates,omitempty"`
	// Theme is "system", "light", "dark", "contrast" or "print"; empty
	// means system. The palettes live in frontend utils/themes.js only.
	Theme string `json:"theme,omitempty"`

	// Watched folder: files appearing in it are imported while blunderDB
	// runs. Off by default. An interval of 0 means watch.DefaultInterval;
	// watch.ClampInterval owns the bounds.
	WatchFolder                bool   `json:"watch_folder,omitempty"`
	WatchFolderPath            string `json:"watch_folder_path,omitempty"`
	WatchFolderIntervalSeconds int    `json:"watch_folder_interval_seconds,omitempty"`
}

// clampUIScale clamps scale to the supported range, 0 meaning the default.
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

// sanitizePanelPosition maps an empty or unknown position to bottom.
func sanitizePanelPosition(pos string) string {
	switch pos {
	case PanelPositionBottom, PanelPositionSide, PanelPositionAuto:
		return pos
	default:
		return DefaultPanelPosition
	}
}

// clampPanelHeight clamps height to the supported range, 0 meaning the default.
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

// clampPanelWidth is clampPanelHeight for the side-mode width.
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

// clampGammonNetPly clamps an explicit depth; 0 is a valid depth, not unset.
func clampGammonNetPly(ply int) int {
	if ply < MinGammonNetPly {
		return MinGammonNetPly
	}
	if ply > MaxGammonNetPly {
		return MaxGammonNetPly
	}
	return ply
}

// clampGammonNetPruneK clamps k, 0 meaning the canonical default.
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

// clampGammonNetCandidates clamps n, 0 meaning the default.
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

// LoadConfig reads the persisted config. No file: a default is created. Only
// the legacy name: read, then re-saved under the current one. Unparseable:
// backed up to config.json.bak and replaced by a default, so the app starts.
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
		// Nothing to migrate yet: only stamp the version.
		config.ConfigVersion = currentConfigVersion
	}

	// The Wails-bound getters read the receiver.
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

// SaveConfig writes config as indented JSON atomically (temp file, fsync,
// rename), so a crash never leaves config.json half-written.
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

// GetLikeLimit reports how many neighbours a ranking returns (DefaultLikeLimit
// when unset).
func (c *Config) GetLikeLimit() int {
	if c.LikeLimit <= 0 {
		return DefaultLikeLimit
	}
	if c.LikeLimit > MaxLikeLimit {
		return MaxLikeLimit
	}
	return c.LikeLimit
}

// SaveLikeLimit persists how many neighbours a ranking returns.
func (c *Config) SaveLikeLimit(n int) error {
	if n < 0 {
		n = 0
	}
	if n > MaxLikeLimit {
		n = MaxLikeLimit
	}
	c.LikeLimit = n
	return c.SaveConfig(c)
}

// GetLikeMaxDistance reports the ceiling, in checker-pips, beyond which a
// position stops being a neighbour. Zero means no ceiling.
func (c *Config) GetLikeMaxDistance() int {
	if c.LikeMaxDistance < 0 {
		return 0
	}
	return c.LikeMaxDistance
}

// SaveLikeMaxDistance persists that ceiling. Zero removes it.
func (c *Config) SaveLikeMaxDistance(d int) error {
	if d < 0 {
		d = 0
	}
	c.LikeMaxDistance = d
	return c.SaveConfig(c)
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

// GetTrainingSeedSources returns the remembered seed source of each Training
// exercise, empty when none was ever chosen.
func (c *Config) GetTrainingSeedSources() map[string]string {
	out := make(map[string]string, len(c.TrainingSeedSources))
	for exercise, source := range c.TrainingSeedSources {
		out[exercise] = source
	}
	return out
}

// SaveTrainingSeedSource remembers the source one exercise was last started
// with. An empty exercise is ignored rather than stored under "": a key nobody
// can ask for again is a leak, not a memory.
func (c *Config) SaveTrainingSeedSource(exercise, source string) error {
	if exercise == "" {
		return nil
	}
	if c.TrainingSeedSources == nil {
		c.TrainingSeedSources = map[string]string{}
	}
	c.TrainingSeedSources[exercise] = source
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
// (clamped; defaults to 2-ply when unset) — what the batch writes to
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
// triggers the batch job automatically.
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

// WatchFolderSettings is the watched folder as one value, so the frontend
// never half-reads it (on with no path). A struct because Wails binds only
// (value) or (value, error): a third return silently resolves to null.
type WatchFolderSettings struct {
	On              bool   `json:"on"`
	Path            string `json:"path"`
	IntervalSeconds int    `json:"intervalSeconds"` // 0 = the default
}

// GetWatchFolder returns whether the watched folder is on, its path, and the
// interval in seconds.
func (c *Config) GetWatchFolder() WatchFolderSettings {
	return WatchFolderSettings{
		On:              c.WatchFolder,
		Path:            c.WatchFolderPath,
		IntervalSeconds: c.WatchFolderIntervalSeconds,
	}
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
// query the GitHub Releases API. Off by default.
func (c *Config) GetCheckForUpdates() bool {
	return c.CheckForUpdates
}

// SaveCheckForUpdates persists the update-check opt-in.
func (c *Config) SaveCheckForUpdates(on bool) error {
	c.CheckForUpdates = on
	return c.SaveConfig(c)
}
