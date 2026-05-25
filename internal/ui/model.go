package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Furthen64/lltop/internal/config"
	"github.com/Furthen64/lltop/internal/history"
	"github.com/Furthen64/lltop/internal/parser"
	"github.com/Furthen64/lltop/internal/runner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	cfg           *config.GlobalConfig
	profiles      []*config.Profile
	selectedIdx   int
	runner        *runner.Runner
	logViewport   viewport.Model
	stats         ServerStats
	width         int
	height        int
	showHelp      bool
	confirmMode   bool
	confirmPrompt string
	confirmAction func()
	statusMsg     string
	logAutoScroll bool

	logLines       []string
	issues         []history.Issue
	pendingQuit    bool
	afterStop      func() tea.Cmd
	currentCommand string
	openedProfile  string
}

type ServerStats struct {
	PromptTokensPerSec  float64
	EvalTokensPerSec    float64
	OffloadedLayers     int
	TotalLayers         int
	Progress            float64
	ChatFormat          string
	CtxSlotSize         int
	LastError           string
	LastGeneratedTokens int
	LastPromptTokens    int
	GPUTotalMiB         int
	GPUFreeMiB          int
	GPUModelMiB         int
	GPUContextMiB       int
	GPUComputeMiB       int
}

type logMsg string
type runnerDoneMsg struct{ info runner.ExitInfo }
type editorDoneMsg struct{ err error }

