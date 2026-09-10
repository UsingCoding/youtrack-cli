# Configuration, Authentication, and Output

## Configuration

Use `github.com/BurntSushi/toml` for decoding and encoding.

Default config:

```text
$XDG_CONFIG_HOME/youtrack/config.toml
# fallback ~/.config/youtrack/config.toml
```

Example:

```toml
current = "company"

[profiles.company]
url = "https://company.youtrack.cloud"
```

Tokens are not stored in the ordinary config.

## Credential store

MVP1 uses a separate TOML credential file with mode `0600`:

```text
~/.config/youtrack/credentials.toml
```

It is hidden behind `CredentialStore` so a future Keychain/Secret Service backend does not change CLI commands.

## Runtime precedence

```text
explicit CLI flags
> environment
> selected profile
> defaults
```

Environment:

```text
YOUTRACK_PROFILE
YOUTRACK_URL
YOUTRACK_TOKEN
```

An explicit server URL must not silently reuse credentials from a different stored profile. Provide an explicit token with a URL override.

## Login

`auth login` obtains URL/profile/token from flags/env/config or prompts, verifies them with `/api/users/me`, then writes profile + credential and selects that profile. Terminal token entry is hidden with `x/term`.

## Stable output model

Domain structs do not contain JSON tags. `internal/output/model.go` maps domain models into explicit compatibility-sensitive JSON DTOs.

The stable issue JSON includes readable ID, entity ID, summary/description, project, semantic fields, and tags. REST `$type` is not the primary JSON contract.
