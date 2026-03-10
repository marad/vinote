package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the application configuration.
type Config struct {
	NotesDir       string   `toml:"notes_dir"`
	Editor         string   `toml:"editor"`
	WeeklyDir      string   `toml:"weekly_dir"`
	WeeklyTemplate string   `toml:"weekly_template"`
	SkipDirs       []string `toml:"skip_dirs"`
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() Config {
	return Config{
		NotesDir:       "~/notes",
		Editor:         "nvim",
		WeeklyDir:      "Allegro/Journal/Week",
		WeeklyTemplate: "templates/Weekly.md",
		SkipDirs:       []string{"_plug", "Library", ".git", "archive"},
	}
}

// Load reads the config from ~/.config/vinote/config.toml.
// If the file doesn't exist, defaults are used.
func Load() (Config, error) {
	cfg := DefaultConfig()

	configPath := filepath.Join(configDir(), "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return expandPaths(cfg), nil
	}

	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		return Config{}, err
	}

	// Apply defaults for unset skip_dirs
	if cfg.SkipDirs == nil {
		cfg.SkipDirs = DefaultConfig().SkipDirs
	}

	return expandPaths(cfg), nil
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "vinote")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "vinote")
}

// ConfigDir returns the configuration directory path (for cache storage etc.).
func ConfigDir() string {
	return configDir()
}

func expandTilde(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, path[1:])
}

func expandPaths(cfg Config) Config {
	cfg.NotesDir = expandTilde(cfg.NotesDir)
	// WeeklyDir and WeeklyTemplate are relative to NotesDir, no expansion needed
	return cfg
}

// NotesAbsPath returns the absolute path to the notes directory.
func (c Config) NotesAbsPath() string {
	return c.NotesDir
}

// WeeklyAbsDir returns the absolute path to the weekly notes directory.
func (c Config) WeeklyAbsDir() string {
	return filepath.Join(c.NotesDir, c.WeeklyDir)
}

// WeeklyTemplateAbsPath returns the absolute path to the weekly template.
func (c Config) WeeklyTemplateAbsPath() string {
	return filepath.Join(c.NotesDir, c.WeeklyTemplate)
}
