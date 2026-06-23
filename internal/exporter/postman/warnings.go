package postman

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/billhuanglofi/flowman/internal/model"
)

func flowWarnings(workspace model.Workspace) []model.ImportWarning {
	warnings := make([]model.ImportWarning, 0)
	requestPaths := map[string]bool{}
	for _, path := range workspace.Project.Requests {
		requestPaths[path] = true
	}
	for flowIndex, flow := range workspace.Flows {
		if len(flow.Steps) > 1 {
			warnings = append(warnings, model.ImportWarning{
				Code:    "postman-flow-multistep-unsupported",
				Path:    fmt.Sprintf("$.flows[%d].steps", flowIndex),
				Message: "Postman export keeps request folders but cannot represent multi-step Flowman flow sequencing losslessly.",
			})
		}
		for stepIndex, step := range flow.Steps {
			if !requestPaths[step.Request] {
				warnings = append(warnings, model.ImportWarning{
					Code:    "postman-flow-request-missing",
					Path:    fmt.Sprintf("$.flows[%d].steps[%d].request", flowIndex, stepIndex),
					Message: "Flow step references a request path that is not part of the workspace request list.",
				})
			}
		}
	}
	return warnings
}

func writeWarningReport(path string, warnings []model.ImportWarning) error {
	if path == "" {
		return nil
	}
	payload := struct {
		Warnings []model.ImportWarning `json:"warnings"`
	}{Warnings: warnings}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal postman warning report: %w", err)
	}
	content = append(content, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create warning report directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write warning report %s: %w", path, err)
	}
	return nil
}
