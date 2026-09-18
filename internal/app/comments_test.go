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

type commentScript struct {
	items []domain.Comment
	err   error
}

type commentStoreFake struct {
	listScripts []commentScript
	listPages   []Page
	listIssues  []domain.IssueRef
	createTexts []string
	createIssue []domain.IssueRef
	create      domain.Comment
	createErr   error
	editIssue   []domain.IssueRef
	editIDs     []string
	editTexts   []string
	edit        domain.Comment
	editErr     error
	removeIDs   []string
	removeIssue []domain.IssueRef
	removeErr   error
}

func (f *commentStoreFake) ListComments(_ context.Context, issue domain.IssueRef, page Page) ([]domain.Comment, error) {
	f.listIssues = append(f.listIssues, issue)
	f.listPages = append(f.listPages, page)
	if len(f.listScripts) == 0 {
		return nil, errors.New("unexpected list call")
	}
	script := f.listScripts[0]
	f.listScripts = f.listScripts[1:]
	return script.items, script.err
}

func (f *commentStoreFake) CreateComment(_ context.Context, issue domain.IssueRef, text string) (domain.Comment, error) {
	f.createIssue = append(f.createIssue, issue)
	f.createTexts = append(f.createTexts, text)
	return f.create, f.createErr
}

func (f *commentStoreFake) EditComment(_ context.Context, issue domain.IssueRef, commentID, text string) (domain.Comment, error) {
	f.editIssue = append(f.editIssue, issue)
	f.editIDs = append(f.editIDs, commentID)
	f.editTexts = append(f.editTexts, text)
	return f.edit, f.editErr
}

func (f *commentStoreFake) SoftRemoveComment(_ context.Context, issue domain.IssueRef, commentID string) error {
	f.removeIssue = append(f.removeIssue, issue)
	f.removeIDs = append(f.removeIDs, commentID)
	return f.removeErr
}

func commentService(store *commentStoreFake) *Service {
	return NewService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, store)
}

func testComment(id string, deleted bool) domain.Comment {
	return domain.Comment{ID: id, Created: time.Unix(0, 0), Deleted: deleted}
}

func TestListCommentsRejectsInvalidInputBeforeStoreCall(t *testing.T) {
	cases := []struct {
		name, issue, message string
		request              PageRequest
	}{
		{name: "blank issue", issue: " \t", message: "issue reference must not be blank"},
		{name: "negative offset", issue: "TT-1", request: PageRequest{Offset: -1}, message: "offset must be non-negative"},
		{name: "zero limit", issue: "TT-1", request: PageRequest{Limit: new(0)}, message: "limit must be positive"},
		{name: "all and limit", issue: "TT-1", request: PageRequest{All: true, Limit: new(1)}, message: "--all and --limit are mutually exclusive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &commentStoreFake{}
			got, err := commentService(store).ListComments(context.Background(), domain.IssueRef(tc.issue), tc.request)
			assert.Nil(t, got)
			require.Error(t, err)
			assert.Equal(t, ErrorValidation, KindOf(err))
			assert.Equal(t, tc.message, err.Error())
			assert.Empty(t, store.listPages)
		})
	}
}

func TestListCommentsPaginatesByActualResults(t *testing.T) {
	store := &commentStoreFake{listScripts: []commentScript{
		{items: []domain.Comment{testComment("4-3", false), testComment("4-1", true)}},
		{items: []domain.Comment{testComment("4-2", false)}},
		{items: []domain.Comment{}},
	}}
	got, err := commentService(store).ListComments(context.Background(), "TT-1", PageRequest{Offset: 7, Limit: new(5)})
	require.NoError(t, err)
	assert.Equal(t, []Page{{Offset: 7, Limit: 5}, {Offset: 9, Limit: 3}, {Offset: 10, Limit: 2}}, store.listPages)
	assert.Equal(t, []domain.Comment{testComment("4-3", false), testComment("4-1", true), testComment("4-2", false)}, got)
}

func TestListCommentsUsesDefaultTruncatesAndAll(t *testing.T) {
	oversized := make([]domain.Comment, 51)
	for i := range oversized {
		oversized[i] = testComment(string(rune('a'+i%26)), false)
	}
	store := &commentStoreFake{listScripts: []commentScript{{items: oversized}}}
	got, err := commentService(store).ListComments(context.Background(), "TT-1", PageRequest{})
	require.NoError(t, err)
	assert.Equal(t, []Page{{Offset: 0, Limit: DefaultPageLimit}}, store.listPages)
	assert.Len(t, got, DefaultPageLimit)

	page42 := make([]domain.Comment, 42)
	for i := range page42 {
		page42[i] = testComment(string(rune('a'+i%26)), false)
	}
	all := &commentStoreFake{listScripts: []commentScript{{items: page42}, {items: []domain.Comment{testComment("after", true)}}, {items: []domain.Comment{}}}}
	got, err = commentService(all).ListComments(context.Background(), "TT-1", PageRequest{Offset: 4, All: true})
	require.NoError(t, err)
	assert.Equal(t, []Page{{Offset: 4, Limit: 50}, {Offset: 46, Limit: 50}, {Offset: 47, Limit: 50}}, all.listPages)
	assert.Len(t, got, 43)
	assert.True(t, got[42].Deleted)
}

