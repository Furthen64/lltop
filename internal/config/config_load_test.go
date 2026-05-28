package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobalConfigMissingDoesNotCreateFilesBeforeWizard(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, created, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("LoadGlobalConfig failed: %v", err)
	}
	if !created {
		t.Fatal("expected created=true when config is missing")
	}
	if cfg == nil {
		t.Fatal("expected config defaults")
	}

	appDir := filepath.Join(home, ".config", "lltop")
	if _, err := os.Stat(appDir); !os.IsNotExist(err) {
		t.Fatalf("expected no app dir before wizard completes, got err=%v", err)
	}
}
