package dto

type SavedQuery struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Query *string `json:"query"`
	Owner *User   `json:"owner"`
}
