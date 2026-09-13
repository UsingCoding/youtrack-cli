# Custom Fields and Resolution

## Semantic model

The domain uses `FieldKind`:

```text
string, integer, float, date, datetime, period, text,
enum, state, user, version, build, owned, group, board, unknown

Cardinality is independent:

```text
single | multi
```

A `FieldDefinition` contains ID, name, kind, cardinality, `canBeEmpty`, and bundle ID when applicable.

## REST type separation

YouTrack's semantic field type is not the same as `IssueCustomField.$type`.

Examples:

```text
integer/date-time/string -> SimpleIssueCustomField
date                     -> DateIssueCustomField
period                   -> PeriodIssueCustomField
text                     -> TextIssueCustomField
enum[1]                  -> SingleEnumIssueCustomField
enum[*]                  -> MultiEnumIssueCustomField
user[1]                  -> SingleUserIssueCustomField
user[*]                  -> MultiUserIssueCustomField
```

Only `internal/youtrack` may know these REST type names.

## Metadata

Project field definitions come from:

```text
GET /api/admin/projects/{project}/customFields
```

Request `field(id,name,fieldType(id,valueType,isMultiValue))`, project-field ID, `canBeEmpty`, and bundle ID where available.

All collection endpoints paginate using `$skip`/`$top`. The pager continues until an empty page instead of assuming `len(page) < requestedTop` means completion.

## Bundle-backed values

Enum, state, version, build, and owned fields resolve values using the project custom field bundle resource:

```text
/api/admin/projects/{projectID}/customFields/{fieldID}/bundle/values
```

Resolution order:

1. exact database ID
2. exact name
3. unique case-insensitive name
4. ambiguous/not found error

Do not fuzzy-match.

## Users

User fields resolve against their `UserBundle` allowed users, not all YouTrack users. `@me` resolves `/api/users/me` and is then verified against the field's allowed users.

## Groups

Group values resolve through `/api/groups?query=...`, followed by exact/unique-case-insensitive matching.

## Empty values

`@none` means clear. A dedicated `field clear` command does the same. Required fields (`canBeEmpty=false`) are rejected before mutation.

An empty string is not the same thing as clear for string/text fields.

## Multi-value input

Dedicated command:

```bash
youtrack issue field set TT-123 "Fix versions" 2026.2 2026.3
```

Combined edit repeats the field:

```bash
youtrack issue edit TT-123 \
  --field "Fix versions=2026.2" \
  --field "Fix versions=2026.3"
```

A scalar field supplied multiple times is a validation error.

## Board membership

`Board` is a reserved case-insensitive synthetic multi-value field, not a project custom field. Reads use issue sprint memberships and expose board IDs/names. Set and repeated combined-edit values replace the exact membership set; values resolve by database ID, exact name, then unique case-insensitive name among current or project-eligible boards. `@none` alone clears it; it cannot be combined with board values. YouTrack selects the current/default sprint, so no sprint syntax is accepted. A colliding custom field is addressable by project-field ID only.

## State-machine fields

A state field managed by a workflow state machine is not updated with an ordinary state bundle value. Issue reads request `possibleEvents(id,presentation)`. Such fields are exposed semantically as state fields with available transitions. `field set State <transition>` resolves the requested transition and the adapter serializes a `StateMachineIssueCustomField` event. MVP1 rejects direct clear/`@none` for state-machine fields; callers must choose an available transition.

This preserves the domain/REST boundary while supporting workflow-regulated states correctly.

## Dates and periods

- date: `YYYY-MM-DD`
- date-time: RFC3339
- REST wire timestamps: Unix milliseconds
- date-only wire value: UTC midday for the selected date
- period MVP1: `m`, `h`, and combinations such as `1h30m`

Days/weeks are intentionally not interpreted because project working-calendar semantics must not be invented by the CLI.

## Unknown types

An unknown REST `$type` must remain readable as an `UnknownValue`. Mutation of unsupported semantic field kinds fails with a clear validation error.
