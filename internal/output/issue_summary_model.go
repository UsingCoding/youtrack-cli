package output

import "time"

type IssueSummaryJSON struct {
	ID       string      `json:"id"`
	EntityID string      `json:"entityId"`
	Summary  string      `json:"summary"`
	Project  ProjectJSON `json:"project"`
	Created  time.Time   `json:"created"`
	Updated  time.Time   `json:"updated"`
	Resolved *time.Time  `json:"resolved"`
}
