# CLI Contract

## Command tree additions

MVP2 adds these commands to the MVP1 tree:

```text
youtrack
├── issue
│   ├── search
│   ├── create
│   └── comment
│       ├── list
│       ├── add
│       ├── edit
│       └── remove
└── saved-search
    └── view
```

All MVP1 commands remain. Global flags, including `--profile`, `--url`, `--token`, `--timeout`, `--json`, `--plain`, and `--debug`, must work from every new nested command.

## Issue search

```bash
youtrack issue search 'project: APP #Unresolved sort by: updated desc'
youtrack issue search 'assignee: me' --limit 20
youtrack issue search 'project: APP' --offset 100 --limit 50
youtrack issue search 'project: APP' --all
```

Contract:

```text
youtrack issue search <query> [--limit <n>] [--offset <n>] [--all]
```

Exactly one non-blank query argument is required. Shell quoting is the caller's responsibility. `--limit` must be positive, `--offset` must be non-negative, and explicit `--limit` is mutually exclusive with `--all`. Defaults are `--limit 50 --offset 0`.

Queries that begin with a hyphen can follow `--`:

```bash
youtrack issue search -- '-State: Done project: APP'
```

## Saved-search view

```bash
youtrack saved-search view 'Assigned to me'
youtrack saved-search view 51-33 --limit 20
youtrack saved-search view 'Release blockers' --all
```

Contract:

```text
youtrack saved-search view <saved-search> [--limit <n>] [--offset <n>] [--all]
```

The reference is a database ID or visible name. Pagination flags apply to matching issues. There are no `saved-search create`, `edit`, or `remove` commands in MVP2.

## Issue create

```bash
youtrack issue create APP \
  --summary 'Login fails after token rotation' \
  --description-file ./description.md \
  --field Type=Bug \
  --field Priority=Critical \
  --field Assignee=@me \
  --tag backend
```

Contract:

```text
youtrack issue create <project> --summary <text>
  [--description <text> | --description-file <path>]
  [--field <name=value>]...
  [--tag <tag>]...
```

`--summary` is required and must contain a non-whitespace character. `--field` splits on the first `=` only. Slice-flag comma splitting is disabled, so commas remain literal. Repeated values for one multi-value field replace that field's complete initial value; a scalar field supplied more than once is invalid.

`--description` and `--description-file` are mutually exclusive. There is no Board, state, attachment, link, comment, or silent-notification flag in MVP2. Supplying `Board` or a state field through `--field` is a validation error.

All references and values are resolved before one create request. Any validation or resolution error performs zero mutation requests.

## Comments

List comments:

```bash
youtrack issue comment list APP-123
youtrack issue comment list APP-123 --limit 20 --offset 20
youtrack issue comment list APP-123 --all --json
```

Contract:

```text
youtrack issue comment list <issue> [--limit <n>] [--offset <n>] [--all]
```

Pagination validation and defaults match `issue search`.

Add a comment:

```bash
youtrack issue comment add APP-123 --text 'Ready for review.'
youtrack issue comment add APP-123 --file ./comment.md
```

Contract:

```text
youtrack issue comment add <issue> (--text <text> | --file <path>)
```

Exactly one text source is required and it must contain a non-whitespace character. No attachment or visibility flag is accepted.

Edit a comment:

```bash
youtrack issue comment edit APP-123 4-17 --text 'Ready after all.'
youtrack issue comment edit APP-123 4-17 --file ./revised-comment.md
```

Contract:

```text
youtrack issue comment edit <issue> <comment-entity-id> (--text <text> | --file <path>)
```

Exactly one replacement-text source is required and it must contain a non-whitespace character. The update sends only replacement text; it does not accept deletion, restoration, attachment, or visibility flags.

Remove a comment:

```bash
youtrack issue comment remove APP-123 4-17
```

Contract:

```text
youtrack issue comment remove <issue> <comment-entity-id>
```

