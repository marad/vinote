package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/marad/vinote/internal/config"
)

func TestInitCmd_CreatesConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cmd := InitCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(tmpDir, "vinote", "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	content, _ := os.ReadFile(configPath)
	if !strings.Contains(string(content), "notes_dir") {
		t.Error("config file missing notes_dir field")
	}

	if !strings.Contains(out.String(), "created") {
		t.Error("output should mention config was created")
	}
}

func TestInitCmd_ExistingConfigSkips(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	configDir := filepath.Join(tmpDir, "vinote")
	os.MkdirAll(configDir, 0o755)
	original := []byte("notes_dir = \"~/custom\"\n")
	os.WriteFile(filepath.Join(configDir, "config.toml"), original, 0o644)

	cmd := InitCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(configDir, "config.toml"))
	if string(content) != string(original) {
		t.Error("existing config should not be modified without --force")
	}

	if !strings.Contains(out.String(), "already exists") {
		t.Error("output should mention config already exists")
	}
}

func TestInitCmd_ForceOverwrites(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	configDir := filepath.Join(tmpDir, "vinote")
	os.MkdirAll(configDir, 0o755)
	os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("old content"), 0o644)

	cmd := InitCmd()
	cmd.SetArgs([]string{"--force"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(configDir, "config.toml"))
	if !strings.Contains(string(content), "notes_dir") {
		t.Error("config should be overwritten with defaults when --force is used")
	}

	if !strings.Contains(out.String(), "created") {
		t.Error("output should mention config was created")
	}
}

func TestInitCmd_MissingNotesDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir) // notes_dir defaults to ~/notes which won't exist

	cmd := InitCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(stderr.String(), "Warning") {
		t.Error("should warn about missing notes directory")
	}

	if !strings.Contains(stdout.String(), "does not exist") {
		t.Error("summary should indicate notes directory does not exist")
	}
}

func TestDefaultConfigTOML_MatchesDefaults(t *testing.T) {
	var parsed config.Config
	if _, err := toml.Decode(config.DefaultConfigTOML(), &parsed); err != nil {
		t.Fatalf("DefaultConfigTOML is not valid TOML: %v", err)
	}

	defaults := config.DefaultConfig()

	if parsed.NotesDir != defaults.NotesDir {
		t.Errorf("NotesDir: got %q, want %q", parsed.NotesDir, defaults.NotesDir)
	}
	if parsed.Editor != defaults.Editor {
		t.Errorf("Editor: got %q, want %q", parsed.Editor, defaults.Editor)
	}
	if parsed.WeeklyDir != defaults.WeeklyDir {
		t.Errorf("WeeklyDir: got %q, want %q", parsed.WeeklyDir, defaults.WeeklyDir)
	}
	if parsed.WeeklyTemplate != defaults.WeeklyTemplate {
		t.Errorf("WeeklyTemplate: got %q, want %q", parsed.WeeklyTemplate, defaults.WeeklyTemplate)
	}
	if len(parsed.SkipDirs) != len(defaults.SkipDirs) {
		t.Errorf("SkipDirs length: got %d, want %d", len(parsed.SkipDirs), len(defaults.SkipDirs))
	}
}
