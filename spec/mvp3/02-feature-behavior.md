# Feature Behavior

## Shared resolution and paging

References resolve by database ID, then unique exact name, then unique case-insensitive name. User references use ID/login semantics and `@me`, not ambiguous display names. Fully paginate visible candidates before choosing a name match. An ambiguity is not permission to select the first result. Do not conflate a permission failure with an empty collection or not-found fallback.

Public lists reuse MVP2 pagination: default 50 from offset 0, positive explicit limit, non-negative offset, and `--all` mutually exclusive with an explicit limit. Continue through short non-empty pages. Lookup resolution is exhaustive regardless of public list defaults.

## Issue spent time

### Resource and discovery

Spent time is managed as individual work items, not by assigning a total to the issue's Spent time custom field. `issue time list` lists entries; `view` reads one entry under the supplied issue. Work-item references are database IDs, never row numbers or text matches.

`issue time types <issue>` resolves the issue project and lists that project's available work-item types. Do not resolve against a global type list or invent a default type. Add/edit reject an observed disabled time-tracking setting with an actionable validation error; read/list/types and removal remain available subject to server permission. Commands never enable time tracking. Permission failures are surfaced, not interpreted as an empty configuration.

### Add

Require a positive `--duration` and explicit `--date YYYY-MM-DD`. Explicit dates avoid dependence on workstation, server, or user timezones. Accept integer `m` and `h` components, including `1h30m`, using the existing period grammar. Check conversion and sum overflow. Reject zero, negative, fractional, bare numeric, seconds, days, and weeks. Work calendars must not be guessed.

Optional values:

- `--text` or `--file`: mutually exclusive; preserve content exactly, including an explicitly empty string
- `--type`: resolve against the project's enabled work-item types
- `--author`: user ID/login or `@me`; omitted means the authenticated user via server behavior

The creator is server-managed and cannot be set. An author other than the caller requires server permission; a successful user lookup is not proof that the caller can log their time. No CLI impersonation mechanism is introduced.

Resolve the issue, time-tracking configuration, supplied type, and supplied author before one create request. Omitted text/type are left to the server. Read the returned item rather than predicting workflow/default changes.

### Edit

Require at least one supplied change. Duration and date replace their previous values; they are not increments. Text/file, type, and author use add semantics. `--clear-type` explicitly clears the type and conflicts with `--type`. An empty `--text ''` clears text; omitted text preserves it.

Read the specific item under the supplied issue before editing. Resolve new values, then submit only the supplied fields in one update. Do not round-trip creator, timestamps, unknown work attributes, or other omitted properties. Omitted retired types remain untouched; a newly supplied type must belong to the current project choices.

### Remove

`issue time remove <issue> <work-item-id> --yes` permanently deletes one work item. `--yes` is mandatory in all output/terminal modes; there is no interactive prompt or TTY-dependent behavior. Read the issue-scoped item before the delete. A missing item is not-found, including a repeated deletion. There is no restore command or synthetic compensation entry.

YouTrack owns recalculation of spent-time totals, workflow effects, and permissions. The CLI does not separately update a total field. A failed mutation is not retried automatically.

## Issue links

### Discovery and direction

`issue link types` lists configured link types with their database ID, name, directed/aggregation properties, and outward/inward labels. Types are server-defined; English words such as `subtask of`, `parent for`, `relates to`, or `duplicates` are examples, not built-in aliases.

A relation selector consists of `--type <reference>` and, when needed, `--direction outward|inward`:

1. Prefer database type ID, then unique exact/unique case-insensitive type name. A directed type selected this way requires an explicit direction.
2. If no type identity/name matches, resolve a unique outward/inward label across all visible types. The label determines direction.
3. A supplied direction must agree with a matched label. Conflicting direction is validation failure.
4. An undirected type has direction `undirected` internally and rejects `--direction`.
5. Duplicate label matches are ambiguous. Use type ID plus direction to disambiguate.

`outward` means source-to-target from the first issue in the command; `inward` means target-to-source relative to that issue. Do not expose REST directed-ID suffixes as a CLI reference format.

For a configured Subtask type with outward label `parent for` and inward label `subtask of`:

```bash
youtrack issue link add APP-CHILD APP-PARENT --type 'subtask of'
```