Removal is a soft removal. It sets `deleted=true`; it is not permanent deletion. A successful repeated removal is allowed.

## Output and exit codes

New commands obey the MVP1 format selection and exit codes. Human formatting is not stable. JSON field names, nullability, array/object shape, plain columns, and empty-output behavior are compatibility-sensitive.

Domain structs remain free of JSON tags. `internal/output` maps them to explicit DTOs.

### Stable JSON models

#### Issue search

`issue search --json` returns an array. Each item is:

```json
{
  "id": "APP-123",
  "entityId": "2-31",
  "summary": "Fix token rotation",
  "project": {
    "entityId": "0-1",
    "name": "Application",
    "shortName": "APP"
  },
  "created": "2026-09-14T10:00:00Z",
  "updated": "2026-09-14T12:00:00Z",
  "resolved": null
}
```

`resolved` is always present and is either an RFC3339 timestamp or `null`. An empty result is `[]`, not `null`.

#### Saved-search view

`saved-search view --json` returns:

```json
{
  "entityId": "51-33",
  "name": "Assigned to me",
  "query": "for: me #Unresolved",
  "owner": {
    "entityId": "1-2",
    "login": "jane.doe",
    "name": "Jane Doe"
  },
  "issues": []
}
```

`owner` is nullable. `issues` contains the same issue-summary objects as `issue search` and is always an array.

#### Issue creation

`issue create --json` returns the existing stable full issue JSON object defined by MVP1. Creation does not introduce a second issue schema.

#### Comments

`issue comment list --json` returns an array. `issue comment add --json` and `issue comment edit --json` return one object of the same shape:

```json
{
  "entityId": "4-17",
  "author": {
    "entityId": "1-2",
    "login": "jane.doe",
    "name": "Jane Doe"
  },
  "text": "Ready for review.",
  "created": "2026-09-14T12:30:00Z",
  "updated": null,
  "deleted": false
}
```

`author`, `text`, and `updated` are nullable and always present. An empty comment list is `[]`.

`issue comment remove --json` returns:

```json
{
  "entityId": "4-17",
  "removed": true
}
```

This result means the soft-removal update succeeded; it does not claim permanent deletion.

### Human output

Human output is terminal-oriented and not formatting-stable.

- issue search: table with readable ID, project short name, updated time, and summary
- saved-search view: saved-search name, stored query, optional owner, then the issue table
- issue create: existing full issue renderer
- comment list: readable blocks containing comment ID, author, timestamps, deletion state, and text
- comment add and edit: one comment block
- comment remove: concise soft-removal confirmation

Multiline descriptions and comments are preserved in human output.

### Plain output

Plain output contains no ANSI or labels.

- issue search: one line per issue as `<readable-id>\t<summary>`
- saved-search view: the same issue lines, with no metadata header
- issue create: existing `<readable-id>\t<summary>` issue line
- comment list: one comment entity ID per line
- comment add and edit: the returned comment entity ID followed by a newline
- comment remove: no output on success

Comment text is intentionally not placed in plain list output because embedded tabs and newlines would violate the one-record-per-line contract. Use JSON when comment content is needed by a script or agent.

### Compatibility

Within the current major release, the output contracts above are stable. Human table layout is not.

## Acceptance workflow

```bash
youtrack issue search 'project: APP #Unresolved'
youtrack issue search 'project: APP' --limit 5 --json
youtrack saved-search view 'Assigned to me'
youtrack issue create APP --summary 'MVP2 acceptance issue'
youtrack issue create APP --summary 'Configured issue' \
  --description 'Created by acceptance workflow.' \
  --field Type=Task \
  --tag backend \
  --json
youtrack issue comment add APP-123 --text 'MVP2 acceptance comment' --json
youtrack issue comment edit APP-123 4-17 --text 'MVP2 acceptance comment, revised' --json
youtrack issue comment list APP-123 --all
youtrack issue comment remove APP-123 4-17 --json
```
