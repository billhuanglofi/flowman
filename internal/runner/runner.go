package runner

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/billhuanglofi/flowman/internal/secrets"
)

type Options struct {
	Timeout time.Duration
	Client  *http.Client
}

type Runner struct {
	client  *http.Client
	timeout time.Duration
}

func New(options Options) Runner {
	client := options.Client
	if client == nil {
		client = http.DefaultClient
	}
	return Runner{client: client, timeout: options.Timeout}
}

func (runner Runner) Run(ctx context.Context, execution Execution) (Response, error) {
	prepared, err := prepareRequest(execution)
	if err != nil {
		return Response{}, err
	}
	requestURL, err := applyQuery(prepared.url, execution.Request.Query)
	if err != nil {
		return Response{}, fmt.Errorf("request %q query: %w", execution.Request.Name, err)
	}
	ctx, cancel := runner.contextWithTimeout(ctx)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(ctx, prepared.method, requestURL, prepared.body)
	if err != nil {
		return Response{}, fmt.Errorf("request %q build http request: %w", execution.Request.Name, err)
	}
	if err := applyHeaders(httpRequest, execution.Request.Headers); err != nil {
		return Response{}, fmt.Errorf("request %q headers: %w", execution.Request.Name, err)
	}
	startedAt := time.Now()
	httpResponse, err := runner.client.Do(httpRequest)
	duration := time.Since(startedAt)
	if err != nil {
		return Response{}, fmt.Errorf("request %q send: %w", execution.Request.Name, err)
	}
	defer httpResponse.Body.Close()
	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return Response{}, fmt.Errorf("request %q read response body: %w", execution.Request.Name, err)
	}
	headers := cloneHeaders(httpResponse.Header)
	contentType := httpResponse.Header.Get("Content-Type")
	return Response{
		StatusCode:     httpResponse.StatusCode,
		Headers:        headers,
		DisplayHeaders: redactedHeaders(headers),
		Body:           body,
		DisplayBody:    redactResponseBody(body, contentType),
		Duration:       duration,
	}, nil
}

func (runner Runner) contextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if runner.timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, runner.timeout)
}

func redactResponseBody(body []byte, contentType string) []byte {
	return secrets.RedactResponseBody(body, contentType)
}
