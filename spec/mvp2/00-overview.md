# YouTrack CLI MVP2 — Overview

## Goal

MVP2 extends the MVP1 issue workflow with server-side issue search, read-only saved-search execution, issue creation, and comment management. MVP1 commands and compatibility guarantees remain in force unless this specification explicitly adds a contract.

## Baseline

MVP2 is additive to [`spec/mvp1`](../mvp1/00-overview.md). It keeps the existing dependency boundaries, authentication and configuration precedence, semantic custom-field model, output modes, exit codes, pagination rule, and mutation-safety requirements.

## MVP2 scope

- execute YouTrack issue queries using the same query language as the YouTrack search panel
- view a visible saved search and its current issue results
- create an issue with project, summary, description, supported custom fields, and existing tags
- list issue comments
- add text comments
- remove comments reversibly
- stable JSON and shell-friendly plain output for every new command
- updated embedded coding-agent skill

## Non-goals

- creating, editing, deleting, or sharing saved searches
- parsing, validating, completing, or rewriting YouTrack search syntax in the CLI
- issue deletion, drafts, bulk issue creation, or bulk mutation
- attachments on issues or comments
- comment editing, permanent deletion, restoration, visibility controls, reactions, or pinning
- setting Board membership or a state field during issue creation
- links, work items, votes, watchers, project mutation, user administration, or a TUI

The authenticated raw REST command remains the escape hatch for capabilities outside this scope.

## Core invariants

1. Every MVP1 architecture and security boundary remains valid.
2. A direct search query is an opaque string. The server, not the CLI, defines its syntax, matching, and sort semantics.
3. Search and comment result limits are explicit and bounded by default. `--all` paginates until an empty page; no collection assumes a short page is final.
4. Saved searches are read-only. A saved-search name is resolved across all visible saved searches before its query is executed.
5. A null or blank saved-search query is rejected and is never converted into an unfiltered issue search.
6. Issue creation resolves the project, every supplied custom-field value, and every tag before its only mutation request.
7. A valid issue creation uses one `POST /api/issues`; tags and custom fields are part of that request. No follow-up write is allowed.
8. `Board` and state fields are rejected during creation because they cannot be safely represented with the pre-create information available to the CLI.
9. Comment removal is a reversible soft removal using `deleted=true`, not permanent REST deletion.
10. REST DTOs, `$type` values, and wire-level saved-query terminology remain inside `internal/youtrack`.
11. JSON output is built from explicit compatibility-sensitive output DTOs.
12. Tokens and Authorization headers are never printed or logged.

## Definition of done

- all acceptance commands in [`03-cli-contract.md`](03-cli-contract.md) work
- direct search forwards the exact supplied YouTrack query and preserves server ordering
- limited and `--all` searches work across multiple API pages
- saved-search resolution handles more than 42 visible saved searches and rejects ambiguous names
- invalid issue-create input performs zero mutation requests
- valid issue creation performs exactly one mutation request with project, summary, description, fields, and tags in one body
- comment listing handles pagination
- comment add creates a text-only comment
- comment remove performs a soft removal and never calls HTTP `DELETE`
- stable JSON output matches [`03-cli-contract.md`](03-cli-contract.md)
- embedded skill documentation matches the implemented commands
- `mise run ci` passes
