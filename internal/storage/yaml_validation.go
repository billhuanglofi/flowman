package storage

import (
	"errors"
	"fmt"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

func validateCanonical(path string, value any) error {
	switch typed := value.(type) {
	case model.Project:
		return validateProject(path, typed)
	case model.Environment:
		return validateEnvironment(path, typed)
	case model.Request:
		return validateRequest(path, typed)
	case model.Flow:
		return validateFlow(path, typed)
	default:
		return nil
	}
}

func validateProject(path string, project model.Project) error {
	return errors.Join(
		requireString(path, "version", project.Version),
		requireString(path, "name", project.Name),
		requireString(path, "default_env", project.DefaultEnv),
		requireList(path, "environments", len(project.Environments)),
	)
}

func validateEnvironment(path string, environment model.Environment) error {
	return errors.Join(
		requireString(path, "version", environment.Version),
		requireString(path, "name", environment.Name),
		requireString(path, "base_url", environment.BaseURL),
		requireString(path, "oracle.dsn_env", environment.Oracle.DSNEnv),
		requireString(path, "oracle.user_env", environment.Oracle.UserEnv),
		requireString(path, "oracle.password_env", environment.Oracle.PasswordEnv),
		validateTrace(path, "trace", environment.Trace),
	)
}

func validateRequest(path string, request model.Request) error {
	return errors.Join(
		requireString(path, "version", request.Version),
		requireString(path, "name", request.Name),
		requireString(path, "method", request.Method),
		validateParameters(path, "query", request.Query),
		validateHeaders(path, request.Headers),
		validateBody(path, request.Body),
		validateOptionalTrace(path, "trace", request.Trace),
	)
}

func validateFlow(path string, flow model.Flow) error {
	var errs []error
	errs = append(errs,
		requireString(path, "version", flow.Version),
		requireString(path, "name", flow.Name),
		requireList(path, "steps", len(flow.Steps)),
	)
	for index, step := range flow.Steps {
		prefix := fmt.Sprintf("steps[%d]", index)
		errs = append(errs,
			requireString(path, prefix+".name", step.Name),
			requireString(path, prefix+".request", step.Request),
		)
	}
	return errors.Join(errs...)
}

func validateParameters(path string, field string, parameters []model.Parameter) error {
	var errs []error
	for index, parameter := range parameters {
		prefix := fmt.Sprintf("%s[%d]", field, index)
		errs = append(errs,
			requireString(path, prefix+".name", parameter.Name),
			requireString(path, prefix+".value", parameter.Value),
		)
	}
	return errors.Join(errs...)
}

func validateHeaders(path string, headers []model.Header) error {
	var errs []error
	for index, header := range headers {
		prefix := fmt.Sprintf("headers[%d]", index)
		errs = append(errs, requireString(path, prefix+".name", header.Name))
		if strings.TrimSpace(header.Value) == "" && strings.TrimSpace(header.Env) == "" {
			errs = append(errs, missingRequiredField(path, prefix+".value or "+prefix+".env"))
		}
	}
	return errors.Join(errs...)
}

func validateBody(path string, body model.RequestBody) error {
	if body.Mode == "" && body.Raw == "" && len(body.Form) == 0 {
		return nil
	}
	return errors.Join(
		requireString(path, "body.mode", body.Mode),
		validateParameters(path, "body.form", body.Form),
	)
}

func validateOptionalTrace(path string, field string, trace model.TraceConfig) error {
	if trace.TransactionID.Header == "" && len(trace.TransactionID.JSONPaths) == 0 && trace.TransactionID.Regex == "" && trace.PollInterval.Duration() == 0 && trace.Timeout.Duration() == 0 {
		return nil
	}
	return validateTrace(path, field, trace)
}

func validateTrace(path string, field string, trace model.TraceConfig) error {
	return errors.Join(
		requireTraceExtractor(path, field+".transaction_id", trace.TransactionID),
		requireDuration(path, field+".poll_interval", trace.PollInterval),
		requireDuration(path, field+".timeout", trace.Timeout),
	)
}

func requireTraceExtractor(path string, field string, extraction model.TransactionIDExtraction) error {
	if strings.TrimSpace(extraction.Header) != "" || len(extraction.JSONPaths) > 0 || strings.TrimSpace(extraction.Regex) != "" {
		return nil
	}
	return missingRequiredField(path, field+".header or "+field+".json_paths or "+field+".regex")
}

func requireString(path string, field string, value string) error {
	if strings.TrimSpace(value) != "" {
		return nil
	}
	return missingRequiredField(path, field)
}

func requireDuration(path string, field string, value model.Duration) error {
	if value.Duration() > 0 {
		return nil
	}
	return missingRequiredField(path, field)
}

func requireList(path string, field string, length int) error {
	if length > 0 {
		return nil
	}
	return missingRequiredField(path, field)
}

func missingRequiredField(path string, field string) error {
	return fmt.Errorf("%s: required field %s: %w", path, field, ErrInvalidCanonicalYAML)
}
