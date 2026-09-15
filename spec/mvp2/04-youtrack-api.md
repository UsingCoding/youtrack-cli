# YouTrack REST Adapter

## Existing client contract

MVP2 uses the MVP1 client, base-URL handling, headers, timeout, error mapping, and GET-only retry policy. POST requests are never automatically retried.

## Issue search and creation

```text
GET  /api/issues
POST /api/issues
```

Search sends:

```text
query=<opaque YouTrack query>
$skip=<offset>
$top=<page size>
fields=<central issue-summary projection>
```

The adapter URL-encodes the query once and does not parse, alter, filter, or sort it. It requests pages sequentially. A caller limit may span multiple API pages when the server caps `$top` below the requested size.

Create sends one body containing:

```text
project: { id }
summary
description          # when supplied
customFields         # when supplied
tags                 # when supplied
```

The request uses a centralized full-issue response projection so the adapter can return the created semantic issue. Write DTOs map resolved semantic field assignments to the required YouTrack `$type` and value shapes. No REST DTO or `$type` escapes `internal/youtrack`.

MVP2 does not use `draftId` or `muteUpdateNotifications`.

## Saved searches

```text
GET /api/savedQueries/{queryID}
GET /api/savedQueries?$skip=...&$top=...
```

A direct database-ID lookup requests `id,name,query,owner(id,login,fullName)`. Name resolution paginates the visible saved-search collection with the same projection until an empty page.

The adapter does not request or trust the embedded `issues` collection. After resolution, the application executes the saved query through `GET /api/issues`, which provides explicit paging and the normal issue-summary projection.

No `POST` or `DELETE` request to `/api/savedQueries` is part of MVP2.

## Comments

```text
GET  /api/issues/{issue}/comments
POST /api/issues/{issue}/comments
POST /api/issues/{issue}/comments/{commentID}
```

List sends `$skip`, `$top`, and a centralized comment projection containing:

```text
id
text
author(id,login,fullName)
created
updated
deleted
```

Add sends only:

```json
{"text":"..."}
```

and requests the same comment projection in the response.

Remove sends only:

```json
{"deleted":true}
```

MVP2 does not call `DELETE /api/issues/{issue}/comments/{commentID}`. Comment attachments, visibility, reactions, pinning, and rendered text are deliberately absent from request and response projections.

## Fields projections

All new `fields=` expressions are constants in `internal/youtrack/fields_query.go`. CLI and application code never build REST projections.

Use separate projections for:

- lightweight issue search summaries
- full created-issue output
- saved-search metadata
- comments

Do not expand every search result through the existing single-issue read path; that would introduce per-result requests and change server ordering behavior.

## Pagination

Collection requests continue by advancing `$skip` by the number of entities actually returned. Continue until the application limit is reached or an empty page is returned. A short non-empty page is not an end-of-collection signal.

Saved-search name resolution always paginates the complete visible collection so ambiguity can be detected. Search and comment listing stop at the caller's limit unless `--all` is selected.

## Errors

Existing `APIError` and exit-code mappings remain. In addition:

- invalid server-side search syntax remains an API/runtime error, not CLI usage error
- a missing saved search or comment maps to not-found
- duplicate saved-search name matches map to ambiguous reference
- create/comment permission failures map through the existing 401/403 behavior
- error strings never include tokens, Authorization headers, or full sensitive request headers

## Official API references

- [Issues resource](https://www.jetbrains.com/help/youtrack/devportal/resource-api-issues.html)
- [Saved Queries resource](https://www.jetbrains.com/help/youtrack/devportal/resource-api-savedQueries.html)
- [Specific SavedQuery operations](https://www.jetbrains.com/help/youtrack/devportal/operations-api-savedQueries.html)
- [Issue Comments resource](https://www.jetbrains.com/help/youtrack/devportal/resource-api-issues-issueID-comments.html)
- [Specific IssueComment operations](https://www.jetbrains.com/help/youtrack/devportal/operations-api-issues-issueID-comments.html)
- [Create an issue with custom fields](https://www.jetbrains.com/help/youtrack/devportal/api-howto-create-issue-with-fields.html)
