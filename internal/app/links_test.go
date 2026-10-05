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

type linkIdentityFake struct {
	identities map[domain.IssueRef]domain.IssueIdentity
	calls      []domain.IssueRef
	err        error
}

func (f *linkIdentityFake) GetIssueIdentity(_ context.Context, ref domain.IssueRef) (domain.IssueIdentity, error) {
	f.calls = append(f.calls, ref)
	if f.err != nil {
		return domain.IssueIdentity{}, f.err
	}
	return f.identities[ref], nil
}

type linkStoreFake struct {
	types       []domain.LinkType
	linked      []domain.IssueSummary
	typePages   []Page
	linkedPages []Page
	sources     []domain.IssueRef
	relations   []domain.LinkRelation
	targets     []domain.IssueRef
	addCalls    int
	removeCalls int
	typeErr     error
	linkedErr   error
	addErr      error
	removeErr   error
}

func (f *linkStoreFake) ListLinkTypes(_ context.Context, page Page) ([]domain.LinkType, error) {
	f.typePages = append(f.typePages, page)
	if f.typeErr != nil {
		return nil, f.typeErr
	}
	if page.Offset >= len(f.types) {
		return []domain.LinkType{}, nil
	}
	end := page.Offset + page.Limit
	if end > len(f.types) {
		end = len(f.types)
	}
	return append([]domain.LinkType(nil), f.types[page.Offset:end]...), nil
}

func (f *linkStoreFake) ListLinkedIssues(_ context.Context, source domain.IssueRef, relation domain.LinkRelation, page Page) ([]domain.IssueSummary, error) {
	f.sources = append(f.sources, source)
	f.relations = append(f.relations, relation)
	f.linkedPages = append(f.linkedPages, page)
	if f.linkedErr != nil {
		return nil, f.linkedErr
	}
	if page.Offset >= len(f.linked) {
		return []domain.IssueSummary{}, nil
	}
	end := page.Offset + page.Limit
	if end > len(f.linked) {
		end = len(f.linked)
	}
	return append([]domain.IssueSummary(nil), f.linked[page.Offset:end]...), nil
}

func (f *linkStoreFake) AddIssueLink(_ context.Context, source domain.IssueRef, relation domain.LinkRelation, target domain.IssueRef) error {
	f.addCalls++
	f.sources = append(f.sources, source)
	f.relations = append(f.relations, relation)
	f.targets = append(f.targets, target)
	return f.addErr
}

func (f *linkStoreFake) RemoveIssueLink(_ context.Context, source domain.IssueRef, relation domain.LinkRelation, target domain.IssueRef) error {
	f.removeCalls++
	f.sources = append(f.sources, source)
	f.relations = append(f.relations, relation)
	f.targets = append(f.targets, target)
	return f.removeErr
}

func linkService(types []domain.LinkType, linked []domain.IssueSummary) (*Service, *linkIdentityFake, *linkStoreFake) {
	identities := &linkIdentityFake{identities: map[domain.IssueRef]domain.IssueIdentity{
		"APP-1":       {ID: "2-1", IDReadable: "APP-1"},
		"APP-1-alias": {ID: "2-1", IDReadable: "APP-1"},
		"APP-2":       {ID: "2-2", IDReadable: "APP-2"},
	}}
	store := &linkStoreFake{types: types, linked: linked}
	return NewService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, identities, store), identities, store
}

func linkSummary(id string) domain.IssueSummary {
	return domain.IssueSummary{ID: id, IDReadable: id, Created: time.Unix(0, 0)}
}

func TestResolveLinkRelationUsesPrecedenceAndValidatesDirection(t *testing.T) {
	directed := domain.LinkType{ID: "parent", Name: "Parent", Directed: true, Outward: "parent for", Inward: "subtask of"}
	undirected := domain.LinkType{ID: "relates", Name: "Relates", Outward: "relates to"}
	service, _, _ := linkService([]domain.LinkType{directed, {ID: "same", Name: "PARENT", Directed: true, Outward: "duplicates", Inward: "is duplicated by"}, undirected}, nil)

	relation, err := service.resolveLinkRelation(context.Background(), LinkRelationReference{Type: "parent", Direction: new("inward")})
	require.NoError(t, err)
	assert.Equal(t, "parent", relation.Type.ID)
	assert.Equal(t, domain.LinkDirectionInward, relation.Direction)
	assert.Equal(t, "subtask of", relation.Label)

	relation, err = service.resolveLinkRelation(context.Background(), LinkRelationReference{Type: "subtask of"})
	require.NoError(t, err)
	assert.Equal(t, domain.LinkDirectionInward, relation.Direction)
	_, err = service.resolveLinkRelation(context.Background(), LinkRelationReference{Type: "Parent", Direction: new("outward")})
	require.NoError(t, err)
	_, err = service.resolveLinkRelation(context.Background(), LinkRelationReference{Type: "PARENT", Direction: new("outward")})
	require.NoError(t, err)
	_, err = service.resolveLinkRelation(context.Background(), LinkRelationReference{Type: "parent"})
	assert.Equal(t, ErrorValidation, KindOf(err))
	_, err = service.resolveLinkRelation(context.Background(), LinkRelationReference{Type: "relates", Direction: new("outward")})
	assert.Equal(t, ErrorValidation, KindOf(err))
	_, err = service.resolveLinkRelation(context.Background(), LinkRelationReference{Type: "subtask of", Direction: new("outward")})
	assert.Equal(t, ErrorValidation, KindOf(err))
}

