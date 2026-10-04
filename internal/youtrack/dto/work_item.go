package dto

type WorkItem struct {
	ID       string        `json:"id"`
	Date     int64         `json:"date"`
	Duration WorkDuration  `json:"duration"`
	Text     *string       `json:"text"`
	Author   *User         `json:"author"`
	Creator  *User         `json:"creator"`
	Type     *WorkItemType `json:"type"`
	Created  int64         `json:"created"`
	Updated  *int64        `json:"updated"`
}

type WorkDuration struct {
	Minutes int64 `json:"minutes"`
}

type WorkItemType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TimeTrackingSettings struct {
	Enabled bool `json:"enabled"`
}
