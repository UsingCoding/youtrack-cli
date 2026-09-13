# Commands

```bash
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

`Board` is available through the field commands and `--field Board=...`. Board set is exact-set replacement; values are board IDs or names, not sprint syntax.

Global connection flags: `--profile`, `--url`, `--token`, `--timeout`. Global output flags: `--json`, `--plain`.
