package domain

// Board is an agile board eligible to contain an issue.
type Board struct {
	ID         string
	Name       string
	ProjectIDs []string
}
