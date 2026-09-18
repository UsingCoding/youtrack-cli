package youtrack

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

func TestNewClientValidatesURLAndSetsDefaults(t *testing.T) {
	for _, baseURL := range []string{"", "ftp://youtrack.example", "https://"} {
		_, err := NewClient(Options{BaseURL: baseURL})
		require.Error(t, err)
	}
	client, err := NewClient(Options{BaseURL: "https://youtrack.example/base/?ignored=true#fragment"})
	require.NoError(t, err)
	assert.Equal(t, "https://youtrack.example/base", client.baseURL.String())
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
	assert.Equal(t, "youtrack-cli/dev", client.userAgent)
}

func TestRawAPIRejectsUnsafeEndpointAndForwardsRequest(t *testing.T) {
	var gotMethod, gotPath string
	var gotHeaders http.Header
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotHeaders = r.Header.Clone()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		gotBody = string(body)
		w.Header().Set("X-Result", "ok")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"created":true}`))
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, Token: "test-token", HTTPClient: server.Client(), UserAgent: "test-agent"})
	require.NoError(t, err)
	for _, endpoint := range []string{"https://example.com/api/issues", "issues", "%"} {
		_, err := client.DoRaw(context.Background(), http.MethodGet, endpoint, nil, nil)
		require.Error(t, err)
		assert.Equal(t, app.ErrorValidation, app.KindOf(err))
	}

	response, err := client.DoRaw(context.Background(), http.MethodPost, "/api/issues?fields=id", []byte(`{"summary":"New"}`), http.Header{"X-Test": {"one", "two"}})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, response.StatusCode)
	assert.Equal(t, []byte(`{"created":true}`), response.Body)
	assert.Equal(t, "ok", response.Header.Get("X-Result"))
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/api/issues", gotPath)
	assert.Equal(t, "application/json", gotHeaders.Get("Content-Type"))
	assert.Equal(t, "test-agent", gotHeaders.Get("User-Agent"))
	assert.Len(t, gotHeaders.Values("X-Test"), 2)
	assert.Equal(t, `{"summary":"New"}`, gotBody)
}

func TestGetRetryStopsWhenContextIsCancelled(t *testing.T) {
	calls := 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		cancel()
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	_, err = client.DoRaw(ctx, http.MethodGet, "/api/issues", nil, nil)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
}

func TestRawAPIErrorRetainsApplicationKind(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error_description":"missing"}`, http.StatusNotFound)
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	_, err = client.DoRaw(context.Background(), http.MethodGet, "/api/issues/missing", nil, nil)
	require.Error(t, err)
	assert.Equal(t, app.ErrorNotFound, app.KindOf(err))
	var apiErr *APIError
	assert.True(t, errors.As(err, &apiErr))
	assert.Equal(t, "/api/issues/missing", apiErr.Path)
}
