package tui

import "charm.land/lipgloss/v2"

var (
	// High contrast colors that work in both light and dark terminals
	colorDeepBlue   = lipgloss.Color("21")   // Deep blue for titles
	colorBrightCyan = lipgloss.Color("51")   // Bright cyan for accents
	colorDarkGray   = lipgloss.Color("240")  // Dark gray for text (visible on light)
	colorBlack      = lipgloss.Color("16")   // Black for primary text
	colorMedGray    = lipgloss.Color("244")  // Medium gray for meta
	colorOrange     = lipgloss.Color("208")  // Orange for selection
	colorGreen      = lipgloss.Color("34")   // Green for success
	colorYellow     = lipgloss.Color("220")  // Yellow for warnings
	colorRed        = lipgloss.Color("196")  // Red for errors
	colorBorder     = lipgloss.Color("240")  // Border color
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")). // White text
			Background(colorDeepBlue).
			Padding(0, 1)
	metaStyle  = lipgloss.NewStyle().Foreground(colorMedGray)
	bodyStyle  = lipgloss.NewStyle().Foreground(colorBlack)
	codeStyle  = lipgloss.NewStyle().Foreground(colorDeepBlue).Bold(true)
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)
	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorDeepBlue)
	selectedStyle   = lipgloss.NewStyle().Foreground(colorOrange).Bold(true)
	warningStyle    = lipgloss.NewStyle().Foreground(colorYellow)
	errorStyle      = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	oliveStyle      = lipgloss.NewStyle().Foreground(colorGreen)
)
