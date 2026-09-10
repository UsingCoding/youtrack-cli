package app

import (
	"context"
	"net/http"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type FieldAssignment struct {
	Field domain.FieldDefinition
	Value domain.FieldValue
}

type IssuePatch struct {
	Summary     *string
	Description *string
	Fields      []FieldAssignment
	Tags        *[]domain.Tag
}

type IssueStore interface {
	GetIssue(context.Context, domain.IssueRef) (domain.Issue, error)
	UpdateIssue(context.Context, domain.IssueRef, IssuePatch) (domain.Issue, error)
	MoveIssue(context.Context, domain.IssueRef, string) (domain.Issue, error)
	AddIssueTag(context.Context, domain.IssueRef, domain.Tag) error
	RemoveIssueTag(context.Context, domain.IssueRef, domain.Tag) error
}

type ProjectFieldStore interface {
	ListProjectFields(context.Context, string) ([]domain.FieldDefinition, error)
	ListFieldOptions(context.Context, string, domain.FieldDefinition) ([]domain.FieldOption, error)
	ListFieldUsers(context.Context, string, domain.FieldDefinition) ([]domain.User, error)
}

type ProjectStore interface {
	GetProject(context.Context, domain.ProjectRef) (domain.Project, error)
	SearchProjects(context.Context, string) ([]domain.Project, error)
}

type TagStore interface {
	SearchTags(context.Context, string) ([]domain.Tag, error)
}

type UserStore interface {
	Me(context.Context) (domain.User, error)
}

type GroupStore interface {
	SearchGroups(context.Context, string) ([]domain.Group, error)
}

type RawResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

type RawAPI interface {
	DoRaw(context.Context, string, string, []byte, http.Header) (RawResponse, error)
}
