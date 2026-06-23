package safestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/storage"
)

const (
	SelectFirst  Selection = "first"
	SelectLatest Selection = "latest"
)

var ErrCannotResolveURL = errors.New("safestore import: cannot resolve request URL")

var ErrCannotResolveURLNonInteractive = errors.New("safestore import: cannot resolve request URL in noninteractive mode")

var ErrInteractiveURLResolutionNotImplemented = errors.New("safestore import: interactive URL resolution is not implemented in core importer")

var ErrInvalidHeaderJSON = errors.New("safestore import: invalid header json")

var ErrMultipleRequestRows = errors.New("safestore import: multiple request rows require selection")

var ErrOutputExists = errors.New("safestore import: output exists")

var ErrNoRequestRows = errors.New("safestore import: no request rows")

var ErrInvalidSelection = errors.New("safestore import: invalid selection")

type Selection string

type ImportOptions struct {
	TransactionID  string
	Environment    model.Environment
	URL            string
	Endpoint       string
	Path           string
	Method         string
	OutputPath     string
	FlowOutputPath string
	Selection      Selection
	All            bool
	Overwrite      bool
	NonInteractive bool
}

type ImportResult struct {
	Request     model.Request
	Requests    []model.Request
	Flow        model.Flow
	Flows       []model.Flow
	OutputPath  string
	OutputPaths []string
	FlowPath    string
	FlowPaths   []string
}

func ImportReplayRequest(ctx context.Context, store RequestStore, options ImportOptions) (ImportResult, error) {
	rows, err := store.RequestRows(ctx, options.TransactionID)
	if err != nil {
		return ImportResult{}, fmt.Errorf("query Safestore request rows for transaction %q: %w", options.TransactionID, err)
	}
	selectedRows, err := selectRows(rows, options)
	if err != nil {
		return ImportResult{}, err
	}
	requests, err := buildRequests(selectedRows, options)
	if err != nil {
		return ImportResult{}, err
	}
	paths := outputPaths(options.OutputPath, len(requests))
	if err := ensureWritable(paths, options.Overwrite); err != nil {
		return ImportResult{}, err
	}
	flowPaths := optionalOutputPaths(options.FlowOutputPath, len(requests))
	if err := ensureWritable(flowPaths, options.Overwrite); err != nil {
		return ImportResult{}, err
	}
	for index, request := range requests {
		if err := storage.WriteCanonical(paths[index], request); err != nil {
			return ImportResult{}, fmt.Errorf("write Safestore replay request %s: %w", paths[index], err)
		}
	}
	flows, err := writeFlows(requests, paths, flowPaths)
	if err != nil {
		return ImportResult{}, err
	}
	return importResult(requests, paths, flows, flowPaths), nil
}

func buildRequests(rows []RequestRow, options ImportOptions) ([]model.Request, error) {
	requests := make([]model.Request, 0, len(rows))
	for index, row := range rows {
		request, err := buildRequest(row, options, index)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func buildRequest(row RequestRow, options ImportOptions, index int) (model.Request, error) {
	resolved, err := resolveURL(options)
	if err != nil {
		return model.Request{}, err
	}
	headers, warnings, err := parseHeaders(row.Headers, row.TransactionID)
	if err != nil {
		return model.Request{}, err
	}
	return model.Request{
		Version:        "v1",
		Name:           requestName(row.TransactionID, index),
		Method:         requestMethod(options.Method),
		URL:            resolved.url,
		Endpoint:       resolved.endpoint,
		Path:           resolved.path,
		Headers:        headers,
		Body:           model.RequestBody{Mode: bodyMode(row.Body), Raw: row.Body},
		ImportWarnings: warnings,
	}, nil
}

type resolvedURL struct {
	url      string
	endpoint string
	path     string
}

func requestMethod(method string) string {
	trimmed := strings.TrimSpace(method)
	if trimmed == "" {
		return "POST"
	}
	return strings.ToUpper(trimmed)
}

func requestName(transactionID string, index int) string {
	if index == 0 {
		return "replay-" + transactionID
	}
	return fmt.Sprintf("replay-%s-%d", transactionID, index)
}

func bodyMode(body string) string {
	var raw json.RawMessage
	if json.Unmarshal([]byte(body), &raw) == nil {
		return "json"
	}
	return "raw"
}
