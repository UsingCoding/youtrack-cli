# Search, Issue Creation, and Comments

## Direct issue search

The CLI accepts one non-blank YouTrack issue query and sends it as the `query` parameter to `/api/issues`. It supports the same syntax the configured YouTrack server accepts in its issue search panel.

The query is opaque:

- preserve its content when URL-encoding it
- do not parse or rewrite field names, values, saved-search references, sorting, or `me`
- do not add an implicit project, unresolved filter, or sort clause
- preserve the order returned by YouTrack
- surface invalid-query responses as API errors

An omitted or blank direct query is a CLI validation error. MVP2 does not expose an accidental “all accessible issues” command.

## Search pagination

Search commands default to 50 results from offset 0. A positive `--limit`, a non-negative `--offset`, or `--all` may change this behavior.

`--all` and an explicitly supplied `--limit` are mutually exclusive. `--all` starts at the requested offset and fetches until the API returns an empty page. A limited search continues through server-capped pages until it has collected the requested number of issues or receives an empty page.

The adapter never treats `len(page) < requestedTop` as proof that the collection ended.

## Saved-search view

A saved-search reference may be its database ID or visible name. Name resolution examines every saved search visible to the authenticated user and follows this order:

1. exact database ID
2. unique exact name
3. unique case-insensitive name
4. ambiguous or not-found error

After resolution, the stored query is executed through the direct search path with the caller's limit and offset options. This avoids relying on an embedded, unpaged `issues` field in the saved-search REST response.

A null or whitespace-only stored query is an error. It must not become an unfiltered search.

MVP2 reads saved searches only. It defines no command or application port for create, update, delete, or sharing changes.

## Issue creation

Required inputs:

- target project reference
- non-blank summary

Optional inputs:

- description text or one description file
- repeated custom-field assignments
- repeated existing tags

Project references use MVP1 project resolution. Description text is preserved exactly; `--description` and `--description-file` are mutually exclusive.

`--field` uses the MVP1 first-`=` split and value syntax. Repeated assignments for one multi-value field form its complete value. Repeating a scalar field is invalid. `@me`, `@none`, primitive parsing, bundle/user/group resolution, exact and case-insensitive matching, and ambiguity behavior follow [`spec/mvp1/02-custom-fields.md`](../mvp1/02-custom-fields.md).

Creation-specific field rules:

- string and text empty strings remain values and are not interpreted as clear
- `@none` is accepted only for a field that may be empty
- unknown or unsupported field kinds fail before mutation
- `Board` is rejected before mutation
- every state field is rejected before mutation; YouTrack defaults and workflows choose the initial state, and later changes use the MVP1 state-transition path
- omitted fields are not assigned client-side defaults

Tags are existing tag IDs or names and use MVP1 resolution. Duplicate requested tags collapse to one value while preserving first occurrence order.

The complete request is serialized into one issue-create POST. It includes tags as the issue's top-level `tags` collection, not as follow-up tag requests. Attachments, comments, links, Board membership, and any other second-step mutation are not part of creation.

## Comment listing

Comment listing returns all comments visible to the authenticated user within the selected page range. It preserves server order and retains soft-deleted entries when YouTrack returns them.

Each semantic comment includes:

- comment database ID
- optional author
- optional raw text
- created timestamp
- optional updated timestamp
- deleted boolean

Attachments, visibility details, reactions, and rendered `textPreview` are not part of the MVP2 domain or output contract.

Comment pagination uses the same `--limit`, `--offset`, and `--all` semantics as issue search.

## Comment add

Exactly one source is required: inline text or a file. The file's contents become the comment text without Markdown rendering or normalization by the CLI. A value with no non-whitespace characters is invalid.

The request contains only `text`. YouTrack applies its normal visibility and workflow rules. MVP2 does not accept attachments or visibility flags.

## Comment remove

A comment reference is its database ID. Removal sets the comment's `deleted` property to `true` through the specific-comment update resource. This is the reversible removal supported by YouTrack.

MVP2 never calls the permanent comment `DELETE` operation. It also does not expose restore or edit commands. Permission failures are ordinary API errors.
