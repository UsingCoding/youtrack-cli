package output

type SavedSearchJSON struct {
	EntityID string             `json:"entityId"`
	Name     string             `json:"name"`
	Query    string             `json:"query"`
	Owner    *UserJSON          `json:"owner"`
	Issues   []IssueSummaryJSON `json:"issues"`
}
