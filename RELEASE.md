# Release

## Publish a release

Pushing a `v*` tag runs GoReleaser and publishes a GitHub release:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The workflow is defined in [`.github/workflows/release.yml`](.github/workflows/release.yml). It checks out the full tag history, installs the pinned tools with mise, and runs `goreleaser release --clean`.

## GitHub token permissions

The repository-scoped GitHub Actions `GITHUB_TOKEN` is sufficient for this release. The workflow grants it `contents: write` to create the GitHub release. Do not configure `SECRET_TOKEN` or a personal access token for this purpose.

If publishing GitHub Packages owned by this repository is added, grant the workflow `packages: write`; the default `GITHUB_TOKEN` remains sufficient. Use a personal access token only when publishing to another repository or when an organization policy prevents the default token from writing.
