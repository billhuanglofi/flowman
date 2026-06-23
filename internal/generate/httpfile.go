package generate

import (
	"strings"

	"github.com/billhuanglofi/flowman/internal/model"
)

func RenderHTTP(request model.Request, options Options) ([]byte, error) {
	projection, err := newProjection(request, options)
	if err != nil {
		return nil, err
	}
	var builder strings.Builder
	builder.WriteString("### ")
	builder.WriteString(DoNotEditMarker)
	builder.WriteByte('\n')
	if projection.environmentName != "" {
		builder.WriteString("# Environment: ")
		builder.WriteString(projection.environmentName)
		builder.WriteByte('\n')
	}
	builder.WriteString(projection.method)
	builder.WriteByte(' ')
	builder.WriteString(projection.url)
	builder.WriteString(" HTTP/1.1\n")
	for _, header := range projection.httpHeaders {
		builder.WriteString(header)
		builder.WriteByte('\n')
	}
	builder.WriteByte('\n')
	if projection.body != "" {
		builder.WriteString(projection.body)
		builder.WriteByte('\n')
	}
	return []byte(builder.String()), nil
}
