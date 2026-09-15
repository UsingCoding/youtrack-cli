package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type searchScript struct {
	items []domain.IssueSummary
	err   error
}

type searchStoreFake struct {
	scripts []searchScript
	queries []string
	pages   []Page
}

func (f *searchStoreFake) SearchIssues(_ context.Context, query string, page Page) ([]domain.IssueSummary, error) {
	f.queries = append(f.queries, query)
	f.pages = append(f.pages, page)
	if len(f.scripts) == 0 {
		return nil, errors.New("unexpected search call")
	}
	script := f.scripts[0]
	f.scripts = f.scripts[1:]
	return script.items, script.err
}

func searchService(store *searchStoreFake) *Service {
	return NewService(nil, store, nil, nil, nil, nil, nil, nil, nil, nil)
}

func summary(id string) domain.IssueSummary {
	return domain.IssueSummary{ID: id, IDReadable: id, Project: domain.Project{ShortName: "TT"}, Created: time.Unix(0, 0)}
}

func TestSearchIssuesForwardsOpaqueQueryAndPreservesOrder(t *testing.T) {
	store := &searchStoreFake{scripts: []searchScript{{items: []domain.IssueSummary{summary("TT-3"), summary("TT-1")}}}}
	query := "project: {Tools} #unresolved -State: Done sort by: updated desc"

	got, err := searchService(store).SearchIssues(context.Background(), query, PageRequest{Limit: new(2)})

	require.NoError(t, err)
	assert.Equal(t, []string{query}, store.queries)
	assert.Equal(t, []Page{{Offset: 0, Limit: 2}}, store.pages)
	assert.Equal(t, []domain.IssueSummary{summary("TT-3"), summary("TT-1")}, got)
}

func TestSearchIssuesRejectsInvalidInputBeforeStoreCall(t *testing.T) {
	cases := []struct {
		name    string
		query   string
		request PageRequest
		message string
	}{
		{name: "blank query", query: " \t", message: "search query must not be blank"},
		{name: "negative offset", query: "project: TT", request: PageRequest{Offset: -1}, message: "offset must be non-negative"},
		{name: "all with explicit limit", query: "project: TT", request: PageRequest{All: true, Limit: new(50)}, message: "--all and --limit are mutually exclusive"},
		{name: "zero limit", query: "project: TT", request: PageRequest{Limit: new(0)}, message: "limit must be positive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &searchStoreFake{}
			got, err := searchService(store).SearchIssues(context.Background(), tc.query, tc.request)
			assert.Nil(t, got)
			require.Error(t, err)
			assert.Equal(t, ErrorValidation, KindOf(err))
			assert.Equal(t, tc.message, err.Error())
			assert.Empty(t, store.pages)
		})
	}
}

func TestSearchIssuesPaginatesFiniteRequestsByActualResults(t *testing.T) {
	store := &searchStoreFake{scripts: []searchScript{
		{items: []domain.IssueSummary{summary("TT-4"), summary("TT-2")}},
		{items: []domain.IssueSummary{summary("TT-9")}},
		{items: []domain.IssueSummary{}},
	}}
	query := "project: TT sort by: created desc"

	got, err := searchService(store).SearchIssues(context.Background(), query, PageRequest{Offset: 7, Limit: new(5)})

	require.NoError(t, err)
	assert.Equal(t, []string{query, query, query}, store.queries)
	assert.Equal(t, []Page{{Offset: 7, Limit: 5}, {Offset: 9, Limit: 3}, {Offset: 10, Limit: 2}}, store.pages)
	assert.Equal(t, []domain.IssueSummary{summary("TT-4"), summary("TT-2"), summary("TT-9")}, got)
}

func TestSearchIssuesUsesDefaultFiniteLimitAndTruncatesOversizedPage(t *testing.T) {
	items := make([]domain.IssueSummary, 51)
	for i := range items {
		items[i] = summary(string(rune('a' + i%26)))
	}
	store := &searchStoreFake{scripts: []searchScript{{items: items}}}

	got, err := searchService(store).SearchIssues(context.Background(), "project: TT", PageRequest{})

	require.NoError(t, err)
	assert.Equal(t, []Page{{Offset: 0, Limit: DefaultPageLimit}}, store.pages)
	assert.Len(t, got, DefaultPageLimit)
}

func TestSearchIssuesAllContinuesFromActualPageLength(t *testing.T) {
	store := &searchStoreFake{scripts: []searchScript{
		{items: []domain.IssueSummary{summary("TT-8"), summary("TT-3")}},
		{items: []domain.IssueSummary{summary("TT-1")}},
		{items: []domain.IssueSummary{}},
	}}

	got, err := searchService(store).SearchIssues(context.Background(), "-State: Done", PageRequest{Offset: 11, All: true})

	require.NoError(t, err)
	assert.Equal(t, []Page{{Offset: 11, Limit: DefaultPageLimit}, {Offset: 13, Limit: DefaultPageLimit}, {Offset: 14, Limit: DefaultPageLimit}}, store.pages)
	assert.Equal(t, []domain.IssueSummary{summary("TT-8"), summary("TT-3"), summary("TT-1")}, got)
}

func TestSearchIssuesReturnsNonNilEmptySliceAndPropagatesStoreError(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		store := &searchStoreFake{scripts: []searchScript{{items: []domain.IssueSummary{}}}}
		got, err := searchService(store).SearchIssues(context.Background(), "project: missing", PageRequest{})
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
	t.Run("error", func(t *testing.T) {
		want := errors.New("server failed")
		store := &searchStoreFake{scripts: []searchScript{{err: want}}}
		got, err := searchService(store).SearchIssues(context.Background(), "project: TT", PageRequest{})
		assert.Nil(t, got)
		assert.ErrorIs(t, err, want)
	})
}
