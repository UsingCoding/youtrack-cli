---
name: youtrack-cli
version: 0.2.0
description: Use when working with YouTrack issues, visible saved searches, and comments from a coding agent; drives the `youtrack` CLI for inspection, fields including Board membership, tags, project moves, comment management, and raw API access.
---

# YouTrack CLI (`youtrack`)

## Quick start

```bash
youtrack auth status
youtrack issue search 'project: APP #Unresolved' --limit 20 --json
youtrack issue view TT-123 --json
youtrack issue create APP --summary 'Clear reproduction steps' --description-file ./description.md --json
youtrack saved-search view 'Assigned to me' --limit 20 --json
youtrack issue comment list TT-123 --limit 20 --json
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
- For `issue create`, choose the project deliberately, use a clear summary, use `--description-file` for substantial multiline content, resolve field and tag values rather than guessing, and inspect the returned issue when workflow defaults matter.
- Do not supply `Board` or any state field to `issue create`; creation does not accept attachments, links, comments, or notification inputs.
- Treat `Board` as a reserved multi-value field: inspect it first, use board IDs/names only, and remember that set replaces the full membership set.
- Do not invent sprint syntax. YouTrack selects the board's current/default sprint.
- A mixed normal-field and Board edit validates first but is not transactionally atomic if Board command execution later fails.
- Quote the single `issue search` query, use YouTrack's server-side search rather than local filtering, and preserve explicit server sorting.
- Prefer a bounded `--limit` for discovery; reserve `--all` for necessary full scans.
- Use `saved-search view` with an existing visible saved search instead of recreating its server-side filter locally.
- Saved searches are view-only: do not use structured create, update, delete, or sharing operations.
- List comments before choosing a comment ID; use `--file` for substantial replacement text.
- Do not put permanent tokens in command logs.
- `issue comment edit` replaces text only; `issue comment remove` is reversible soft removal, not restoration or permanent deletion.

## Core commands

| Area | Commands |
| --- | --- |
| Auth | `auth login`, `auth logout`, `auth status` |
| Issue | `issue search`, `issue view`, `issue create`, `issue edit`, `issue move` |
| Comments | `issue comment list/add/edit/remove` |
| Fields | `issue field list/get/set/clear` |
| Saved searches | `saved-search view` |
| API | `api <endpoint>` |
| Config | `config list/get/set` |

See `references/commands.md`, `references/fields.md`, and `references/workflows.md` for details.
