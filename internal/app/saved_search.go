package app

import (
	"context"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type SavedSearchView struct {
	Search domain.SavedSearch
	Issues []domain.IssueSummary
}

// ViewSavedSearch resolves a visible saved search then executes its stored query.
func (s *Service) ViewSavedSearch(ctx context.Context, ref string, request PageRequest) (SavedSearchView, error) {
	if strings.TrimSpace(ref) == "" {
		return SavedSearchView{}, Validationf("saved search reference must not be blank")
	}
	if err := request.Validate(); err != nil {
		return SavedSearchView{}, err
	}

	search, err := s.savedSearches.GetSavedSearch(ctx, ref)
	if err != nil {
		if KindOf(err) != ErrorNotFound {
			return SavedSearchView{}, err
		}
		search, err = s.resolveSavedSearch(ctx, ref)
		if err != nil {
			return SavedSearchView{}, err
		}
	}
	if strings.TrimSpace(search.Query) == "" {
		return SavedSearchView{}, Validationf("saved search %q has a blank query", search.Name)
	}
	issues, err := s.SearchIssues(ctx, search.Query, request)
	if err != nil {
		return SavedSearchView{}, err
	}
	return SavedSearchView{Search: search, Issues: issues}, nil
}

func (s *Service) resolveSavedSearch(ctx context.Context, ref string) (domain.SavedSearch, error) {
	searches := make([]domain.SavedSearch, 0)
	for offset := 0; ; {
		page, err := s.savedSearches.ListSavedSearches(ctx, Page{Offset: offset, Limit: DefaultPageLimit})
		if err != nil {
			return domain.SavedSearch{}, err
		}
		if len(page) == 0 {
			break
		}
		searches = append(searches, page...)
		offset += len(page)
	}

	exact := make([]domain.SavedSearch, 0)
	for _, search := range searches {
		if search.Name == ref {
			exact = append(exact, search)
		}
	}
	if len(exact) == 1 {
		return exact[0], nil
	}
	if len(exact) > 1 {
		return domain.SavedSearch{}, Ambiguousf("saved search %q is ambiguous", ref)
	}

	folded := make([]domain.SavedSearch, 0)
	for _, search := range searches {
		if strings.EqualFold(search.Name, ref) {
			folded = append(folded, search)
		}
	}
	if len(folded) == 1 {
		return folded[0], nil
	}
	if len(folded) > 1 {
		return domain.SavedSearch{}, Ambiguousf("saved search %q is ambiguous", ref)
	}
	return domain.SavedSearch{}, NotFoundf("saved search %q was not found", ref)
}
