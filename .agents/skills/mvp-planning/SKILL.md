---
name: mvp-planning
description: Plan the next YouTrack CLI MVP or epic while preserving the established architecture, contracts, and delivery standards.
---

# MVP planning

## Current structure

- `spec/mvp1/` defines the baseline: issue reads/mutations, configuration/authentication, output, REST boundaries, and release rules.
- `spec/mvp2/` adds issue search, saved-search viewing, issue creation, and comments.
- `spec/mvp3/` adds spent time, links, and browser handoff; its browser epic remains specified until implemented.
- Every MVP has `00-overview.md` through `07-testing-release.md`; use `08-implementation-plan.md` for feature-sliced execution.
- Preserve `cli -> app -> domain`; `youtrack` implements app-owned ports and is the only REST/wire-shape layer; `output` owns explicit stable JSON DTOs.

## Plan a new MVP

1. Write the overview, architecture, feature behavior, CLI/output contract, YouTrack API evidence, agent-skill requirements, verification/release gates, and feature-sliced implementation plan.
2. State scope, non-goals, compatibility decisions, invariants, mutation ordering, pagination, error/exit-code behavior, and acceptance scenarios.
3. Specify semantic domain models and consumer-owned app ports. Keep REST DTOs, `$type` values, and protocol encodings inside `internal/youtrack`.
4. Plan each epic as a reachable vertical slice: domain, app, adapter, CLI, output, focused tests/smoke proof, README, and `skills/youtrack-cli/` updates. No placeholder commands, compatibility shims, or undocumented behavior.
5. Every command MUST support `--json` and define stable JSON: explicit DTO shape, key/nullability/empty-collection rules, timestamps, and plain-output semantics. Preserve existing command names, flags, JSON fields, exit codes, and nested global flags unless the MVP explicitly changes them.
6. Every subcommand that manages an entity MUST include an `open` command, and that command MUST support `--print-url`. Specify its canonical YouTrack route, URL-only behavior, JSON/plain result, opener failure semantics, URL encoding/context-path rules, and credential/token isolation.
7. For mutations, resolve and validate all supplied references before the first write; document no-op, confirmation, retry, and uncertain-outcome behavior.
8. Require complete pagination for collections and resolution, including server caps and short non-empty pages.

## Evidence and completion

- Ground REST endpoints, permissions, and wire behavior in supported-version/API evidence; do not infer from a UI alone.
- Require focused regression coverage for observable boundaries and a real built-CLI smoke scenario for every epic.
- Release gates retain formatting, lint, test, race, build, CI, coverage, README, and embedded-skill verification.
- Update the embedded skill only when behavior is implemented; plans must not advertise unavailable commands.
