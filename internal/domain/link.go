package domain

type LinkDirection string

const (
	LinkDirectionOutward    LinkDirection = "outward"
	LinkDirectionInward     LinkDirection = "inward"
	LinkDirectionUndirected LinkDirection = "undirected"
)

type LinkType struct {
	ID          string
	Name        string
	Directed    bool
	Aggregation bool
	Outward     string
	Inward      string
}

type LinkRelation struct {
	Type      LinkType
	Direction LinkDirection
	Label     string
}

type IssueIdentity struct {
	ID         string
	IDReadable string
}

type IssueLinkList struct {
	IssueID  string
	Relation LinkRelation
	Issues   []IssueSummary
}

type IssueLinkChange struct {
	IssueID       string
	TargetIssueID string
	Relation      LinkRelation
	Present       bool
	Changed       bool
}
