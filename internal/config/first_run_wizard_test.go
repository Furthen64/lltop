package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverModelFilesRespectsDepth(t *testing.T) {
	root := t.TempDir()

	mkFile := func(rel string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}

	mkFile("top.gguf")
	mkFile("one/two/model.bin")
	mkFile("one/two/three/deep.gguf")
	mkFile("not-a-model.txt")

	models, err := DiscoverModelFiles(root, 3)
	if err != nil {
		t.Fatalf("DiscoverModelFiles failed: %v", err)
	}

	want := map[string]struct{}{
		filepath.Join(root, "top.gguf"):          {},
		filepath.Join(root, "one/two/model.bin"): {},
	}
	if len(models) != len(want) {
		t.Fatalf("expected %d models, got %d: %v", len(want), len(models), models)
	}
	for _, model := range models {
		if _, ok := want[model]; !ok {
			t.Fatalf("unexpected model path: %s", model)
		}
	}
}

func TestGenerateProfilesForModelsCreatesUniqueProfileFiles(t *testing.T) {
	profilesDir := t.TempDir()
	cfg := DefaultGlobalConfig()
	cfg.ProfilesDir = profilesDir
	cfg.LlamaServer = "/usr/bin/llama-server"

	existing := DefaultProfile(cfg, "foo")
	existing.Model = "/models/existing.gguf"
	if err := SaveProfile(filepath.Join(profilesDir, "foo.toml"), existing); err != nil {
		t.Fatalf("failed to seed existing profile: %v", err)
	}

	models := []string{
		"/models/foo.gguf",
		"/models/bar.gguf",
	}
	created, err := GenerateProfilesForModels(cfg, models)
	if err != nil {
		t.Fatalf("GenerateProfilesForModels failed: %v", err)
	}
	if created != 2 {
		t.Fatalf("expected 2 profiles created, got %d", created)
	}

	if _, err := os.Stat(filepath.Join(profilesDir, "foo-2.toml")); err != nil {
		t.Fatalf("expected foo-2.toml: %v", err)
	}
	if _, err := os.Stat(filepath.Join(profilesDir, "bar.toml")); err != nil {
		t.Fatalf("expected bar.toml: %v", err)
	}
}
