package postman

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

type importPlan struct {
	files    []plannedFile
	warnings []model.ImportWarning
}

type plannedFile struct {
	request     model.Request
	flow        model.Flow
	requestPath string
	flowPath    string
}

func buildPlan(collection collectionEnvelope) (importPlan, error) {
	plan := importPlan{}
	if len(collection.Variable) > 0 {
		plan.warnings = append(plan.warnings, warning("postman-collection-js-variable-unsupported", "$.variable[0]", "Collection-level Postman variables are not imported; use Flowman env files instead."))
	}
	appendExtraWarnings(&plan.warnings, "$", collection.Extra)
	for itemIndex, item := range collection.Items {
		walkItems(&plan, item, []string{}, fmt.Sprintf("$.item[%d]", itemIndex))
	}
	return plan, nil
}

func walkItems(plan *importPlan, item itemObject, folders []string, pointer string) {
	appendExtraWarnings(&plan.warnings, pointer, item.Extra)
	if len(item.Items) > 0 {
		nextFolders := folders
		if item.Name != "" {
			nextFolders = append(append([]string(nil), folders...), item.Name)
		}
		for index, child := range item.Items {
			walkItems(plan, child, nextFolders, fmt.Sprintf("%s.item[%d]", pointer, index))
		}
		return
	}
	planned, warnings, err := planRequest(item, folders, pointer)
	plan.warnings = append(plan.warnings, warnings...)
	if err != nil {
		plan.warnings = append(plan.warnings, warning("postman-request-import-error", pointer, err.Error()))
		return
	}
	plan.files = append(plan.files, planned)
}

func planRequest(item itemObject, folders []string, pointer string) (plannedFile, []model.ImportWarning, error) {
	warnings := scriptWarnings(pointer, item.Event)
	headers := make([]model.Header, 0, len(item.Request.Header)+1)
	for index, header := range item.Request.Header {
		if strings.TrimSpace(header.Key) == "" {
			continue
		}
		headers = append(headers, importHeader(header.Key, header.Value, envNameFor(folders, item.Name, header.Key)))
		if strings.EqualFold(header.Type, "file") {
			warnings = append(warnings, warning("postman-multipart-file-unsupported", fmt.Sprintf("%s.request.header[%d]", pointer, index), "Multipart file-style header metadata is unsupported and was not imported specially."))
		}
	}
	authHeaders, authWarnings := importAuth(pointer+".request.auth", folders, item.Name, item.Request.Auth)
	headers = append(headers, authHeaders...)
	warnings = append(warnings, authWarnings...)
	importedURL, query, urlWarnings, err := importURL(item.Request.URL)
	if err != nil {
		return plannedFile{}, warnings, err
	}
	warnings = append(warnings, urlWarnings...)
	body, bodyWarnings := importBody(pointer+".request.body", item.Request.Body)
	warnings = append(warnings, bodyWarnings...)
	if len(item.Request.Certificate) > 0 {
		warnings = append(warnings, warning("postman-client-certificate-unsupported", pointer+".request.certificate", "Client certificates are not imported into canonical Flowman YAML."))
	}
	appendExtraWarnings(&warnings, pointer+".request", item.Request.Extra)
	requestPath := buildOutputPath("requests", folders, item.Name, ".request.yaml")
	flowPath := buildOutputPath("flows", folders, item.Name, ".flow.yaml")
	request := model.Request{
		Version:        "v1",
		Name:           item.Name,
		Method:         strings.ToUpper(strings.TrimSpace(item.Request.Method)),
		URL:            importedURL.raw,
		Path:           importedURL.path,
		Query:          query,
		Headers:        dedupeHeaders(headers),
		Body:           body,
		ImportWarnings: append([]model.ImportWarning(nil), warnings...),
	}
	flow := model.Flow{
		Version:        "v1",
		Name:           item.Name,
		Steps:          []model.FlowStep{{Name: item.Name, Request: filepath.ToSlash(requestPath), Trace: true}},
		ImportWarnings: append([]model.ImportWarning(nil), warnings...),
	}
	return plannedFile{request: request, flow: flow, requestPath: requestPath, flowPath: flowPath}, warnings, nil
}

type importedURL struct {
	raw  string
	path string
}

func importURL(raw json.RawMessage) (importedURL, []model.Parameter, []model.ImportWarning, error) {
	if len(raw) == 0 {
		return importedURL{}, nil, nil, nil
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return importedURL{}, nil, nil, fmt.Errorf("decode request URL string: %w", err)
		}
		return importedURL{raw: value}, nil, nil, nil
	}
	var value urlObject
	if err := json.Unmarshal(raw, &value); err != nil {
		return importedURL{}, nil, nil, fmt.Errorf("decode request URL object: %w", err)
	}
	query := make([]model.Parameter, 0, len(value.Query))
	for _, parameter := range value.Query {
		query = append(query, model.Parameter{Name: parameter.Key, Value: parameter.Value})
	}
	sort.Slice(query, func(left int, right int) bool {
		if query[left].Name == query[right].Name {
			return query[left].Value < query[right].Value
		}
		return query[left].Name < query[right].Name
	})
	path := "/" + strings.Trim(strings.Join(value.Path, "/"), "/")
	if path == "/" && len(value.Path) == 0 {
		path = ""
	}
	if strings.TrimSpace(value.Raw) != "" {
		return importedURL{path: path}, query, nil, nil
	}
	return importedURL{raw: joinURLParts(value), path: path}, query, nil, nil
}

func joinURLParts(value urlObject) string {
	host := strings.Join(value.Host, ".")
	path := strings.Join(value.Path, "/")
	base := host
	if value.Protocol != "" {
		base = value.Protocol + "://" + host
	}
	if path == "" {
		return base
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}
