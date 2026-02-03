package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"zonekit/pkg/errors"
)

// Client is a generic HTTP client for DNS provider APIs
type Client struct {
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
	timeout    time.Duration
	retries    int
}

// ClientConfig configures the HTTP client
type ClientConfig struct {
	BaseURL string
	Headers map[string]string
	Timeout time.Duration // in seconds
	Retries int
}

// NewClient creates a new HTTP client with the given configuration
func NewClient(config ClientConfig) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	retries := config.Retries
	if retries == 0 {
		retries = 3
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: config.BaseURL,
		headers: config.Headers,
		timeout: timeout,
		retries: retries,
	}
}

// RequestOptions contains options for making HTTP requests
type RequestOptions struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    interface{}
	Query   map[string]string
}

// Do performs an HTTP request with retry logic
func (c *Client) Do(ctx context.Context, opts RequestOptions) (*http.Response, error) {
	url := c.baseURL + opts.Path

	// Build request body
	var bodyReader io.Reader
	if opts.Body != nil {
		bodyBytes, err := json.Marshal(opts.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, opts.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// Set request-specific headers
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	// Set content-type if body is present
	if opts.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add query parameters
	if len(opts.Query) > 0 {
		q := req.URL.Query()
		for key, value := range opts.Query {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	// Perform request with retry logic
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Check if status code indicates retryable error
		if shouldRetry(resp.StatusCode) && attempt < c.retries {
			resp.Body.Close()
			lastErr = fmt.Errorf("received status %d", resp.StatusCode)
			continue
		}

		// Check for non-2xx status codes
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, errors.NewAPI(
				opts.Method,
				fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)),
				fmt.Errorf("HTTP %d", resp.StatusCode),
			)
		}

		return resp, nil
	}

	return nil, errors.NewAPI(
		opts.Method,
		fmt.Sprintf("request failed after %d retries", c.retries),
		lastErr,
	)
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string, query map[string]string) (*http.Response, error) {
	return c.Do(ctx, RequestOptions{
		Method: http.MethodGet,
		Path:   path,
		Query:  query,
	})
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	return c.Do(ctx, RequestOptions{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
	})
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	return c.Do(ctx, RequestOptions{
		Method: http.MethodPut,
		Path:   path,
		Body:   body,
	})
}

// Patch performs a PATCH request
func (c *Client) Patch(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	return c.Do(ctx, RequestOptions{
		Method: http.MethodPatch,
		Path:   path,
		Body:   body,
	})
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	return c.Do(ctx, RequestOptions{
		Method: http.MethodDelete,
		Path:   path,
	})
}

// shouldRetry determines if a status code indicates a retryable error
func shouldRetry(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusInternalServerError ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}

// ParseJSONResponse parses a JSON response into the given struct
func ParseJSONResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	return nil
}
