package youtrack

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

func TestMapAPIErrorUsesStatusKindsAndDescriptionFallbacks(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		kind   app.ErrorKind
		want   string
	}{
		{name: "auth", status: http.StatusUnauthorized, body: `{"error_description":"invalid credentials"}`, kind: app.ErrorAuth, want: "invalid credentials"},
		{name: "forbidden", status: http.StatusForbidden, body: `{"description":"denied"}`, kind: app.ErrorAuth, want: "denied"},
		{name: "not found", status: http.StatusNotFound, body: `{"error":"missing"}`, kind: app.ErrorNotFound, want: "missing"},
		{name: "runtime", status: http.StatusConflict, body: "  conflict details  ", kind: app.ErrorRuntime, want: "conflict details"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &http.Response{StatusCode: tt.status, Header: make(http.Header)}
			err := mapAPIError(http.MethodGet, "/api/issues/TT-1", response, []byte(tt.body))
			require.Error(t, err)
			assert.Equal(t, tt.kind, app.KindOf(err))
			assert.Contains(t, err.Error(), tt.want)
		})
	}
	long := strings.Repeat("x", 400)
	err := mapAPIError(http.MethodGet, "/api/issues/TT-1", &http.Response{StatusCode: http.StatusBadRequest, Header: make(http.Header)}, []byte(long))
	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Len(t, apiErr.Description, 303)
	assert.True(t, strings.HasSuffix(apiErr.Description, "…"))
}

func TestAPIErrorDebugStringIsBoundedAndSanitized(t *testing.T) {
	apiErr := &APIError{
		Method:    http.MethodPost,
		Path:      "/api/issues/TT-1",
		RequestID: "request-1",
		Body:      []byte(strings.Repeat("x", 5000)),
	}
	debug := apiErr.DebugString()
	assert.Contains(t, debug, "POST /api/issues/TT-1")
	assert.Contains(t, debug, "request-id=request-1")
	assert.LessOrEqual(t, len(debug), 4096+100)
	assert.NotContains(t, debug, "Authorization")
	assert.NotContains(t, debug, "Bearer ")
}

func TestAPIErrorFallsBackToStatusText(t *testing.T) {
	apiErr := &APIError{StatusCode: http.StatusTeapot}
	assert.Contains(t, apiErr.Error(), http.StatusText(http.StatusTeapot))
}
