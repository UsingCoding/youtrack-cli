# Testing, Tooling, and Release

## Existing standards

MVP2 keeps the MVP1 Testify, handwritten-fake, `httptest.Server`, coverage, mise, race-test, build, and release requirements from [`spec/mvp1/07-testing-release.md`](../mvp1/07-testing-release.md).

New tests must assert observable behavior, serialization boundaries, pagination, error mapping, output contracts, and validate-before-write guarantees. Do not assert source text or incidental wiring.

## Search tests

Application and adapter coverage includes:

- exact query forwarding, including spaces, braces, `#`, minus filters, and explicit sort clauses
- server ordering preserved without client-side sorting
- blank direct query rejected before an HTTP request
- invalid server query mapped as an API/runtime error
- default, explicit limit, offset, and `--all` behavior
- a requested limit filled across multiple server-capped pages
- short non-empty pages followed until an empty page
- empty results rendered as non-null empty arrays
- no per-result issue enrichment requests

## Saved-search tests

Coverage includes:

- direct database-ID lookup
- unique exact-name and unique case-insensitive-name resolution
- duplicate exact and case-insensitive names rejected as ambiguous
- not-found behavior
- visible-search pagination beyond 42 entries
- null and blank stored queries rejected without calling `/api/issues`
- stored query executed through the ordinary issue-search path with caller paging
- no saved-search POST or DELETE operation
- JSON metadata and issue-summary contract

## Issue-creation tests

Field resolution cases from MVP1 are reused for project-scoped creation. Critical integration coverage proves:

- missing project, blank summary, malformed field input, unknown project, field, value, user, group, or tag causes zero mutation requests
- a late invalid value after earlier valid values still causes zero mutation requests
- repeated scalar values fail; repeated multi-values serialize as one complete assignment
- `@me`, `@none`, primitive, bundle, user, and group values follow MVP1 semantics
- Board and every state field are rejected before mutation
- description text and file inputs are mutually exclusive and preserve content
- duplicate tags collapse deterministically
- a valid create performs exactly one `POST /api/issues`
- project, summary, optional description, all custom fields, and all tags appear in the same request body
- the response maps to the existing semantic issue and stable full issue JSON
- POST failures are not retried

## Comment tests

Coverage includes:

- listing default, limited, offset, and `--all` pages
- pagination beyond 42 comments and short non-empty pages
- nullable author, text, and updated time mapping
- deleted comments remain distinguishable
- add requires exactly one non-blank text source
- file content is preserved
- add serializes only `text` and is not retried
- edit requires exactly one non-blank replacement-text source and preserves inline and file text
- edit posts only `text` to the selected issue/comment resource and is not retried
- remove posts only `deleted=true` to the selected issue/comment resource
- remove never uses HTTP `DELETE`
- not-found and permission errors retain existing exit-code semantics
- JSON, human, and plain contracts, including multiline comment text

## Security tests

Regression coverage verifies that query, creation, and comment failures do not expose permanent tokens or Authorization headers. Debug mode may include method and sanitized path data but never authenticated headers.

## Acceptance and CI

The command workflow in [`03-cli-contract.md`](03-cli-contract.md) is the manual acceptance surface against a disposable YouTrack project. Test-created issues and comments use unique names and are cleaned up where the available MVP2 operations permit; issue deletion and permanent comment deletion are not added solely for test cleanup.

Required final checks remain:

```bash
mise run fmt
mise run lint
mise run test
mise run test:race
mise run build
mise run ci
```

The existing overall and package coverage targets remain unchanged. MVP2 does not change artifact formats, supported platforms, or GoReleaser behavior.
