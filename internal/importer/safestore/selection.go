package safestore

import (
	"fmt"
	"strconv"
	"strings"
)

func selectRows(rows []RequestRow, options ImportOptions) ([]RequestRow, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("transaction %q: %w", options.TransactionID, ErrNoRequestRows)
	}
	if options.All {
		return rows, nil
	}
	if len(rows) == 1 && options.Selection == "" {
		return rows, nil
	}
	row, err := selectOneRow(rows, options.Selection)
	if err != nil {
		return nil, err
	}
	return []RequestRow{row}, nil
}

func selectOneRow(rows []RequestRow, selection Selection) (RequestRow, error) {
	switch selection {
	case "":
		return RequestRow{}, fmt.Errorf("got %d Safestore request rows: use --select first|latest|index:N or --all: %w", len(rows), ErrMultipleRequestRows)
	case SelectFirst:
		return rows[0], nil
	case SelectLatest:
		return latestRow(rows), nil
	default:
		return indexedRow(rows, selection)
	}
}

func latestRow(rows []RequestRow) RequestRow {
	latest := rows[0]
	for _, row := range rows[1:] {
		if row.Timestamp.After(latest.Timestamp) {
			latest = row
		}
	}
	return latest
}

func indexedRow(rows []RequestRow, selection Selection) (RequestRow, error) {
	rawIndex, ok := strings.CutPrefix(string(selection), "index:")
	if !ok {
		return RequestRow{}, fmt.Errorf("selection %q: %w", selection, ErrInvalidSelection)
	}
	index, err := strconv.Atoi(rawIndex)
	if err != nil || index < 0 || index >= len(rows) {
		return RequestRow{}, fmt.Errorf("selection %q outside 0..%d: %w", selection, len(rows)-1, ErrInvalidSelection)
	}
	return rows[index], nil
}
