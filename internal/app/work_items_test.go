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

type workItemIssueFake struct{ issue domain.Issue }

func (f workItemIssueFake) GetIssue(context.Context, domain.IssueRef) (domain.Issue, error) {
	return f.issue, nil
}
func (workItemIssueFake) UpdateIssue(context.Context, domain.IssueRef, IssuePatch) error { return nil }
func (workItemIssueFake) MoveIssue(context.Context, domain.IssueRef, string) (domain.Issue, error) {
	return domain.Issue{}, nil
}
func (workItemIssueFake) AddIssueTag(context.Context, domain.IssueRef, domain.Tag) error { return nil }
func (workItemIssueFake) RemoveIssueTag(context.Context, domain.IssueRef, domain.Tag) error {
	return nil
}

type workItemStoreFake struct {
	items       []domain.WorkItem
	getItem     domain.WorkItem
	getErr      error
	create      WorkItemCreate
	patch       WorkItemPatch
	listPages   []Page
	getIssues   []domain.IssueRef
	createCalls int
	updateCalls int
	deleteCalls int
	updateErr   error
}

func (f *workItemStoreFake) ListWorkItems(_ context.Context, issue domain.IssueRef, page Page) ([]domain.WorkItem, error) {
	f.getIssues = append(f.getIssues, issue)
	f.listPages = append(f.listPages, page)
	if page.Offset >= len(f.items) {
		return []domain.WorkItem{}, nil
	}
	end := page.Offset + page.Limit
	if end > len(f.items) {
		end = len(f.items)
	}
	return append([]domain.WorkItem(nil), f.items[page.Offset:end]...), nil
}
func (f *workItemStoreFake) GetWorkItem(_ context.Context, issue domain.IssueRef, _ string) (domain.WorkItem, error) {
	f.getIssues = append(f.getIssues, issue)
	if f.getErr != nil {
		return domain.WorkItem{}, f.getErr
	}
	return f.getItem, nil
}
func (f *workItemStoreFake) CreateWorkItem(_ context.Context, issue domain.IssueRef, create WorkItemCreate) (domain.WorkItem, error) {
	f.getIssues = append(f.getIssues, issue)
	f.create = create
	f.createCalls++
	return domain.WorkItem{ID: "115-7"}, nil
}
func (f *workItemStoreFake) UpdateWorkItem(_ context.Context, issue domain.IssueRef, _ string, patch WorkItemPatch) (domain.WorkItem, error) {
	f.getIssues = append(f.getIssues, issue)
	f.patch = patch
	f.updateCalls++
	if f.updateErr != nil {
		return domain.WorkItem{}, f.updateErr
	}
	return domain.WorkItem{ID: "115-7"}, nil
}
func (f *workItemStoreFake) DeleteWorkItem(_ context.Context, issue domain.IssueRef, _ string) error {
	f.getIssues = append(f.getIssues, issue)
	f.deleteCalls++
	return nil
}

type timeTrackingFake struct {
	enabled bool
	types   []domain.WorkItemType
	pages   []Page
	err     error
}

func (f *timeTrackingFake) TimeTrackingEnabled(context.Context, string) (bool, error) {
	return f.enabled, f.err
}
func (f *timeTrackingFake) ListWorkItemTypes(_ context.Context, _ string, page Page) ([]domain.WorkItemType, error) {
	f.pages = append(f.pages, page)
	if f.err != nil {
		return nil, f.err
	}
	if page.Offset >= len(f.types) {
		return []domain.WorkItemType{}, nil
	}
	end := page.Offset + page.Limit
	if end > len(f.types) {
		end = len(f.types)
	}
	return append([]domain.WorkItemType(nil), f.types[page.Offset:end]...), nil
}

type userDirectoryFake struct {
	users []domain.User
	pages []Page
	err   error
}

func (f *userDirectoryFake) ListUsers(_ context.Context, page Page) ([]domain.User, error) {
	f.pages = append(f.pages, page)
	if f.err != nil {
		return nil, f.err
	}
	if page.Offset >= len(f.users) {
		return []domain.User{}, nil
	}
	end := page.Offset + page.Limit
	if end > len(f.users) {
		end = len(f.users)
	}
	return append([]domain.User(nil), f.users[page.Offset:end]...), nil
}

type workItemUserFake struct{ me domain.User }

func (f workItemUserFake) Me(context.Context) (domain.User, error) { return f.me, nil }

func workItemService(store *workItemStoreFake, tracking *timeTrackingFake, directory *userDirectoryFake) *Service {
	issue := workItemIssueFake{issue: domain.Issue{ID: "2-1", Project: domain.Project{ID: "0-1"}}}
	return NewService(issue, nil, nil, nil, nil, nil, nil, workItemUserFake{me: domain.User{ID: "me", Login: "me"}}, nil, nil, nil, store, tracking, directory, nil, nil)
}

func TestWorkItemListsUseCanonicalIssueAndPublicPaging(t *testing.T) {
	items := make([]domain.WorkItem, 43)
	for i := range items {
		items[i] = domain.WorkItem{ID: string(rune('a' + i))}
	}
	store := &workItemStoreFake{items: items}
	tracking := &timeTrackingFake{enabled: true, types: []domain.WorkItemType{{ID: "type"}}}
	service := workItemService(store, tracking, &userDirectoryFake{})

	got, err := service.ListWorkItems(context.Background(), "APP-1", PageRequest{All: true})
	require.NoError(t, err)
	assert.Len(t, got, 43)
	assert.Equal(t, []Page{{Offset: 0, Limit: 50}, {Offset: 43, Limit: 50}}, store.listPages)
	assert.Equal(t, domain.IssueRef("2-1"), store.getIssues[0])

	types, err := service.ListWorkItemTypes(context.Background(), "APP-1", PageRequest{Limit: new(1)})
	require.NoError(t, err)
	assert.Equal(t, []domain.WorkItemType{{ID: "type"}}, types)
	assert.NotNil(t, types)
}

