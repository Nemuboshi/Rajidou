package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config defines runtime settings loaded from YAML.
type Config struct {
	// Links contains input URLs to resolve and download.
	Links []string `yaml:"links"`
	// OutputDir is the target directory for downloaded files.
	OutputDir string `yaml:"outputDir"`
	// AreaID optionally scopes downloads to a specific area.
	AreaID string `yaml:"areaId"`
	// Jobs controls maximum parallel downloads.
	Jobs int `yaml:"jobs"`
}

// Load reads, validates, and normalizes config from a YAML file path.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return Config{}, err
	}
	if len(c.Links) == 0 {
		return Config{}, fmt.Errorf("config must contain a non-empty `links` array")
	}
	// Keep defaults centralized so callers can rely on normalized values.
	if c.OutputDir == "" {
		c.OutputDir = "downloads"
	}
	if c.Jobs <= 0 {
		c.Jobs = 2
	}
	return c, nil
}
