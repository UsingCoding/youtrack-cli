package dto

type Comment struct {
	ID      string  `json:"id"`
	Text    *string `json:"text"`
	Author  *User   `json:"author"`
	Created int64   `json:"created"`
	Updated *int64  `json:"updated"`
	Deleted bool    `json:"deleted"`
}
