# Custom fields

The CLI resolves human values against project-specific YouTrack field metadata. Do not construct YouTrack `$type` JSON manually unless using the raw API escape hatch.

Supported MVP field kinds: string, integer, float, date, date-time, period, text, enum, state, user, version, build, owned field, and group.

Examples:

```bash
youtrack issue field set TT-123 Priority Critical
youtrack issue field set TT-123 State "In Progress"
youtrack issue field set TT-123 Assignee @me
youtrack issue field set TT-123 "Fix versions" 2026.2 2026.3
youtrack issue field clear TT-123 Assignee
```

`@none` clears a nullable field. Period input in MVP supports hours and minutes, e.g. `2h`, `30m`, `1h30m`.
