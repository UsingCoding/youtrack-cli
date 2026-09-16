# Commands

```bash
youtrack issue search <query> [--limit <n>] [--offset <n>] [--all]
youtrack saved-search view <saved-search> [--limit <n>] [--offset <n>] [--all]
youtrack issue comment list <issue> [--limit <n>] [--offset <n>] [--all]
youtrack issue comment add <issue> (--text <text> | --file <path>)
youtrack issue comment edit <issue> <comment-entity-id> (--text <text> | --file <path>)
youtrack issue comment remove <issue> <comment-entity-id>


youtrack issue view <issue> [--json|--plain]

youtrack issue create <project> \
  --summary TEXT \
  [--description TEXT | --description-file FILE] \
  [--field NAME=VALUE ...] \
  [--tag TAG ...]

youtrack issue edit <issue> \
  [--summary TEXT] \
  [--description TEXT | --description-file FILE] \
  [--field NAME=VALUE ...] \
  [--tag TAG ...] \
  [--remove-tag TAG ...]

youtrack issue move <issue> <project>

youtrack issue field list <issue>
youtrack issue field get <issue> <field>
youtrack issue field set <issue> <field> <value>...
youtrack issue field clear <issue> <field>

youtrack issue tag list <issue>
youtrack issue tag add <issue> <tag>
youtrack issue tag remove <issue> <tag>

youtrack api <endpoint> [--method METHOD] [--data JSON | --data-file FILE]
```

`issue search` accepts exactly one non-blank opaque query. Quote it so YouTrack receives its filters and explicit sort clause unchanged. It starts at offset `0` and returns `50` results by default. `--limit` must be positive, `--offset` must be non-negative, and `--all` cannot be combined with an explicitly supplied `--limit`. Use `--` when the query begins with a hyphen:

```bash
youtrack issue search -- '-State: Done project: APP'
```

Use `--json` for structured result consumption. Global flags work from nested commands.

`saved-search view` accepts one database ID or quoted visible name. It resolves the visible server-side saved query, then pages matching issues with the same defaults and validation as `issue search`: offset `0`, limit `50`, a positive explicit `--limit`, non-negative `--offset`, and no explicit `--limit` with `--all`. Use `--json` for the saved-search metadata and issue summaries, or `--plain` for matching issue lines only. The command is view-only; structured saved-search create, update, delete, and sharing operations are unavailable.

`issue comment list` uses the same pagination rules as `issue search`: default limit 50, non-negative offset, positive explicit limit, and no explicit `--limit` with `--all`. It includes visible soft-removed comments. `issue comment add` and `issue comment edit` require exactly one source selected with `--text` or `--file`; both preserve raw supplied bytes, and blank/whitespace-only text is rejected. Edit sends a text-only POST to the selected comment and does not restore a removed comment. Prefer `--file` for substantial multiline content. `issue comment remove` performs reversible soft removal only. Restoration, permanent deletion, attachments, visibility, reactions, and pinning are unsupported. Use `--json` for stable comment objects, `--plain` for IDs (and no bytes on successful removal), or human output for readable blocks.

`issue create` accepts exactly one project and a non-blank `--summary`. `--description` and `--description-file` are mutually exclusive and preserve supplied bytes. Each `--field` is split only on its first `=`, so commas and later equals signs remain literal; repeat a multi-value field to provide its complete value set. The CLI resolves the project, all fields, and all tags before issuing one non-retried issue POST, then renders the returned full issue. `Board`, state fields, attachments, links, comments, and notification inputs are unavailable during creation.

`Board` is available through the field commands and `--field Board=...`. Board set is exact-set replacement; values are board IDs or names, not sprint syntax.

Global connection flags: `--profile`, `--url`, `--token`, `--timeout`. Global output flags: `--json`, `--plain`.
