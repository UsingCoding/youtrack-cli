package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

func TestAPICommandForwardsBodyHeadersAndFormatsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/test", r.URL.Path)
		assert.Equal(t, "one", r.Header.Get("X-Test"))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	out := &bytes.Buffer{}
	err := issueSearchRoot(out, server).Run(context.Background(), []string{"youtrack", "api", "/api/test", "--method", "POST", "--data", `{"x":1}`, "--header", "X-Test: one", "--url", server.URL, "--token", "secret", "--json"})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "\"ok\": true")
}

func TestAPICommandRejectsMalformedAndConflictingRequestsBeforeHTTP(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	for _, args := range [][]string{
		{"youtrack", "api", "/api/test", "--header", "invalid"},
		{"youtrack", "api", "/api/test", "--header", "Authorization: custom"},
		{"youtrack", "api", "/api/test", "--data", "x", "--data-file", "file"},
	} {
		err := issueSearchRoot(&bytes.Buffer{}, server).Run(context.Background(), args)
		require.Error(t, err)
		assert.Equal(t, app.ErrorValidation, app.KindOf(err))
	}
	assert.Zero(t, calls)
}
