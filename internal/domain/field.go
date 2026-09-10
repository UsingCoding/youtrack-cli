package domain

import "time"

type FieldKind string

const (
	FieldUnknown  FieldKind = "unknown"
	FieldString   FieldKind = "string"
	FieldInteger  FieldKind = "integer"
	FieldFloat    FieldKind = "float"
	FieldDate     FieldKind = "date"
	FieldDateTime FieldKind = "datetime"
	FieldPeriod   FieldKind = "period"
	FieldText     FieldKind = "text"
	FieldEnum     FieldKind = "enum"
	FieldState    FieldKind = "state"
	FieldUser     FieldKind = "user"
	FieldVersion  FieldKind = "version"
	FieldBuild    FieldKind = "build"
	FieldOwned    FieldKind = "owned"
	FieldGroup    FieldKind = "group"
)

type Cardinality string

const (
	CardinalitySingle Cardinality = "single"
	CardinalityMulti  Cardinality = "multi"
)

type FieldDefinition struct {
	ID          string
	Name        string
	Kind        FieldKind
	Cardinality Cardinality
	CanBeEmpty  bool
	BundleID    string
}

type FieldOption struct {
	ID   string
	Name string
}

type FieldValue interface {
	isFieldValue()
}

type StringValue struct{ Value string }
type IntegerValue struct{ Value int64 }
type FloatValue struct{ Value float64 }
type DateValue struct{ Value time.Time }
type PeriodValue struct{ Minutes int64 }
type TextValue struct{ Value string }
type EntityValue struct{ ID, Name string }
type UserValue struct{ ID, Login, FullName string }
type StateTransitionValue struct{ ID, Presentation string }
type MultiValue struct{ Values []FieldValue }
type EmptyValue struct{}
type UnknownValue struct{ Type string }

func (StringValue) isFieldValue()          {}
func (IntegerValue) isFieldValue()         {}
func (FloatValue) isFieldValue()           {}
func (DateValue) isFieldValue()            {}
func (PeriodValue) isFieldValue()          {}
func (TextValue) isFieldValue()            {}
func (EntityValue) isFieldValue()          {}
func (UserValue) isFieldValue()            {}
func (StateTransitionValue) isFieldValue() {}
func (MultiValue) isFieldValue()           {}
func (EmptyValue) isFieldValue()           {}
func (UnknownValue) isFieldValue()         {}

type IssueField struct {
	ID           string
	Name         string
	Kind         FieldKind
	Cardinality  Cardinality
	Value        FieldValue
	StateMachine bool
	Transitions  []FieldOption
}
