package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/adrg/xdg"
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

type Config struct {
	WindowWidth      int                  `json:"window_width"`
	WindowHeight     int                  `json:"window_height"`
	LastDatabasePath string               `json:"last_database_path"`
	StatsFilter      StatsFilterPersisted `json:"stats_filter,omitempty"`
	Language         string               `json:"language,omitempty"`
	BoardColors      BoardColors          `json:"board_colors,omitempty"`
	UIScale          int                  `json:"ui_scale,omitempty"`
	TourSeen         bool                 `json:"tour_seen,omitempty"`
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

func NewConfig() *Config {
	initialWidth, initialHeight := calculateInitialDimensions()
	return &Config{
		WindowWidth:  initialWidth,
		WindowHeight: initialHeight,
		Language:     "en",
		BoardColors:  DefaultBoardColors(),
		UIScale:      DefaultUIScale,
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
	c.TourSeen = config.TourSeen

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

// GetTourSeen reports whether the first-run guided-tour catalog has been shown.
func (c *Config) GetTourSeen() bool {
	return c.TourSeen
}

// SaveTourSeen persists whether the first-run guided-tour catalog has been shown.
func (c *Config) SaveTourSeen(seen bool) error {
	c.TourSeen = seen
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
