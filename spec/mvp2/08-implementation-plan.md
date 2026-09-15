# MVP2 Feature-Sliced Implementation Plan

## Execution model

Implement and merge the epics sequentially, from top to bottom. Each epic is a complete vertical slice across domain, application, YouTrack adapter, CLI, output, tests, README, and embedded agent skill.

For worktree-based implementation:

1. Create the epic worktree from the branch containing every completed earlier epic.
2. Implement the complete vertical slice.
3. Run the epic's focused verification.
4. Merge the epic into the current branch.
5. Create the next worktree from that merged state.

Do not leave dormant commands, placeholder ports, compatibility constructors, temporary aliases, or documentation for behavior that is not reachable. Shared files such as `internal/app/ports.go`, `internal/app/service.go`, `internal/cli/runtime.go`, `internal/cli/issue.go`, `internal/cli/root.go`, README, and skill references are intentionally updated by multiple epics in sequence.

The source contracts are:

- [`00-overview.md`](00-overview.md)
- [`01-architecture.md`](01-architecture.md)
- [`02-feature-behavior.md`](02-feature-behavior.md)
- [`03-cli-contract.md`](03-cli-contract.md)
- [`04-youtrack-api.md`](04-youtrack-api.md)
- [`06-agent-skill.md`](06-agent-skill.md)
- [`07-testing-release.md`](07-testing-release.md)

## Epic checklist

- [ ] Epic 1 — Implement issue search end to end
- [ ] Epic 2 — Implement saved-search view end to end
- [ ] Epic 3 — Implement issue creation end to end
- [ ] Epic 4 — Implement comment list/add/soft-remove end to end
- [ ] Epic 5 — Complete cross-feature release verification

## Epic 1 — Implement issue search end to end

### Product result

Users can execute one non-blank YouTrack query with:

```bash
youtrack issue search 'project: APP #Unresolved sort by: updated desc'
youtrack issue search 'assignee: me' --limit 20
youtrack issue search 'project: APP' --offset 100 --limit 50
youtrack issue search 'project: APP' --all
```

The query is passed to YouTrack unchanged, results preserve server order, and output contains the lightweight issue-summary fields defined in [`03-cli-contract.md`](03-cli-contract.md).

### Domain and application

1. Add `domain.IssueSummary` with entity ID, readable ID, summary, project, created time, updated time, and optional resolved time. Keep it separate from the full `domain.Issue`.
2. Add application pagination types:

   ```go
   type PageRequest struct {
       Offset int
       Limit  *int
       All    bool
   }

   type Page struct {
       Offset int
       Limit  int
   }
   ```

   `Limit == nil` means the default limit of 50. Explicit limits must be positive. Offset must be non-negative. `All` and an explicit limit are mutually exclusive.
3. Add an app-owned `IssueSearchStore` with a one-page method. REST `$skip` and `$top` must not enter the domain.
4. Extend `Service` composition as a clean cutover and update every constructor caller. Do not add a compatibility constructor.
5. Implement `Service.SearchIssues`:
   - reject blank queries before a store call
   - pass the original non-blank query unchanged
   - request pages sequentially
   - advance offsets by the actual number returned
   - continue after short non-empty pages
   - stop when the requested limit is filled or an empty page is returned
   - never parse, rewrite, filter, or sort the query/results
   - never call full issue reads or Board enrichment

### YouTrack adapter

1. Add a centralized lightweight issue-summary projection.
2. Implement the search port with one paged `GET /api/issues` request using `query`, `$skip`, `$top`, and `fields`.
3. Map REST issues directly to `domain.IssueSummary`.
4. Reuse the existing client transport and GET-only retry behavior.
5. Do not call `GetIssue`, `/sprints`, or any per-result endpoint.

### CLI and output

1. Add `issue search` to the issue command tree.
2. Add shared `--limit`, `--offset`, and `--all` flag helpers that later epics reuse.
3. Use `cmd.IsSet("limit")` so the default value is distinguishable from an explicit limit when validating `--all`.
4. Preserve nested global flags and support a leading-hyphen query after `--`.
5. Add explicit issue-summary JSON DTOs and renderer methods in new output files.
6. Implement:
   - human table: readable ID, project short name, updated time, summary
   - JSON array with the stable summary schema and explicit `resolved: null`
   - plain lines as `<readable-id>\t<summary>`
   - empty JSON as `[]`

### Tests

