package domain

import "time"

type WorkItem struct {
	ID              string
	Date            time.Time
	DurationMinutes int64
	Text            *string
	Author          *User
	Creator         *User
	Type            *WorkItemType
	Created         time.Time
	Updated         *time.Time
}

type WorkItemType struct {
	ID   string
	Name string
}
