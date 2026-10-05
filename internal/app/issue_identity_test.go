package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type issueIdentityFake struct {
	identity domain.IssueIdentity
	calls    []domain.IssueRef
}

func (f *issueIdentityFake) GetIssueIdentity(_ context.Context, ref domain.IssueRef) (domain.IssueIdentity, error) {
	f.calls = append(f.calls, ref)
	return f.identity, nil
}

func TestResolveIssueIdentityUsesIdentityPortOnly(t *testing.T) {
	identity := &issueIdentityFake{identity: domain.IssueIdentity{ID: "2-1", IDReadable: "APP-1"}}
	service := NewService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, identity, nil)

	got, err := service.ResolveIssueIdentity(context.Background(), "APP-1")

	require.NoError(t, err)
	assert.Equal(t, identity.identity, got)
	assert.Equal(t, []domain.IssueRef{"APP-1"}, identity.calls)
}

func TestResolveIssueIdentityRejectsBlankReference(t *testing.T) {
	identity := &issueIdentityFake{}
	service := NewService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, identity, nil)

	_, err := service.ResolveIssueIdentity(context.Background(), " \t")

	require.Error(t, err)
	assert.Equal(t, ErrorValidation, KindOf(err))
	assert.Empty(t, identity.calls)
}
