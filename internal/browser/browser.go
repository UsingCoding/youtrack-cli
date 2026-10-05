package browser

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

type Opener interface {
	Open(string) error
}

type DefaultOpener struct{}

func BuildIssueURL(baseURL, readableID string) (string, error) {
	if strings.TrimSpace(readableID) == "" {
		return "", fmt.Errorf("issue readable ID must not be blank")
	}
	base, err := parseBaseURL(baseURL)
	if err != nil {
		return "", err
	}
	return appendPath(base, "issue", readableID), nil
}

func BuildSearchURL(baseURL, query string) (string, error) {
	if strings.TrimSpace(query) == "" {
		return "", fmt.Errorf("search query must not be blank")
	}
	base, err := parseBaseURL(baseURL)
	if err != nil {
		return "", err
	}
	result := appendPath(base, "issues", "")
	u, err := url.Parse(result)
	if err != nil {
		return "", fmt.Errorf("build search URL: %w", err)
	}
	u.RawQuery = url.Values{"q": {query}}.Encode()
	return u.String(), nil
}

func (DefaultOpener) Open(rawURL string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.CommandContext(context.Background(), "open", rawURL) // #nosec G204 -- rawURL is built from a validated service URL and escaped route values.
	case "linux":
		command = exec.CommandContext(context.Background(), "xdg-open", rawURL) // #nosec G204 -- rawURL is built from a validated service URL and escaped route values.
	case "windows":
		command = exec.CommandContext(context.Background(), "rundll32", "url.dll,FileProtocolHandler", rawURL) // #nosec G204 -- rawURL is built from a validated service URL and escaped route values.
	default:
		return fmt.Errorf("opening a browser is unsupported on %s", runtime.GOOS)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("dispatch browser opener: %w", err)
	}
	return nil
}

func parseBaseURL(rawURL string) (*url.URL, error) {
	base, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse service URL: %w", err)
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return nil, fmt.Errorf("service URL must use http or https")
	}
	if base.Host == "" {
		return nil, fmt.Errorf("service URL must include a host")
	}
	if base.User != nil {
		return nil, fmt.Errorf("service URL must not include user information")
	}
	if base.RawQuery != "" {
		return nil, fmt.Errorf("service URL must not include a query")
	}
	if base.Fragment != "" {
		return nil, fmt.Errorf("service URL must not include a fragment")
	}
	return base, nil
}

func appendPath(base *url.URL, route, segment string) string {
	path := strings.TrimRight(base.EscapedPath(), "/") + "/" + route
	if segment != "" {
		path += "/" + escapePathSegment(segment)
	}
	result := *base
	unescaped, err := url.PathUnescape(path)
	if err != nil {
		panic(err)
	}
	result.Path = unescaped
	result.RawPath = path
	return result.String()
}

func escapePathSegment(value string) string {
	escaped := url.PathEscape(value)
	if value == "." || value == ".." {
		return strings.ReplaceAll(escaped, ".", "%2E")
	}
	return escaped
}
