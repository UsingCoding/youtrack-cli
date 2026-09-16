# youtrack-cli

A Go CLI for JetBrains YouTrack, inspired by the entity-oriented structure and agent-friendly workflows of [`teamcity-cli`](https://github.com/jetbrains/teamcity-cli).

MVP1 focuses on safe issue inspection and mutation: custom fields, tags, summary/description changes, and project moves. MVP2 issue search, read-only saved-search view, comment management, and issue creation are implemented; release verification remains unavailable.

## Status

MVP1 implementation plus MVP2 issue search, read-only saved-search view, comment management, and issue creation. The implementation specification is split across [`spec/mvp1`](spec/mvp1/00-overview.md) and [`spec/mvp2`](spec/mvp2/00-overview.md).

## Install for development

```bash
mise install
mise run build
./bin/youtrack version
```

Or:

```bash
go install ./cmd/youtrack
```

## Authentication

```bash
youtrack auth login
```

You can also provide:

```text
YOUTRACK_PROFILE
YOUTRACK_URL
YOUTRACK_TOKEN
```

Configuration and credentials are stored separately. Permanent tokens are not written into the normal profile config.

```bash
youtrack auth status
youtrack auth logout
```

## Issue inspection

```bash
youtrack issue view TT-123
youtrack issue view TT-123 --json

youtrack issue field list TT-123
youtrack issue field get TT-123 Priority --plain
```

## Issue search

```bash
youtrack issue search 'project: APP'
youtrack issue search 'project: APP #Unresolved sort by: updated desc' --limit 20
youtrack issue search 'project: APP' --offset 100 --limit 50
youtrack issue search 'project: APP' --all
youtrack issue search 'project: APP #Unresolved' --json
```

YouTrack performs filtering and sorting; the CLI preserves the quoted query and any explicit sort clause. Prefer bounded discovery with `--limit`; reserve `--all` for a genuinely complete collection.

## Saved searches

```bash
youtrack saved-search view 'Assigned to me'
youtrack saved-search view 51-33 --limit 20
youtrack saved-search view 'Release blockers' --all
```

`saved-search view` runs the visible server-side saved query. Result flags page its matching issues; the command is view-only.

## Comments

```bash
youtrack issue comment list TT-123 --limit 20 --json
youtrack issue comment add TT-123 --text 'Short update'
youtrack issue comment add TT-123 --file ./comment.md
youtrack issue comment edit TT-123 4-17 --text 'Revised update'
youtrack issue comment edit TT-123 4-17 --file ./revised-comment.md
youtrack issue comment remove TT-123 4-17
```

Comment lists use bounded pagination by default (50); use `--all` only for a complete scan. Prefer `--file` for substantial multiline text because its bytes are preserved exactly. Edit replaces only the selected comment text; it is distinct from reversible `deleted=true` removal. Restoration, permanent deletion, attachments, visibility, reactions, and pinning remain unsupported.

## Issue creation

```bash
youtrack issue create APP \
  --summary 'Login fails after token rotation' \
  --description-file ./description.md \
  --field Type=Bug \
  --field 'Fix versions=2026.2' \
  --field 'Fix versions=2026.3' \
  --field Assignee=@me \
  --tag backend \
  --json
```

`issue create` resolves the project, fields, and tags before one `POST /api/issues`. `--description` and `--description-file` are mutually exclusive; supplied description bytes are preserved exactly. Each `--field` splits on its first `=`, so commas are literal, and repeat a multi-value field to supply all values. `Board` and every state field are unavailable during creation. The returned issue uses the existing full issue output contract.

## Custom fields

```bash
youtrack issue field set TT-123 Priority Critical
youtrack issue field set TT-123 State "In Progress"
youtrack issue field set TT-123 Assignee @me
youtrack issue field clear TT-123 Assignee
```

Multi-value fields take multiple positional values:

```bash
youtrack issue field set TT-123 "Fix versions" 2026.2 2026.3
```

Supported MVP1 semantic kinds include string, integer, float, date, date-time, period, text, enum, state, user, version, build, owned, and group fields.

Periods support `m` and `h`, for example `30m` and `1h30m`.

## Combined edit

```bash
youtrack issue edit TT-123 \
  --summary "Improve authentication" \
  --description-file ./description.md \
  --field Priority=Critical \
  --field "State=In Progress" \
  --tag backend \
  --remove-tag legacy
```

For multi-value fields, repeat the field:

```bash
youtrack issue edit TT-123 \
  --field "Fix versions=2026.2" \
  --field "Fix versions=2026.3"
```

All supplied references are resolved before mutation. When the requested changes can be represented by one YouTrack issue update, the CLI sends one mutation request.

## Board membership

`Board` is a reserved synthetic multi-value field. It lists the issue's agile-board memberships; values are board database IDs or names, never sprint syntax. Setting it replaces the complete set, and YouTrack selects a board's current/default sprint:

```bash
youtrack issue field get TT-123 Board --json
youtrack issue field set TT-123 Board "Platform Board" "Operations Board"
youtrack issue field clear TT-123 Board
youtrack issue edit TT-123 --field "Board=Platform Board" --field "Board=Operations Board"
```

Board removals use YouTrack command parsing and can be rejected for query-driven sprint-disabled boards. The CLI validates the complete command first. A mixed edit validates all values first, then sends the ordinary issue update before the Board command; these two endpoints are not atomic, so an execution-time Board failure leaves a successful ordinary update intact. A real custom field named Board is addressable only by its project-field ID.

## Tags

```bash
youtrack issue tag list TT-123
youtrack issue tag add TT-123 backend
youtrack issue tag remove TT-123 backend
```

## Move an issue

```bash
youtrack issue move TT-123 PLATFORM
```

Project movement is deliberately separate from generic issue editing.

## Raw REST API

For endpoints not yet modeled by the CLI:

```bash
youtrack api /api/users/me

youtrack api /api/issues/TT-123 \
  --method POST \
  --data '{"summary":"Updated"}'
```

Only relative `/api` paths are accepted and the Authorization header cannot be overridden.

## Output

Normal output is human-oriented. `--json` returns the CLI's stable output model; it does not expose YouTrack REST DTOs. `--plain` is intended for shell pipelines.

```bash
youtrack issue view TT-123 --json
youtrack issue field get TT-123 Priority --plain
```

## Agent skill

The binary embeds an agent skill:

```bash
youtrack skill list
youtrack skill install
youtrack skill install --project
youtrack skill update
youtrack skill remove
```

The skill teaches coding agents to inspect before mutation, discover custom fields instead of guessing, and prefer structured commands over raw REST calls.

## Development

```bash
mise run fmt
mise run lint
mise run test
mise run test:race
mise run coverage
mise run ci
```

Tests use Testify assertions and handwritten fakes. REST tests use `httptest.Server`.

- [Release instructions](RELEASE.md)

## Design/specification

- [Overview](spec/mvp1/00-overview.md)
- [Architecture](spec/mvp1/01-architecture.md)
- [Custom fields](spec/mvp1/02-custom-fields.md)
- [CLI contract](spec/mvp1/03-cli-contract.md)
- [YouTrack REST adapter](spec/mvp1/04-youtrack-api.md)
- [Config/auth/output](spec/mvp1/05-config-auth-output.md)
- [Agent skill](spec/mvp1/06-agent-skill.md)
- [Testing/release](spec/mvp1/07-testing-release.md)
