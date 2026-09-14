# Configuration, Authentication, and Output

## Configuration and authentication

MVP2 adds no configuration keys, environment variables, credential behavior, or authentication flow. The precedence and explicit-URL credential safety rule from [`spec/mvp1/05-config-auth-output.md`](../mvp1/05-config-auth-output.md) remain unchanged.

Search text, issue descriptions, and comment text are request data. Debug logging must not emit Authorization headers or tokens. Normal debug policy may identify methods and paths without dumping authenticated headers.

## Stable JSON models

Domain structs remain free of JSON tags. `internal/output` maps them to explicit DTOs.

### Issue search

`issue search --json` returns an array. Each item is:

```json
{
  "id": "APP-123",
  "entityId": "2-31",
  "summary": "Fix token rotation",
  "project": {
    "entityId": "0-1",
    "name": "Application",
    "shortName": "APP"
  },
  "created": "2026-09-14T10:00:00Z",
  "updated": "2026-09-14T12:00:00Z",
  "resolved": null
}
```

`resolved` is always present and is either an RFC3339 timestamp or `null`. An empty result is `[]`, not `null`.

### Saved-search view

`saved-search view --json` returns:

```json
{
  "entityId": "51-33",
  "name": "Assigned to me",
  "query": "for: me #Unresolved",
  "owner": {
    "entityId": "1-2",
    "login": "jane.doe",
    "name": "Jane Doe"
  },
  "issues": []
}
```

`owner` is nullable. `issues` contains the same issue-summary objects as `issue search` and is always an array.

### Issue creation

`issue create --json` returns the existing stable full issue JSON object defined by MVP1. Creation does not introduce a second issue schema.

### Comments

`issue comment list --json` returns an array. `issue comment add --json` returns one object of the same shape:

```json
{
  "entityId": "4-17",
  "author": {
    "entityId": "1-2",
    "login": "jane.doe",
    "name": "Jane Doe"
  },
  "text": "Ready for review.",
  "created": "2026-09-14T12:30:00Z",
  "updated": null,
  "deleted": false
}
```

`author`, `text`, and `updated` are nullable and always present. An empty comment list is `[]`.

`issue comment remove --json` returns:

```json
{
  "entityId": "4-17",
  "removed": true
}
```

This result means the soft-removal update succeeded; it does not claim permanent deletion.

## Human output

Human output is terminal-oriented and not formatting-stable.

- issue search: table with readable ID, project short name, updated time, and summary
- saved-search view: saved-search name, stored query, optional owner, then the issue table
- issue create: existing full issue renderer
- comment list: readable blocks containing comment ID, author, timestamps, deletion state, and text
- comment add: one comment block
- comment remove: concise soft-removal confirmation

Multiline descriptions and comments are preserved in human output.

## Plain output

Plain output contains no ANSI or labels.

- issue search: one line per issue as `<readable-id>\t<summary>`
- saved-search view: the same issue lines, with no metadata header
- issue create: existing `<readable-id>\t<summary>` issue line
- comment list: one comment entity ID per line
- comment add: the created comment entity ID followed by a newline
- comment remove: no output on success

Comment text is intentionally not placed in plain list output because embedded tabs and newlines would violate the one-record-per-line contract. Use JSON when comment content is needed by a script or agent.

## Compatibility

Within the current major release, JSON field names, nullability, array/object shape, plain columns, and empty-output behavior above are compatibility-sensitive. Human table layout is not.
