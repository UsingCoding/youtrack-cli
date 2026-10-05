package output

import "github.com/UsingCoding/youtrack-cli/internal/domain"

type LinkTypeJSON struct {
	EntityID    string  `json:"entityId"`
	Name        string  `json:"name"`
	Directed    bool    `json:"directed"`
	Aggregation bool    `json:"aggregation"`
	Outward     string  `json:"outward"`
	Inward      *string `json:"inward"`
}

type LinkRelationTypeJSON struct {
	EntityID string `json:"entityId"`
	Name     string `json:"name"`
}

type LinkRelationJSON struct {
	Type      LinkRelationTypeJSON `json:"type"`
	Direction string               `json:"direction"`
	Label     string               `json:"label"`
}

type IssueLinkListJSON struct {
	IssueID  string             `json:"issueId"`
	Relation LinkRelationJSON   `json:"relation"`
	Issues   []IssueSummaryJSON `json:"issues"`
}

type IssueLinkChangeJSON struct {
	IssueID       string           `json:"issueId"`
	TargetIssueID string           `json:"targetIssueId"`
	Relation      LinkRelationJSON `json:"relation"`
	Present       bool             `json:"present"`
	Changed       bool             `json:"changed"`
}

func linkTypeJSON(item domain.LinkType) LinkTypeJSON {
	out := LinkTypeJSON{
		EntityID: item.ID, Name: item.Name, Directed: item.Directed, Aggregation: item.Aggregation, Outward: item.Outward,
	}
	if item.Directed {
		inward := item.Inward
		out.Inward = &inward
	}
	return out
}

func linkTypesJSON(items []domain.LinkType) []LinkTypeJSON {
	out := make([]LinkTypeJSON, 0, len(items))
	for _, item := range items {
		out = append(out, linkTypeJSON(item))
	}
	return out
}

func linkRelationJSON(relation domain.LinkRelation) LinkRelationJSON {
	return LinkRelationJSON{
		Type:      LinkRelationTypeJSON{EntityID: relation.Type.ID, Name: relation.Type.Name},
		Direction: string(relation.Direction), Label: relation.Label,
	}
}

func issueLinkListJSON(item domain.IssueLinkList) IssueLinkListJSON {
	return IssueLinkListJSON{IssueID: item.IssueID, Relation: linkRelationJSON(item.Relation), Issues: issueSummariesJSON(item.Issues)}
}

func issueLinkChangeJSON(item domain.IssueLinkChange) IssueLinkChangeJSON {
	return IssueLinkChangeJSON{
		IssueID: item.IssueID, TargetIssueID: item.TargetIssueID, Relation: linkRelationJSON(item.Relation), Present: item.Present, Changed: item.Changed,
	}
}
