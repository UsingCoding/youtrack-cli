package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type savedSearchScript struct {
	items []domain.SavedSearch
	err   error
}

type savedSearchStoreFake struct {
	direct    domain.SavedSearch
	directErr error
	list      []savedSearchScript
	refs      []string
	pages     []Page
}

func (f *savedSearchStoreFake) GetSavedSearch(_ context.Context, ref string) (domain.SavedSearch, error) {
	f.refs = append(f.refs, ref)
	return f.direct, f.directErr
}

func (f *savedSearchStoreFake) ListSavedSearches(_ context.Context, page Page) ([]domain.SavedSearch, error) {
	f.pages = append(f.pages, page)
	if len(f.list) == 0 {
		return nil, errors.New("unexpected saved-search list call")
	}
	script := f.list[0]
	f.list = f.list[1:]
	return script.items, script.err
}

func savedSearchService(saved *savedSearchStoreFake, issues *searchStoreFake) *Service {
	return NewService(nil, issues, saved, nil, nil, nil, nil, nil, nil, nil)
}

func TestViewSavedSearchUsesDirectLookupAndStoredQuery(t *testing.T) {
	stored := "project: {Tools} #Unresolved -State: Done sort by: updated desc"
	saved := &savedSearchStoreFake{direct: domain.SavedSearch{ID: "51-33", Name: "Mine", Query: stored}}
	issues := &searchStoreFake{scripts: []searchScript{{items: []domain.IssueSummary{summary("TT-2"), summary("TT-1")}}}}

	got, err := savedSearchService(saved, issues).ViewSavedSearch(context.Background(), "51-33", PageRequest{Offset: 7, Limit: new(2)})

	require.NoError(t, err)
	assert.Equal(t, []string{"51-33"}, saved.refs)
	assert.Empty(t, saved.pages)
	assert.Equal(t, []string{stored}, issues.queries)
	assert.Equal(t, []Page{{Offset: 7, Limit: 2}}, issues.pages)
	assert.Equal(t, saved.direct, got.Search)
	assert.Equal(t, []domain.IssueSummary{summary("TT-2"), summary("TT-1")}, got.Issues)
}

func TestViewSavedSearchFallsBackToCompleteVisibleList(t *testing.T) {
	first := make([]domain.SavedSearch, 42)
	for i := range first {
		first[i] = domain.SavedSearch{ID: "first", Name: "other", Query: "project: OTHER"}
	}
	query := "project: APP"
	saved := &savedSearchStoreFake{
		directErr: NotFoundf("missing"),
		list: []savedSearchScript{
			{items: first},
			{items: []domain.SavedSearch{{ID: "target", Name: "Release blockers", Query: query}}},
			{items: []domain.SavedSearch{}},
		},
	}
	issues := &searchStoreFake{scripts: []searchScript{{items: []domain.IssueSummary{}}}}

	got, err := savedSearchService(saved, issues).ViewSavedSearch(context.Background(), "Release blockers", PageRequest{})

	require.NoError(t, err)
	assert.Equal(t, "target", got.Search.ID)
	assert.Equal(t, []Page{{Offset: 0, Limit: 50}, {Offset: 42, Limit: 50}, {Offset: 43, Limit: 50}}, saved.pages)
	assert.Equal(t, []string{query}, issues.queries)
	assert.NotNil(t, got.Issues)
}

func TestViewSavedSearchResolutionRules(t *testing.T) {
	cases := []struct {
		name    string
		ref     string
		items   []domain.SavedSearch
		wantID  string
		kind    ErrorKind
		message string
	}{
		{name: "unique exact", ref: "Mine", items: []domain.SavedSearch{{ID: "1", Name: "Mine", Query: "q"}}, wantID: "1"},
		{name: "unique folded", ref: "mine", items: []domain.SavedSearch{{ID: "1", Name: "Mine", Query: "q"}}, wantID: "1"},
		{name: "exact wins over folded", ref: "Mine", items: []domain.SavedSearch{{ID: "1", Name: "Mine", Query: "q"}, {ID: "2", Name: "MINE", Query: "other"}}, wantID: "1"},
		{name: "duplicate exact", ref: "Mine", items: []domain.SavedSearch{{Name: "Mine"}, {Name: "Mine"}}, kind: ErrorAmbiguous, message: `saved search "Mine" is ambiguous`},
		{name: "duplicate folded", ref: "mine", items: []domain.SavedSearch{{Name: "Mine"}, {Name: "MINE"}}, kind: ErrorAmbiguous, message: `saved search "mine" is ambiguous`},
		{name: "missing", ref: "Mine", items: []domain.SavedSearch{{Name: "Other"}}, kind: ErrorNotFound, message: `saved search "Mine" was not found`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saved := &savedSearchStoreFake{directErr: NotFoundf("missing"), list: []savedSearchScript{{items: tc.items}, {items: []domain.SavedSearch{}}}}
			issues := &searchStoreFake{scripts: []searchScript{{items: []domain.IssueSummary{}}}}
			got, err := savedSearchService(saved, issues).ViewSavedSearch(context.Background(), tc.ref, PageRequest{})
			if tc.kind != ErrorRuntime {
				require.Error(t, err)
				assert.Equal(t, tc.kind, KindOf(err))
				assert.Equal(t, tc.message, err.Error())
				assert.Empty(t, issues.pages)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantID, got.Search.ID)
		})
	}
}

func TestViewSavedSearchRejectsInvalidInputBeforeStores(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ref     string
		request PageRequest
		message string
	}{
		{name: "blank reference", ref: " \t", message: "saved search reference must not be blank"},
		{name: "negative offset", ref: "Mine", request: PageRequest{Offset: -1}, message: "offset must be non-negative"},
		{name: "zero limit", ref: "Mine", request: PageRequest{Limit: new(0)}, message: "limit must be positive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			saved := &savedSearchStoreFake{}
			issues := &searchStoreFake{}
			_, err := savedSearchService(saved, issues).ViewSavedSearch(context.Background(), tc.ref, tc.request)
			require.Error(t, err)
			assert.Equal(t, tc.message, err.Error())
			assert.Empty(t, saved.refs)
			assert.Empty(t, saved.pages)
			assert.Empty(t, issues.pages)
		})
	}
}

func TestViewSavedSearchPropagatesDirectErrorAndRejectsBlankQuery(t *testing.T) {
	t.Run("direct error", func(t *testing.T) {
		want := errors.New("server failed")
		saved := &savedSearchStoreFake{directErr: want}
		_, err := savedSearchService(saved, &searchStoreFake{}).ViewSavedSearch(context.Background(), "Mine", PageRequest{})
		require.ErrorIs(t, err, want)
		assert.Empty(t, saved.pages)
	})
	for _, query := range []string{"", " \t"} {
		t.Run("blank query", func(t *testing.T) {
			saved := &savedSearchStoreFake{direct: domain.SavedSearch{Name: "Mine", Query: query}}
			issues := &searchStoreFake{}
			_, err := savedSearchService(saved, issues).ViewSavedSearch(context.Background(), "Mine", PageRequest{})
			require.Error(t, err)
			assert.Equal(t, ErrorValidation, KindOf(err))
			assert.Empty(t, issues.pages)
		})
	}
}
