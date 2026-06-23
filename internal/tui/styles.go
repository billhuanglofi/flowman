package tui

import "charm.land/lipgloss/v2"

var (
	colorParchment  = lipgloss.Color("230")
	colorIvory      = lipgloss.Color("231")
	colorSand       = lipgloss.Color("223")
	colorCharcoal   = lipgloss.Color("236")
	colorMuted      = lipgloss.Color("244")
	colorTerracotta = lipgloss.Color("167")
	colorOlive      = lipgloss.Color("108")
	colorWarning    = lipgloss.Color("179")
	colorError      = lipgloss.Color("131")
	colorBorder     = lipgloss.Color("240")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorIvory).
			Background(colorTerracotta).
			Padding(0, 1)
	metaStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	bodyStyle  = lipgloss.NewStyle().Foreground(colorParchment)
	codeStyle  = lipgloss.NewStyle().Foreground(colorSand)
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)
	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorSand)
	selectedStyle   = lipgloss.NewStyle().Foreground(colorTerracotta).Bold(true)
	warningStyle    = lipgloss.NewStyle().Foreground(colorWarning)
	errorStyle      = lipgloss.NewStyle().Foreground(colorError).Bold(true)
	oliveStyle      = lipgloss.NewStyle().Foreground(colorOlive)
)
