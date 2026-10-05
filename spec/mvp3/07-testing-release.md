# Verification and Release

## Scope of this change

Epic 1 spent-time and Epic 2 relationship behavior are implemented. Validate their focused contracts, built-CLI smoke scenarios, output compatibility, and documentation. Browser-handoff requirements remain specification-only until Epic 3 is implemented.
The requirements below apply when implementing MVP3. Keep the existing Testify, handwritten fake, `httptest.Server`, mise, race/build, and coverage conventions. Tests must defend observable contracts and plausible boundary failures, not source text, constructor wiring, or incidental human wording.

## Spent time

High-value behavior/adapter coverage:

- explicit valid dates round-trip without local timezone/day shifts; REST midnight differs from custom date-field midday
- leap-day invalidity, malformed periods, zero/negative/fractional units, and arithmetic overflow fail before mutation
- add resolves project types beyond 42 values and rejects an unrelated project's type
- author and creator remain distinct; lookup success is not treated as permission approval
- disabled project settings reject add/edit without changing settings; existing item reads/removal remain server-authorized
- date/duration required on add, empty edit rejected, conflicting text sources/type-clear flags rejected
- explicit empty text clears, omitted text/type/date/duration remain unchanged, unknown attributes survive an edit
- type clearing serializes null; minutes serialize numerically; date is UTC epoch milliseconds
- specific item read/edit/remove stays scoped to the requested issue
- deletion requires `--yes`, uses permanent DELETE once, and does not masquerade as comment soft removal
- list paging handles server-capped short pages, empty arrays, limit/offset/all, and nullable returned fields
- uncertain POST/DELETE failures never trigger automatic retries or compensating time entries

Live disposable-project acceptance: discover types, add a dated item, read it, replace its duration/text/type, verify through the server, remove it, and confirm it is gone. Exercise alternate-author permission and type-clear behavior on supported versions. Observe server-computed spent-time totals without writing a total custom field.

## Links

High-value behavior/adapter coverage:

- custom/renamed/localized type names and labels resolve without built-in English aliases
- name and label ambiguity across later pages is rejected; type ID plus direction disambiguates
- directed type identity requires direction, label direction conflicts fail, undirected input rejects direction
- outward maps to `s`, inward to `t`, undirected has no suffix; only adapter code knows this encoding
- both issues resolve before a write; two aliases for the same issue fail self-link validation
- selected-relation list preserves server order and paginates the dedicated linked-issues endpoint
- add-existing/remove-missing membership checks find an edge beyond page 1 and produce no write
- an actual add sends one target database ID; a remove deletes one selected edge, not either issue or all links
- there is no reciprocal second mutation, implicit reparent, or workflow bypass
- permission/cycle/second-parent/race failures are surfaced without inventing server guarantees

Live acceptance: create a relationship between two disposable issues, inspect it from both sides, list each correct direction, remove it, and confirm neither issue was deleted. Include an undirected type and a subtask relation, using server-discovered identities.

## Browser handoff

Parser and URL behavior coverage:

- normal one-argument search is byte/output-compatible with MVP2
- `search 'open'` is REST search, while `search 'open' open` is browser search
- leading-hyphen queries after `--` work; unexpected extra arguments fail
- browser search rejects explicitly set pagination flags, including default-valued ones
- direct browser search makes no REST request and never reads stored credentials
- issue opening resolves readable/database IDs through a lightweight read; saved-search opening resolves metadata without fetching results
- blank saved queries fail before launching; nested global profile/URL/token flags retain precedence and explicit-URL credential isolation
- URL encoding round-trips `#`, `&`, `+`, `%`, Unicode, braces, shell metacharacters, and quotes exactly once
- a `/youtrack` context path is preserved; malicious path/query text cannot switch origin or become a shell command
- non-HTTP(S), missing-host, userinfo, query-bearing, and fragment-bearing service bases fail
- `--print-url` never calls the opener; JSON/plain alone do not suppress launch
- launcher errors yield nonzero and no false success object; no token or Authorization value appears in URLs/errors/debug logs
- plain output is one URL/newline, and JSON uses explicit URL/opened fields

Proof must exercise the actual CLI and platform integration, not only a fake opener:

1. Run URL-only mode headlessly and inspect the exact output for root/context-path service URLs.
2. Use a disposable local HTTP page or authorized test server to observe a real browser handoff on macOS, Linux, and Windows release targets. Platform launch paths must use arguments/native APIs without shell interpolation (including Windows, not `cmd /c start` with user input).
3. Verify issue and query web routes in an actual supported YouTrack browser surface, including an opaque query with reserved characters and a self-hosted context path where available.
4. Verify a missing/unavailable opener produces the documented failure and URL-only guidance. OS dispatch does not prove page authentication; record that distinction.

## Cross-feature output and compatibility

Cover exact stable JSON key/nullability/array shapes and plain bytes, especially empty lists, no-op links, and permanent deletion results. Preserve existing MVP1/MVP2 output/exit codes and global flag behavior. Do not pin terminal table layout or wording.

Each epic must have a real built-CLI smoke scenario against `httptest` or a disposable server in addition to focused regression tests where plausible bugs justify them. README and embedded skill must describe only delivered behavior and match the CLI contract.

## Release gates

- live compatibility scenarios cover the supported YouTrack versions and browser routes; record the tested versions and any minimum version requirement
- temporary smoke fixtures/scripts are removed after proof, and production commands contain no stubs/compatibility scaffolding
- normal release platform/artifact policy remains unchanged
- repository checks pass after integration:

```bash
mise run fmt
mise run lint
mise run test
mise run test:race
mise run build
mise run ci
```

Coverage targets remain those in MVP1/AGENTS; this spec does not lower them or prescribe tests merely to increase test count.
