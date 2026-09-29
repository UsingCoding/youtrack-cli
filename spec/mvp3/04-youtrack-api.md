# YouTrack API and Compatibility Evidence

## Evidence status

The endpoints and field declarations below are based on the linked official JetBrains documentation reviewed for this plan. Product choices are not claims about undocumented server behavior. No live server mutation was performed while writing this specification.

Keep the existing transport, service context-path handling, timeout, error mapping, and GET-only retry policy. Never retry POST or DELETE automatically. Centralize new response projections in `internal/youtrack/fields_query.go` and keep request/response DTOs private to the adapter.

## Work items

```text
GET    /api/issues/{issue}/timeTracking/workItems
POST   /api/issues/{issue}/timeTracking/workItems
GET    /api/issues/{issue}/timeTracking/workItems/{itemID}
POST   /api/issues/{issue}/timeTracking/workItems/{itemID}
DELETE /api/issues/{issue}/timeTracking/workItems/{itemID}
GET    /api/admin/projects/{projectID}/timeTrackingSettings
GET    /api/admin/projects/{projectID}/timeTrackingSettings/workItemTypes
```

Resolve the issue/project before project type discovery. Work-item IDs are database IDs scoped under the issue. Settings reads request `enabled`; type listing requests `id,name` with `$skip`/`$top`. Do not use an embedded type collection as the complete choice set or attach global types to a project as a side effect.

Work-item projection:

```text
id,date,duration(minutes),text,
author(id,login,fullName),creator(id,login,fullName),
type(id,name),created,updated
```

A create body after resolution can contain:

```json
{
  "duration": {"minutes": 90},
  "date": 1790640000000,
  "text": "Implement token refresh",
  "type": {"id": "114-0"},
  "author": {"id": "1-2"}
}
```

Duration is required by REST. Date is additionally required by the CLI. Optional keys are omitted unless supplied. Send numeric minutes, never localized duration `presentation`. The CLI's positive duration and supported-unit restrictions are product rules.

The work-item date is documented as UTC epoch milliseconds with its time part set to midnight. Serialize the explicit `2026-09-29` date as `2026-09-29T00:00:00Z`; this is deliberately **not** the MVP1 custom date-field UTC-midday encoding. Never depend on normalization of an arbitrary local-time timestamp. Read work dates in UTC; created/updated remain ordinary instants.

Specific-item POST is a partial update. Serialize only changed fields; clearing type uses `"type": null`, and clearing text uses `"text": ""`. The entity table declares author writable and creator read-only; author changes still depend on server permissions. Verify author changes and nullable type clearing in live compatibility acceptance rather than assuming identical behavior across releases. Do not send creator, created/updated, issue, attributes, or rendered previews.

Read requires issue access and Read Work Item. Update/delete documentation distinguishes an entry created by the caller from one created by another user; do not confuse creator with attributed author. Creating on behalf of another author may require Create Not Own Work Item. Server permission responses remain authoritative.

The CLI deliberately rejects add/edit when the observed project setting is disabled rather than attempting to enable it; reads and deletion of an existing entry remain available subject to server permission. This is a CLI safety choice, not a claim that the server uniformly rejects all writes while disabled. No global working-calendar request is needed for minute/hour input.

## Links

```text
GET    /api/issueLinkTypes
GET    /api/issues/{issue}/links/{linkID}/issues
POST   /api/issues/{issue}/links/{linkID}/issues
DELETE /api/issues/{issue}/links/{linkID}/issues/{targetIssueID}
```

Type projection:

```text
id,name,directed,aggregation,sourceToTarget,targetToSource
```

The adapter maps configured labels into the semantic outward/inward fields. REST `linkID` is constructed from the resolved base type ID only inside the adapter:

| Type/direction relative to first issue | REST link ID |
| --- | --- |
| directed outward | `<typeID>s` |
| directed inward | `<typeID>t` |
| undirected | `<typeID>` |

Resolve both issue references to database IDs before mutation. Add sends only:

```json
{"id":"2-72"}
```

The target in DELETE is also its database ID. One request establishes/removes the reciprocal relation; never write its inverse separately. Link-type administration and `/api/commands` are not used.

