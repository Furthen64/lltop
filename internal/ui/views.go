package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Furthen64/lltop/internal/runner"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	width := m.width
	if width <= 0 {
		width = 120
	}
	height := m.height
	if height <= 0 {
		height = 40
	}

	topH, statusH, keysH := layoutHeights(height, m.showHelp)
	leftW := max(24, int(float64(width)*0.30))
	rightW := max(40, width-leftW)

	profilesPanel := panelStyle.Width(leftW - 2).Height(topH - 2).Render(m.renderProfiles())
	logsPanel := panelStyle.Width(rightW - 2).Height(topH - 2).Render(m.renderLogs())
	top := lipgloss.JoinHorizontal(lipgloss.Top, profilesPanel, logsPanel)

	status := panelStyle.Width(width - 2).Height(statusH - 2).Render(m.renderStatus())
	bottomContent := m.renderKeys()
	if m.confirmMode {
		bottomContent = titleStyle.Render("confirm") + "\n\n" + m.confirmPrompt
	}
	bottom := panelStyle.Width(width - 2).Height(keysH - 2).Render(bottomContent)

	return lipgloss.JoinVertical(lipgloss.Left, top, status, bottom)
}

func (m *Model) renderProfiles() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("profiles"))
	b.WriteString("\n\n")
	if len(m.profiles) == 0 {
		b.WriteString(dimStyle.Render("No profiles found."))
		return b.String()
	}
	for i, profile := range m.profiles {
		line := profile.Name
		if profile.Description != "" {
			line += dimStyle.Render(" — " + profile.Description)
		}
		if i == m.selectedIdx {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		if i < len(m.profiles)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m *Model) renderLogs() string {
	header := titleStyle.Render("live log") + " " + dimStyle.Render(fmt.Sprintf("autoscroll=%t", m.logAutoScroll))
	if m.logViewport.Width <= 0 {
		return header + "\n\n"
	}
	return header + "\n\n" + m.logViewport.View()
}

func (m *Model) renderStatus() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("current server"))
	b.WriteString("\n\n")

	profile := m.selectedProfile()
	if m.runner != nil && m.runner.Profile != nil && m.runner.IsRunning() {
		profile = m.runner.Profile
	}
	if profile != nil {
		b.WriteString(fmt.Sprintf("profile: %s\n", profile.Name))
		b.WriteString(fmt.Sprintf("model: %s\n", profile.Model))
		b.WriteString(fmt.Sprintf("bind: %s:%d\n", profile.Host, profile.Port))
	}
	status := "stopped"
	pid := 0
	externalCmd := ""
	runnerActive := false
	if m.runner != nil {
		status = m.runner.Status
		pid = m.runner.PID
		runnerActive = m.runner.IsRunning()
	}
	if !runnerActive {
		if m.externalProc.PID > 0 {
			status = "externally running"
			pid = m.externalProc.PID
			externalCmd = m.externalProc.Command
		} else if proc, err := detectExternalLlamaServer(os.Getpid()); err == nil && proc.PID > 0 {
			status = "externally running"
			pid = proc.PID
			externalCmd = proc.Command
		}
	}
	b.WriteString(fmt.Sprintf("status: %s", status))
	if pid > 0 {
		b.WriteString(fmt.Sprintf(" (pid %d)", pid))
	}
	if m.runner != nil && !m.runner.StartTime.IsZero() {
		b.WriteString(fmt.Sprintf("  uptime: %s", time.Since(m.runner.StartTime).Truncate(time.Second)))
	}
	b.WriteByte('\n')
	launchText := m.renderedLaunchText(externalCmd)
	if launchText != "" {
		b.WriteString(fmt.Sprintf("launch: %s\n", launchText))
	}
	b.WriteString(fmt.Sprintf("prompt tok/s: %.2f  eval tok/s: %.2f  offload: %d/%d  progress: %.2f\n",
		m.stats.PromptTokensPerSec, m.stats.EvalTokensPerSec, m.stats.OffloadedLayers, m.stats.TotalLayers, m.stats.Progress))
	if m.stats.ChatFormat != "" {
		b.WriteString(fmt.Sprintf("chat format: %s  ctx slot: %d\n", m.stats.ChatFormat, m.stats.CtxSlotSize))
	}
	if m.stats.LastError != "" {
		b.WriteString(errStyle.Render("last error: " + m.stats.LastError))
		b.WriteByte('\n')
	}
	if m.statusMsg != "" {
		b.WriteString(infoStyle.Render(m.statusMsg))
	}
	return b.String()
}

func (m *Model) renderedLaunchText(externalCmd string) string {
	if m.runner != nil && m.runner.IsRunning() && m.currentCommand != "" {
		return m.currentCommand
	}
	if externalCmd != "" {
		return externalCmd
	}
	profile := m.selectedProfile()
	if profile == nil {
		return ""
	}
	spec, err := runner.BuildCommand(m.cfg, profile)
	if err != nil {
		return ""
	}
	return spec.Display
}

func (m *Model) renderKeys() string {
	title := titleStyle.Render("keys")
	if !m.showHelp {
		return title + "\n\nUp/Down move  Enter launch  s stop  S kill  r restart  e edit  n new  d duplicate  v command  c copy command  l autoscroll  h/? more help  q quit"
	}
	return strings.Join([]string{
		title,
		"",
		"navigation: Up/Down select profile  Enter launch  q quit",
		"server: s stop gracefully  S force kill  r restart  l toggle log autoscroll",
		"log: when autoscroll=false, PgUp/PgDown or Ctrl+U/Ctrl+D scroll; Home/End jump",
		"profile: e edit selected  n new profile  d duplicate selected  v show command  c copy command",
		"help: h/? hide this help",
	}, "\n")
}

func layoutHeights(height int, showHelp bool) (topH, statusH, keysH int) {
	topH = max(8, int(float64(height)*0.60))
	statusH = max(6, int(float64(height)*0.25))
	keysH = max(4, height-topH-statusH)
	if !showHelp || keysH >= 7 {
		return topH, statusH, keysH
	}

	deficit := 7 - keysH
	reduceStatus := min(deficit, statusH-6)
	statusH -= reduceStatus
	deficit -= reduceStatus

	reduceTop := min(deficit, topH-8)
	topH -= reduceTop

	return topH, statusH, height - topH - statusH
}

func colorizeLogLine(line string) string {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "error") || strings.Contains(lower, "failed"):
		return errStyle.Render(line)
	case strings.Contains(lower, "warning") || strings.Contains(lower, "warn"):
		return warnStyle.Render(line)
	case strings.Contains(lower, "tokens per second"):
		return okStyle.Render(line)
	case strings.Contains(lower, "offloaded"):
		return blueStyle.Render(line)
	case strings.Contains(lower, "progress"):
		return dimStyle.Render(line)
	default:
		return line
	}
}