func NewModel(cfg *config.GlobalConfig, profiles []*config.Profile, statusMsg string) *Model {
	vp := viewport.New(0, 0)
	vp.SetContent("")
	m := &Model{
		cfg:           cfg,
		profiles:      profiles,
		runner:        runner.New(),
		logViewport:   vp,
		logAutoScroll: true,
		statusMsg:     statusMsg,
		logLines:      []string{},
		issues:        []history.Issue{},
	}
	return m
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		topH := max(8, int(float64(m.height)*0.60))
		rightW := max(40, m.width-max(24, int(float64(m.width)*0.30)))
		m.logViewport.Width = rightW - 6
		m.logViewport.Height = topH - 6
		m.refreshViewport()
		return m, nil
	case tea.KeyMsg:
		if m.confirmMode {
			switch msg.String() {
			case "y", "Y":
				m.confirmMode = false
				action := m.confirmAction
				m.confirmAction = nil
				if action != nil {
					action()
				}
				return m, m.followUpCmd()
			case "n", "N", "esc":
				m.confirmMode = false
				m.confirmAction = nil
				m.statusMsg = "Cancelled."
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "up":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case "down":
			if m.selectedIdx < len(m.profiles)-1 {
				m.selectedIdx++
			}
		case "enter":
			return m, m.launchSelectedCmd(false)
		case "s":
			if m.runner.IsRunning() {
				if err := m.runner.Stop(); err != nil {
					m.statusMsg = err.Error()
				} else {
					m.statusMsg = "Sent SIGINT."
				}
			}
		case "S":
			if m.runner.IsRunning() {
				if err := m.runner.Kill(); err != nil {
					m.statusMsg = err.Error()
				} else {
					m.statusMsg = "Sent SIGKILL."
				}
			}
		case "r":
			if m.runner.IsRunning() && m.cfg.ConfirmRestart {
				m.confirmPrompt = "Restart current server? [y/N]"
				m.confirmMode = true
				m.confirmAction = func() {
					m.afterStop = func() tea.Cmd { return m.launchSelectedCmd(true) }
					_ = m.runner.Stop()
					m.statusMsg = "Stopping current server for restart..."
				}
				return m, nil
			}
			return m, m.restartNowCmd()
		case "e":
			return m, m.editSelectedCmd()
		case "n":
			return m, m.newProfileCmd()
		case "d":
			if err := m.duplicateSelected(); err != nil {
				m.statusMsg = err.Error()
			} else {
				m.statusMsg = "Profile duplicated."
			}
			return m, nil
		case "v":
			if profile := m.selectedProfile(); profile != nil {
				spec, err := runner.BuildCommand(m.cfg, profile)
				if err != nil {
					m.statusMsg = err.Error()
				} else {
					m.statusMsg = spec.Display
				}
			}
		case "l":
			m.logAutoScroll = !m.logAutoScroll
			if m.logAutoScroll {
				m.logViewport.GotoBottom()
			}
			m.statusMsg = fmt.Sprintf("Log auto-scroll = %t", m.logAutoScroll)
		case "h", "?":
			m.showHelp = !m.showHelp
		case "q":
			if m.runner.IsRunning() {
				m.confirmMode = true
				m.confirmPrompt = "Server is running and cannot be detached here. Stop it and quit? [y/N]"
				m.confirmAction = func() {
					m.pendingQuit = true
					_ = m.runner.Stop()
					m.statusMsg = "Stopping server before quit..."
				}
				return m, nil
			}
			return m, tea.Quit
		}
	case logMsg:
		line := string(msg)
		m.logLines = append(m.logLines, line)
		if len(m.logLines) > 500 {
			m.logLines = append([]string(nil), m.logLines[len(m.logLines)-500:]...)
		}
		m.consumeParsedLine(line)
		m.refreshViewport()
		return m, waitForLog(m.runner)
	case runnerDoneMsg:
		end := time.Now()
		started := m.runner.StartTime
		reason := "exit"
		if msg.info.Err != nil {
			reason = msg.info.Err.Error()
		}
		if profile := m.runner.Profile; profile != nil {
			snapshot := history.StatsSnapshot{
				PromptTokensPerSec: m.stats.PromptTokensPerSec,
				EvalTokensPerSec:   m.stats.EvalTokensPerSec,
				GeneratedTokens:    m.stats.LastGeneratedTokens,
				PromptTokens:       m.stats.LastPromptTokens,
				OffloadedLayers:    m.stats.OffloadedLayers,
				TotalLayers:        m.stats.TotalLayers,
				GPUTotalMiB:        m.stats.GPUTotalMiB,
				GPUFreeMiB:         m.stats.GPUFreeMiB,
				GPUModelMiB:        m.stats.GPUModelMiB,
				GPUContextMiB:      m.stats.GPUContextMiB,
				GPUComputeMiB:      m.stats.GPUComputeMiB,
				Issues:             append([]history.Issue(nil), m.issues...),
			}
			record := history.NewRunRecord(m.cfg, profile, m.currentCommand, started, end, msg.info.ExitCode, reason, snapshot)
			if _, err := history.SaveRunRecord(m.cfg.RunsDir, record); err != nil {
				m.statusMsg = "failed to store run record: " + err.Error()
			} else {
				m.statusMsg = fmt.Sprintf("Run ended with exit code %d.", msg.info.ExitCode)
			}
		}
		if m.pendingQuit {
			return m, tea.Quit
		}
		if m.afterStop != nil {
			next := m.afterStop
			m.afterStop = nil
			return m, next()
		}
		return m, nil
	case editorDoneMsg:
		if msg.err != nil {
			m.statusMsg = "editor failed: " + msg.err.Error()
			return m, nil
		}
		if err := m.reloadProfiles(); err != nil {
			m.statusMsg = err.Error()
		} else if m.openedProfile != "" {
			m.statusMsg = "Saved profile: " + m.openedProfile
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) selectedProfile() *config.Profile {
	if len(m.profiles) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.profiles) {
		return nil
	}
	return m.profiles[m.selectedIdx]
}

func (m *Model) launchSelectedCmd(skipFailureCheck bool) tea.Cmd {
	profile := m.selectedProfile()
	if profile == nil {
		m.statusMsg = "No profile selected."
		return nil
	}
	if m.runner.IsRunning() {
		m.statusMsg = "A server is already running."
		return nil
	}
	if !skipFailureCheck && m.cfg.ConfirmRecentFailure {
		recent, err := history.FindRecentFailure(m.cfg.RunsDir, history.BuildScenarioKey(profile), m.cfg.RecentFailureWindowSecs, m.cfg.StartupFailureSecs)
		if err != nil {
			m.statusMsg = err.Error()
			return nil
		}
		if recent != nil {
			ago := int(time.Since(recent.StartedAt).Seconds())
			issue := "startup failure"
			if len(recent.Issues) > 0 {
				issue = recent.Issues[0].Kind
			}
			m.confirmMode = true
			m.confirmPrompt = fmt.Sprintf("This same scenario failed %ds ago (exit code %d).\nProfile: %s  Issue: %s\nRun it again? [y/N]", ago, recent.ExitCode, recent.ProfileName, issue)
			m.confirmAction = func() {
				m.afterStop = nil
				m.statusMsg = "Launching profile..."
				if err := m.startSelected(); err != nil {
					m.statusMsg = err.Error()
				}
			}
			return nil
		}
	}
	if err := m.startSelected(); err != nil {
		m.statusMsg = err.Error()
		return nil
	}
	return tea.Batch(waitForLog(m.runner), waitForDone(m.runner))
}

func (m *Model) restartNowCmd() tea.Cmd {
	if m.runner.IsRunning() {
		m.afterStop = func() tea.Cmd { return m.launchSelectedCmd(true) }
		if err := m.runner.Stop(); err != nil {
			m.statusMsg = err.Error()
			return nil
		}
		return waitForDone(m.runner)
	}
	return m.launchSelectedCmd(true)
}

func (m *Model) startSelected() error {
	profile := m.selectedProfile()
	if profile == nil {
		return fmt.Errorf("no profile selected")
	}
	m.stats = ServerStats{}
	m.issues = nil
	m.logLines = nil
	m.refreshViewport()
	spec, err := runner.BuildCommand(m.cfg, profile)
	if err != nil {
		return err
	}
	m.currentCommand = spec.Display
	if err := m.runner.Launch(m.cfg, profile); err != nil {
		return err
	}
	m.statusMsg = "Launched " + profile.Name
	return nil
}

func (m *Model) consumeParsedLine(line string) {
	parsed := parser.ParseLine(line)
	if parsed.PromptTokensPerSec > 0 {
		m.stats.PromptTokensPerSec = parsed.PromptTokensPerSec
		m.stats.LastPromptTokens = parsed.PromptTokens
	}
	if parsed.EvalTokensPerSec > 0 {
		m.stats.EvalTokensPerSec = parsed.EvalTokensPerSec
		m.stats.LastGeneratedTokens = parsed.EvalTokens
	}
	if parsed.OffloadedLayers > 0 || parsed.TotalLayers > 0 {
		m.stats.OffloadedLayers = parsed.OffloadedLayers
		m.stats.TotalLayers = parsed.TotalLayers
	}
	if parsed.Progress > 0 {
		m.stats.Progress = parsed.Progress
	}
	if parsed.ChatFormat != "" {
		m.stats.ChatFormat = parsed.ChatFormat
	}
	if parsed.CtxSlotSize > 0 {
		m.stats.CtxSlotSize = parsed.CtxSlotSize
	}
	if parsed.GPUTotalMiB > 0 {
		m.stats.GPUTotalMiB = parsed.GPUTotalMiB
		m.stats.GPUFreeMiB = parsed.GPUFreeMiB
		m.stats.GPUModelMiB = parsed.GPUModelMiB
		m.stats.GPUContextMiB = parsed.GPUContextMiB
		m.stats.GPUComputeMiB = parsed.GPUComputeMiB
	}
	if parsed.IsError {
		m.stats.LastError = parsed.ErrorMessage
		seenAt := 0.0
		if !m.runner.StartTime.IsZero() {
			seenAt = time.Since(m.runner.StartTime).Seconds()
		}
		m.issues = append(m.issues, history.Issue{
			Severity:   "error",
			Kind:       parsed.ErrorKind,
			Message:    parsed.ErrorMessage,
			SeenAtSecs: seenAt,
		})
	}
}

func (m *Model) refreshViewport() {
	rendered := make([]string, 0, len(m.logLines))
	for _, line := range m.logLines {
		rendered = append(rendered, colorizeLogLine(line))
	}
	m.logViewport.SetContent(strings.Join(rendered, "\n"))
	if m.logAutoScroll {
		m.logViewport.GotoBottom()
	}
}

func (m *Model) reloadProfiles() error {
	profiles, err := config.LoadProfiles(m.cfg.ProfilesDir)
	if err != nil {
		return err
	}
	m.profiles = profiles
	if len(m.profiles) == 0 {
		m.selectedIdx = 0
		return nil
	}
	for i, profile := range m.profiles {
		if profile.Name == m.openedProfile {
			m.selectedIdx = i
			return nil
		}
	}
	if m.selectedIdx >= len(m.profiles) {
		m.selectedIdx = len(m.profiles) - 1
	}
	return nil
}

func (m *Model) editSelectedCmd() tea.Cmd {
	profile := m.selectedProfile()
	if profile == nil {
		m.statusMsg = "No profile selected."
		return nil
	}
	path := filepath.Join(m.cfg.ProfilesDir, config.SlugifyName(profile.Name)+".toml")
	m.openedProfile = profile.Name
	return openEditor(m.cfg.Editor, path)
}

func (m *Model) newProfileCmd() tea.Cmd {
	base := "new-profile"
	file := base + ".toml"
	path := filepath.Join(m.cfg.ProfilesDir, file)
	for i := 2; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			break
		}
		file = fmt.Sprintf("%s-%d.toml", base, i)
		path = filepath.Join(m.cfg.ProfilesDir, file)
	}
	name := strings.TrimSuffix(file, ".toml")
	profile := config.DefaultProfile(m.cfg, name)
	profile.Description = "New profile"
	if err := config.SaveProfile(path, profile); err != nil {
		m.statusMsg = err.Error()
		return nil
	}
	m.openedProfile = profile.Name
	return openEditor(m.cfg.Editor, path)
}

