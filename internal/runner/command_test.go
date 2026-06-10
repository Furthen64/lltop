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
	want := "--jinja --chat-template chatml"
	if !strings.Contains(got, want) {
		t.Fatalf("expected args to contain %q in order, got %q", want, got)
	}
	if !strings.Contains(spec.Display, "--chat-template chatml") {
		t.Fatalf("expected display command to include chat template, got %q", spec.Display)
	}
}
