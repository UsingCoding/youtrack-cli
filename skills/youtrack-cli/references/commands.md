# Commands

```bash
youtrack issue search <query> [--limit <n>] [--offset <n>] [--all]

youtrack issue view <issue> [--json|--plain]

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

`Board` is available through the field commands and `--field Board=...`. Board set is exact-set replacement; values are board IDs or names, not sprint syntax.

Global connection flags: `--profile`, `--url`, `--token`, `--timeout`. Global output flags: `--json`, `--plain`.
