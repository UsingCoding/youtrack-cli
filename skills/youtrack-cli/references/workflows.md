# Workflows

## Safe issue update

```bash
youtrack issue view TT-123 --json
youtrack issue field list TT-123 --json
youtrack issue edit TT-123 --field Priority=Critical --tag backend
youtrack issue view TT-123 --json
```

`issue edit` resolves every field and tag before mutation. Use it when applying several changes together.

## Move an issue

```bash
youtrack issue view TT-123 --json
youtrack issue move TT-123 PLATFORM
youtrack issue view TT-123 --json
```

Moving is intentionally separate because the set of available custom fields can change with the project.
