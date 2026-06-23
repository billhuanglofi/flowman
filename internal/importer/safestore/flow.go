package safestore

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/storage"
)

func writeFlows(requests []model.Request, requestPaths []string, flowPaths []string) ([]model.Flow, error) {
	if len(flowPaths) == 0 {
		return nil, nil
	}
	flows := buildFlows(requests, requestPaths)
	for index, flow := range flows {
		if err := storage.WriteCanonical(flowPaths[index], flow); err != nil {
			return nil, fmt.Errorf("write Safestore replay flow %s: %w", flowPaths[index], err)
		}
	}
	return flows, nil
}

func buildFlows(requests []model.Request, requestPaths []string) []model.Flow {
	flows := make([]model.Flow, 0, len(requests))
	for index, request := range requests {
		flows = append(flows, model.Flow{
			Version: "v1",
			Name:    request.Name,
			Steps: []model.FlowStep{{
				Name:    "replay request",
				Request: requestReference(requestPaths[index]),
				Trace:   true,
			}},
			ImportWarnings: request.ImportWarnings,
		})
	}
	return flows
}

func requestReference(path string) string {
	clean := filepath.ToSlash(filepath.Clean(path))
	if index := strings.Index(clean, "requests/"); index >= 0 {
		return clean[index:]
	}
	return filepath.Base(clean)
}

func importResult(requests []model.Request, paths []string, flows []model.Flow, flowPaths []string) ImportResult {
	result := ImportResult{Request: requests[0], Requests: requests, OutputPath: paths[0], OutputPaths: paths}
	if len(flows) > 0 {
		result.Flow = flows[0]
		result.Flows = flows
		result.FlowPath = flowPaths[0]
		result.FlowPaths = flowPaths
	}
	return result
}
