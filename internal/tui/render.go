package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/billhuanglofi/flowman/internal/model"
	flowtrace "github.com/billhuanglofi/flowman/internal/trace"
	"github.com/mattn/go-runewidth"
)

type ViewState struct {
	Status         string
	Message        string
	InputMode      string
	InputValue     string
	ImportURL      string
	ImportEndpoint string
	ImportPath     string
}

func RenderPreview(preview Preview, size RenderSize, states ...ViewState) string {
	width := normalizedWidth(size.Width)
	state := ViewState{Status: statusReady, Message: "r run | t trace tx | i import safestore | up/down select | q quit"}
	if len(states) > 0 {
		state = states[0]
	}
	if preview.LoadError != nil {
		return renderError(preview, width)
	}
	header := renderHeader(preview)
	left := renderLists(preview, bodyWidth(width, 2))
	right := renderSelected(preview, bodyWidth(width, 2))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
	response := renderResponse(preview, width)
	journey := renderJourney(preview.Journey, width)
	footer := renderFooter(state, width)
	return strings.Join([]string{header, body, response, journey, footer}, "\n")
}

func renderHeader(preview Preview) string {
	environment, ok := selectedEnvironment(preview.Workspace, preview.SelectedEnv)
	baseURL := "base URL unavailable"
	if ok {
		baseURL = safeText(environment.BaseURL)
	}
	parts := []string{
		titleStyle.Render("Flowman"),
		"  ",
		bodyStyle.Render("Environment: " + safeText(preview.SelectedEnv)),
		"  ",
		metaStyle.Render("Base URL: " + baseURL),
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

func renderRequests(preview Preview, width int) string {
	lines := []string{panelTitleStyle.Render("Requests")}
	if len(preview.Workspace.Requests) == 0 {
		lines = append(lines, warningStyle.Render("No requests found in YAML"))
	} else {
		for index, request := range preview.Workspace.Requests {
			line := requestLine(request)
			if index == preview.Selected {
				line = selectedStyle.Render("> " + line)
			} else {
				line = bodyStyle.Render("  " + line)
			}
			lines = append(lines, line)
		}
	}
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func renderLists(preview Preview, width int) string {
	sections := []string{
		renderEnvironments(preview, width),
		renderFlows(preview, width),
		renderRequests(preview, width),
	}
	return strings.Join(sections, "\n")
}

func renderEnvironments(preview Preview, width int) string {
	lines := []string{panelTitleStyle.Render("Environments")}
	if len(preview.Workspace.Environments) == 0 {
		lines = append(lines, warningStyle.Render("No environments found in YAML"))
	} else {
		for _, environment := range preview.Workspace.Environments {
			line := environmentLine(environment)
			if environment.Name == preview.SelectedEnv {
				line = selectedStyle.Render("> " + line)
			} else {
				line = bodyStyle.Render("  " + line)
			}
			lines = append(lines, line)
		}
	}
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func environmentLine(environment model.Environment) string {
	return safeText(environment.Name) + " " + safeText(environment.BaseURL)
}

func renderFlows(preview Preview, width int) string {
	lines := []string{panelTitleStyle.Render("Flows")}
	if len(preview.Workspace.Flows) == 0 {
		lines = append(lines, warningStyle.Render("No flows found in YAML"))
	} else {
		for _, flow := range preview.Workspace.Flows {
			lines = append(lines, bodyStyle.Render("  "+flowLine(flow)))
		}
	}
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func flowLine(flow model.Flow) string {
	return fmt.Sprintf("%s (%d steps)", safeText(flow.Name), len(flow.Steps))
}

func requestLine(request model.Request) string {
	method := safeText(request.Method)
	name := safeText(request.Name)
	return fmt.Sprintf("%s %s", method, name)
}

func renderSelected(preview Preview, width int) string {
	lines := []string{panelTitleStyle.Render("Selected Request")}
	request, ok := selectedRequest(preview.Workspace, preview.Selected)
	if !ok {
		lines = append(lines, warningStyle.Render("No selected request"))
		return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
	}
	lines = append(lines,
		bodyStyle.Render("Name: "+safeText(request.Name)),
		codeStyle.Render("Method: "+safeText(request.Method)),
		metaStyle.Render("Endpoint: "+safeText(request.Endpoint)),
		metaStyle.Render("Path: "+safeText(request.Path)),
		warningStyle.Render("Run: "+safeText(preview.RunState)),
		warningStyle.Render("Trace: "+safeText(preview.TraceState)),
	)
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func renderResponse(preview Preview, width int) string {
	lines := []string{panelTitleStyle.Render("Response")}
	lines = append(lines,
		bodyStyle.Render("State: "+safeText(preview.Response.State)),
		bodyStyle.Render(safeText(preview.Response.Status)),
		codeStyle.Render("Transaction: "+safeText(preview.Response.TransactionID)),
		metaStyle.Render("Duration: "+safeText(preview.Response.Duration)),
	)
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func renderJourney(journey flowtrace.Journey, width int) string {
	lines := []string{panelTitleStyle.Render("Trace Journey")}
	if journey.TransactionID != "" {
		lines = append(lines, codeStyle.Render("Transaction: "+safeText(journey.TransactionID)))
	}
	if len(journey.Rows) == 0 {
		lines = append(lines, warningStyle.Render("Trace rows will appear after a run"))
		for _, warning := range journey.Warnings {
			lines = append(lines, warningStyle.Render(safeText(warning.Message)))
		}
		return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
	}
	lines = append(lines, metaStyle.Render("Step  Service       State       Outcome     Message"))
	for _, row := range journey.Rows {
		line := journeyLine(row)
		lines = append(lines, journeyRowStyle(row.DisplayLabel).Render(line))
	}
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func journeyLine(row flowtrace.JourneyRow) string {
	parts := []string{
		widthCell(fmt.Sprintf("%d", row.Step), 5),
		widthCell(safeText(row.Service), 13),
		widthCell(safeText(row.State), 11),
		widthCell(safeText(row.Outcome), 11),
		widthCell(safeText(row.Message), 34),
	}
	return strings.Join(parts, " ")
}

func widthCell(value string, width int) string {
	truncated := runewidth.Truncate(value, width, "…")
	return runewidth.FillRight(truncated, width)
}

func renderFooter(state ViewState, width int) string {
	lines := []string{panelTitleStyle.Render("Safestore Import")}
	if state.Status != "" {
		lines = append(lines, bodyStyle.Render("Status: "+safeText(state.Status)))
	}
	if state.InputMode != "" && state.InputMode != string(inputModeNone) {
		lines = append(lines, codeStyle.Render("Input "+safeText(state.InputMode)+": "+safeText(state.InputValue)))
	}
	lines = append(lines, metaStyle.Render("URL: "+resolutionValue(state.ImportURL)))
	lines = append(lines, metaStyle.Render("Endpoint: "+resolutionValue(state.ImportEndpoint)))
	lines = append(lines, metaStyle.Render("Path: "+resolutionValue(state.ImportPath)))
	if state.Message != "" {
		lines = append(lines, warningStyle.Render(safeText(state.Message)))
	}
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func resolutionValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "required"
	}
	return safeText(value)
}

func journeyRowStyle(label string) lipgloss.Style {
	switch label {
	case flowtrace.DisplayLabelFailed:
		return errorStyle
	case flowtrace.DisplayLabelStuck:
		return selectedStyle
	case flowtrace.DisplayLabelTerminal:
		return oliveStyle
	default:
		return bodyStyle
	}
}

func renderError(preview Preview, width int) string {
	lines := []string{
		titleStyle.Render("Flowman"),
		errorStyle.Render("Unable to render preview"),
		bodyStyle.Render(safeText(preview.LoadError.Error())),
		warningStyle.Render("check --config path; check --env names a configured environment."),
	}
	if preview.SelectedEnv != "" {
		lines = append(lines, metaStyle.Render("Requested environment: "+safeText(preview.SelectedEnv)))
	}
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func normalizedWidth(width int) int {
	if width < 60 {
		return 60
	}
	return width
}

func bodyWidth(width int, columns int) int {
	return (width - columns - 1) / columns
}
