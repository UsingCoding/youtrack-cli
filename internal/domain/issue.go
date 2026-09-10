package domain

import "time"

type Issue struct {
	ID          string
	IDReadable  string
	Summary     string
	Description string
	Project     Project
	Reporter    *User
	Updater     *User
	Created     time.Time
	Updated     time.Time
	Resolved    *time.Time
	Fields      []IssueField
	Tags        []Tag
}
