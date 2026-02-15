package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Color is an RGB triplet [R, G, B].
type Color [3]uint8

// Config holds user-customizable settings.
type Config struct {
	Events map[string]Color `json:"events"`
}

// defaults returns the built-in color map (current hardcoded values).
func defaults() map[string]Color {
	return map[string]Color{
		"session_start":       {80, 160, 255},
		"session_end":         {80, 160, 255},
		"prompt_acknowledged": {0, 255, 255},
		"tool_executing":      {255, 255, 0},
		"tool_completed":      {0, 255, 0},
		"task_complete":       {0, 255, 0},
		"subtask_complete":    {0, 255, 0},
		"needs_attention":     {255, 0, 0},
		"waiting_for_input":   {255, 165, 0},
		"context_compacting":  {128, 0, 255},
		"notification":        {255, 255, 255},
	}
}

// configPath returns the platform-appropriate config file path.
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(dir, "agent-chroma", "config.json"), nil
}

// writeDefaults creates the config file with default values so users can
// easily discover and edit it.
func writeDefaults(path string, cfg *Config) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[agent-chroma] Failed to create config dir: %v\n", err)
		return
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "[agent-chroma] Failed to write default config: %v\n", err)
	}
}

// Load reads the config file and merges it with defaults.
// If the file doesn't exist, it is created with default values.
func Load() *Config {
	cfg := &Config{Events: defaults()}

	path, err := configPath()
	if err != nil {
		return cfg
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist — create it with defaults.
		writeDefaults(path, cfg)
		return cfg
	}

	var userCfg Config
	if err := json.Unmarshal(data, &userCfg); err != nil {
		fmt.Fprintf(os.Stderr, "[agent-chroma] Invalid config %s: %v\n", path, err)
		return cfg
	}

	// Merge: user values override defaults.
	for k, v := range userCfg.Events {
		cfg.Events[k] = v
	}

	return cfg
}
