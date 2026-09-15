package dto

import "encoding/json"

type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
}

type User struct {
	ID       string `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"fullName"`
}

type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Event struct {
	ID           string `json:"id"`
	Presentation string `json:"presentation"`
}

type IssueCustomField struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Type           string          `json:"$type"`
	Value          json.RawMessage `json:"value"`
	PossibleEvents []Event         `json:"possibleEvents"`
}

type Issue struct {
	ID           string             `json:"id"`
	IDReadable   string             `json:"idReadable"`
	Summary      string             `json:"summary"`
	Description  string             `json:"description"`
	Created      int64              `json:"created"`
	Updated      int64              `json:"updated"`
	Resolved     *int64             `json:"resolved"`
	Project      Project            `json:"project"`
	Reporter     *User              `json:"reporter"`
	Updater      *User              `json:"updater"`
	Tags         []Tag              `json:"tags"`
	CustomFields []IssueCustomField `json:"customFields"`
}

type IssueSummary struct {
	ID         string  `json:"id"`
	IDReadable string  `json:"idReadable"`
	Summary    string  `json:"summary"`
	Project    Project `json:"project"`
	Created    int64   `json:"created"`
	Updated    int64   `json:"updated"`
	Resolved   *int64  `json:"resolved"`
}
