package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MCPServer holds configuration for an MCP server
type MCPServer struct {
	Name        string   `json:"name"`
	Transport   string   `json:"transport,omitempty"`
	Command     string   `json:"command"`
	Args        []string `json:"args,omitempty"`
	Env         []string `json:"env,omitempty"`
	WorkingDir  string   `json:"working_dir,omitempty"`
	Description string   `json:"description,omitempty"`
}

// Settings represents the main Soren settings
type Settings struct {
	MCPServers []MCPServer `json:"mcpServers,omitempty"`
}

// DefaultSettingsPath returns the default path for the settings file
func DefaultSettingsPath() string {
	return filepath.Join(".soren", "settings.json")
}

// LoadSettings loads the settings from the default location
func LoadSettings() (*Settings, error) {
	return LoadSettingsFromFile(DefaultSettingsPath())
}

// LoadSettingsFromFile loads settings from a specific file
func LoadSettingsFromFile(settingsPath string) (*Settings, error) {
	// Check if settings file exists
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		// Return empty settings if file doesn't exist
		return &Settings{}, nil
	}

	file, err := os.Open(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open settings file %s: %w", settingsPath, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings file: %w", err)
	}

	return &settings, nil
}

// SaveSettings saves settings to the default location
func SaveSettings(settings *Settings) error {
	return SaveSettingsToFile(DefaultSettingsPath(), settings)
}

// SaveSettingsToFile saves settings to a specific file
func SaveSettingsToFile(settingsPath string, settings *Settings) error {
	// Ensure directory exists
	dir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}
