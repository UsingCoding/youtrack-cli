# CLI and Output Contract

## Command additions

```text
youtrack
├── issue
│   ├── open
│   ├── search <query> [open]
│   ├── time
│   │   ├── types
│   │   ├── list
│   │   ├── view
│   │   ├── add
│   │   ├── edit
│   │   └── remove
│   └── link
│       ├── types
│       ├── list
│       ├── add
│       └── remove
└── saved-search
    └── open
```

All existing commands remain. `issue search` gains an optional positional mode, not a child command. All global flags work at every nested command. `--json` and `--plain` remain mutually exclusive. Repeatable values do not split commas.

In signatures below, `[page flags]` means `[--limit <positive-n>] [--offset <non-negative-n>] [--all]`, with defaults 50/0 and `--all` conflicting with an explicitly supplied limit. `[relation]` means required `--type <type-id|type-name|direction-label>` and optional `--direction outward|inward`, subject to [direction resolution](02-feature-behavior.md#discovery-and-direction).

## Spent time

```text
youtrack issue time types <issue> [page flags]
youtrack issue time list <issue> [page flags]
youtrack issue time view <issue> <work-item-id>
youtrack issue time add <issue> --duration <period> --date <YYYY-MM-DD>
  [--text <text> | --file <path>] [--type <type>] [--author <user>]
youtrack issue time edit <issue> <work-item-id>
  [--duration <period>] [--date <YYYY-MM-DD>]
  [--text <text> | --file <path>] [--type <type> | --clear-type] [--author <user>]
youtrack issue time remove <issue> <work-item-id> --yes
```

Edit requires at least one change. Duration is positive integer minutes/hours (`45m`, `2h`, `1h30m`); date is required on add. Removal is permanent; `--yes` is a required boolean acknowledgement, not a request for interactive confirmation. Add does not accept `--clear-type`.

```bash
youtrack issue time types APP-123 --all --json
youtrack issue time add APP-123 --duration 1h30m --date 2026-09-29 \
  --type Development --text 'Implement token refresh' --json
youtrack issue time list APP-123 --all --json
youtrack issue time view APP-123 115-7
youtrack issue time edit APP-123 115-7 --duration 2h --text ''
youtrack issue time remove APP-123 115-7 --yes
```

## Links

```text
youtrack issue link types [page flags]
youtrack issue link list <issue> [relation] [page flags]
youtrack issue link add <issue> <target-issue> [relation]
youtrack issue link remove <issue> <target-issue> [relation]
```

`--type` is required for list/add/remove. Only one target is accepted. Use discovered labels, not assumed English aliases. Type ID/name selection requires direction for directed types. Remove deletes an edge only, so it does not require `--yes`.

```bash
youtrack issue link types --all --json
youtrack issue link add APP-124 APP-123 --type 'subtask of'
youtrack issue link list APP-123 --type 'parent for' --all
youtrack issue link add APP-123 APP-125 --type 'relates to'
youtrack issue link list APP-123 --type 80-2 --direction outward --limit 20
youtrack issue link remove APP-124 APP-123 --type 'subtask of'
```

Names and IDs above are illustrative and must be taken from the target server's discovery output.

## Browser handoff

```text
youtrack issue open <issue> [--print-url]
youtrack issue search <query> open [--print-url]
youtrack saved-search open <saved-search> [--print-url]
```

```bash
youtrack issue open APP-123
youtrack issue search 'project: APP #Unresolved sort by: updated desc' open
youtrack issue search 'open'                  # existing REST search
youtrack issue search 'open' open             # browser search
youtrack issue search -- '-State: Done' open
youtrack issue search 'project: APP' open --print-url --plain
youtrack saved-search open 'Assigned to me'
youtrack issue open APP-123 --print-url --json
```

Browser search rejects explicitly supplied `--limit`, `--offset`, and `--all`. Without the suffix, search keeps its existing JSON/plain output and pagination. `--print-url` is valid only for opening commands/browser search mode. There is no `issue search open <query>` alternative.

## Stable JSON output

JSON never directly serializes REST/domain structs. All keys listed below are always present unless explicitly stated otherwise. Nullable keys use `null`; empty collections use `[]`. Timestamps use UTC RFC3339; date-only values use `YYYY-MM-DD`. Identity objects use `entityId`, not REST `id`.

### Work items

List returns an array; view/add/edit return one object:

```json
{
  "entityId": "115-7",
  "date": "2026-09-29",
  "durationMinutes": 90,
  "text": "Implement token refresh",
  "author": {"entityId": "1-2", "login": "alice", "name": "Alice"},
  "creator": {"entityId": "1-2", "login": "alice", "name": "Alice"},
  "type": {"entityId": "114-0", "name": "Development"},
  "created": "2026-09-29T12:00:00Z",
  "updated": null
}
```

`text`, `author`, `creator`, `type`, and `updated` are nullable. Types returns an array of `{ "entityId": "114-0", "name": "Development" }` objects. Remove returns `{ "entityId": "115-7", "removed": true }`, meaning permanent deletion was accepted.

### Link types

Types returns an array:

```json
[
  {
    "entityId": "80-2",
    "name": "Subtask",
    "directed": true,
    "aggregation": true,
    "outward": "parent for",
    "inward": "subtask of"
  }
]
```

For undirected types, `outward` is the common label and `inward` is `null`. A selected relation object is:

```json
{
  "type": {"entityId": "80-2", "name": "Subtask"},
  "direction": "inward",
  "label": "subtask of"
}
```

Direction is `outward`, `inward`, or `undirected`. List returns an object with `issueId` (source readable ID), `relation` (above), and `issues` (array using the exact MVP2 issue-summary schema). It never flattens different relations into one ambiguous page.

Add/remove return:

```json
{
  "issueId": "APP-124",
  "targetIssueId": "APP-123",
  "relation": {
    "type": {"entityId": "80-2", "name": "Subtask"},
    "direction": "inward",
    "label": "subtask of"
  },
  "present": true,
  "changed": true
}
```

`present` is the requested final edge state: true for add, false for remove. `changed` is false for the preflight no-op and true for a successful mutation request; it is not a claim about concurrent writers or server-internal work.

### Browser results

Every open operation returns:

```json
{"url":"https://example.youtrack.cloud/issue/APP-123","opened":true}
```

`opened` is true only after successful dispatch; `--print-url` returns false. Launcher failure returns nonzero and no success object. Ordinary REST search output is unchanged.

## Human and plain output

Human output is not layout-stable:

- time list/view/add/edit: ID, date, duration, type, author, and text; distinguish creator where useful
- link types: IDs, type names, directions/labels, and aggregation; link list: selected relation and issue table
- deletes: explicit permanent deletion confirmation; link changes distinguish no-op from change
- open: destination plus dispatch confirmation, or just destination in URL-only mode

Plain output contains no labels or ANSI. Stable lines:

| Operation | Plain stdout |
| --- | --- |
| time list/view/add/edit | One work-item entity ID per line |
| time types | One type entity ID per line |
| time remove | Empty on success |
| link types | One type entity ID per line |
| link list | MVP2 `<readable-id>\t<summary>` per linked issue |
| link add/remove | Empty on success |
| every open / URL-only operation | Exactly the destination URL and newline |

Use JSON for directions, durations, and multiline text. Empty lists emit no plain bytes. Open output is produced after dispatch succeeds unless `--print-url` was requested.

## Errors and side effects

Keep exit codes: 0 success, 1 runtime/API, 2 validation/usage, 3 authentication/configuration, 4 not-found, 5 ambiguous reference. Server permission failures retain the current adapter mapping; do not reclassify them as successful emptiness.

Missing `--yes`, invalid date/duration, contradictory directions, missing mutation flags, or incompatible browser pagination are usage failures before any write/launch. Failure after submitting a write may have an uncertain outcome; do not promise rollback or blindly suggest retrying creation. No new command prompts for input implicitly.
