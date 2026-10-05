package youtrack

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func TestLinkAdapterMapsTypesPagesAndAllDirectionPaths(t *testing.T) {
	var postBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /api/issueLinkTypes":
			assert.Equal(t, linkTypeFields, r.URL.Query().Get("fields"))
			assert.Equal(t, "3", r.URL.Query().Get("$skip"))
			assert.Equal(t, "2", r.URL.Query().Get("$top"))
			_, _ = w.Write([]byte(`[{"id":"parent","name":"Parent","directed":true,"aggregation":true,"sourceToTarget":"parent for","targetToSource":"subtask of"},{"id":"relates","name":"Relates","directed":false,"aggregation":false,"sourceToTarget":"relates to","targetToSource":""}]`))
		case "GET /api/issues/2-1/links/parents/issues", "GET /api/issues/2-1/links/parentt/issues", "GET /api/issues/2-1/links/relates/issues":
			assert.Equal(t, issueSummaryFields, r.URL.Query().Get("fields"))
			assert.Equal(t, "3", r.URL.Query().Get("$skip"))
			assert.Equal(t, "2", r.URL.Query().Get("$top"))
			_, _ = w.Write([]byte(`[{"id":"2-3","idReadable":"APP-3","summary":"Third","project":{"id":"0-1","name":"App","shortName":"APP"},"created":1,"updated":2,"resolved":null},{"id":"2-2","idReadable":"APP-2","summary":"Second","project":{"id":"0-1","name":"App","shortName":"APP"},"created":3,"updated":4,"resolved":5}]`))
		case "POST /api/issues/2-1/links/parentt/issues":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			postBody = string(body)
			w.WriteHeader(http.StatusNoContent)
		case "DELETE /api/issues/2-1/links/parentt/issues/2-2":
			assert.Equal(t, int64(0), r.ContentLength)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	types, err := client.ListLinkTypes(context.Background(), app.Page{Offset: 3, Limit: 2})
	require.NoError(t, err)
	require.Len(t, types, 2)
	assert.Equal(t, domain.LinkType{ID: "parent", Name: "Parent", Directed: true, Aggregation: true, Outward: "parent for", Inward: "subtask of"}, types[0])

	outward := domain.LinkRelation{Type: types[0], Direction: domain.LinkDirectionOutward}
	inward := domain.LinkRelation{Type: types[0], Direction: domain.LinkDirectionInward}
	undirected := domain.LinkRelation{Type: types[1], Direction: domain.LinkDirectionUndirected}
	for _, relation := range []domain.LinkRelation{outward, inward, undirected} {
		issues, err := client.ListLinkedIssues(context.Background(), "2-1", relation, app.Page{Offset: 3, Limit: 2})
		require.NoError(t, err)
		require.Len(t, issues, 2)
		assert.Equal(t, []string{"2-3", "2-2"}, []string{issues[0].ID, issues[1].ID})
		assert.Equal(t, time.UnixMilli(5), *issues[1].Resolved)
	}
	require.NoError(t, client.AddIssueLink(context.Background(), "2-1", inward, "2-2"))
	assert.Equal(t, `{"id":"2-2"}`, postBody)
	require.NoError(t, client.RemoveIssueLink(context.Background(), "2-1", inward, "2-2"))
}

func TestGetIssueIdentityAndLinkMutationsFailOnce(t *testing.T) {
	postCalls, deleteCalls := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /api/issues/APP-1":
			assert.Equal(t, issueIdentityFields, r.URL.Query().Get("fields"))
			_, _ = w.Write([]byte(`{"id":"2-1","idReadable":"APP-1"}`))
		case "POST /api/issues/2-1/links/parentt/issues":
			postCalls++
			http.Error(w, `{"error":"no"}`, http.StatusServiceUnavailable)
		case "DELETE /api/issues/2-1/links/parentt/issues/2-2":
			deleteCalls++
			http.Error(w, `{"error":"no"}`, http.StatusServiceUnavailable)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)
	identity, err := client.GetIssueIdentity(context.Background(), "APP-1")
	require.NoError(t, err)
	assert.Equal(t, domain.IssueIdentity{ID: "2-1", IDReadable: "APP-1"}, identity)
	relation := domain.LinkRelation{Type: domain.LinkType{ID: "parent", Name: "Parent", Directed: true}, Direction: domain.LinkDirectionInward}
	assert.Error(t, client.AddIssueLink(context.Background(), "2-1", relation, "2-2"))
	assert.Error(t, client.RemoveIssueLink(context.Background(), "2-1", relation, "2-2"))
	assert.Equal(t, 1, postCalls)
	assert.Equal(t, 1, deleteCalls)
}

func TestLinkIDRejectsImpossibleRelations(t *testing.T) {
	_, err := linkID(domain.LinkRelation{Type: domain.LinkType{ID: "parent", Directed: true}, Direction: domain.LinkDirectionUndirected})
	assert.Equal(t, app.ErrorValidation, app.KindOf(err))
	_, err = linkID(domain.LinkRelation{Type: domain.LinkType{ID: "relates"}, Direction: domain.LinkDirectionOutward})
	assert.Equal(t, app.ErrorValidation, app.KindOf(err))
}