func (m *Model) duplicateSelected() error {
	profile := m.selectedProfile()
	if profile == nil {
		return fmt.Errorf("no profile selected")
	}
	dup := *profile
	base := profile.Name + "-copy"
	dup.Name = base
	path := filepath.Join(m.cfg.ProfilesDir, config.SlugifyName(dup.Name)+".toml")
	for i := 2; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			break
		}
		dup.Name = fmt.Sprintf("%s-%d", base, i)
		path = filepath.Join(m.cfg.ProfilesDir, config.SlugifyName(dup.Name)+".toml")
	}
	if err := config.SaveProfile(path, &dup); err != nil {
		return err
	}
	m.openedProfile = dup.Name
	return m.reloadProfiles()
}

func waitForLog(r *runner.Runner) tea.Cmd {
	if r == nil {
		return nil
	}
	return func() tea.Msg {
		line, ok := <-r.LogCh
		if !ok {
			return nil
		}
		return logMsg(line)
	}
}

func waitForDone(r *runner.Runner) tea.Cmd {
	if r == nil {
		return nil
	}
	return func() tea.Msg {
		info := <-r.DoneCh
		return runnerDoneMsg{info: info}
	}
}

func openEditor(editor, path string) tea.Cmd {
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		parts = []string{"nano"}
	}
	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorDoneMsg{err: err}
	})
}

func (m *Model) followUpCmd() tea.Cmd {
	if m.runner == nil {
		return nil
	}
	if m.pendingQuit && m.runner.IsRunning() {
		return tea.Batch(waitForLog(m.runner), waitForDone(m.runner))
	}
	if m.runner.IsRunning() {
		return tea.Batch(waitForLog(m.runner), waitForDone(m.runner))
	}
	return nil
}
