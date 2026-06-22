package ui

import "github.com/charmbracelet/lipgloss"

const (
	colorPanelBorder   = lipgloss.Color("63")
	colorTitleText     = lipgloss.Color("86")
	colorMutedText     = lipgloss.Color("82")
	colorSuccessText   = lipgloss.Color("42")
	colorWarningText   = lipgloss.Color("214")
	colorErrorText     = lipgloss.Color("196")
	colorInfoText      = lipgloss.Color("81")
	colorHighlightText = lipgloss.Color("75")
	colorSelectedText  = lipgloss.Color("229")
	colorSelectedBg    = lipgloss.Color("63")
)

var (
	panelStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorPanelBorder).Padding(0, 1)
	titleStyle    = lipgloss.NewStyle().Foreground(colorTitleText).Bold(true)
	dimStyle      = lipgloss.NewStyle().Foreground(colorMutedText)
	okStyle       = lipgloss.NewStyle().Foreground(colorSuccessText)
	warnStyle     = lipgloss.NewStyle().Foreground(colorWarningText)
	errStyle      = lipgloss.NewStyle().Foreground(colorErrorText)
	infoStyle     = lipgloss.NewStyle().Foreground(colorInfoText)
	blueStyle     = lipgloss.NewStyle().Foreground(colorHighlightText)
	selectedStyle = lipgloss.NewStyle().Foreground(colorSelectedText).Background(colorSelectedBg).Bold(true)
)
