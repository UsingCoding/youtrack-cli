package youtrack

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func TestGetSavedSearchRequestsProjectionAndMapsOwner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/savedQueries/51-33", r.URL.Path)
		assert.Equal(t, savedQueryFields, r.URL.Query().Get("fields"))
		assert.Len(t, r.URL.Query(), 1)
		_, _ = w.Write([]byte(`{"id":"51-33","name":"Mine","query":"project: APP","owner":{"id":"1-2","login":"ada","fullName":"Ada Lovelace"}}`))
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	got, err := client.GetSavedSearch(context.Background(), "51-33")

	require.NoError(t, err)
	assert.Equal(t, domain.SavedSearch{ID: "51-33", Name: "Mine", Query: "project: APP", Owner: &domain.User{ID: "1-2", Login: "ada", FullName: "Ada Lovelace"}}, got)
}

func TestListSavedSearchesRequestsOnePageAndMapsNulls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/savedQueries", r.URL.Path)
		assert.Equal(t, savedQueryFields, r.URL.Query().Get("fields"))
		assert.Equal(t, "12", r.URL.Query().Get("$skip"))
		assert.Equal(t, "2", r.URL.Query().Get("$top"))
		assert.Len(t, r.URL.Query(), 3)
		_, _ = w.Write([]byte(`[
			{"id":"2","name":"second","query":null,"owner":null},
			{"id":"1","name":"first","query":"project: APP","owner":{"id":"u","login":"lin","fullName":"Lin User"}}
		]`))
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	got, err := client.ListSavedSearches(context.Background(), app.Page{Offset: 12, Limit: 2})

	require.NoError(t, err)
	assert.Equal(t, []domain.SavedSearch{
		{ID: "2", Name: "second"},
		{ID: "1", Name: "first", Query: "project: APP", Owner: &domain.User{ID: "u", Login: "lin", FullName: "Lin User"}},
	}, got)
}

func TestSavedSearchAdapterMapsNotFoundAndNonNilEmptyPage(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_description":"missing"}`))
		}))
		defer server.Close()
		client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
		require.NoError(t, err)
		_, err = client.GetSavedSearch(context.Background(), "missing")
		require.Error(t, err)
		assert.Equal(t, app.ErrorNotFound, app.KindOf(err))
		var apiErr *APIError
		assert.True(t, errors.As(err, &apiErr))
	})
	t.Run("empty page", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`[]`))
		}))
		defer server.Close()
		client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
		require.NoError(t, err)
		got, err := client.ListSavedSearches(context.Background(), app.Page{Limit: 50})
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}
