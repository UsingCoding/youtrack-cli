# Workflows

## Search then inspect

```bash
youtrack issue search 'project: APP #Unresolved sort by: updated desc' --limit 20 --json
youtrack issue view APP-123 --json
youtrack issue field list APP-123 --json
```

Use a quoted bounded search first. Select the readable ID from the server-ordered results, then inspect the issue and fields before mutating it.

## Saved search then inspect

```bash
youtrack saved-search view 'Release blockers' --limit 20 --json
youtrack issue view APP-123 --json
youtrack issue field list APP-123 --json
```

Use an existing visible saved search when its server-side filter captures the intended work. Inspect server-ordered results; do not copy or recreate the filter locally.

## Create then inspect

```bash
youtrack issue create APP \
  --summary 'Describe the failure clearly' \
  --description-file ./description.md \
  --field Priority=Critical \
  --tag backend \
  --json
youtrack issue view APP-123 --json
```

Choose the project deliberately and use known project field and tag values. Creation resolves all supplied values before one POST, but workflow defaults are server-controlled; inspect the returned issue or re-read it before work that depends on its state or Board membership.

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

## Comment workflow

```bash
youtrack issue comment list APP-123 --limit 20 --json
youtrack issue comment add APP-123 --file ./comment.md
youtrack issue comment edit APP-123 4-17 --file ./revised-comment.md
youtrack issue comment remove APP-123 4-17
youtrack issue comment list APP-123 --limit 20 --json
```

List first to obtain the server comment ID. Use a file for multiline text. Editing replaces text only; it does not restore a removed comment. Re-list after adding, editing, or removing when verification matters; removal is reversible soft removal.
