package youtrack

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func commentClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)
	return client
}

func TestListCommentsRequestsPageAndMapsNullableValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/issues/APP-123/comments", r.URL.Path)
		assert.Equal(t, commentFields, r.URL.Query().Get("fields"))
		assert.Equal(t, "12", r.URL.Query().Get("$skip"))
		assert.Equal(t, "2", r.URL.Query().Get("$top"))
		assert.Len(t, r.URL.Query(), 3)
		_, _ = w.Write([]byte(`[{"id":"4-2","text":null,"author":null,"created":1700000000123,"updated":null,"deleted":true},{"id":"4-1","text":"hello","author":{"id":"u-1","login":"ada","fullName":"Ada Lovelace"},"created":1700000000456,"updated":1700000000789,"deleted":false}]`))
	}))
	defer server.Close()
	got, err := commentClient(t, server).ListComments(context.Background(), "APP-123", app.Page{Offset: 12, Limit: 2})
	require.NoError(t, err)
	updated := time.UnixMilli(1700000000789)
	assert.Equal(t, []domain.Comment{
		{ID: "4-2", Created: time.UnixMilli(1700000000123), Deleted: true},
		{ID: "4-1", Text: new("hello"), Author: &domain.User{ID: "u-1", Login: "ada", FullName: "Ada Lovelace"}, Created: time.UnixMilli(1700000000456), Updated: &updated},
	}, got)
}

func TestListCommentsMapsNotFoundAndEmptyPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/issues/missing/comments" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_description":"missing"}`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	client := commentClient(t, server)
	_, err := client.ListComments(context.Background(), "missing", app.Page{Limit: 1})
	require.Error(t, err)
	assert.Equal(t, app.ErrorNotFound, app.KindOf(err))
	var apiErr *APIError
	assert.True(t, errors.As(err, &apiErr))
	got, err := client.ListComments(context.Background(), "empty", app.Page{Limit: 1})
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestCreateCommentSendsOnlyTextAndDoesNotRetry(t *testing.T) {
	calls := 0
	text := "first line\n\tsecond line\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/issues/APP-123/comments", r.URL.Path)
		assert.Equal(t, commentFields, r.URL.Query().Get("fields"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]any{"text": text}, body)
		_, _ = w.Write([]byte(`{"id":"4-9","text":"first line\n\tsecond line\n","author":null,"created":1700000000000,"updated":null,"deleted":false}`))
	}))
	defer server.Close()
	got, err := commentClient(t, server).CreateComment(context.Background(), "APP-123", text)
	require.NoError(t, err)
	assert.Equal(t, "4-9", got.ID)
	assert.Equal(t, text, *got.Text)
	assert.Equal(t, 1, calls)

	calls = 0
	failed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(http.StatusServiceUnavailable) }))
	defer failed.Close()
	_, err = commentClient(t, failed).CreateComment(context.Background(), "APP-123", "once")
	require.Error(t, err)
	assert.Equal(t, app.ErrorRuntime, app.KindOf(err))
	assert.Equal(t, 1, calls)
}

func TestSoftRemoveCommentPostsDeletedWithoutRetry(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/issues/APP-123/comments/4-17", r.URL.Path)
		assert.Empty(t, r.URL.Query())
		var body map[string]bool
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]bool{"deleted": true}, body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client := commentClient(t, server)
	require.NoError(t, client.SoftRemoveComment(context.Background(), "APP-123", "4-17"))
	require.NoError(t, client.SoftRemoveComment(context.Background(), "APP-123", "4-17"))
	assert.Equal(t, 2, calls)

	for _, status := range []int{http.StatusServiceUnavailable, http.StatusForbidden, http.StatusNotFound} {
		calls = 0
		failed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(status) }))
		err := commentClient(t, failed).SoftRemoveComment(context.Background(), "APP-123", "4-17")
		failed.Close()
		require.Error(t, err)
		assert.Equal(t, 1, calls)
		if status == http.StatusForbidden {
			assert.Equal(t, app.ErrorAuth, app.KindOf(err))
		}
		if status == http.StatusNotFound {
			assert.Equal(t, app.ErrorNotFound, app.KindOf(err))
		}
	}
}
