package youtrack

import (
	"context"
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

func TestSearchIssuesRequestsOneProjectedPageAndMapsSummaries(t *testing.T) {
	query := "project: {Tools} #unresolved -State: Done sort by: updated desc"
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/issues", r.URL.Path)
		require.Equal(t, query, r.URL.Query().Get("query"))
		require.Equal(t, "12", r.URL.Query().Get("$skip"))
		require.Equal(t, "2", r.URL.Query().Get("$top"))
		require.Equal(t, issueSummaryFields, r.URL.Query().Get("fields"))
		require.Len(t, r.URL.Query(), 4)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"2-9","idReadable":"TT-9","summary":"later first","project":{"id":"0-1","name":"Tools","shortName":"TT"},"created":1700000000000,"updated":1700000002000,"resolved":1700000003000},
			{"id":"2-1","idReadable":"TT-1","summary":"earlier second","project":{"id":"0-1","name":"Tools","shortName":"TT"},"created":1700000001000,"updated":1700000004000,"resolved":null}
		]`))
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	got, err := client.SearchIssues(context.Background(), query, app.Page{Offset: 12, Limit: 2})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	resolved := time.UnixMilli(1700000003000)
	assert.Equal(t, []domain.IssueSummary{
		{ID: "2-9", IDReadable: "TT-9", Summary: "later first", Project: domain.Project{ID: "0-1", Name: "Tools", ShortName: "TT"}, Created: time.UnixMilli(1700000000000), Updated: time.UnixMilli(1700000002000), Resolved: &resolved},
		{ID: "2-1", IDReadable: "TT-1", Summary: "earlier second", Project: domain.Project{ID: "0-1", Name: "Tools", ShortName: "TT"}, Created: time.UnixMilli(1700000001000), Updated: time.UnixMilli(1700000004000)},
	}, got)
}

func TestSearchIssuesMapsInvalidQueryToRuntimeErrorWithoutRetry(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error_description":"invalid query"}`))
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	_, err = client.SearchIssues(context.Background(), "bad", app.Page{Limit: 50})

	require.Error(t, err)
	assert.Equal(t, app.ErrorRuntime, app.KindOf(err))
	var apiErr *APIError
	assert.True(t, errors.As(err, &apiErr))
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Equal(t, 1, calls)
}

func TestSearchIssuesRetriesTransientGetFailures(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	got, err := client.SearchIssues(context.Background(), "project: TT", app.Page{Limit: 50})

	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, 3, calls)
}
