package ui

import (
	"strings"
	"testing"

	"github.com/Furthen64/lltop/internal/config"
)

func testGlobalConfig() *config.GlobalConfig {
	cfg := config.DefaultGlobalConfig()
	cfg.LlamaServer = "/usr/bin/llama-server"
	return cfg
}

func testProfile() *config.Profile {
	p := config.DefaultProfile(testGlobalConfig(), "qwen")
	p.Model = "/models/qwen.gguf"
	return p
}

func TestRenderKeys_TogglesExpandedHelp(t *testing.T) {
	m := &Model{}

	collapsed := m.renderKeys()
	if !strings.Contains(collapsed, "h/? more help") {
		t.Fatalf("expected collapsed keys to advertise more help, got %q", collapsed)
	}
	if strings.Contains(collapsed, "navigation:") {
		t.Fatalf("expected collapsed keys to stay compact, got %q", collapsed)
	}

	m.showHelp = true
	expanded := m.renderKeys()
	for _, want := range []string{
		"navigation:",
		"server:",
		"profile:",
		"help: h/? hide this help",
	} {
		if !strings.Contains(expanded, want) {
			t.Fatalf("expected expanded help to contain %q, got %q", want, expanded)
		}
	}
}

func TestLayoutHeights_ExpandsHelpArea(t *testing.T) {
	topH, statusH, keysH := layoutHeights(40, false)
	helpTopH, helpStatusH, helpKeysH := layoutHeights(40, true)

	if topH+statusH+keysH != 40 {
		t.Fatalf("expected default layout to fill height, got %d", topH+statusH+keysH)
	}
	if helpTopH+helpStatusH+helpKeysH != 40 {
		t.Fatalf("expected help layout to fill height, got %d", helpTopH+helpStatusH+helpKeysH)
	}
	if helpKeysH <= keysH {
		t.Fatalf("expected help layout to grow keys area, got default=%d help=%d", keysH, helpKeysH)
	}
}

func TestRenderStatusShowsLaunchCommand(t *testing.T) {
	m := NewModel(testGlobalConfig(), nil, "")
	profile := testProfile()
	m.profiles = []*config.Profile{profile}

	status := m.renderStatus()
	if !strings.Contains(status, "launch:") {
		t.Fatalf("expected current server status to include launch text, got %q", status)
	}
	if !strings.Contains(status, "--chat-template chatml") {
		t.Fatalf("expected launch text to include chat template, got %q", status)
	}
}

func TestHandleLogScrollKeyRequiresAutoscrollOff(t *testing.T) {
	m := NewModel(testGlobalConfig(), nil, "")
	m.logViewport.Width = 80
	m.logViewport.Height = 4
	m.logLines = []string{"one", "two", "three", "four", "five", "six"}
	m.refreshViewport()

	if m.handleLogScrollKey("pgup") {
		t.Fatal("expected scroll key to be ignored while autoscroll is enabled")
	}

	m.logAutoScroll = false
	m.logViewport.GotoBottom()
	before := m.logViewport.YOffset
	if !m.handleLogScrollKey("pgup") {
		t.Fatal("expected page up to scroll when autoscroll is disabled")
	}
	if m.logViewport.YOffset >= before {
		t.Fatalf("expected viewport to move up from %d, got %d", before, m.logViewport.YOffset)
	}
}
