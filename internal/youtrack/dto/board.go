package dto

type Agile struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Projects []Project `json:"projects"`
}

type Sprint struct {
	Agile *Agile `json:"agile"`
}

type ParsedCommand struct {
	Error       bool   `json:"error"`
	Delete      bool   `json:"delete"`
	Description string `json:"description"`
}

type CommandList struct {
	Commands []ParsedCommand `json:"commands"`
}
