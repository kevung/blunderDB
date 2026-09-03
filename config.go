package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/adrg/xdg"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/race"
)

const configFilePath = "blunderDB/config.yaml"

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
// Both default to the canonical parameters — 2-ply, pruning k=12 — matching
// gammonnet.DefaultConfig / gammonnet.DefaultPruneK.
const (
	MinGammonNetPly     = 0
	MaxGammonNetPly     = 4 // gammonnet.MaxPly
	DefaultGammonNetPly = 2

	MinGammonNetPruneK     = 1
	MaxGammonNetPruneK     = 64
	DefaultGammonNetPruneK = 12 // gammonnet.DefaultPruneK

	MinGammonNetCandidates     = 1
	MaxGammonNetCandidates     = 50
	DefaultGammonNetCandidates = 10
)

type Config struct {
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
	// (.bd) widening the embedded TS-06-06 (ADR-0009). Empty = none.
	BearoffTSPath string `json:"bearoff_ts_path,omitempty"`
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

func (c *Config) LoadConfig() (*Config, error) {
	configPath, err := xdg.SearchConfigFile(configFilePath)
	if err != nil {
		log.Println("Config file not found, creating a new one.")
		config := NewConfig()
		if err := c.SaveConfig(config); err != nil {
			return nil, err
		}
		return config, nil
	}
	log.Println("Config file was found at:", configPath)

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, err
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
	c.EpcChallenge = config.EpcChallenge
	c.GammonNetDisplayPly = config.GammonNetDisplayPly
	c.GammonNetAnalysisPly = config.GammonNetAnalysisPly
	c.GammonNetPruneK = clampGammonNetPruneK(config.GammonNetPruneK)
	config.GammonNetPruneK = c.GammonNetPruneK
	c.GammonNetCandidates = clampGammonNetCandidates(config.GammonNetCandidates)
	config.GammonNetCandidates = c.GammonNetCandidates
	c.GammonNetAutoAnalyze = config.GammonNetAutoAnalyze

	return &config, nil
}

func (c *Config) SaveConfig(config *Config) error {
	configPath, err := xdg.ConfigFile(configFilePath)
	if err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, bytes, 0644)
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

// GetStatsFilter returns the persisted stats filter (called from the frontend).
func (c *Config) GetStatsFilter() StatsFilterPersisted {
	return c.StatsFilter
}

// SaveStatsFilter persists the given stats filter to disk.
func (c *Config) SaveStatsFilter(filter StatsFilterPersisted) error {
	c.StatsFilter = filter
	return c.SaveConfig(c)
}
