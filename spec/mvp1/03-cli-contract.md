# CLI Contract

## Command tree

```text
youtrack
├── auth
│   ├── login
│   ├── logout
│   └── status
├── issue
│   ├── view
│   ├── edit
│   ├── move
│   ├── field
│   │   ├── list
│   │   ├── get
│   │   ├── set
│   │   └── clear
│   └── tag
│       ├── list
│       ├── add
│       └── remove
├── api
├── config
│   ├── list
│   ├── get
│   └── set
├── skill
│   ├── list
│   ├── install
│   ├── update
│   └── remove
└── version
```

## Global flags

```text
--profile <name>
--url <service-url>
--token <token>
--timeout <duration>   default 30s
--json
--plain
--debug
```

`--json` and `--plain` are mutually exclusive.

## Issue edit

```bash
youtrack issue edit TT-123 \
  --summary "New summary" \
  --description-file description.md \
  --field Priority=Critical \
  --field "State=In Progress" \
  --tag backend \
  --remove-tag legacy
```

`--field` splits on the first `=` only. `--description` and `--description-file` are mutually exclusive. Adding and removing the same tag is invalid. No mutation flags is invalid. Slice-flag comma splitting is disabled, so commas inside a field value, tag, or raw API header remain literal.

Repeated `--field Board=...` values replace the Board membership set. All resolution and Board command-assist validation happen before any mutation. Mixed edits may use one issue POST followed by one Board command POST; execution is non-atomic after validation.

## Field commands

```bash
youtrack issue field list TT-123
youtrack issue field get TT-123 Priority
youtrack issue field set TT-123 Priority Critical
youtrack issue field set TT-123 "Fix versions" 2026.2 2026.3
youtrack issue field set TT-123 Assignee @me
youtrack issue field clear TT-123 Assignee
```

`Board` is a reserved synthetic field available from view/list/get/set/clear and `--field Board=...`. Values are board IDs or names; set replaces the whole membership set and clear yields an empty set. The CLI does not accept sprint syntax.

## Tags

```bash
youtrack issue tag list TT-123
youtrack issue tag add TT-123 backend
youtrack issue tag remove TT-123 backend
```

Add-existing and remove-missing are CLI-level no-op successes.

## Move

```bash
youtrack issue move TT-123 PLATFORM
```

Project resolution prefers direct database ID/short-name lookup, then exact name, then a unique case-insensitive name.

## Raw API

```bash
youtrack api /api/users/me
youtrack api /api/issues/TT-123 --method POST --data '{"summary":"Updated"}'
```

Only relative `/api` paths are accepted. `Authorization` cannot be overridden by `--header`.

## Output

Human output is optimized for a terminal and is not formatting-stable.

JSON is an explicit, stable CLI model and never emits REST DTOs directly.

Plain output contains no decoration/ANSI and is intended for shell pipelines. Single field get prints one scalar when possible; tag lists print one tag per line.

## Exit codes

```text
0 success
1 runtime/API error
2 CLI validation/usage
3 authentication/configuration
4 not found
5 ambiguous reference
```

## Acceptance workflow

```bash
youtrack auth login
youtrack auth status
youtrack issue view TT-123
youtrack issue field list TT-123
youtrack issue field get TT-123 Priority
youtrack issue field set TT-123 Priority Critical
youtrack issue field set TT-123 "Fix versions" 2026.2 2026.3
youtrack issue field set TT-123 Assignee @me
youtrack issue field clear TT-123 Assignee
youtrack issue edit TT-123 --summary "New" --field Priority=Critical --tag backend
youtrack issue tag add TT-123 backend
youtrack issue tag remove TT-123 backend
youtrack issue move TT-123 PLATFORM
youtrack issue view TT-123 --json
youtrack api /api/users/me
youtrack skill install
```
