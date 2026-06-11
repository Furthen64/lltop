package runner

import (
	"strings"
	"testing"

	"github.com/Furthen64/lltop/internal/config"
)

func TestBuildCommandIncludesChatTemplateAfterJinja(t *testing.T) {
	cfg := config.DefaultGlobalConfig()
	cfg.LlamaServer = "/usr/bin/llama-server"
	profile := config.DefaultProfile(cfg, "chatml")
	profile.Model = "/models/qwen.gguf"
	profile.ChatTemplate = "chatml"

	spec, err := BuildCommand(cfg, profile)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	got := strings.Join(spec.Args, " ")
	want := "--jinja --no-mmap --chat-template chatml"
	if !strings.Contains(got, want) {
		t.Fatalf("expected args to contain %q in order, got %q", want, got)
	}
	if !strings.Contains(spec.Display, "--chat-template chatml") {
		t.Fatalf("expected display command to include chat template, got %q", spec.Display)
	}
}

func TestBuildCommandIncludesNoMmapFlag(t *testing.T) {
	cfg := config.DefaultGlobalConfig()
	cfg.LlamaServer = "/usr/bin/llama-server"
	profile := config.DefaultProfile(cfg, "no-mmap")
	profile.Model = "/models/qwen.gguf"

	spec, err := BuildCommand(cfg, profile)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	got := strings.Join(spec.Args, " ")
	if !strings.Contains(got, "--no-mmap") {
		t.Fatalf("expected args to contain --no-mmap, got %q", got)
	}
	if !strings.Contains(spec.Display, "--no-mmap") {
		t.Fatalf("expected display command to include --no-mmap, got %q", spec.Display)
	}
}