func TestResolveLinkRelationReportsLaterPageAmbiguity(t *testing.T) {
	types := make([]domain.LinkType, 43)
	for i := range 42 {
		types[i] = domain.LinkType{ID: string(rune('a' + i)), Name: "other"}
	}
	types[42] = domain.LinkType{ID: "later", Name: "Same", Directed: true, Outward: "out", Inward: "in"}
	types = append(types, domain.LinkType{ID: "later-two", Name: "Same", Directed: true, Outward: "out-two", Inward: "in-two"})
	service, _, store := linkService(types, nil)
	_, err := service.resolveLinkRelation(context.Background(), LinkRelationReference{Type: "Same", Direction: new("outward")})
	assert.Equal(t, ErrorAmbiguous, KindOf(err))
	assert.Equal(t, []Page{{Offset: 0, Limit: 50}, {Offset: 44, Limit: 50}}, store.typePages)
}

func TestListIssueLinksUsesCanonicalIdentityAndShortPagePaging(t *testing.T) {
	typeRelation := domain.LinkType{ID: "relates", Name: "Relates", Outward: "relates to"}
	linked := make([]domain.IssueSummary, 43)
	for i := range linked {
		linked[i] = linkSummary(string(rune('a' + i)))
	}
	service, identities, store := linkService([]domain.LinkType{typeRelation}, linked)
	got, err := service.ListIssueLinks(context.Background(), "APP-1", LinkRelationReference{Type: "relates"}, PageRequest{All: true})
	require.NoError(t, err)
	assert.Equal(t, "APP-1", got.IssueID)
	assert.Equal(t, linked, got.Issues)
	assert.Equal(t, []domain.IssueRef{"APP-1"}, identities.calls)
	assert.Equal(t, []Page{{Offset: 0, Limit: 50}, {Offset: 43, Limit: 50}}, store.linkedPages)
	assert.Equal(t, domain.IssueRef("2-1"), store.sources[0])
}

func TestIssueLinkChangesDetectLaterPageNoopsAndWriteOnce(t *testing.T) {
	relation := domain.LinkType{ID: "parent", Name: "Parent", Directed: true, Outward: "parent for", Inward: "subtask of"}
	linked := make([]domain.IssueSummary, 43)
	for i := range linked {
		linked[i] = linkSummary("other")
	}
	linked[42] = linkSummary("2-2")
	service, _, store := linkService([]domain.LinkType{relation}, linked)

	change, err := service.AddIssueLink(context.Background(), "APP-1", "APP-2", LinkRelationReference{Type: "subtask of"})
	require.NoError(t, err)
	assert.True(t, change.Present)
	assert.False(t, change.Changed)
	assert.Zero(t, store.addCalls)

	store.linked = nil
	change, err = service.AddIssueLink(context.Background(), "APP-1", "APP-2", LinkRelationReference{Type: "parent", Direction: new("inward")})
	require.NoError(t, err)
	assert.True(t, change.Present)
	assert.True(t, change.Changed)
	assert.Equal(t, 1, store.addCalls)
	assert.Equal(t, domain.IssueRef("2-1"), store.sources[len(store.sources)-1])
	assert.Equal(t, domain.IssueRef("2-2"), store.targets[0])

	change, err = service.RemoveIssueLink(context.Background(), "APP-1", "APP-2", LinkRelationReference{Type: "parent", Direction: new("inward")})
	require.NoError(t, err)
	assert.False(t, change.Present)
	assert.False(t, change.Changed)
	assert.Zero(t, store.removeCalls)
}

