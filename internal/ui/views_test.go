package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Furthen64/lltop/internal/config"
	"github.com/Furthen64/lltop/internal/history"
	tea "github.com/charmbracelet/bubbletea"
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
		"annotate latest run",
		"help:",
		"h/? hide this help",
	} {
		if !strings.Contains(expanded, want) {
			t.Fatalf("expected expanded help to contain %q, got %q", want, expanded)
		}
	}
}

func TestRenderKeys_NotesViewShowsNotesHelp(t *testing.T) {
	m := &Model{viewMode: notesView}

	collapsed := m.renderKeys()
	for _, want := range []string{"N/Esc back to main", "a annotate run"} {
		if !strings.Contains(collapsed, want) {
			t.Fatalf("expected notes help to contain %q, got %q", want, collapsed)
		}
	}

	m.showHelp = true
	expanded := m.renderKeys()
	for _, want := range []string{"selected run", "return to main view", "PgUp/PgDown"} {
		if !strings.Contains(expanded, want) {
			t.Fatalf("expected expanded notes help to contain %q, got %q", want, expanded)
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

func TestRenderStatusShowsHistorySummary(t *testing.T) {
	m := NewModel(testGlobalConfig(), nil, "")
	profile := testProfile()
	m.profiles = []*config.Profile{profile}
	m.historySummary = history.ProfileSummary{
		ProfileName: profile.Name,
		RunCount:    3,
		GenerationSpeed: history.MetricSummary{
			Count:   3,
			Latest:  4,
			Average: 5,
			Median:  4.5,
			Min:     3,
			Max:     8,
			Series:  []float64{3, 8, 4},
		},
		PromptSpeed: history.MetricSummary{
			Count:   2,
			Latest:  100,
			Average: 95,
			Median:  95,
			Min:     90,
			Max:     100,
			Series:  []float64{90, 100},
		},
	}

	status := m.renderStatus()
	for _, want := range []string{"history: 3 run(s)", "gen tok/s latest 4.00", "ingest tok/s latest 100.00"} {
		if !strings.Contains(status, want) {
			t.Fatalf("expected status to contain %q, got %q", want, status)
		}
	}
}

func TestRenderProfilesShowsModelFileSize(t *testing.T) {
	modelPath := filepath.Join(t.TempDir(), "qwen.gguf")
	file, err := os.Create(modelPath)
	if err != nil {
		t.Fatalf("failed to create model file: %v", err)
	}
	if err := file.Truncate(1536); err != nil {
		t.Fatalf("failed to size model file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("failed to close model file: %v", err)
	}

	m := NewModel(testGlobalConfig(), nil, "")
	profile := testProfile()
	profile.Model = modelPath
	m.profiles = []*config.Profile{profile}

	profiles := m.renderProfiles()
	if !strings.Contains(profiles, "1.5 KiB") {
		t.Fatalf("expected profile list to include model file size, got %q", profiles)
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := map[int64]string{
		512:     "512 B",
		1536:    "1.5 KiB",
		5 << 20: "5.0 MiB",
		7 << 30: "7.0 GiB",
		3 << 40: "3.0 TiB",
		5 << 50: "5.0 PiB",
	}

	for size, want := range tests {
		if got := formatFileSize(size); got != want {
			t.Fatalf("formatFileSize(%d) = %q, want %q", size, got, want)
		}
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

func TestEnterNotesViewLoadsProfileRuns(t *testing.T) {
	cfg := testGlobalConfig()
	cfg.RunsDir = t.TempDir()
	profile := testProfile()
	m := NewModel(cfg, []*config.Profile{profile}, "")
	m.width = 120
	m.height = 40
	m.updateLayout()

	record1 := history.NewRunRecord(cfg, profile, "cmd1", time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC), time.Date(2026, 6, 23, 10, 0, 5, 0, time.UTC), 0, "exit", history.StatsSnapshot{})
	record1.Notes = "first note"
	if _, err := history.SaveRunRecord(cfg.RunsDir, record1); err != nil {
		t.Fatalf("failed to save run record 1: %v", err)
	}

	record2 := history.NewRunRecord(cfg, profile, "cmd2", time.Date(2026, 6, 23, 11, 0, 0, 0, time.UTC), time.Date(2026, 6, 23, 11, 0, 5, 0, time.UTC), 0, "exit", history.StatsSnapshot{})
	record2.Notes = "second note"
	if _, err := history.SaveRunRecord(cfg.RunsDir, record2); err != nil {
		t.Fatalf("failed to save run record 2: %v", err)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})

	if m.viewMode != notesView {
		t.Fatalf("expected notes view, got %q", m.viewMode)
	}
	if len(m.noteEntries) != 2 {
		t.Fatalf("expected 2 note entries, got %d", len(m.noteEntries))
	}
	if m.noteEntries[0].Record == nil || m.noteEntries[0].Record.Notes != "second note" {
		t.Fatalf("expected newest record first, got %#v", m.noteEntries[0].Record)
	}
	if !strings.Contains(m.noteViewport.View(), "second note") {
		t.Fatalf("expected note viewport to show selected note, got %q", m.noteViewport.View())
	}
}

func TestAnnotationTemplateIncludesRunParameters(t *testing.T) {
	record := &history.RunRecord{
		RunID:                  "20260623_123456_qwen",
		StartedAt:              time.Date(2026, 6, 23, 12, 34, 56, 0, time.UTC),
		DurationSeconds:        12.34,
		ExitCode:               0,
		ExitReason:             "exit",
		LlamaServer:            "/usr/bin/llama-server",
		Model:                  "/models/qwen.gguf",
		Host:                   "0.0.0.0",
		Port:                   8080,
		Alias:                  "qwen",
		Ctx:                    65536,
		NGL:                    99,
		CacheK:                 "q4_0",
		CacheV:                 "q4_0",
		Temp:                   0.1,
		TopP:                   0.95,
		TopK:                   40,
		MinP:                   0.05,
		Batch:                  512,
		UBatch:                 256,
		Parallel:               1,
		Threads:                16,
		Metrics:                true,
		Jinja:                  true,
		NoMmap:                 true,
		ChatTemplate:           "chatml",
		ExtraArgs:              []string{"--flash-attn", "--prio", "2"},
		GeneratedCommand:       "/usr/bin/llama-server -m /models/qwen.gguf --temp 0.1",
		LastEvalTokensPerSec:   42.5,
		LastPromptTokensPerSec: 1234.5,
	}

	got := annotationTemplate("qwen", record)
	for _, want := range []string{
		"profile: qwen",
		"run_parameters:",
		"ctx: 65536",
		"temp: 0.1",
		"threads: 16",
		"extra_args: --flash-attn --prio 2",
		"command:",
		"/usr/bin/llama-server -m /models/qwen.gguf --temp 0.1",
		"notes:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected annotation template to contain %q, got %q", want, got)
		}
	}
}
