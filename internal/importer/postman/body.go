package postman

import (
	"fmt"
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

func importBody(pointer string, body bodyObject) (model.RequestBody, []model.ImportWarning) {
	warnings := make([]model.ImportWarning, 0)
	appendExtraWarnings(&warnings, pointer, body.Extra)
	mode := strings.TrimSpace(strings.ToLower(body.Mode))
	switch mode {
	case "", "raw":
		if len(body.GraphQL) > 0 {
			warnings = append(warnings, warning("postman-graphql-unsupported", pointer+".graphql", "GraphQL schema-specific body handling is unsupported; raw body was kept if present."))
		}
		if strings.EqualFold(stringValue(body.Options.Raw["language"]), "json") {
			return model.RequestBody{Mode: "json", Raw: body.Raw}, warnings
		}
		return model.RequestBody{Mode: "raw", Raw: body.Raw}, warnings
	case "urlencoded":
		form := make([]model.Parameter, 0, len(body.URLEnc))
		for _, parameter := range body.URLEnc {
			form = append(form, model.Parameter{Name: parameter.Key, Value: parameter.Value})
		}
		return model.RequestBody{Mode: "form", Form: form}, warnings
	case "formdata":
		for index, field := range body.FormData {
			if strings.EqualFold(field.Type, "file") || field.Src != nil {
				warnings = append(warnings, warning("postman-multipart-file-unsupported", fmt.Sprintf("%s.formdata[%d].src", pointer, index), "Multipart file uploads are unsupported and were not imported."))
			}
		}
		return model.RequestBody{}, warnings
	case "graphql":
		warnings = append(warnings, warning("postman-graphql-unsupported", pointer, "GraphQL schema-specific body handling is unsupported."))
		return model.RequestBody{}, warnings
	default:
		warnings = append(warnings, warning("postman-body-mode-unmapped", pointer+".mode", fmt.Sprintf("Body mode %q is unsupported and was not imported.", body.Mode)))
		return model.RequestBody{}, warnings
	}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
