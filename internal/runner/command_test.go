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

func TestBuildCommandIncludesFlashAttnAfterCacheFlags(t *testing.T) {
	cfg := config.DefaultGlobalConfig()
	cfg.LlamaServer = "/usr/bin/llama-server"
	profile := config.DefaultProfile(cfg, "flash-attn")
	profile.Model = "/models/qwen.gguf"
	profile.CacheK = "q4_0"
	profile.CacheV = "q8_0"
	profile.FlashAttn = "on"

	spec, err := BuildCommand(cfg, profile)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	got := strings.Join(spec.Args, " ")
	want := "--cache-type-k q4_0 --cache-type-v q8_0 --flash-attn on"
	if !strings.Contains(got, want) {
		t.Fatalf("expected args to contain %q, got %q", want, got)
	}
}

func TestBuildCommandDefaultsFlashAttnToAuto(t *testing.T) {
	cfg := config.DefaultGlobalConfig()
	cfg.LlamaServer = "/usr/bin/llama-server"
	profile := config.DefaultProfile(cfg, "default-flash-attn")
	profile.Model = "/models/qwen.gguf"
	profile.FlashAttn = ""

	spec, err := BuildCommand(cfg, profile)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	got := strings.Join(spec.Args, " ")
	if !strings.Contains(got, "--flash-attn auto") {
		t.Fatalf("expected args to contain default flash attention setting, got %q", got)
	}
}

func TestBuildCommandPrefersProfileFlashAttnOverExtraArgs(t *testing.T) {
	cfg := config.DefaultGlobalConfig()
	cfg.LlamaServer = "/usr/bin/llama-server"
	profile := config.DefaultProfile(cfg, "dedupe-flash-attn")
	profile.Model = "/models/qwen.gguf"
	profile.FlashAttn = "on"
	profile.ExtraArgs = []string{"--flash-attn", "off", "-fa=auto", "--prio", "2"}

	spec, err := BuildCommand(cfg, profile)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	got := strings.Join(spec.Args, " ")
	if strings.Count(got, "--flash-attn") != 1 {
		t.Fatalf("expected exactly one flash attention flag, got %q", got)
	}
	if !strings.Contains(got, "--flash-attn on") {
		t.Fatalf("expected profile flash attention value to win, got %q", got)
	}
	if !strings.Contains(got, "--prio 2") {
		t.Fatalf("expected unrelated extra args to remain, got %q", got)
	}
}
