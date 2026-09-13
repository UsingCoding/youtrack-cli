# Workflows

## Safe issue update

```bash
youtrack issue view TT-123 --json
youtrack issue field list TT-123 --json
youtrack issue edit TT-123 --field Priority=Critical --tag backend
youtrack issue view TT-123 --json
```

`issue edit` resolves every field and tag before mutation. Use it when applying several changes together.

## Board update

```bash
youtrack issue field get TT-123 Board --json
youtrack issue field set TT-123 Board "Platform Board"
youtrack issue field clear TT-123 Board
```

For a mixed update, use `issue edit ... --field Board=...`. All values and Board command parsing are validated before writes. The normal issue POST precedes the Board command, so this mixed operation is not atomic after validation; re-read the issue when command execution matters.

## Move an issue

```bash
youtrack issue view TT-123 --json
youtrack issue move TT-123 PLATFORM
youtrack issue view TT-123 --json
```

Moving is intentionally separate because the set of available custom fields can change with the project.