func TestIssueLinkRejectsInvalidInputBeforeStoreCalls(t *testing.T) {
	service, identities, store := linkService([]domain.LinkType{{ID: "relates", Name: "Relates", Outward: "relates to"}}, nil)
	_, err := service.AddIssueLink(context.Background(), "", "APP-2", LinkRelationReference{Type: "relates"})
	assert.Equal(t, ErrorValidation, KindOf(err))
	_, err = service.RemoveIssueLink(context.Background(), "APP-1", "APP-1-alias", LinkRelationReference{Type: "relates"})
	assert.Equal(t, ErrorValidation, KindOf(err))
	assert.Equal(t, 2, len(identities.calls))
	assert.Zero(t, store.addCalls)
	assert.Zero(t, store.removeCalls)
	typeCalls := len(store.typePages)
	_, err = service.ListIssueLinks(context.Background(), "APP-1", LinkRelationReference{Type: ""}, PageRequest{})
	assert.Equal(t, ErrorValidation, KindOf(err))
	assert.Equal(t, typeCalls, len(store.typePages))
}

func TestLinkPublicListsValidateAndUseRequestedPage(t *testing.T) {
	linkType := domain.LinkType{ID: "relates", Name: "Relates", Outward: "relates to"}
	service, _, store := linkService([]domain.LinkType{linkType}, []domain.IssueSummary{linkSummary("2-2")})

	types, err := service.ListLinkTypes(context.Background(), PageRequest{Offset: 0, Limit: new(1)})
	require.NoError(t, err)
	assert.Equal(t, []domain.LinkType{linkType}, types)
	assert.Equal(t, []Page{{Offset: 0, Limit: 1}}, store.typePages)
	_, err = service.ListLinkTypes(context.Background(), PageRequest{Offset: -1})
	assert.Equal(t, ErrorValidation, KindOf(err))
	_, err = service.ListIssueLinks(context.Background(), "", LinkRelationReference{Type: "relates"}, PageRequest{})
	assert.Equal(t, ErrorValidation, KindOf(err))
	_, err = service.ListIssueLinks(context.Background(), "APP-1", LinkRelationReference{Type: "relates"}, PageRequest{Offset: -1})
	assert.Equal(t, ErrorValidation, KindOf(err))
}

func TestIssueLinkRemovalAndRelationLookupFailures(t *testing.T) {
	parent := domain.LinkType{ID: "parent", Name: "Parent", Directed: true, Outward: "parent for", Inward: "subtask of"}
	service, _, store := linkService([]domain.LinkType{parent}, []domain.IssueSummary{linkSummary("2-2")})

	change, err := service.RemoveIssueLink(context.Background(), "APP-1", "APP-2", LinkRelationReference{Type: "parent", Direction: new("inward")})
	require.NoError(t, err)
	assert.False(t, change.Present)
	assert.True(t, change.Changed)
	assert.Equal(t, 1, store.removeCalls)
	assert.Equal(t, domain.IssueRef("2-2"), store.targets[0])

	_, err = service.resolveLinkRelation(context.Background(), LinkRelationReference{Type: "missing"})
	assert.Equal(t, ErrorNotFound, KindOf(err))
	ambiguous, _, _ := linkService([]domain.LinkType{
		{ID: "one", Name: "One", Directed: true, Outward: "same", Inward: "inverse one"},
		{ID: "two", Name: "Two", Directed: true, Outward: "other", Inward: "same"},
	}, nil)
	_, err = ambiguous.resolveLinkRelation(context.Background(), LinkRelationReference{Type: "same"})
	assert.Equal(t, ErrorAmbiguous, KindOf(err))
}

func TestLinkOperationsPropagateReadAndWriteErrors(t *testing.T) {
	expected := errors.New("unavailable")
	linkType := domain.LinkType{ID: "relates", Name: "Relates", Outward: "relates to"}

	service, identities, _ := linkService([]domain.LinkType{linkType}, nil)
	identities.err = expected
	_, err := service.ListIssueLinks(context.Background(), "APP-1", LinkRelationReference{Type: "relates"}, PageRequest{})
	assert.ErrorIs(t, err, expected)

	service, _, store := linkService([]domain.LinkType{linkType}, nil)
	store.typeErr = expected
	_, err = service.ListLinkTypes(context.Background(), PageRequest{})
	assert.ErrorIs(t, err, expected)

	service, _, store = linkService([]domain.LinkType{linkType}, nil)
	store.linkedErr = expected
	_, err = service.ListIssueLinks(context.Background(), "APP-1", LinkRelationReference{Type: "relates"}, PageRequest{})
	assert.ErrorIs(t, err, expected)

	service, _, store = linkService([]domain.LinkType{linkType}, nil)
	store.addErr = expected
	_, err = service.AddIssueLink(context.Background(), "APP-1", "APP-2", LinkRelationReference{Type: "relates"})
	assert.ErrorIs(t, err, expected)

	service, _, store = linkService([]domain.LinkType{linkType}, []domain.IssueSummary{linkSummary("2-2")})
	store.removeErr = expected
	_, err = service.RemoveIssueLink(context.Background(), "APP-1", "APP-2", LinkRelationReference{Type: "relates"})
	assert.ErrorIs(t, err, expected)
}
