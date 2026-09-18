# Fields

The CLI resolves human values against project-specific YouTrack field metadata. Do not construct YouTrack `$type` JSON manually unless using the raw API escape hatch.

Supported custom-field kinds: string, integer, float, date, date-time, period, text, enum, state, user, version, build, owned field, and group.

Examples:

```bash
youtrack issue field set TT-123 Priority Critical
youtrack issue field set TT-123 State "In Progress"
youtrack issue field set TT-123 Assignee @me
youtrack issue field set TT-123 "Fix versions" 2026.2 2026.3
youtrack issue field clear TT-123 Assignee
```

`@none` clears a nullable custom field. Period input in MVP supports hours and minutes, e.g. `2h`, `30m`, `1h30m`.

## Creation restrictions

The issue-field commands and `issue edit` support state fields, including state-machine transitions on existing issues. `issue create` does not: every state field is rejected, and it also rejects the reserved synthetic `Board` field. Use only supported non-state project fields during creation; inspect the returned issue before relying on workflow defaults.

## Board membership

`Board` is a reserved synthetic multi-value field. `issue field set <issue> Board <board>...` and repeated `--field Board=<board>` replace the entire membership set; `clear` (or `@none` alone) removes all memberships. Use a board database ID or name only. Do not provide a sprint: YouTrack chooses the board's current/default sprint. Query-driven sprint-disabled board removals can be rejected by YouTrack command parsing. A project custom field named Board remains addressable by its project-field ID only.
