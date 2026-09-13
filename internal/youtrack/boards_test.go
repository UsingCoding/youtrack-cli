package youtrack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func TestListIssueBoardsPaginatesAndBuildsSyntheticField(t *testing.T) {
	var skips []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/issues/TT-1" {
			_, _ = w.Write([]byte(`{"id":"2-1","idReadable":"TT-1","summary":"S","description":"","created":1,"updated":2,"project":{"id":"0-1","name":"Tools","shortName":"TT"},"tags":[],"customFields":[]}`))
			return
		}
		require.Equal(t, "/api/issues/TT-1/sprints", r.URL.Path)
		require.Equal(t, issueSprintFields, r.URL.Query().Get("fields"))
		skips = append(skips, r.URL.Query().Get("$skip"))
		switch r.URL.Query().Get("$skip") {
		case "0":
			_, _ = w.Write([]byte(`[{"agile":{"id":"a-1","name":"Alpha"}},{"agile":null}]`))
		case "2":
			_, _ = w.Write([]byte(`[{"agile":{"id":"a-1","name":"Alpha"}},{"agile":{"id":"","name":"Ignored"}},{"agile":{"id":"a-2","name":"Beta"}}]`))
		default:
			_, _ = w.Write([]byte(`[]`))
		}
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)

	issue, err := client.GetIssue(context.Background(), "TT-1")
	require.NoError(t, err)
	require.Equal(t, []string{"0", "2", "5"}, skips)
	require.Len(t, issue.Fields, 1)
	board := issue.Fields[0]
	assert.Equal(t, domain.FieldBoard, board.Kind)
	values := board.Value.(domain.MultiValue).Values
	assert.Equal(t, []domain.FieldValue{domain.EntityValue{ID: "a-1", Name: "Alpha"}, domain.EntityValue{ID: "a-2", Name: "Beta"}}, values)
}

func TestGetIssueEmptyBoardsUsesNonNilValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/issues/TT-1/sprints" {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"2-1","idReadable":"TT-1","summary":"S","description":"","created":1,"updated":2,"project":{"id":"0-1","name":"Tools","shortName":"TT"},"tags":[],"customFields":[]}`))
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)
	issue, err := client.GetIssue(context.Background(), "TT-1")
	require.NoError(t, err)
	values := issue.Fields[0].Value.(domain.MultiValue).Values
	assert.NotNil(t, values)
	assert.Empty(t, values)
}

func TestListBoardsMapsProjectsAndPaginates(t *testing.T) {
	var skips []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/agiles", r.URL.Path)
		require.Equal(t, boardFields, r.URL.Query().Get("fields"))
		skips = append(skips, r.URL.Query().Get("$skip"))
		if r.URL.Query().Get("$skip") == "0" {
			_, _ = w.Write([]byte(`[{"id":"a-1","name":"Alpha","projects":[{"id":"p-1"},{"id":"p-2"}]}]`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)
	boards, err := client.ListBoards(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"0", "1"}, skips)
	assert.Equal(t, []domain.Board{{ID: "a-1", Name: "Alpha", ProjectIDs: []string{"p-1", "p-2"}}}, boards)
}

func TestBoardCommandsValidateThenApply(t *testing.T) {
	var assist, commands int
	var queries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		queries = append(queries, body["query"].(string))
		issues := body["issues"].([]any)
		require.Equal(t, map[string]any{"id": "2-1"}, issues[0])
		switch r.URL.Path {
		case "/api/commands/assist":
			assist++
			_, _ = w.Write([]byte(`{"commands":[{"error":false,"delete":false,"description":"ok"},{"error":false,"delete":false,"description":"ok"}]}`))
		case "/api/commands":
			commands++
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
	require.NoError(t, err)
	add, remove := []domain.Board{{ID: "new", Name: "Platform Board"}}, []domain.Board{{ID: "old", Name: "Legacy Board"}}
	require.NoError(t, client.ValidateIssueBoardChange(context.Background(), "2-1", add, remove))
	require.NoError(t, client.ApplyIssueBoardChange(context.Background(), "2-1", add, remove))
	assert.Equal(t, 1, assist)
	assert.Equal(t, 1, commands)
	assert.Equal(t, []string{"remove Board Legacy Board add Board Platform Board", "remove Board Legacy Board add Board Platform Board"}, queries)
}

func TestBoardCommandValidationRejectsInvalidParsing(t *testing.T) {
	for _, response := range []string{
		`{"commands":[{"error":true,"delete":false,"description":"bad"}]}`,
		`{"commands":[]}`,
		`{"commands":[{"error":false,"delete":true,"description":"delete"}]}`,
	} {
		t.Run(response, func(t *testing.T) {
			commands := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/commands" {
					commands++
				}
				_, _ = w.Write([]byte(response))
			}))
			defer server.Close()
			client, err := NewClient(Options{BaseURL: server.URL, HTTPClient: server.Client()})
			require.NoError(t, err)
			err = client.ValidateIssueBoardChange(context.Background(), "2-1", []domain.Board{{ID: "a", Name: "A"}}, nil)
			assert.Equal(t, app.ErrorValidation, app.KindOf(err))
			assert.Zero(t, commands)
		})
	}
}
