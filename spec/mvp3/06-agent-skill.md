# Embedded Agent Skill

This specification does not change the currently embedded skill: it must describe implemented commands, not future ones. Each implementation epic updates `skills/youtrack-cli/SKILL.md`, `skills/youtrack-cli/references/commands.md`, and `skills/youtrack-cli/references/workflows.md` alongside the working feature. Keep MVP1/MVP2 guidance intact.

## Spent-time guidance

Teach agents to:

- use `issue time`, not direct edits to a total Spent time field
- discover project work-item types and list existing entries before editing/removing an entry
- use database work-item IDs from JSON, never row numbers
- supply an explicit calendar date and positive minute/hour duration; do not assume a workday length
- treat duration edits as replacement, not additional logged time
- distinguish attributed author from creator and respect server permission failures
- omit unknown attributes and unchanged fields, including types no longer offered by the project
- recognize that `remove --yes` is permanent and never use it as a soft-delete operation
- inspect current entries after an uncertain mutation outcome instead of blindly repeating an add

## Link guidance

Teach agents to:

- discover server link labels and IDs with `issue link types --all --json`
- resolve direction relative to the first issue, especially `subtask of` versus `parent for`
- use type ID plus explicit direction when labels are ambiguous
- never assume English built-in labels or hard-code REST `s`/`t` suffixes
- list the selected relation with bounded discovery or `--all` when complete membership is needed
- recognize existing-add/missing-remove as no-op success
- understand that link removal deletes only a relationship
- never issue a second reciprocal mutation or silently reparent an issue

## Browser guidance

Teach agents to:

- prefer `--print-url --json` in unattended/headless sessions
- launch a browser only when requested by the user or appropriate for an interactive workflow
- use `issue open <issue>` or the exact `issue search '<query>' open` suffix form
- understand that `issue search 'open'` is still an ordinary query
- use `saved-search open` to hand off its current stored query without downloading its results
- omit pagination flags in browser search mode
- preserve query text, quote shell-sensitive input, and never append tokens to URLs
- distinguish OS dispatch success from a loaded/authenticated web page
- know that browser login may differ from CLI authentication
- split mutation and browser opening into explicit commands; a launcher failure must never motivate repeating an issue creation

## Output and safety

Use JSON for structured work items and relations. Plain list output is intentionally identifier-oriented where text would be ambiguous. Explain permanent work-item deletion separately from existing reversible comment removal. Update examples against [03-cli-contract.md](03-cli-contract.md), including global flags at nested commands and token-free direct browser search.
