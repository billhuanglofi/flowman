package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func renderResponseWithTabs(preview Preview, state ViewState, width int) string {
	// Render tab bar
	tabs := []string{
		renderTab("Body", state.ActiveTab == string(tabBody)),
		renderTab("Headers", state.ActiveTab == string(tabHeaders)),
		renderTab("Trace", state.ActiveTab == string(tabTrace)),
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Render content based on active tab
	var content string
	switch responseTab(state.ActiveTab) {
	case tabBody:
		content = renderBodyTab(preview, width)
	case tabHeaders:
		content = renderHeadersTab(preview, width)
	case tabTrace:
		content = renderTraceTab(preview, width)
	default:
		content = renderBodyTab(preview, width)
	}

	lines := []string{
		panelTitleStyle.Render("Response"),
		tabBar,
		"",
		content,
	}

	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func renderTab(label string, active bool) string {
	style := tabInactiveStyle
	if active {
		style = tabActiveStyle
	}
	return style.Render(label)
}

func renderBodyTab(preview Preview, width int) string {
	lines := []string{
		bodyStyle.Render("State: " + safeText(preview.Response.State)),
		bodyStyle.Render(safeText(preview.Response.Status)),
		codeStyle.Render("Transaction: " + safeText(preview.Response.TransactionID)),
		metaStyle.Render("Duration: " + safeText(preview.Response.Duration)),
	}

	// If we have a response body, format it
	if preview.FullResponse.StatusCode > 0 && len(preview.FullResponse.DisplayBody) > 0 {
		lines = append(lines, "")
		lines = append(lines, metaStyle.Render("Body:"))

		// Try to format as JSON
		formatted := formatJSON(preview.FullResponse.DisplayBody, width-4)
		if formatted != "" {
			lines = append(lines, formatted)
		} else {
			// Not JSON, show raw with truncation
			body := string(preview.FullResponse.DisplayBody)
			if len(body) > 500 {
				body = body[:500] + "..."
			}
			lines = append(lines, bodyStyle.Render(body))
		}

		// Show size
		size := formatBytes(len(preview.FullResponse.DisplayBody))
		lines = append(lines, "", metaStyle.Render(fmt.Sprintf("Size: %s", size)))
	}

	return strings.Join(lines, "\n")
}

func renderHeadersTab(preview Preview, width int) string {
	if preview.FullResponse.StatusCode == 0 {
		return warningStyle.Render("No response headers yet")
	}

	lines := []string{
		bodyStyle.Render(safeText(preview.Response.Status)),
		"",
		metaStyle.Render("Response Headers:"),
	}

	// Display headers
	for name, values := range preview.FullResponse.DisplayHeaders {
		for _, value := range values {
			headerLine := fmt.Sprintf("%s: %s", name, value)
			if len(headerLine) > width-4 {
				headerLine = headerLine[:width-7] + "..."
			}
			lines = append(lines, bodyStyle.Render(headerLine))
		}
	}

	if len(preview.FullResponse.DisplayHeaders) == 0 {
		lines = append(lines, warningStyle.Render("No headers"))
	}

	return strings.Join(lines, "\n")
}

func renderTraceTab(preview Preview, width int) string {
	if preview.Journey.TransactionID == "" {
		return warningStyle.Render("No trace data yet. Run a request first.")
	}

	lines := []string{
		codeStyle.Render("Transaction: " + safeText(preview.Journey.TransactionID)),
		"",
	}

	if len(preview.Journey.Rows) == 0 {
		lines = append(lines, warningStyle.Render("Trace rows will appear after tracing"))
		for _, warning := range preview.Journey.Warnings {
			lines = append(lines, warningStyle.Render(safeText(warning.Message)))
		}
		return strings.Join(lines, "\n")
	}

	lines = append(lines, metaStyle.Render("Step  Service       State       Outcome     Message"))
	for _, row := range preview.Journey.Rows {
		line := journeyLine(row)
		lines = append(lines, journeyRowStyle(row.DisplayLabel).Render(line))
	}

	return strings.Join(lines, "\n")
}

func formatJSON(data []byte, maxWidth int) string {
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "" // Not valid JSON
	}

	// Pretty print with indentation
	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return ""
	}

	result := string(formatted)

	// Truncate if too long (show first 20 lines)
	lines := strings.Split(result, "\n")
	if len(lines) > 20 {
		lines = lines[:20]
		lines = append(lines, "... (truncated)")
	}

	// Syntax highlight (simple version)
	highlighted := make([]string, 0, len(lines))
	for _, line := range lines {
		highlighted = append(highlighted, syntaxHighlightJSON(line))
	}

	return strings.Join(highlighted, "\n")
}

func syntaxHighlightJSON(line string) string {
	// Simple syntax highlighting for JSON
	// Keys in yellow, strings in green, numbers in blue

	// This is a simplified version - just colorize the line
	if strings.Contains(line, ":") {
		// Likely a key-value pair
		return codeStyle.Render(line)
	}
	return bodyStyle.Render(line)
}

func formatBytes(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	kb := float64(bytes) / 1024.0
	if kb < 1024 {
		return fmt.Sprintf("%.2f KB", kb)
	}
	mb := kb / 1024.0
	return fmt.Sprintf("%.2f MB", mb)
}

var (
	tabActiveStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("167")).
			Foreground(lipgloss.Color("231")).
			Bold(true).
			Padding(0, 2)

	tabInactiveStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("250")).
				Padding(0, 2)
)
