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

type Config struct {
	WindowWidth      int    `json:"window_width"`
	WindowHeight     int    `json:"window_height"`
	LastDatabasePath string `json:"last_database_path"`
}

func NewConfig() *Config {
	initialWidth, initialHeight := calculateInitialDimensions()
	return &Config{
		WindowWidth:  initialWidth,
		WindowHeight: initialHeight,
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
