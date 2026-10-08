package app

import (
	"context"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

// ResolveIssueIdentity resolves canonical issue identity without loading issue details.
func (s *Service) ResolveIssueIdentity(ctx context.Context, ref domain.IssueRef) (domain.IssueIdentity, error) {
	if strings.TrimSpace(string(ref)) == "" {
		return domain.IssueIdentity{}, Validationf("issue reference must not be blank")
	}
	return s.issueIdentity.GetIssueIdentity(ctx, ref)
}