Add focused domain/application, adapter, CLI, output, and integration coverage for:

- exact query forwarding with spaces, braces, `#`, minus filters, and sort clauses
- blank query rejected before HTTP
- server order preserved
- default, explicit limit, offset, and `--all`
- invalid pagination combinations
- short non-empty pages followed until empty
- JSON nullability and empty-array behavior
- exact plain bytes
- no per-result enrichment calls
- invalid server query mapped as an API/runtime error

### Documentation and skill

Update in the same epic:

- `README.md` with concise direct-search examples
- `skills/youtrack-cli/SKILL.md` with bounded-search guidance
- `skills/youtrack-cli/references/commands.md` with exact syntax and flags
- `skills/youtrack-cli/references/workflows.md` with search-then-inspect guidance

Teach agents to quote query strings, use bounded discovery by default, request `--all` only when necessary, preserve explicit server sorting, and use `--json` for structured consumption.

### Complete when

- the command is reachable through the real root command
- focused package and integration tests pass
- the built CLI executes search against an `httptest` server or disposable YouTrack instance
- README and embedded skill match the implemented behavior
- all MVP1 commands remain compatible

## Epic 2 — Implement saved-search view end to end

### Product result

Users can resolve a visible saved search by database ID or name and execute its current query:

```bash
youtrack saved-search view 'Assigned to me'
youtrack saved-search view 51-33 --limit 20
youtrack saved-search view 'Release blockers' --all
```

MVP2 does not create, edit, delete, or change sharing for saved searches.

### Domain and application

1. Add `domain.SavedSearch` with entity ID, name, query, and optional owner.
2. Add an app-owned `SavedSearchStore` with:
   - direct database-ID lookup
   - one-page visible saved-search listing
3. Implement `Service.ViewSavedSearch`:
   - attempt direct database-ID lookup first
   - otherwise paginate every visible saved search
   - resolve a unique exact name, then a unique case-insensitive name
   - reject duplicate matches as ambiguous
   - reject missing references as not found
   - reject null or blank stored queries before issue search
   - call the existing `SearchIssues` path with the stored query and caller pagination
4. Return saved-search metadata plus issue summaries without introducing a second search implementation.

### YouTrack adapter

1. Add a private saved-search DTO and centralized projection for ID, name, query, and owner.
2. Implement:
   - `GET /api/savedQueries/{id}`
   - paged `GET /api/savedQueries`
3. Map wire-level `SavedQuery` objects into semantic `domain.SavedSearch`.
4. Do not request or rely on the saved query's embedded `issues` collection.
5. Do not add POST or DELETE methods for saved searches.

### CLI and output

1. Add the top-level `saved-search view` command.
2. Reuse Epic 1 pagination flags and validation.
3. Add an explicit saved-search JSON DTO:
   - `entityId`
   - `name`
   - `query`
   - nullable `owner`
   - non-null `issues` array using Epic 1 summary DTOs
4. Implement:
   - human metadata followed by the issue table
   - JSON object with metadata and summaries
   - plain output containing only issue lines

### Tests

Add focused coverage for:

- direct database-ID lookup
- exact and unique case-insensitive name resolution
- duplicate exact and case-insensitive names
- not found
- visible-search pagination beyond 42 entries
- short non-empty pages
- null and blank stored queries causing no `/api/issues` request
- stored query passed unchanged through Epic 1 search
- caller offset/limit/`--all` applied to matching issues
- nullable owner and non-null empty issue array
- absence of saved-search mutation requests

### Documentation and skill

Update README and all affected skill/reference files in the same epic. Document saved searches as view-only, show both ID and quoted-name use, and tell agents to use `saved-search view` instead of copying or recreating server-side filters locally.

### Complete when

- the command is registered and works through the built CLI
- name resolution reads all visible pages and detects ambiguity
- saved query execution reuses direct search behavior
- no saved-search mutation API exists in structured application or adapter code
- focused and integration tests pass
- README and embedded skill match the implementation

## Epic 3 — Implement issue creation end to end

### Product result

Users can create an issue with project, summary, optional description, supported fields, and existing tags:

```bash
youtrack issue create APP \
  --summary 'Login fails after token rotation' \
  --description-file ./description.md \
  --field Type=Bug \
  --field Priority=Critical \
  --field Assignee=@me \
  --tag backend
```

A valid creation performs exactly one mutation request. Any client-side validation or resolution failure performs zero mutations.