List the specific relation's `/issues` collection with `$skip`, `$top`, and the existing full lightweight MVP2 issue-summary projection (including its timestamps/project). Preserve server order. Do not use the embedded `issues`/`trimmedIssues` arrays from `/api/issues/{issue}/links` for exhaustive membership or pagination.

The official API documents Link Issue permission but does not establish all duplicate, missing-edge, second-parent, hierarchy-cycle, or race outcomes. CLI no-op behavior uses preflight membership; concurrent outcomes and hierarchy rules remain server-controlled. No workflow bypass or `muteUpdateNotifications` flag is added.

## Work-item author resolution

```text
GET /api/users
GET /api/users/{userID-or-login}
GET /api/users/me
```

Reuse existing user ID/login resolution and `@me` semantics for work-item authors. Paginate the complete visible user collection when name resolution requires it; do not assume the users endpoint supports a `query` parameter. Permission failures are real errors, not empty candidate sets. Resolving an author does not establish permission to create or edit work attributed to that user.

## Browser destinations and authentication

Browser URLs are web routes, not REST endpoints:

```text
<service-context>/issue/<escaped-readable-ID>
<service-context>/issues?q=<encoded-query>
```

Issue opening uses a lightweight `GET /api/issues/{reference}?fields=id,idReadable` before URL construction. Saved-search opening reuses MVP2 metadata resolution (`id,name,query,owner(...)`) without calling the issue search endpoint. Direct browser search makes no API request and needs no token. All browser paths use the same selected service URL; API paths retain normal credential safety.

Encode `q` once with a URL library; decode must recover the exact original string, including `#`, `&`, `+`, percent signs, braces, quotes, Unicode, and newlines. Append paths without discarding an installed context path. Browser routes are product routing conventions, not a REST compatibility guarantee: verify both routes against the target Cloud/self-hosted versions before release.

Do not claim query validity or web access merely because a URL was built or dispatched. Browser session identity can differ from the permanent-token user.

## Official references

### Work items and links

- [Issue work-item collection/create](https://www.jetbrains.com/help/youtrack/devportal/resource-api-issues-issueID-timeTracking-workItems.html)
- [Specific work-item CRUD](https://www.jetbrains.com/help/youtrack/devportal/operations-api-issues-issueID-timeTracking-workItems.html)
- [Work-item entity](https://www.jetbrains.com/help/youtrack/devportal/api-entity-IssueWorkItem.html)
- [Project time-tracking settings](https://www.jetbrains.com/help/youtrack/devportal/operations-api-admin-projects-projectID-timeTrackingSettings.html)
- [Project work-item types](https://www.jetbrains.com/help/youtrack/devportal/resource-api-admin-projects-projectID-timeTrackingSettings-workItemTypes.html)
- [Link types](https://www.jetbrains.com/help/youtrack/devportal/resource-api-issueLinkTypes.html)
- [Linked issue listing/add](https://www.jetbrains.com/help/youtrack/devportal/resource-api-issues-issueID-links-linkID-issues.html)
- [Linked issue removal](https://www.jetbrains.com/help/youtrack/devportal/operations-api-issues-issueID-links-linkID-issues.html)
- [Link product behavior](https://www.jetbrains.com/help/youtrack/cloud/link-issues.html)

### Users

- [Users](https://www.jetbrains.com/help/youtrack/devportal/resource-api-users.html)
- [Specific user operations](https://www.jetbrains.com/help/youtrack/devportal/operations-api-users.html)

### Shared conventions and browser search

- [REST pagination](https://www.jetbrains.com/help/youtrack/devportal/api-concept-pagination.html)
- [REST fields syntax](https://www.jetbrains.com/help/youtrack/devportal/api-fields-syntax.html)
- [Service URL/context paths](https://www.jetbrains.com/help/youtrack/devportal/api-url-and-endpoints.html)
- [Issue search product behavior](https://www.jetbrains.com/help/youtrack/cloud/search-for-issues.html)
- [JetBrains public browser search route](https://youtrack.jetbrains.com/issues?q=%23jt)
