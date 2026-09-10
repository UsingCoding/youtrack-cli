# YouTrack CLI MVP1 — Overview

## Goal

`youtrack` is an entity-oriented Go CLI inspired by JetBrains TeamCity CLI. MVP1 focuses on safe issue inspection and mutation for humans, shell scripts, and coding agents.

## Stack

- Go 1.27
- `github.com/urfave/cli/v3`
- `github.com/BurntSushi/toml`
- `github.com/stretchr/testify`
- `golang.org/x/term`
- `net/http`, `httptest`, `log/slog`, `embed`
- mise, golangci-lint, GoReleaser

## MVP1 scope

- profiles and permanent-token authentication
- issue view
- summary and description editing
- custom-field list/get/set/clear
- tag list/add/remove
- combined `issue edit`
- move issue to another project
- stable JSON and shell-friendly plain output
- authenticated raw REST escape hatch
- embedded coding-agent skill

## Non-goals

Issue creation/deletion, comments, attachments, links, work items, issue search, boards/agile, project mutation, user administration, saved searches, workflows, bulk updates, and a TUI are outside MVP1.

## Core invariants

1. CLI/domain code models YouTrack concepts, not REST JSON.
2. REST DTOs and `$type` values stay inside `internal/youtrack`.
3. Semantic field kind is separate from REST `IssueCustomField.$type`.
4. Every collection adapter handles pagination.
5. Project custom-field values are resolved against project metadata/bundles.
6. `issue edit` resolves and validates all values before any mutation.
7. Combined issue edits use one `POST /api/issues/{issue}` when representable as one issue update.
8. Project movement is a dedicated operation.
9. JSON output uses explicit compatibility-sensitive output DTOs.
10. Tests use Testify assertions and handwritten fakes by default.

## Definition of done

- acceptance commands in `03-cli-contract.md` work
- overall test coverage target >= 80%
- resolver target >= 95%
- `go test -race ./...` passes
- no token leakage
- unknown REST custom-field types do not break issue reads
- invalid combined edit performs zero mutation requests
- valid combined edit performs one issue update request
- embedded skill matches actual CLI behavior
- `mise run ci` passes
