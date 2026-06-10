package ui

import (
	"strings"
	"testing"
)

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
