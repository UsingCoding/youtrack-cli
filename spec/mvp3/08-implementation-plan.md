# MVP3 Feature-Sliced Implementation Plan

## Status and execution model

This is a plan, not an implementation report. All checklist items start unchecked. Implement each epic as an end-to-end slice across application/domain, adapter, CLI, output, focused verification, README, and embedded skill. Reuse [the source contracts](00-overview.md#specification-map).

Default merge order follows the three epics below: spent time, issue relationships, then browser handoff.

Create each implementation worktree from the integrated earlier changes. Shared composition files (`internal/app/ports.go`, service constructors, `internal/cli/runtime.go`, issue/root commands, output, and skill references) require one integration owner and sequential integration. Independent research can run alongside feature work; do not have two slices independently redesign the same runtime.

Each merged implementation slice must contain reachable behavior, not dormant ports, placeholder commands, compatibility constructors, or documentation claiming unsupported capabilities. API compatibility probes use only disposable resources.

## Epic checklist

- [ ] Epic 1 — Issue spent time CRUD and type discovery
- [ ] Epic 2 — Directed and undirected issue relationships
- [ ] Epic 3 — Browser handoff and URL-only workflows
- [ ] Release — Cross-feature compatibility and documentation verification

## Epic 1 — Issue spent time CRUD

### Product result

A user can discover an issue project's work-item types, list/read time entries, add a dated duration, change one entry, and permanently delete one with acknowledgement. The CLI never treats logging time as assignment to a total field.

### Implementation

1. Add semantic work-item/type models and narrow app-owned store ports, keeping creator separate from author and omission separate from clearing.
2. Reuse existing minute/hour parsing semantics with checked arithmetic; add a work-date conversion distinct from MVP1 custom date-field midday handling.
3. Resolve issue/project and project-scoped types; validate positive durations, explicit dates, text source exclusivity, enabled settings for add/edit, and author references before one write.
4. Implement the documented list/read/create/partial-update/delete endpoints, project settings read, and paged project type discovery. Do not write project settings or unknown work attributes.
5. Register `issue time types/list/view/add/edit/remove`; reuse pagination and nested global flag helpers. Require `--yes` for permanent remove in every mode.
6. Add explicit work-item/type output DTOs and identifier-oriented plain output. Do not change full issue JSON or add per-issue time enrichment.
7. Update README and agent commands/workflows with explicit dates, replace-not-increment editing, and permanent deletion semantics.

### Complete when

- the built CLI performs the full create/read/update/delete lifecycle against a disposable fixture/server
- a type invalid on the target project, an invalid late author, or a date/duration overflow results in zero writes
- short-page pagination, UTC midnight dates, clear-versus-omit behavior, and non-retried writes have focused coverage
- supported-version acceptance verifies nullable type clearing, author changes, and server-computed totals
- command/output/help/skill contracts agree

## Epic 2 — Issue relationships

### Product result

A user can discover custom relations, inspect one relation's linked issues, and add/remove directed or undirected edges, including parent/subtask relationships, without reversing direction or duplicating writes.

### Implementation

1. Add semantic link-type/relation models and lightweight issue identity resolution where needed.
2. Implement paginated type discovery and the [type/name/label precedence](02-feature-behavior.md#discovery-and-direction), preserving direction as an application concept.
3. Implement selected-relation linked-issue pages using existing issue-summary DTOs, not embedded trimmed link arrays.
4. Resolve both issues/type and reject self-link aliases before mutation. Fully inspect selected-relation membership for documented no-op behavior.
5. Encode adapter-only `s`/`t` suffixes and use one typed resource POST/DELETE. Do not use command strings or write the reciprocal edge separately.
6. Register `issue link types/list/add/remove`, output relation metadata and result state, and document required `--type` plus conditional `--direction`.
7. Update skill examples to discover server-specific labels and warn against implicit reparenting.

### Complete when

- the built CLI creates, lists from both sides, and removes a subtask relation without deleting issues
- an undirected relation and a custom/ambiguous label work according to the contract
- invalid target/type/direction causes zero writes and late-page existing edges produce no-op results
- pagination/output/error contracts and no reciprocal write are verified

## Epic 3 — Browser handoff

### Product result

The requested issue/search opening commands and an additional saved-search opening command launch the default browser safely or print the same destination for headless callers, without altering existing REST search behavior.

### Implementation

1. Separate reusable URL/profile selection from token-requiring runtime setup without duplicating configuration precedence or weakening explicit-URL credential isolation.
2. Add lightweight issue identity resolution and reuse saved-search metadata resolution without executing a search. Keep direct query opening token-free and HTTP-free.
3. Add one origin/context-preserving URL builder for issue and query destinations. Reject unsafe service bases and encode query/path values exactly once.
4. Add an injectable OS opener using direct native/process invocation, covering current macOS/Linux/Windows targets without a shell. Avoid arbitrary `BROWSER` command evaluation.
5. Register `issue open` and `saved-search open`. Extend existing search argument parsing to recognize only a second literal `open`, preserving one-argument queries including the word `open`.
6. Reject explicit pagination in browser mode and browser-only flags in REST mode. Keep nested global flags working.
7. Implement `--print-url`, explicit URL/opened JSON, one-line plain output, and truthful launch failure behavior. JSON/plain alone do not suppress launch.
8. Update README and skill with headless workflows, saved-query snapshot semantics, separate browser authentication, and no automatic mutation retry after launch failure.

### Complete when

- real built-CLI parsing distinguishes REST and browser modes without breaking MVP2 acceptance
- URL-only mode works without an opener; direct browser search works without a token or credentials read
- reserved characters and self-hosted context paths round-trip correctly without origin changes
- real OS handoff and supported YouTrack web routes are observed, not only fake-opener calls
- launch failures never render `opened:true` or cause a preceding mutation to repeat

## Release — Integration verification

1. Execute [the verification matrix](07-testing-release.md), including every epic's actual CLI smoke and high-value regressions.
2. Record supported YouTrack versions/API evidence and supported-platform browser dispatch results.
3. Run repository formatting/lint/tests/race/build/CI after integration; preserve existing coverage targets and release artifacts.
4. Review README, embedded skill, and references against shipped commands. Do not update MVP1/MVP2 specs to pretend MVP3 already existed there; keep this additive scope and any explicit compatibility decisions here.
5. Remove throwaway probes/fixtures after proof and ensure no incomplete behavior is hidden by unchecked acceptance or fabricated capability.

Release completion requires all three product epics. A scope revision needs an explicit product decision and a corresponding specification update.
