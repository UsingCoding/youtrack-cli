# Architecture

## Existing boundaries

Keep the MVP1/MVP2 direction:

```text
cli -> app -> domain
youtrack -> app ports + domain
output -> domain
config, credentials, skill -> supporting utilities
```

The CLI parses arguments and file input, the application resolves/validates intent, the adapter maps REST, and output owns explicit compatibility DTOs. Do not route structured operations through the raw API or YouTrack command-language strings.

## Semantic models

Add only models needed by reachable features:

- `WorkItem`: entity ID, work date, duration in integer minutes, optional text, optional author/creator/type, created and optional updated timestamps.
- `WorkItemType`: entity ID and name; choices are scoped to the issue project.
- `LinkType`: entity ID, name, directed/aggregation booleans, source-to-target label, target-to-source label.
- `LinkRelation`: type plus semantic direction (`outward`, `inward`, `undirected`) and display label, relative to one source issue.
- linked issue: a link relation plus an existing lightweight issue summary.

Dates, timestamps, directions, and durations are semantic values, not REST payload fragments. Unknown work-item attributes remain untouched by partial edits; they do not need a public editing model in this release.

## Consumer-owned ports

Extend app-owned ports narrowly:

- work-item page/read/create/partial-update/delete operations
- project-scoped time-tracking status and work-item type listing
- link-type listing and per-relation linked-issue pages, plus one-edge add/remove
- work-item author lookup using existing user identity semantics, with complete pagination where resolution requires it
- lightweight issue identity/project reads where full issue/Board enrichment is unnecessary
- saved-search metadata resolution reusable without executing its issues query

Share resolvers with existing field and saved-search behavior instead of introducing new fuzzy or first-match resolution rules. Work-item authors are not custom-field values: do not restrict author lookup to an issue field's UserBundle.

Ports accept resolved identities and semantic changes. An update distinguishes omission from explicit clear; a zero value must never accidentally reset an omitted duration, type, date, or text.

## Mutation flow

```text
validate CLI shape and required confirmations
 -> read target and required metadata
 -> resolve every supplied reference across complete visible candidates
 -> validate the complete intent
 -> one mutation request
 -> map server result or render explicit deletion/relationship confirmation
```

No rollback, read-modify-write retry, or distributed transaction is promised. Work-item edits send only supplied fields, preserving unknown attributes and server-managed values.

## Browser handoff

Separate URL resolution from process launching and API authentication:

- service URL selection reuses flags > environment > selected profile > defaults
- token-free URL construction does not read the credential store or create an API client
- API-dependent issue/saved-search resolution retains ordinary credential rules, including no stored-token reuse for an explicit URL override
- application behavior resolves an issue identity or saved query without per-result enrichment
- one small URL builder preserves the configured context path and encodes path/query values once
- an injectable platform opener accepts one already validated absolute URL, not a shell command
- the CLI coordinates output and opening; domain and YouTrack HTTP code never start local processes

Reuse configuration selection logic rather than cloning its precedence in browser commands. Add a URL-only resolution path without weakening authenticated runtime requirements.

The opener invokes the platform mechanism directly with argument arrays. It must not use a shell or interpolate query text into a command string. It reports dispatch success, not browser navigation, HTTP authorization, or successful page rendering.

## Pagination and caching

Reuse MVP2 `--limit 50`, `--offset 0`, `--all`, and explicit-limit conflict rules. Resolution caches live for one invocation only. Advance collection offsets by actual returned count; stop only at caller limit or an empty page. No persistent directory/link/type cache is introduced.

Linked-issue listing is scoped to one resolved relation so pagination has a single unambiguous collection. Listing link types is a separate discovery operation; never assume embedded linked issues include the whole collection.

## Integration cutover

Update service composition, CLI dependencies, fakes, renderer entry points, and all constructor callers in each implementation slice. Do not leave compatibility constructors, duplicate parsers/resolvers, unused ports, or pre-registered placeholder commands. Existing full issue reads and outputs need not acquire time or link enrichment.
