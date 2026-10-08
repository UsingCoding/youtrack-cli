package dto

type LinkType struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Directed       bool   `json:"directed"`
	Aggregation    bool   `json:"aggregation"`
	SourceToTarget string `json:"sourceToTarget"`
	TargetToSource string `json:"targetToSource"`
}

type IssueIdentity struct {
	ID         string `json:"id"`
	IDReadable string `json:"idReadable"`
}
