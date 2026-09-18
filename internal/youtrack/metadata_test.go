package youtrack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func TestMetadataEndpointsMapAndPaginate(t *testing.T) {
	var tagSkips, groupSkips []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/users/me":
			assert.Equal(t, userFields, r.URL.Query().Get("fields"))
			_, _ = w.Write([]byte(`{"id":"u-1","login":"alice","fullName":"Alice"}`))
		case "/api/tags":
			assert.Equal(t, "back", r.URL.Query().Get("query"))
			tagSkips = append(tagSkips, r.URL.Query().Get("$skip"))
			if r.URL.Query().Get("$skip") == "0" {
				_, _ = w.Write([]byte(`[{"id":"t-1","name":"backend"}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		case "/api/groups":
			assert.Equal(t, "plat", r.URL.Query().Get("query"))
			groupSkips = append(groupSkips, r.URL.Query().Get("$skip"))
			if r.URL.Query().Get("$skip") == "0" {
				_, _ = w.Write([]byte(`[{"id":"g-1","name":"Platform"}]`))
				return
			}
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	me, err := client.Me(context.Background())
	require.NoError(t, err)
	tags, err := client.SearchTags(context.Background(), "back")
	require.NoError(t, err)
	groups, err := client.SearchGroups(context.Background(), "plat")
	require.NoError(t, err)

	assert.Equal(t, domain.User{ID: "u-1", Login: "alice", FullName: "Alice"}, me)
	assert.Equal(t, []domain.Tag{{ID: "t-1", Name: "backend"}}, tags)
	assert.Equal(t, []domain.Group{{ID: "g-1", Name: "Platform"}}, groups)
	assert.Equal(t, []string{"0", "1"}, tagSkips)
	assert.Equal(t, []string{"0", "1"}, groupSkips)
}

func TestMetadataEndpointsPropagateAPIFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error_description":"unavailable"}`, http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	_, err = client.SearchTags(context.Background(), "back")
	require.Error(t, err)
	assert.Equal(t, app.ErrorRuntime, app.KindOf(err))
}
