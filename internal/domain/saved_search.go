package domain

// SavedSearch is a visible server-side saved query.
type SavedSearch struct {
	ID    string
	Name  string
	Query string
	Owner *User
}
