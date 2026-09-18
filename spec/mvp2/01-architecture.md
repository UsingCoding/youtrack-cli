# Architecture

## Relationship to MVP1

MVP2 extends the package layout and dependency graph defined in [`spec/mvp1/01-architecture.md`](../mvp1/01-architecture.md). The direction remains:

```text
cli -> app -> domain
youtrack -> app ports + domain
output -> domain
config, credentials, skill -> independent utilities
```

No new package may bypass these boundaries.

## Domain models

`internal/domain` adds semantic models without JSON tags or REST names:

- `IssueSummary`: entity ID, readable ID, summary, project, created, updated, and optional resolved time
- `SavedSearch`: entity ID, name, query, and optional owner
- `Comment`: entity ID, optional author, optional text, created time, optional updated time, and deleted state

The full existing `Issue` remains the result of issue creation. `SavedSearch` is the domain term even though the YouTrack REST resource is named `SavedQuery`.

Pagination controls are application concepts: non-negative offset, positive limit, and an explicit all-results mode. They are not copied from REST `$skip` and `$top` DTO fields into the domain.

## Consumer-owned ports

Add narrow interfaces in `internal/app/ports.go` instead of exposing the HTTP client:

- an issue-search port that accepts an opaque query and page request and returns issue summaries
- an issue-creation port that accepts a resolved project, summary, optional description, resolved field assignments, and resolved tags
- a saved-search port that reads one search by database ID and paginates all visible searches for name resolution
- a comment port that lists, creates, edits, and soft-removes comments

The application layer owns these contracts. Adapter request and response DTOs remain private to `internal/youtrack`.

## Search flow

Direct issue search is read-only:

```text
validate query and page options
 -> GET /api/issues with the exact query
 -> continue pages until the requested limit is filled or an empty page is returned
 -> map REST issues to domain IssueSummary values
 -> render in server order
```

The application does not parse or normalize the query and does not apply client-side filtering or sorting.

Saved-search view is also read-only:

```text
resolve saved search by database ID
 or paginate every visible saved search and resolve its name
 -> reject null/blank query
 -> execute the stored query through the ordinary issue-search path
 -> render saved-search metadata and issue summaries
```

Resolution order is exact database ID, unique exact name, then unique case-insensitive name. Duplicate matches are ambiguous. There is no fuzzy matching.

## Issue-creation flow

```text
validate required CLI values
 -> resolve project
 -> load/cache project field definitions
 -> group and resolve every supplied field value
 -> reject Board and state fields
 -> resolve every supplied tag
 -> validate the complete create request
 -> one POST /api/issues containing all values
 -> map and render the created issue
```

No mutation occurs before all client-side resolution succeeds. Omitted project fields are left to YouTrack defaults and workflows; the CLI does not invent defaults. An API rejection after the POST is a runtime/API error and is not retried automatically.

Creation reuses the MVP1 custom-field resolver semantics for supported non-state project fields. The resolver gains a project-scoped path so it does not need a fabricated issue. The existing issue-scoped path remains responsible for state-machine transitions during edits.

`Board` stays a synthetic issue field. It cannot be assigned before an issue exists, and adding it after creation would require a second non-atomic mutation, so MVP2 rejects it before the create POST.

## Comment flows

List:

```text
validate issue reference and page options
 -> GET issue comments
 -> paginate to the requested limit or an empty page
 -> map visible comments
 -> render in server order
```

Add:

```text
read exactly one text source
 -> reject blank text
 -> POST one text-only comment
 -> map and render the returned comment
```

Edit:

```text
read exactly one replacement-text source
-> reject blank text and blank issue or comment references
-> POST replacement text to the specific comment
-> map and render the returned comment
```

Remove:

```text
validate issue and comment references
 -> POST deleted=true to the specific comment
 -> render removal result
```

Writes are sequential and are never automatically retried. Re-removing an already removed comment is an idempotent success when YouTrack accepts the same deleted state.

## Command-scoped caches

MVP1 project-field, bundle, allowed-user, current-user, tag, and board rules remain. Saved-search name resolution may cache its fully paginated visible-search list for one invocation only. No persistent cache is introduced.

## Concurrency

Search page requests are sequential so order and server behavior remain predictable. Issue creation and comment mutations are sequential. Do not parallelize writes.