The current specification rejects `Board` and every state field during creation. If this product decision changes, update the specification before implementing this epic because application validation, adapter serialization, CLI help, tests, README, and skill guidance all depend on it.

### Domain and application

1. Add the app-owned unresolved `CreateIssueRequest` used by the CLI and fully resolved `IssueCreate` used by the adapter.
2. Add an `IssueCreator` port returning the existing full `domain.Issue`.
3. Add a project-scoped field-resolution path that reuses existing:
   - project field definitions
   - bundle values
   - allowed user bundles
   - group resolution
   - `@me` and `@none`
   - primitive/date/period parsing
   - cardinality and ambiguity rules
   - command-scoped caches
4. Do not fabricate an issue to reuse issue-scoped state-machine behavior.
5. Implement `Service.CreateIssue`:
   - require a project reference and non-blank summary
   - preserve non-blank summary and description content exactly
   - resolve the project first
   - group repeated field inputs in first-seen order
   - reject repeated scalar fields
   - reject `Board`, all state fields, and unsupported kinds
   - resolve every supplied field and tag before mutation
   - deduplicate tags by resolved entity ID while preserving first occurrence
   - call `IssueCreator.CreateIssue` exactly once
6. Reuse existing project/tag matching rules rather than introducing a second resolver convention.

### YouTrack adapter

1. Implement one `POST /api/issues` request.
2. Include in the same request body:
   - resolved project
   - summary
   - optional description
   - all custom fields
   - all tags
3. Use a creation-specific field serializer where create wire shapes differ from update:
   - field name
   - adapter-private `$type`
   - serialized semantic value
4. Include resolved tags in the top-level `tags` array.
5. Request the full issue projection and map the response through the existing semantic issue model.
6. Do not send follow-up field, tag, Board, command, or comment requests.
7. Do not retry the POST automatically.

### CLI and output

1. Add `issue create` to the issue command tree.
2. Require one project argument and `--summary`.
3. Support mutually exclusive `--description` and `--description-file`.
4. Support repeated `--field NAME=VALUE` with first-`=` splitting and comma literals.
5. Support repeated `--tag`.
6. Preserve description file bytes exactly.
7. Reuse the existing full issue renderer and JSON schema; do not create another issue output model.
8. Ensure CLI validation finishes before runtime mutation calls.

### Tests

Add focused coverage for:

- missing project or blank summary
- description source exclusivity and exact file content
- malformed field input and first-`=` behavior
- repeated multi-value and invalid repeated scalar fields
- primitive, bundle, user, group, `@me`, and `@none` resolution
- Board, state, and unsupported kinds rejected before mutation
- unknown/ambiguous project, field, value, user, group, or tag
- a late invalid value causing zero mutation requests
- deterministic tag deduplication
- exactly one valid `POST /api/issues`
- project, summary, description, fields, and tags in the same body
- no follow-up tag request
- full issue output through human, JSON, and plain modes
- POST errors not retried

### Documentation and skill

Update README and all affected skill/reference files in the same epic. Document safe field input, `--description-file`, repeated multi-value fields, one-request creation, the Board/state restriction, and inspecting the returned issue when workflows may affect subsequent work.

### Complete when

- issue creation is reachable through the built CLI
- invalid inputs are proven to perform zero mutations
- valid creation is proven to perform one POST containing fields and tags
- output reuses the MVP1 full issue contract
- focused and integration tests pass
- README and embedded skill match the implementation

## Epic 4 — Implement comment list/add/soft-remove end to end

### Product result

Users can list comments, add a text comment, and remove a comment reversibly:

```bash
youtrack issue comment list APP-123 --all
youtrack issue comment add APP-123 --text 'Ready for review.'
youtrack issue comment add APP-123 --file ./comment.md
youtrack issue comment remove APP-123 4-17
```

This is the complete MVP2 comment scope, not full REST CRUD. Comment editing, restoration, permanent deletion, attachments, visibility controls, reactions, and pinning remain excluded.

### Domain and application

1. Add `domain.Comment` with:
   - entity ID
   - optional author
   - optional raw text
   - created time
   - optional updated time
   - deleted state
2. Add an app-owned `CommentStore` with one-page list, create, and explicit soft-remove methods.
3. Implement `Service.ListComments` using Epic 1 pagination while retaining server order and returned deleted entries.
4. Implement `Service.AddComment`:
   - require non-blank issue reference
   - reject text containing no non-whitespace characters
   - pass original non-blank text unchanged
