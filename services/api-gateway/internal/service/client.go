package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ForwardRequest struct {
	Method   string
	Path     string
	RawQuery string
	Headers  http.Header
	Body     []byte
}

type ForwardResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

type ServiceClient interface {
	Forward(ctx context.Context, baseURL string, req ForwardRequest) (*ForwardResponse, error)
}

type HTTPClient struct {
	httpClient *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *HTTPClient) Forward(ctx context.Context, baseURL string, req ForwardRequest) (*ForwardResponse, error) {
	url := strings.TrimRight(baseURL, "/") + req.Path
	if req.RawQuery != "" {
		url += "?" + req.RawQuery
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, bytes.NewReader(req.Body))
	if err != nil {
		return nil, fmt.Errorf("build upstream request: %w", err)
	}

	httpReq.Header = cloneHeaders(req.Headers)
	httpReq.Host = ""

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read upstream response: %w", err)
	}

	return &ForwardResponse{
		StatusCode: resp.StatusCode,
		Headers:    cloneHeaders(resp.Header),
		Body:       body,
	}, nil
}

func cloneHeaders(headers http.Header) http.Header {
	cloned := make(http.Header, len(headers))
	for key, values := range headers {
		for _, value := range values {
			cloned.Add(key, value)
		}
	}
	return cloned
}
