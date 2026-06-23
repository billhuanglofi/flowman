package insomnia

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
	"github.com/billhuanglofi/flowman/internal/secrets"
)

type plannedFile struct {
	request     model.Request
	flow        model.Flow
	requestPath string
	flowPath    string
}

func buildPlan(doc document) ([]plannedFile, []model.ImportWarning) {
	resourcesByID := map[string]resource{}
	for _, entry := range doc.Resources {
		resourcesByID[entry.ID] = entry
	}
	files := make([]plannedFile, 0)
	warnings := make([]model.ImportWarning, 0)
	requestIndex := 0
	for _, entry := range doc.Resources {
		if entry.Type != "request" {
			continue
		}
		if len(entry.CookieJar) > 0 {
			warnings = append(warnings, model.ImportWarning{Code: "insomnia-cookie-jar-unsupported", Path: fmt.Sprintf("$.resources[%d].cookieJar", requestIndex+2), Message: "Cookie jars are not canonical Flowman source and were not imported."})
			requestIndex++
			continue
		}
		file, requestWarnings := planRequest(entry, resourcesByID, requestIndex)
		requestIndex++
		warnings = append(warnings, requestWarnings...)
		files = append(files, file)
	}
	return files, warnings
}

func planRequest(entry resource, resourcesByID map[string]resource, requestIndex int) (plannedFile, []model.ImportWarning) {
	folders := requestFolders(entry, resourcesByID)
	warnings := make([]model.ImportWarning, 0)
	headers := make([]model.Header, 0, len(entry.Headers))
	for index, item := range entry.Headers {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		headerValue := model.Header{Name: item.Name, Value: item.Value}
		if secrets.IsSensitiveHeader(item.Name) {
			headerValue = model.Header{Name: item.Name, Env: envNameFor(folders, entry.Name, item.Name)}
		}
		headers = append(headers, headerValue)
		if item.Value == "" && strings.EqualFold(item.Name, "Authorization") {
			warnings = append(warnings, model.ImportWarning{Code: "insomnia-auth-unmapped", Path: fmt.Sprintf("$.resources[%d].headers[%d]", requestIndex+2, index), Message: "Authorization header is missing a value and was not imported specially."})
		}
	}
	if len(entry.ClientCertificates) > 0 {
		warnings = append(warnings, model.ImportWarning{Code: "insomnia-client-certificate-unsupported", Path: fmt.Sprintf("$.resources[%d].clientCertificates[0]", requestIndex+2), Message: "Client certificates are not imported into canonical Flowman YAML."})
	}
	if len(entry.Authentication) > 0 {
		authHeaders, authWarnings := importAuthentication(fmt.Sprintf("$.resources[%d].authentication", requestIndex+2), folders, entry.Name, entry.Authentication)
		headers = append(headers, authHeaders...)
		warnings = append(warnings, authWarnings...)
	}
	body, bodyWarnings := importBody(entry, requestIndex+2)
	warnings = append(warnings, bodyWarnings...)
	query := sortedQuery(entry.Parameters)
	requestPath := buildOutputPath("requests", folders, entry.Name, ".request.yaml")
	flowPath := buildOutputPath("flows", folders, entry.Name, ".flow.yaml")
	request := model.Request{Version: "v1", Name: entry.Name, Method: strings.ToUpper(strings.TrimSpace(entry.Method)), URL: explicitURL(entry.URL), Path: inferredPath(entry.URL), Query: query, Headers: dedupeHeaders(headers), Body: body, ImportWarnings: append([]model.ImportWarning(nil), warnings...)}
	flow := model.Flow{Version: "v1", Name: entry.Name, Steps: []model.FlowStep{{Name: entry.Name, Request: filepath.ToSlash(requestPath), Trace: true}}, ImportWarnings: append([]model.ImportWarning(nil), warnings...)}
	return plannedFile{request: request, flow: flow, requestPath: requestPath, flowPath: flowPath}, warnings
}

func requestFolders(entry resource, resourcesByID map[string]resource) []string {
	folders := make([]string, 0)
	current := entry.ParentID
	for current != "" {
		parent, ok := resourcesByID[current]
		if !ok || parent.Type != "request_group" {
			break
		}
		folders = append([]string{parent.Name}, folders...)
		current = parent.ParentID
	}
	return folders
}

func importBody(entry resource, resourceIndex int) (model.RequestBody, []model.ImportWarning) {
	mime := strings.ToLower(strings.TrimSpace(entry.Body.MimeType))
	switch mime {
	case "", "text/plain":
		return model.RequestBody{Mode: "raw", Raw: entry.Body.Text}, nil
	case "application/json":
		return model.RequestBody{Mode: "json", Raw: entry.Body.Text}, nil
	case "application/x-www-form-urlencoded":
		form := make([]model.Parameter, 0, len(entry.Body.Params))
		for _, item := range entry.Body.Params {
			form = append(form, model.Parameter{Name: item.Name, Value: item.Value})
		}
		return model.RequestBody{Mode: "form", Form: form}, nil
	case "multipart/form-data":
		warnings := make([]model.ImportWarning, 0, len(entry.Body.Params))
		for index, item := range entry.Body.Params {
			if strings.TrimSpace(item.FileName) != "" {
				warnings = append(warnings, model.ImportWarning{Code: "insomnia-multipart-file-unsupported", Path: fmt.Sprintf("$.resources[%d].body.params[%d].fileName", resourceIndex, index), Message: "Multipart file uploads are unsupported and were not imported."})
			}
		}
		return model.RequestBody{}, warnings
	default:
		return model.RequestBody{}, []model.ImportWarning{{Code: "insomnia-body-mode-unmapped", Path: fmt.Sprintf("$.resources[%d].body.mimeType", resourceIndex), Message: fmt.Sprintf("Body mime type %q is unsupported and was not imported.", entry.Body.MimeType)}}
	}
}

func sortedQuery(parameters []parameter) []model.Parameter {
	query := make([]model.Parameter, 0, len(parameters))
	for _, item := range parameters {
		query = append(query, model.Parameter{Name: item.Name, Value: item.Value})
	}
	sort.Slice(query, func(left int, right int) bool {
		if query[left].Name == query[right].Name {
			return query[left].Value < query[right].Value
		}
		return query[left].Name < query[right].Name
	})
	return query
}
