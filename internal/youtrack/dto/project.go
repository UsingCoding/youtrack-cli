package dto

type FieldType struct {
	ID           string `json:"id"`
	ValueType    string `json:"valueType"`
	IsMultiValue bool   `json:"isMultiValue"`
}

type CustomField struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	FieldType FieldType `json:"fieldType"`
}

type Bundle struct {
	ID string `json:"id"`
}

type ProjectCustomField struct {
	ID         string      `json:"id"`
	Type       string      `json:"$type"`
	CanBeEmpty bool        `json:"canBeEmpty"`
	Field      CustomField `json:"field"`
	Bundle     *Bundle     `json:"bundle"`
}

type BundleValue struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
}

type Group struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