5. Implement `Service.RemoveComment`:
   - require non-blank issue and comment IDs
   - call only `SoftRemoveComment`
   - do not read first, retry, restore, or permanently delete

### YouTrack adapter

1. Add a private comment DTO and centralized projection for ID, author, text, created, updated, and deleted.
2. Implement:
   - paged `GET /api/issues/{issue}/comments`
   - text-only `POST /api/issues/{issue}/comments`
   - `POST /api/issues/{issue}/comments/{commentID}` with only `deleted=true`
3. Map nullable author, text, and updated timestamp without manufacturing values.
4. Reuse existing transport and error mapping.
5. Never call HTTP `DELETE` for comment removal.

### CLI and output

1. Add the nested commands:
   - `issue comment list`
   - `issue comment add`
   - `issue comment remove`
2. Reuse Epic 1 pagination flags for listing.
3. Require exactly one add source: `--text` or `--file`.
4. Preserve file contents and multiline text exactly.
5. Add explicit comment and removal JSON DTOs.
6. Implement:
   - human comment blocks retaining multiline text
   - JSON comment arrays/objects with explicit nullable `author`, `text`, and `updated`
   - empty JSON list as `[]`
   - plain list as one comment entity ID per line
   - plain add as the created comment ID
   - no plain output after successful removal
   - human wording that says removed, not permanently deleted

### Tests

Add focused coverage for:

- default, limit, offset, and `--all` listing
- pagination beyond 42 and short non-empty pages
- deleted comments retained and distinguishable
- nullable author, text, and updated mapping/output
- exactly one non-blank add source
- exact inline and file text preservation
- add body containing only `text`
- add POST not retried
- remove path, method, and body
- no HTTP DELETE
- repeated soft removal accepted when YouTrack accepts the state
- not-found and permission error mapping
- multiline human output, JSON nullability, and exact plain bytes

### Documentation and skill

Update README and all affected skill/reference files in the same epic. Teach agents to list comments before acting on an ID, use files for substantial multiline text, understand removal as reversible, and avoid unsupported attachments or permanent-delete claims.

### Complete when

- all three comment commands are reachable through the built CLI
- list/add/remove work through the real adapter and application layers
- removal is proven to use POST with `deleted=true` and never DELETE
- JSON and plain contracts match `03-cli-contract.md`
- focused and integration tests pass
- README and embedded skill match the implementation

## Epic 5 — Complete cross-feature release verification

### Product result

MVP2 behaves as one compatible CLI release rather than four separately working feature slices.

### Integration and acceptance

1. Add or consolidate `integration/mvp2_issue_workflow_test.go` without coupling it to the existing MVP1 edit harness.
2. Verify cross-feature flows:
   - direct search, then issue inspection
   - saved-search view using the same issue-search path
   - create, then find the issue through search
   - add/list/remove a comment on the created issue
3. Exercise global flags from every new nested command.
4. Exercise stable JSON and exact plain output across all features.
5. Confirm validation errors perform no mutation requests.
6. Confirm API errors retain existing exit-code semantics.
7. Confirm query, creation, and comment failures—including debug output—never expose permanent tokens or Authorization headers.
8. Run the acceptance workflow from [`03-cli-contract.md`](03-cli-contract.md) against a disposable YouTrack project or an equivalent real CLI harness.

### Documentation and cleanup

1. Read README and the embedded skill as a user and agent after all commands are available.
2. Remove duplicated examples, obsolete MVP1-only status text, temporary smoke-test scaffolding, unused helpers, and stale aliases.
3. Keep `mise.toml` and release configuration unchanged unless verification proves an actual defect; all required tasks already exist.
4. Do not add issue deletion or permanent comment deletion solely for test cleanup.

### Verification

Run focused tests first, then the complete repository checks:

```bash
go test ./internal/app ./internal/youtrack ./internal/output ./internal/cli ./integration
mise run fmt
mise run lint
mise run test
mise run test:race
mise run build
mise run ci
```

Confirm coverage remains at or above the targets in [`07-testing-release.md`](07-testing-release.md).

### Complete when

- every checkbox above represents a merged, reachable vertical feature
- the actual CLI acceptance workflow succeeds
- focused, race, build, and CI checks pass
- all affected docs and embedded skill content match behavior
- there are no REST DTO leaks, second resolution conventions, compatibility shims, placeholders, or unused scaffolding
