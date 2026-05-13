package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.MaxEntries != defaultMaxEntries {
		t.Errorf("expected MaxEntries=%d, got %d", defaultMaxEntries, cfg.MaxEntries)
	}
	if !cfg.KeepWindowOpen {
		t.Error("expected KeepWindowOpen=true by default")
	}
}

func TestLoad_MissingFileReturnsDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading missing config: %v", err)
	}
	if cfg.MaxEntries != defaultMaxEntries {
		t.Errorf("expected default MaxEntries, got %d", cfg.MaxEntries)
	}
	if !cfg.KeepWindowOpen {
		t.Error("expected default KeepWindowOpen=true")
	}
}

func TestSaveAndLoad(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	original := &Config{
		MaxEntries:     100,
		KeepWindowOpen: false,
	}
	if err := original.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if loaded.MaxEntries != 100 {
		t.Errorf("expected MaxEntries=100, got %d", loaded.MaxEntries)
	}
	if loaded.KeepWindowOpen {
		t.Error("expected KeepWindowOpen=false after save/load")
	}
}

func TestLoad_CorruptFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfgDir := filepath.Join(dir, ".config", "clipboard-manager")
	_ = os.MkdirAll(cfgDir, 0700)
	_ = os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte("not valid json{{{"), 0600)

	cfg, err := Load()
	if err == nil {
		t.Error("expected error loading corrupt config")
	}
	if cfg.MaxEntries != defaultMaxEntries {
		t.Errorf("expected default MaxEntries on corrupt file, got %d", cfg.MaxEntries)
	}
}

func TestSave_CreatesDirectoryIfMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := Default()
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save failed when directory did not exist: %v", err)
	}

	home := os.Getenv("HOME")
	path := filepath.Join(home, ".config", "clipboard-manager", "config.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected config file to be created but it does not exist")
	}
}
