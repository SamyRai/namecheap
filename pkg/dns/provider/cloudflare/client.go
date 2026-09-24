package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"zonekit/pkg/errors"
)

// listPerPage is the page size used for every paginated list call. Free
// plan zones rarely have more than a couple hundred records, so one page
// size keeps pagination handling uniform without tuning per endpoint.
const listPerPage = 50

// apiError is one entry in a Cloudflare error envelope.
type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// resultInfo is Cloudflare's pagination envelope, present on list
// endpoints.
type resultInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
	Count      int `json:"count"`
	TotalCount int `json:"total_count"`
}

// apiEnvelope is the outer shape of every Cloudflare API v4 response.
type apiEnvelope struct {
	Success    bool            `json:"success"`
	Errors     []apiError      `json:"errors"`
	Result     json.RawMessage `json:"result"`
	ResultInfo *resultInfo     `json:"result_info,omitempty"`
}

// apiClient is the low-level, stdlib-only HTTP client shared by Provider
// and the ZoneConfigurer methods.
type apiClient struct {
	baseURL    string
	authHeader string
	httpClient *http.Client
}

func newAPIClient(cfg Config) (*apiClient, error) {
	if cfg.APIToken == "" {
		return nil, errors.NewConfiguration("cloudflare: api_token is required")
	}

	return &apiClient{
		baseURL:    cfg.baseURL(),
		authHeader: "Bearer " + cfg.APIToken,
		httpClient: cfg.httpClient(),
	}, nil
}

// do issues one HTTP request and returns the decoded envelope. It treats
// both a non-2xx status and success=false in the envelope as failure,
// surfacing Cloudflare's own error codes/messages when present.
func (c *apiClient) do(ctx context.Context, method, path string, query url.Values, body interface{}) (*apiEnvelope, error) {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("cloudflare: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: build request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.NewAPI(method+" "+path, "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cloudflare: read response body: %w", err)
	}

	var env apiEnvelope
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &env); err != nil {
			return nil, errors.NewAPI(method+" "+path,
				fmt.Sprintf("non-JSON response (status %d): %s", resp.StatusCode, truncate(string(raw), 200)),
				err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !env.Success {
		return nil, errors.NewAPI(method+" "+path, formatAPIErrors(resp.StatusCode, env.Errors), fmt.Errorf("HTTP %d", resp.StatusCode))
	}

	return &env, nil
}

// get performs a GET request with no query parameters. Paginated list
// endpoints go through listPaginated instead, which needs page/per_page
// query params and calls do directly.
func (c *apiClient) get(ctx context.Context, path string, out interface{}) error {
	env, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return err
	}
	return decodeResult(env, out)
}

func (c *apiClient) post(ctx context.Context, path string, body, out interface{}) error {
	env, err := c.do(ctx, http.MethodPost, path, nil, body)
	if err != nil {
		return err
	}
	return decodeResult(env, out)
}

func (c *apiClient) put(ctx context.Context, path string, body, out interface{}) error {
	env, err := c.do(ctx, http.MethodPut, path, nil, body)
	if err != nil {
		return err
	}
	return decodeResult(env, out)
}

func (c *apiClient) patch(ctx context.Context, path string, body, out interface{}) error {
	env, err := c.do(ctx, http.MethodPatch, path, nil, body)
	if err != nil {
		return err
	}
	return decodeResult(env, out)
}

func (c *apiClient) delete(ctx context.Context, path string) error {
	_, err := c.do(ctx, http.MethodDelete, path, nil, nil)
	return err
}

func decodeResult(env *apiEnvelope, out interface{}) error {
	if out == nil || len(env.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Result, out); err != nil {
		return fmt.Errorf("cloudflare: decode result: %w", err)
	}
	return nil
}

// listPaginated fetches every page of a Cloudflare list endpoint and
// returns the concatenated items. It follows result_info.total_pages
// rather than looping until an empty page, since Cloudflare's page
// numbering is 1-based and well-formed lists always report totals.
func listPaginated[T any](ctx context.Context, c *apiClient, path string, query url.Values) ([]T, error) {
	q := cloneValues(query)
	q.Set("per_page", strconv.Itoa(listPerPage))

	var all []T
	for page := 1; ; page++ {
		q.Set("page", strconv.Itoa(page))

		env, err := c.do(ctx, http.MethodGet, path, q, nil)
		if err != nil {
			return nil, err
		}

		var items []T
		if len(env.Result) > 0 {
			if err := json.Unmarshal(env.Result, &items); err != nil {
				return nil, fmt.Errorf("cloudflare: decode page %d: %w", page, err)
			}
		}
		all = append(all, items...)

		if env.ResultInfo == nil || page >= env.ResultInfo.TotalPages {
			break
		}
	}
	return all, nil
}

func cloneValues(v url.Values) url.Values {
	out := url.Values{}
	for k, vals := range v {
		out[k] = append([]string(nil), vals...)
	}
	return out
}

func formatAPIErrors(status int, errs []apiError) string {
	if len(errs) == 0 {
		return fmt.Sprintf("request failed with status %d", status)
	}
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		parts = append(parts, fmt.Sprintf("%d: %s", e.Code, e.Message))
	}
	return fmt.Sprintf("request failed with status %d: %s", status, strings.Join(parts, "; "))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
