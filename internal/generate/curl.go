package generate

import (
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

func RenderCurl(request model.Request, options Options) ([]byte, error) {
	projection, err := newProjection(request, options)
	if err != nil {
		return nil, err
	}
	var builder strings.Builder
	builder.WriteString("#!/bin/sh\n")
	builder.WriteString("# ")
	builder.WriteString(DoNotEditMarker)
	builder.WriteByte('\n')
	if projection.environmentName != "" {
		builder.WriteString("# Environment: ")
		builder.WriteString(projection.environmentName)
		builder.WriteByte('\n')
	}
	builder.WriteString("curl \\\n")
	builder.WriteString("  --request ")
	builder.WriteString(projection.method)
	builder.WriteString(" \\\n")
	builder.WriteString("  --url '")
	builder.WriteString(shellEscapeSingleQuoted(projection.url))
	builder.WriteString("'")
	if len(projection.headers) == 0 && projection.body == "" {
		builder.WriteByte('\n')
		return []byte(builder.String()), nil
	}
	builder.WriteString(" \\\n")
	for index, header := range projection.headers {
		builder.WriteString("  --header '")
		builder.WriteString(shellEscapeSingleQuoted(header))
		builder.WriteString("'")
		if index < len(projection.headers)-1 || projection.body != "" {
			builder.WriteString(" \\\n")
		} else {
			builder.WriteByte('\n')
		}
	}
	if projection.body != "" {
		builder.WriteString("  --data-binary @- <<'FLOWMAN_BODY'\n")
		builder.WriteString(projection.body)
		builder.WriteString("\nFLOWMAN_BODY\n")
	}
	return []byte(builder.String()), nil
}
