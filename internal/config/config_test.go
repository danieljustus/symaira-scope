package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeGlobalConfig writes $HOME/.config/symscope/config.toml and points HOME
// (and USERPROFILE, which os.UserHomeDir reads on Windows) at an isolated temp
// dir so no real user state is touched.
func writeGlobalConfig(t *testing.T, content string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := filepath.Join(home, ".config", "symscope")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDefaults(t *testing.T) {
	d := Defaults()
	if d.Ports.SuggestFrom != 3000 {
		t.Errorf("SuggestFrom = %d, want 3000", d.Ports.SuggestFrom)
	}
	if d.Ports.SuggestTo != 9999 {
		t.Errorf("SuggestTo = %d, want 9999", d.Ports.SuggestTo)
	}
}

func TestLoadNoConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SYMSCOPE_PORTS_SUGGEST_FROM", "")
	t.Setenv("SYMSCOPE_PORTS_SUGGEST_TO", "")
	loader.ResetCache()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Ports.SuggestFrom != 3000 || cfg.Ports.SuggestTo != 9999 {
		t.Errorf("got %d/%d, want defaults 3000/9999", cfg.Ports.SuggestFrom, cfg.Ports.SuggestTo)
	}
}

func TestLoadFileWinsOverDefaults(t *testing.T) {
	writeGlobalConfig(t, "[ports]\nsuggest_from = 4000\nsuggest_to = 5000\n")
	t.Setenv("SYMSCOPE_PORTS_SUGGEST_FROM", "")
	t.Setenv("SYMSCOPE_PORTS_SUGGEST_TO", "")
	loader.ResetCache()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Ports.SuggestFrom != 4000 || cfg.Ports.SuggestTo != 5000 {
		t.Errorf("got %d/%d, want file values 4000/5000", cfg.Ports.SuggestFrom, cfg.Ports.SuggestTo)
	}
}

func TestLoadEnvWinsOverFile(t *testing.T) {
	writeGlobalConfig(t, "[ports]\nsuggest_from = 4000\nsuggest_to = 5000\n")
	t.Setenv("SYMSCOPE_PORTS_SUGGEST_FROM", "6000")
	t.Setenv("SYMSCOPE_PORTS_SUGGEST_TO", "")
	loader.ResetCache()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Ports.SuggestFrom != 6000 {
		t.Errorf("SuggestFrom = %d, want env value 6000", cfg.Ports.SuggestFrom)
	}
	if cfg.Ports.SuggestTo != 5000 {
		t.Errorf("SuggestTo = %d, want file value 5000 (env only overrides SuggestFrom)", cfg.Ports.SuggestTo)
	}
}
