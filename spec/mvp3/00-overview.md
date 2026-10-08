# YouTrack CLI MVP3 — Overview

## Goal

MVP3 adds spent-time management, issue relationships, and browser handoff. All three epics are implemented; Epic 3 live browser acceptance remains pending authorized multi-platform verification. Release verification remains separate.

## Baseline

MVP3 is additive to [MVP1](../mvp1/00-overview.md) and [MVP2](../mvp2/00-overview.md). Existing command names, flags, output contracts, exit codes, authentication precedence, pagination behavior, and dependency boundaries remain valid. The earlier non-goals for links and work items are lifted only for the operations specified here. Existing MVP1 issue-tag commands remain unchanged.

## Epics

1. **Issue spent time CRUD:** list and inspect work items, add time, edit one entry, permanently remove one entry, and discover the issue project's work-item types.
2. **Issue links:** discover configured link types and their directions, list linked issues, add a relationship, and remove a relationship. Include subtasks, relates, and custom link types without hard-coded English aliases.
3. **Browser handoff:** `issue open <issue>`, `issue search <query> open`, and `saved-search open <saved-search>`, with a URL-only mode for headless callers.

## Non-goals

- timers, work reports, bulk time import, work-item attribute editing, or time-tracking administration
- creating/configuring link types, bulk link changes, or implicit issue creation/reparenting
- standalone tag CRUD, tag visibility/sharing controls, or user/group administration
- changing issue/comment visibility
- new issue-create or combined issue-edit mutation flags for time or links
- arbitrary URL opening, browser selection/configuration, browser authentication, or automatic opening after a mutation
- saved-search mutation, new saved-search sharing controls, or a TUI

The raw API remains the escape hatch for operations outside this scope. It is not a fallback that structured commands invoke after unsupported or rejected writes.

## Core invariants

1. Domain/application models use semantic concepts; REST DTOs and directed link ID suffixes stay in `internal/youtrack`.
2. All references and supplied values are resolved before the first mutation. A single command changes one work item or one relationship.
3. Writes are not automatically retried. A transport failure can mean that a mutation committed; errors must not assert rollback.
4. Work-item deletion is permanent and requires `--yes`; comment removal retains its existing soft-removal behavior.
5. Link direction is relative to the first issue in the command. A reciprocal edge is never written separately.
6. Collection paging follows server caps and continues after short non-empty pages. Name resolution examines the complete visible candidate set.
7. Browser opening launches only a URL under the resolved YouTrack service origin/context path. No shell interpolation, token in URL, credential forwarding, or browser-cookie manipulation.
8. Existing issue JSON remains unchanged.

## Specification map

- [Architecture](01-architecture.md)
- [Feature behavior](02-feature-behavior.md)
- [CLI and output contracts](03-cli-contract.md)
- [YouTrack API and compatibility evidence](04-youtrack-api.md)
- [Embedded agent skill requirements](06-agent-skill.md)
- [Verification and release gates](07-testing-release.md)
- [Feature-sliced implementation plan](08-implementation-plan.md)

## Definition of done for implementation

- all three epics are reachable through the real CLI and satisfy their documented acceptance scenarios
- invalid or ambiguous references produce zero mutation requests
- project-scoped work types and directed links work beyond 42 entries
- work-item deletion is never confused with soft comment removal
- browser URL construction and dispatch are verified on supported platforms, including context-path installations and headless failures
- stable JSON/plain output, MVP1/MVP2 compatibility, README, and embedded agent skill are updated with implementation
- existing mise CI and coverage requirements remain satisfied
