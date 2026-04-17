package encrypter

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings holds user-configurable options that are persisted to disk
// as a JSON file in the OS config directory.
//
// Locations:
//   - Windows: %APPDATA%\PersonalSecureEncrypter\config.json
//   - Linux:   ~/.config/PersonalSecureEncrypter/config.json
//   - macOS:   ~/Library/Application Support/PersonalSecureEncrypter/config.json
type Settings struct {
	// Extension is appended to encrypted files (default ".pse").
	// Users can customize this to any value, e.g. ".enc", ".locked".
	Extension string `json:"extension"`

	// Theme is "dark" or "light" — controls the frontend appearance.
	Theme string `json:"theme"`

	// DeleteOriginals controls whether source files are deleted
	// after successful encryption.
	DeleteOriginals bool `json:"deleteOriginals"`

	// OutputFolder, if non-empty, causes all encrypted/decrypted files
	// to be written to this directory instead of next to the originals.
	OutputFolder string `json:"outputFolder"`

	// OutputMode is "alongside" (next to originals) or "separate"
	// (in a dedicated output folder). Currently OutputFolder takes
	// precedence when set.
	OutputMode string `json:"outputMode"`
}

// DefaultSettings returns the factory-default configuration.
func DefaultSettings() *Settings {
	return &Settings{
		Extension:       ".pse",
		Theme:           "dark",
		DeleteOriginals: false,
		OutputFolder:    "",
		OutputMode:      "alongside",
	}
}

// getConfigPath returns the absolute path to the settings JSON file.
func getConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return filepath.Join(configDir, "PersonalSecureEncrypter", "config.json")
}

// LoadSettings reads settings from disk. If the file does not exist or
// is corrupted, default settings are returned silently.
func LoadSettings() *Settings {
	path := getConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultSettings()
	}

	s := DefaultSettings()
	if err := json.Unmarshal(data, s); err != nil {
		return DefaultSettings()
	}
	return s
}

// Save writes the current settings to disk as formatted JSON.
// The config directory is created with 0700 permissions (owner-only).
func (s *Settings) Save() error {
	path := getConfigPath()
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	// 0600 = owner read/write only.
	return os.WriteFile(path, data, 0600)
}
