# AGENTS.md

## Project

`youtrack-cli` is an entity-oriented CLI for JetBrains YouTrack. MVP1 focuses on issue reads and mutations and is designed for both humans and coding agents.

## Development interface

Use mise:

```bash
mise install
mise run fmt
mise run lint
mise run test
mise run test:race
mise run coverage
mise run build
mise run ci
```

## Architecture

Dependency direction:

```text
cli -> app -> domain
youtrack -> app ports + domain
output -> domain
```

`internal/config`, `internal/credentials`, and `internal/skill` are supporting packages.

### Hard boundaries

- `internal/domain` contains no REST DTOs and no JSON compatibility contract.
- `internal/app` contains application behavior and consumer-owned ports.
- `internal/youtrack` is the only package that knows YouTrack REST `$type` names and wire shapes.
- `internal/cli` does not construct REST payloads.
- REST DTOs must never escape `internal/youtrack`.
- Normal JSON output is built from explicit DTOs in `internal/output`.

## Custom fields

Semantic field kind and YouTrack REST `$type` are different concepts. Never put values such as `SingleEnumIssueCustomField` into domain/application logic.

Field values must be resolved from the project field definition. Bundle-backed fields resolve against the project field's bundle. User fields resolve against the allowed `UserBundle`, not all users.

All collection endpoints must handle pagination. Do not assume YouTrack returns all values in one response; its common default collection limit is 42.

Unknown REST custom-field types must remain readable. Unsupported mutations should fail cleanly. State-machine-managed state fields must resolve and serialize transition events from the issue's `possibleEvents`; do not treat them as ordinary state bundle assignment.

## Mutation safety

For combined `issue edit`:

1. Read the issue.
2. Resolve all field metadata and values.
3. Resolve tags.
4. Validate everything.
5. Perform one issue update POST when possible.

If any supplied input is invalid, no mutation must have happened.

Project movement remains a dedicated operation through `issue move`.

## Tests

Use Testify:

```go
require.NoError(t, err)
assert.Equal(t, want, got)
```

Prefer handwritten fakes over `testify/mock`.

High-value tests cover field resolution, pagination, REST serialization, error mapping, configuration precedence, output contracts, and the validate-before-write property.

Targets:

- overall >= 80%
- app >= 90%
- field resolver >= 95%
- YouTrack adapter >= 85%
- config >= 90%

## CLI/output compatibility

Within a major release, treat command names, flags, stable JSON field names, plain output semantics, and exit-code meanings as compatibility-sensitive. Human table formatting is not stable.

Global flags must work from nested commands.

## Agent skill

When command behavior changes, update:

```text
skills/youtrack-cli/SKILL.md
skills/youtrack-cli/references/*.md
```

When architecture or MVP behavior changes, update the relevant file under `spec/mvp1/` in the same change.

## Security

Never print or log permanent tokens or Authorization headers. Do not silently reuse a stored profile token when the server URL has been explicitly overridden.
