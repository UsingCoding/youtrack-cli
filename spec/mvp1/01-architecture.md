# Architecture

## Repository layout

```text
youtrack-cli/
├── cmd/youtrack/main.go
├── internal/
│   ├── cli/
│   ├── app/
│   ├── domain/
│   ├── youtrack/
│   │   └── dto/
│   ├── config/
│   ├── credentials/
│   ├── output/
│   ├── skill/
│   └── version/
├── skills/youtrack-cli/
├── spec/mvp1/
├── integration/
├── testdata/
├── mise.toml
├── .golangci.yml
├── .goreleaser.yaml
├── AGENTS.md
└── README.md
```

## Dependency direction

```text
cli -> app -> domain
youtrack -> app ports + domain
output -> domain
config, credentials, skill -> independent utilities
```

Forbidden dependencies:

- domain -> CLI/config/YouTrack
- app -> REST DTOs
- CLI -> REST DTOs
- output -> REST DTOs

Interfaces are declared by their consumer in `internal/app/ports.go`.

## Main application ports

`IssueStore` provides read/update/move/tag operations. `ProjectFieldStore` provides project field definitions, bundle values, and allowed field users. `ProjectStore`, `TagStore`, `UserStore`, and `GroupStore` provide reference resolution.

Application services receive those interfaces and never instantiate HTTP/config clients themselves.

## Mutation model

Application code builds an `IssuePatch` containing summary, description, resolved field assignments, and an optional complete desired tag set. The YouTrack adapter serializes semantic values into the wire-level `$type` and value shapes.

Combined edit flow:

```text
GET issue
 -> load/cache project metadata
 -> resolve every field
 -> resolve every tag
 -> validate all changes
 -> calculate final tags
 -> one POST /api/issues/{issue}
 -> render returned issue
```

A validation error before the POST must leave the issue untouched.

## Command-scoped caches

The field resolver caches within a command invocation:

- project field definitions
- bundle options
- allowed field users
- current authenticated user

No persistent cache is part of MVP1.

## Concurrency

Mutation operations are sequential. Do not parallelize writes. Metadata lookup optimization is secondary to predictable behavior.
