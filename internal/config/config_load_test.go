package config

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoadProfileDefaultsFlashAttnToAutoWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.toml")
	content := strings.Join([]string{
		`name = "legacy"`,
		`model = "/models/qwen.gguf"`,
		`host = "0.0.0.0"`,
		`port = 8080`,
		`ctx = 65536`,
		`ngl = 99`,
		`cache_k = "q4_0"`,
		`cache_v = "q8_0"`,
		`temp = 0.100`,
		`top_p = 0.950`,
		`top_k = 40`,
		`min_p = 0.050`,
		`batch = 512`,
		`ubatch = 256`,
		`parallel = 1`,
		`threads = 0`,
		`jinja = true`,
		`metrics = true`,
		`no_mmap = true`,
		`chat_template = "chatml"`,
		`extra_args = []`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write legacy profile: %v", err)
	}

	profile, err := LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}
	if profile.FlashAttn != "auto" {
		t.Fatalf("expected missing flash_attn to default to auto, got %q", profile.FlashAttn)
	}
}

func TestLoadProfileRejectsInvalidFlashAttnValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.toml")
	content := strings.Join([]string{
		`name = "invalid"`,
		`model = "/models/qwen.gguf"`,
		`host = "0.0.0.0"`,
		`port = 8080`,
		`flash_attn = "maybe"`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write invalid profile: %v", err)
	}

	_, err := LoadProfile(path)
	if err == nil {
		t.Fatal("expected invalid flash_attn to fail validation")
	}
	if !strings.Contains(err.Error(), "flash_attn must be one of: auto, on, off") {
		t.Fatalf("unexpected error: %v", err)
	}
}
