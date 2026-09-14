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

New commands obey the MVP1 format selection and exit codes. Human formatting is not stable. JSON field names and plain semantics are compatibility-sensitive and are defined in [`05-config-auth-output.md`](05-config-auth-output.md).

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
youtrack issue comment list APP-123 --all
youtrack issue comment remove APP-123 4-17 --json
```