func TestAddWorkItemResolvesLaterPagesBeforeOneCreate(t *testing.T) {
	types := make([]domain.WorkItemType, 43)
	users := make([]domain.User, 43)
	for i := range 42 {
		types[i] = domain.WorkItemType{ID: "t"}
		users[i] = domain.User{ID: "u", Login: "other"}
	}
	types[42] = domain.WorkItemType{ID: "development", Name: "Development"}
	users[42] = domain.User{ID: "alice-id", Login: "alice"}
	store, tracking, directory := &workItemStoreFake{}, &timeTrackingFake{enabled: true, types: types}, &userDirectoryFake{users: users}
	service := workItemService(store, tracking, directory)
	text := ""
	item, err := service.AddWorkItem(context.Background(), "APP-1", AddWorkItemRequest{Duration: "1h30m", Date: "2026-09-29", Text: &text, Type: new("development"), Author: new("alice")})
	require.NoError(t, err)
	assert.Equal(t, "115-7", item.ID)
	assert.Equal(t, 1, store.createCalls)
	assert.Equal(t, int64(90), store.create.DurationMinutes)
	assert.Equal(t, time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC), store.create.Date)
	assert.Equal(t, "", *store.create.Text)
	assert.Equal(t, "development", store.create.Type.ID)
	assert.Equal(t, "alice-id", store.create.Author.ID)
	assert.Equal(t, []Page{{Offset: 0, Limit: 50}, {Offset: 43, Limit: 50}}, tracking.pages)
	assert.Equal(t, []Page{{Offset: 0, Limit: 50}, {Offset: 43, Limit: 50}}, directory.pages)
}

func TestWorkItemAddAndEditRejectInvalidMutationWithoutWrite(t *testing.T) {
	store, tracking, directory := &workItemStoreFake{getItem: domain.WorkItem{ID: "115-7"}}, &timeTrackingFake{enabled: false}, &userDirectoryFake{}
	service := workItemService(store, tracking, directory)
	_, err := service.AddWorkItem(context.Background(), "APP-1", AddWorkItemRequest{Duration: "0m", Date: "2026-09-29"})
	require.Error(t, err)
	assert.Equal(t, ErrorValidation, KindOf(err))
	_, err = service.AddWorkItem(context.Background(), "APP-1", AddWorkItemRequest{Duration: "1h", Date: "2026-09-29"})
	require.Error(t, err)
	assert.Equal(t, ErrorValidation, KindOf(err))
	assert.Zero(t, store.createCalls)

	_, err = service.EditWorkItem(context.Background(), "APP-1", "115-7", EditWorkItemRequest{Type: new("x"), ClearType: true})
	require.Error(t, err)
	assert.Zero(t, store.updateCalls)
}

func TestEditAndRemoveWorkItemUseScopedReadAndPartialPatch(t *testing.T) {
	store := &workItemStoreFake{getItem: domain.WorkItem{ID: "115-7"}}
	tracking := &timeTrackingFake{enabled: true}
	service := workItemService(store, tracking, &userDirectoryFake{})
	text := ""
	_, err := service.EditWorkItem(context.Background(), "APP-1", "115-7", EditWorkItemRequest{Text: &text, ClearType: true})
	require.NoError(t, err)
	require.NotNil(t, store.patch.Text)
	assert.Equal(t, "", *store.patch.Text)
	assert.True(t, store.patch.Type.Set)
	assert.Nil(t, store.patch.Type.Value)
	assert.Nil(t, store.patch.Date)
	assert.Nil(t, store.patch.DurationMinutes)
	assert.Equal(t, domain.IssueRef("2-1"), store.getIssues[0])

	id, err := service.RemoveWorkItem(context.Background(), "APP-1", "115-7")
	require.NoError(t, err)
	assert.Equal(t, "115-7", id)
	assert.Equal(t, 1, store.deleteCalls)
	store.getErr = errors.New("missing")
	_, err = service.RemoveWorkItem(context.Background(), "APP-1", "115-7")
	require.Error(t, err)
	assert.Equal(t, 1, store.deleteCalls)
}

func TestWorkItemResolutionRulesAndPeriodOverflow(t *testing.T) {
	store := &workItemStoreFake{}
	tracking := &timeTrackingFake{enabled: true, types: []domain.WorkItemType{{ID: "one", Name: "Development"}, {ID: "two", Name: "development"}}}
	directory := &userDirectoryFake{users: []domain.User{{ID: "a", Login: "alice", FullName: "Alice Smith"}, {ID: "b", Login: "ALICE"}}}
	service := workItemService(store, tracking, directory)
	_, err := service.AddWorkItem(context.Background(), "APP-1", AddWorkItemRequest{Duration: "1m", Date: "2026-09-29", Type: new("DEVELOPMENT")})
	require.Error(t, err)
	assert.Equal(t, ErrorAmbiguous, KindOf(err))
	_, err = service.AddWorkItem(context.Background(), "APP-1", AddWorkItemRequest{Duration: "1m", Date: "2026-09-29", Author: new("Alice Smith")})
	require.Error(t, err)
	assert.Equal(t, ErrorNotFound, KindOf(err))
	_, err = parseWorkItemDuration("153722867280912931h")
	require.Error(t, err)
	assert.Equal(t, 0, store.createCalls)
}
