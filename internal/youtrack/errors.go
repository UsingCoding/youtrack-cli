package youtrack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

type APIError struct {
	StatusCode  int
	Method      string
	Path        string
	ErrorCode   string
	Description string
	RequestID   string
	Body        []byte
}

func (e *APIError) Error() string {
	message := e.Description
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	return fmt.Sprintf("YouTrack returned %d %s: %s", e.StatusCode, http.StatusText(e.StatusCode), message)
}

func (e *APIError) DebugString() string {
	body := strings.TrimSpace(string(e.Body))
	if len(body) > 4096 {
		body = body[:4096] + "…"
	}
	parts := []string{fmt.Sprintf("%s %s", e.Method, e.Path)}
	if e.RequestID != "" {
		parts = append(parts, "request-id="+e.RequestID)
	}
	if body != "" {
		parts = append(parts, "body="+body)
	}
	return strings.Join(parts, "\n")
}

func mapAPIError(method, path string, resp *http.Response, body []byte) error {
	var data struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		Description      string `json:"description"`
	}
	_ = json.Unmarshal(body, &data)
	desc := data.ErrorDescription
	if desc == "" {
		desc = data.Description
	}
	if desc == "" {
		desc = data.Error
	}
	if desc == "" {
		desc = strings.TrimSpace(string(body))
		if len(desc) > 300 {
			desc = desc[:300] + "…"
		}
	}
	apiErr := &APIError{
		StatusCode: resp.StatusCode, Method: method, Path: path,
		ErrorCode: data.Error, Description: desc, RequestID: resp.Header.Get("X-Request-ID"), Body: body,
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &app.Error{Kind: app.ErrorAuth, Message: apiErr.Error(), Err: apiErr}
	case http.StatusNotFound:
		return &app.Error{Kind: app.ErrorNotFound, Message: apiErr.Error(), Err: apiErr}
	default:
		return &app.Error{Kind: app.ErrorRuntime, Message: apiErr.Error(), Err: apiErr}
	}
}
