package youtrack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
	userAgent  string
}

type Options struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	UserAgent  string
	Timeout    time.Duration
}

func NewClient(opts Options) (*Client, error) {
	if strings.TrimSpace(opts.BaseURL) == "" {
		return nil, app.Authf("YouTrack URL is not configured")
	}
	u, err := url.Parse(opts.BaseURL)
	if err != nil {
		return nil, app.Validationf("invalid YouTrack URL: %v", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, app.Validationf("YouTrack URL must use http or https")
	}
	if u.Host == "" {
		return nil, app.Validationf("YouTrack URL must include a host")
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.Path = strings.TrimSuffix(u.Path, "/")

	hc := opts.HTTPClient
	if hc == nil {
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		hc = &http.Client{Timeout: timeout}
	}
	ua := opts.UserAgent
	if ua == "" {
		ua = "youtrack-cli/dev"
	}
	return &Client{baseURL: u, token: opts.Token, httpClient: hc, userAgent: ua}, nil
}

func (c *Client) apiURL(path string, query url.Values) string {
	u := *c.baseURL
	basePath := strings.TrimSuffix(c.baseURL.Path, "/")
	u.Path = basePath + "/" + strings.TrimPrefix(path, "/")
	u.RawQuery = query.Encode()
	return u.String()
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body, out any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}
	resp, data, err := c.do(ctx, method, path, query, payload, nil) //nolint:bodyclose // do closes the body after buffering it.
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapAPIError(method, path, resp, data)
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode YouTrack response: %w", err)
	}
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body []byte, headers http.Header) (*http.Response, []byte, error) {
	attempts := 1
	if method == http.MethodGet {
		attempts = 3
	}
	backoff := []time.Duration{100 * time.Millisecond, 300 * time.Millisecond}

	for attempt := 0; attempt < attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.apiURL(path, query), bytes.NewReader(body))
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		if len(body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		for key, values := range headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, nil, readErr
		}
		if method == http.MethodGet && (resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504) && attempt+1 < attempts {
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(backoff[attempt]):
			}
			continue
		}
		return resp, data, nil
	}
	return nil, nil, fmt.Errorf("request attempts exhausted")
}

func (c *Client) DoRaw(ctx context.Context, method, endpoint string, body []byte, headers http.Header) (app.RawResponse, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return app.RawResponse{}, app.Validationf("invalid raw API endpoint: %v", err)
	}
	if parsed.IsAbs() || parsed.Host != "" || (!strings.HasPrefix(parsed.Path, "/api/") && parsed.Path != "/api") {
		return app.RawResponse{}, app.Validationf("raw API endpoint must be a relative /api path")
	}
	resp, data, err := c.do(ctx, strings.ToUpper(method), parsed.Path, parsed.Query(), body, headers) //nolint:bodyclose // do closes the body after buffering it.
	if err != nil {
		return app.RawResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return app.RawResponse{}, mapAPIError(method, parsed.Path, resp, data)
	}
	return app.RawResponse{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: data}, nil
}
