# Embedded Agent Skill

MVP2 updates the embedded skill and its command/workflow references in the same implementation change. The MVP1 guidance remains valid.

## Search guidance

The skill teaches agents to:

- pass valid YouTrack search syntax as one quoted query
- use `issue search` instead of recreating server-side filtering locally
- preserve explicit sorting in the query when deterministic order matters
- use `--limit` for bounded discovery and `--all` only when the full collection is necessary
- use `--json` for structured consumption
- use `saved-search view` to execute an existing visible saved search
- never attempt to create, update, or delete a saved search through structured MVP2 commands

## Issue-creation guidance

The skill teaches agents to:

- resolve the target project deliberately
- provide a clear non-blank summary
- use `--description-file` for substantial multiline Markdown
- use repeated `--field` values for multi-value fields
- never guess field names or enum, version, user, group, or tag values
- expect every supplied value to be validated before creation
- omit state to let YouTrack choose its initial value; change it later through an available transition
- never supply `Board` during create
- understand that valid creation uses one mutation request
- inspect the returned issue when subsequent work depends on workflow-applied values

## Comment guidance

The skill teaches agents to:

- use `issue comment list` before acting on a referenced comment ID
- use `--file` for substantial multiline comments
- use comments for useful issue context, not command logs or token-bearing data
- understand that `comment remove` is reversible soft removal
- never claim a comment was permanently deleted
- never attempt comment attachments, edits, visibility changes, reactions, restoration, or pinning through MVP2 structured commands

## Safety and output

The skill continues to prefer structured commands over raw REST and to avoid exposing permanent tokens. It calls out that direct search strings and comment/description text may contain shell-sensitive characters and should be quoted or supplied from files.

Skill command and JSON examples must match [`03-cli-contract.md`](03-cli-contract.md).
