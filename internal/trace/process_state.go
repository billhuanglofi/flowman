package trace

import (
	"strings"
	"time"
)

const WarningProcessStateEmpty = "process_state_empty"

const (
	DisplayLabelActive   = "active"
	DisplayLabelFailed   = "failed"
	DisplayLabelStuck    = "stuck"
	DisplayLabelTerminal = "terminal"
)

type TraceRow struct {
	TransactionID string
	Timestamp     time.Time
	Step          int
	Service       string
	State         string
	Outcome       string
	Message       string
}

type JourneyRow struct {
	TransactionID string
	Timestamp     time.Time
	Step          int
	Service       string
	State         string
	Outcome       string
	Message       string
	DisplayLabel  string
}

type JourneyWarning struct {
	Code    string
	Message string
}

type Journey struct {
	TransactionID   string
	Rows            []JourneyRow
	Warnings        []JourneyWarning
	DisplayStuckRow *JourneyRow
}

type TerminalStateClassification struct {
	TerminalStates []string
	FailedStates   []string
	FailedOutcomes []string
}

func DefaultTerminalStateClassification() TerminalStateClassification {
	return TerminalStateClassification{
		TerminalStates: []string{"COMPLETE", "COMPLETED", "DONE", "SUCCESS", "SUCCEEDED"},
		FailedStates:   []string{"ERROR", "FAILED"},
		FailedOutcomes: []string{"ERROR", "FAILED"},
	}
}

func MapProcessStateJourney(transactionID string, rows []TraceRow, classification TerminalStateClassification) Journey {
	if len(rows) == 0 {
		return Journey{
			TransactionID: transactionID,
			Warnings: []JourneyWarning{{
				Code:    WarningProcessStateEmpty,
				Message: "PROCESS_STATE returned no rows for transaction " + transactionID,
			}},
		}
	}
	classifier := newStateClassifier(classification)
	journey := Journey{TransactionID: transactionID, Rows: make([]JourneyRow, 0, len(rows))}
	for _, row := range rows {
		journey.Rows = append(journey.Rows, mapJourneyRow(row, classifier))
	}
	markDisplayStuckRow(&journey)
	return journey
}

func mapJourneyRow(row TraceRow, classifier stateClassifier) JourneyRow {
	return JourneyRow{
		TransactionID: row.TransactionID,
		Timestamp:     row.Timestamp,
		Step:          row.Step,
		Service:       row.Service,
		State:         row.State,
		Outcome:       row.Outcome,
		Message:       row.Message,
		DisplayLabel:  classifier.displayLabel(row),
	}
}

func markDisplayStuckRow(journey *Journey) {
	for index := len(journey.Rows) - 1; index >= 0; index-- {
		row := journey.Rows[index]
		if row.DisplayLabel == DisplayLabelTerminal {
			continue
		}
		if row.DisplayLabel == DisplayLabelActive {
			journey.Rows[index].DisplayLabel = DisplayLabelStuck
		}
		selected := journey.Rows[index]
		journey.DisplayStuckRow = &selected
		return
	}
}

type stateClassifier struct {
	terminalStates map[string]struct{}
	failedStates   map[string]struct{}
	failedOutcomes map[string]struct{}
}

func newStateClassifier(classification TerminalStateClassification) stateClassifier {
	if len(classification.TerminalStates) == 0 && len(classification.FailedStates) == 0 && len(classification.FailedOutcomes) == 0 {
		classification = DefaultTerminalStateClassification()
	}
	return stateClassifier{
		terminalStates: labelSet(classification.TerminalStates),
		failedStates:   labelSet(classification.FailedStates),
		failedOutcomes: labelSet(classification.FailedOutcomes),
	}
}

func (classifier stateClassifier) displayLabel(row TraceRow) string {
	if hasLabel(classifier.failedStates, row.State) || hasLabel(classifier.failedOutcomes, row.Outcome) {
		return DisplayLabelFailed
	}
	if hasLabel(classifier.terminalStates, row.State) {
		return DisplayLabelTerminal
	}
	return DisplayLabelActive
}

func labelSet(labels []string) map[string]struct{} {
	set := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		canonical := canonicalLabel(label)
		if canonical == "" {
			continue
		}
		set[canonical] = struct{}{}
	}
	return set
}

func hasLabel(labels map[string]struct{}, value string) bool {
	_, ok := labels[canonicalLabel(value)]
	return ok
}

func canonicalLabel(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