This makes the first issue a child of the second. The CLI does not reverse operands, create another issue, or send a reciprocal write.

### List

`issue link list <issue> --type <relation>` lists linked issue summaries for exactly one resolved relation, using ordinary pagination. Requiring a relation avoids applying one ambiguous offset across multiple nested collections. Use `issue link types` to discover choices. Empty linked results are a successful empty array. Do not trust the embedded `issues` collection from a link descriptor as complete.

### Add and remove

Resolve both issues and the relation before mutation. Reject self-links after resolving database IDs, so readable-ID/database-ID aliases cannot bypass the check.

Add-existing and remove-missing are CLI no-op successes, determined by the complete selected relation collection. Return whether a write was required. Check membership across all pages, not only the first displayed page. A concurrent change after the check is still subject to server behavior; no automatic write retry is promised.

A real add/remove uses one REST write to the selected source-side relation. Never rewrite the issue's complete link collection. Removing a relationship does not delete either issue.

Subtask changes respect YouTrack constraints and workflows. Adding a different parent does not implicitly detach the old parent or perform a two-step reparent. Report server rejection; users can explicitly remove and add when appropriate, understanding that these are separate non-atomic commands. Do not attempt to implement a client-side copy of all cycle, hierarchy, or permission rules.

## Browser opening

### Supported destinations

- `issue open <issue>` resolves a readable or database issue ID with a lightweight authenticated issue read, then opens its canonical readable-ID URL.
- `issue search <query> open` opens the supplied opaque query in the web issue list. It does not execute REST search, validate query syntax, or fetch results.
- `saved-search open <saved-search>` resolves the saved search and opens its current stored query through the same search URL builder. It does not fetch matching issues or promise a saved-search management/settings page.

A blank direct/stored query is invalid. The direct query remains byte-for-byte unchanged before one URL encoding step. A saved-search URL captures the query at invocation time, not a live subscription to later edits.

Additional useful cases are covered by composition: create an issue, inspect its returned ID, then explicitly open it; or obtain a URL in a headless session. Root-level arbitrary `open`, comment anchors, and `--open` on mutation commands are intentionally deferred because they add routing or mixed mutation/launcher failure contracts.

### Search grammar and compatibility

Keep the requested suffix syntax. Search accepts either one positional query (existing REST behavior) or exactly two positionals where the second is literal lowercase `open` (browser behavior). Do not register `open` as a conventional child command that steals a query named `open`.

- `issue search 'open'` remains REST search for the literal query `open`.
- `issue search 'open' open` opens that query in a browser.
- `issue search -- '-State: Done' open` supports a leading-hyphen query.
- other extra positional arguments are usage errors.

Browser search rejects explicitly supplied `--limit`, `--offset`, or `--all`, even when the supplied value equals its default. These are REST pagination controls, not browser parameters. Ordinary search rejects browser-only `--print-url` unless the suffix is present.

### URL-only mode, configuration, and output

`--print-url` on an open operation builds/resolves the same destination but does not launch anything. It works in headless environments and is suitable for agents.

Direct search opening needs only the resolved service URL and no API token. Issue/saved-search opening needs normal API authentication, including with `--print-url`, because it resolves IDs/names and permissions. Browser login is independent of the CLI token and may use a different account. Never put the token in a URL or browser session.

Accept only HTTP(S) service URLs with a host and no embedded username/password. Reject configured query/fragment components for browser destinations rather than incorporating them. Preserve a self-hosted context path such as `/youtrack`. Append one escaped readable issue ID path segment or encode one `q` query value; never concatenate unescaped user input. Do not open server-returned arbitrary URLs.

Dispatch via the default OS browser using a direct process/native mechanism, never a shell. Do not interpret `BROWSER` as arbitrary executable shell text in MVP3. Missing launcher, headless rejection, or process-start failure returns runtime error with `--print-url` guidance; no silent fallback that claims success. Success means the dispatch mechanism accepted the URL, not that the browser authenticated or loaded the issue. No indefinite wait for the browser process or page.

`--json` and `--plain` only select result formatting; neither implies URL-only mode. Use `--print-url` to suppress the side effect explicitly. See the exact output contract in [03-cli-contract.md](03-cli-contract.md).
