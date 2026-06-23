package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/billhuanglofi/flowman/internal/importer/safestore"
	"github.com/billhuanglofi/flowman/internal/runner"
	flowtrace "github.com/billhuanglofi/flowman/internal/trace"
)

func startRun(model Model) (tea.Model, tea.Cmd) {
	request, ok := selectedRequest(model.preview.Workspace, model.preview.Selected)
	if !ok {
		model.message = "No request selected"
		return model, nil
	}
	if model.services == nil {
		model.message = "No runner service configured"
		return model, nil
	}
	model.status = statusRunning
	model.preview.RunState = "running selected request"
	return model, func() tea.Msg {
		response, err := model.services.RunRequest(context.Background(), runner.Execution{Request: request, Environment: model.environment})
		if err != nil {
			return runCompleteMsg{err: fmt.Errorf("run selected request: %w", err)}
		}
		transactionID, err := flowtrace.ExtractTransactionID(flowtrace.ExtractionInput{Config: requestTraceConfig(request, model.environment), Response: response})
		if err != nil {
			return runCompleteMsg{response: response, err: fmt.Errorf("extract transaction id: %w", err)}
		}
		return runCompleteMsg{response: response, transactionID: transactionID}
	}
}

func handleRunComplete(model Model, msg runCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		model.status = statusReady
		model.preview.RunState = "request run failed"
		model.message = "Run error: " + safeText(msg.err.Error())
		return model, nil
	}

	// Store full response for display
	model.preview.FullResponse = msg.response
	model.preview.Response = responsePreview(msg.response, msg.transactionID)
	model.preview.RunState = "request run complete"

	// Add to history
	request, ok := selectedRequest(model.preview.Workspace, model.preview.Selected)
	if ok {
		model.addToHistory(
			request.Name,
			msg.response.StatusCode,
			msg.response.Duration.String(),
			msg.transactionID,
		)
	}

	return model.startTrace(msg.transactionID)
}

func startTrace(model Model, transactionID string) (tea.Model, tea.Cmd) {
	if model.services == nil {
		model.status = statusReady
		model.message = "No trace service configured"
		return model, nil
	}
	model.status = statusTracing
	model.preview.TraceState = "querying PROCESS_STATE"
	return model, func() tea.Msg {
		journey, err := model.services.TraceJourney(context.Background(), flowtrace.ProcessStateRequest{
			TransactionID:  transactionID,
			Timeout:        model.environment.Trace.Timeout.Duration(),
			Classification: flowtrace.DefaultTerminalStateClassification(),
		})
		if err != nil {
			return traceCompleteMsg{err: fmt.Errorf("trace transaction %s: %w", transactionID, err)}
		}
		return traceCompleteMsg{journey: journey}
	}
}

func handleTraceComplete(model Model, msg traceCompleteMsg) (tea.Model, tea.Cmd) {
	model.status = statusReady
	if msg.err != nil {
		model.preview.TraceState = "trace failed"
		model.message = "Trace error: " + safeText(msg.err.Error()) + " | Action: check --env Oracle settings or retry after import URL resolution | r run"
		return model, nil
	}
	model.preview.Journey = msg.journey
	model.preview.TraceState = "trace complete"
	model.message = "Trace complete"
	return model, nil
}

func startImport(model Model, transactionID string) (tea.Model, tea.Cmd) {
	if model.services == nil {
		model.status = statusReady
		model.message = "No Safestore import service configured"
		return model, nil
	}
	model.status = statusImporting
	model.message = "Safestore import running for " + safeText(transactionID)
	return model, func() tea.Msg {
		result, err := model.services.ImportSafestore(context.Background(), safestore.ImportOptions{
			TransactionID:  transactionID,
			Environment:    model.environment,
			URL:            model.importURL,
			Endpoint:       model.importEndpoint,
			Path:           model.importPath,
			OutputPath:     model.importOutput,
			Selection:      safestore.SelectLatest,
			Overwrite:      true,
			NonInteractive: true,
		})
		if err != nil {
			return importCompleteMsg{err: fmt.Errorf("import Safestore transaction %s: %w", transactionID, err)}
		}
		return importCompleteMsg{result: result}
	}
}

func handleImportComplete(model Model, msg importCompleteMsg) (tea.Model, tea.Cmd) {
	model.status = statusReady
	if msg.err != nil {
		model.message = "Safestore Import error: " + safeText(msg.err.Error()) + " | requires --url, endpoint alias, or env base_url + path"
		return model, nil
	}
	model.message = "Safestore Import | Imported: " + safeText(msg.result.Request.Name) + " | " + safeText(msg.result.OutputPath)
	return model, nil
}

func responsePreview(response runner.Response, transactionID string) ResponsePreview {
	return ResponsePreview{State: "complete", Status: fmt.Sprintf("Status: %d", response.StatusCode), TransactionID: transactionID, Duration: response.Duration.String()}
}