func TestListCommentsEmptyAndErrors(t *testing.T) {
	empty := &commentStoreFake{listScripts: []commentScript{{items: []domain.Comment{}}}}
	got, err := commentService(empty).ListComments(context.Background(), "TT-1", PageRequest{})
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)

	want := errors.New("server failed")
	failed := &commentStoreFake{listScripts: []commentScript{{err: want}}}
	got, err = commentService(failed).ListComments(context.Background(), "TT-1", PageRequest{})
	assert.Nil(t, got)
	assert.ErrorIs(t, err, want)
}

func TestAddCommentValidatesAndPreservesText(t *testing.T) {
	store := &commentStoreFake{create: testComment("4-1", false)}
	text := "  line one\n\tline two  \n"
	got, err := commentService(store).AddComment(context.Background(), "TT-1", text)
	require.NoError(t, err)
	assert.Equal(t, testComment("4-1", false), got)
	assert.Equal(t, []domain.IssueRef{"TT-1"}, store.createIssue)
	assert.Equal(t, []string{text}, store.createTexts)

	for _, input := range []struct{ issue, text, message string }{{" ", "ok", "issue reference must not be blank"}, {"TT-1", " \t\n", "comment text must not be blank"}} {
		store = &commentStoreFake{}
		_, err := commentService(store).AddComment(context.Background(), domain.IssueRef(input.issue), input.text)
		require.Error(t, err)
		assert.Equal(t, input.message, err.Error())
		assert.Empty(t, store.createTexts)
	}
	want := errors.New("create failed")
	store = &commentStoreFake{createErr: want}
	_, err = commentService(store).AddComment(context.Background(), "TT-1", "ok")
	assert.ErrorIs(t, err, want)
	assert.Len(t, store.createTexts, 1)
}

func TestRemoveCommentValidatesAndCallsDirectly(t *testing.T) {
	store := &commentStoreFake{}
	service := commentService(store)
	for _, input := range []struct{ issue, id, message string }{{" ", "4-1", "issue reference must not be blank"}, {"TT-1", " \t", "comment ID must not be blank"}} {
		err := service.RemoveComment(context.Background(), domain.IssueRef(input.issue), input.id)
		require.Error(t, err)
		assert.Equal(t, input.message, err.Error())
	}
	assert.Empty(t, store.removeIDs)
	require.NoError(t, service.RemoveComment(context.Background(), "TT-1", "4-1"))
	require.NoError(t, service.RemoveComment(context.Background(), "TT-1", "4-2"))
	assert.Equal(t, []string{"4-1", "4-2"}, store.removeIDs)
	assert.Empty(t, store.listPages)
	assert.Empty(t, store.createTexts)
}

func TestEditCommentValidatesAndCallsOnlyEdit(t *testing.T) {
	for _, input := range []struct {
		name, issue, id, text, message string
	}{
		{name: "blank issue", issue: " \t", id: "4-1", text: "replacement", message: "issue reference must not be blank"},
		{name: "blank comment ID", issue: "TT-1", id: " \t", text: "replacement", message: "comment ID must not be blank"},
		{name: "blank replacement text", issue: "TT-1", id: "4-1", text: " \t\n", message: "comment text must not be blank"},
	} {
		t.Run(input.name, func(t *testing.T) {
			store := &commentStoreFake{}
			_, err := commentService(store).EditComment(context.Background(), domain.IssueRef(input.issue), input.id, input.text)
			require.Error(t, err)
			assert.Equal(t, input.message, err.Error())
			assert.Empty(t, store.listPages)
			assert.Empty(t, store.createTexts)
			assert.Empty(t, store.editTexts)
			assert.Empty(t, store.removeIDs)
		})
	}

	text := "  revised line one\n\tline two  \n"
	store := &commentStoreFake{edit: testComment("4-1", false)}
	got, err := commentService(store).EditComment(context.Background(), "TT-1", "4-1", text)
	require.NoError(t, err)
	assert.Equal(t, testComment("4-1", false), got)
	assert.Equal(t, []domain.IssueRef{"TT-1"}, store.editIssue)
	assert.Equal(t, []string{"4-1"}, store.editIDs)
	assert.Equal(t, []string{text}, store.editTexts)
	assert.Empty(t, store.listPages)
	assert.Empty(t, store.createTexts)
	assert.Empty(t, store.removeIDs)
}

func TestEditCommentPropagatesErrorWithoutRetry(t *testing.T) {
	want := errors.New("edit failed")
	store := &commentStoreFake{editErr: want}
	_, err := commentService(store).EditComment(context.Background(), "TT-1", "4-1", "replacement")
	assert.ErrorIs(t, err, want)
	assert.Equal(t, []string{"replacement"}, store.editTexts)
	assert.Empty(t, store.listPages)
	assert.Empty(t, store.createTexts)
	assert.Empty(t, store.removeIDs)
}
