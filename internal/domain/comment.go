package domain

import "time"

type Comment struct {
	ID      string
	Author  *User
	Text    *string
	Created time.Time
	Updated *time.Time
	Deleted bool
}
