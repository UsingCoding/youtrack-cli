---
name: youtrack-cli
version: 0.1.0
description: Use when working with YouTrack issues from a coding agent; drives the `youtrack` CLI for issue inspection, custom fields, tags, project moves, and raw API access.
---

# YouTrack CLI (`youtrack`)

## Quick start

```bash
youtrack auth status
youtrack issue view TT-123 --json
youtrack issue field list TT-123 --json
```

Do not guess command flags, custom-field names, enum/state/version values, or users. Use `youtrack <command> --help`, `issue field list`, and `issue field get` to inspect the current issue before mutating it.

## Rules

- Inspect an issue before changing it.
- Prefer structured commands over `youtrack api`.
- Use `--json` when consuming results programmatically.
- Use `issue move` for project changes; do not try to edit the project as a normal field.
- Use `issue tag add/remove` for isolated tag changes.
- Use `issue edit` when several issue changes should be validated and applied together.
- Use `@me` only for user fields and `@none` to clear nullable fields.
- Re-read the issue after a meaningful mutation when verification matters.

## Core commands

| Area | Commands |
| --- | --- |
| Auth | `auth login`, `auth logout`, `auth status` |
| Issue | `issue view`, `issue edit`, `issue move` |
| Fields | `issue field list/get/set/clear` |
| Tags | `issue tag list/add/remove` |
| API | `api <endpoint>` |
| Config | `config list/get/set` |

See `references/commands.md`, `references/fields.md`, and `references/workflows.md` for details.
