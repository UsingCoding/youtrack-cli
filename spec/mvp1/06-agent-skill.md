# Embedded Agent Skill

The binary embeds `skills/**` with Go `embed`.

```text
skills/youtrack-cli/
├── SKILL.md
└── references/
    ├── commands.md
    ├── fields.md
    └── workflows.md
```

Commands:

```text
youtrack skill list
youtrack skill install
youtrack skill install --project
youtrack skill update
youtrack skill remove
```

The skill teaches workflows, not REST internals:

- check authentication
- inspect before mutation
- prefer structured commands over `api`
- use `--json` for agent consumption
- never guess field names or enum/state/version values
- use field list/get when uncertain
- use `issue move` for project movement
- verify meaningful mutations by reading the issue again when appropriate
- inspect and mutate Board like a field using Board IDs/names only; do not invent sprint syntax
- understand that Board set replaces the membership set and mixed Board edits are non-atomic after validation

When command behavior changes, update the skill/reference docs in the same change.
