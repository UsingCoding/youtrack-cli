# Testing, Tooling, and Release

## Tests

Use:

```go
github.com/stretchr/testify/require
github.com/stretchr/testify/assert
```

Use handwritten fakes for application ports. Avoid `testify/mock` unless it is materially simpler.

HTTP adapter tests use `httptest.Server` and assert method, path, query/fields, headers, body, pagination, mapping, and API errors.

Important resolver cases:

- primitive field kinds
- single/multi enums and bundle values
- user single/multi
- `@me`, `@none`
- versions/build/owned/group
- unknown/ambiguous values
- required clear
- unknown REST `$type`
- pagination beyond YouTrack's common 42-item default

Critical combined-edit integration test:

- all metadata/value resolution happens before POST
- invalid last field => zero mutation POSTs
- valid edit => exactly one issue update POST

Coverage targets:

```text
overall             >= 80%
app                  >= 90%
field resolver       >= 95%
youtrack adapter     >= 85%
config               >= 90%
```

## mise

Required tasks:

```text
fmt
fmt:check
lint
test
test:race
coverage
build
install
release:snapshot
ci
```

`ci` runs formatting check, lint, tests, race tests, and build.

## GoReleaser

Build `youtrack` for:

```text
darwin/amd64
darwin/arm64
linux/amd64
linux/arm64
windows/amd64
```

Inject version, commit, and date with ldflags. Use `tar.gz`, and zip for Windows.
