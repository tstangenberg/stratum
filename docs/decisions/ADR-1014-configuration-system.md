# ADR-1014: YAML-to-env configuration system with self-contained plugins

**Status:** Accepted

## Context and Problem Statement

Stratum's plugin set is growing. Stateful plugins — starting with `api-key-auth` — need runtime configuration (e.g., a list of valid API keys). A system is needed that lets operators supply config values without requiring the core to know about each plugin's internals. The system must also work naturally with container runtimes where env vars are the primary config mechanism.

ADR-1008 already references `stratum.yaml` as the config file for hook ordering. This ADR defines how that file is read and how its values reach plugins.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **Central typed config struct** | Type-safe, single source of truth, validated at startup | Core must know every plugin's config type; adding a plugin requires touching core |
| **Configurable interface** | Plugins declare their own needs; core stays decoupled | Stringly typed `map[string]any`; runtime type assertions; weaker error messages |
| **Blind YAML→env expansion + plugins read own env vars** | Plugins fully self-contained; core knows nothing about plugin config; scales to dynamic plugins | Config keys are distributed across plugins; no central validation |

## Decision Outcome

Chosen: **blind YAML→env expansion with self-contained plugins**.

The core config system (`internal/config`) reads a YAML file and expands every leaf value to an environment variable using a deterministic naming rule — without knowing what plugins exist. Plugins own their env var names and read them directly via `os.Getenv` in their constructor. The core never calls into plugins during config loading.

**Naming rule:** YAML key path → join segments with `_`, uppercase, prepend `STRATUM_`.

| YAML path | Env var |
|-----------|---------|
| `server.addr` | `STRATUM_SERVER_ADDR` |
| `database.url` | `STRATUM_DATABASE_URL` |
| `plugins.auth.api_keys` | `STRATUM_PLUGINS_AUTH_API_KEYS` |

**Precedence:** env vars already set in the process environment are never overwritten. The YAML file provides defaults; the environment overrides. This makes the system container-friendly: `docker run -e STRATUM_PLUGINS_AUTH_API_KEYS=secret ...` always wins.

**Lists** in YAML are comma-joined into a single string: `[key1, key2]` → `"key1,key2"`. Plugins split on comma.

**Plugin convention:** every plugin that needs config must use the prefix `STRATUM_PLUGINS_<PLUGIN>_<KEY>` and document its keys. No registration with the core is required.

**File resolution:** `STRATUM_CONFIG` env var → `./stratum.yaml` → no file (pure env-var mode). Missing file is not an error.

**Consequences:**
- Adding a new plugin never requires changes to the core config system
- Config keys are discoverable only by reading plugin source or documentation — no central registry
- The system is forward-compatible with dynamically loaded plugins

## Addendum (US-0063): Constant declaration and doc comment convention

Every `STRATUM_*` environment variable must be declared as an exported constant in the Go package that reads it:

```go
// Human-readable description of what the variable controls.
// Default: <default value, or "none" if required>
const EnvFoo = "STRATUM_FOO"
```

**Placement rules:**
- Core infrastructure vars (`STRATUM_CONFIG`, `STRATUM_SERVER_ADDR`) → `internal/config/env.go`
- Plugin vars → an `env.go` file in the plugin's own package (e.g. `internal/plugin/database/env.go`)
- Schema vars → `internal/schema/env.go`

**Doc comment format:**
- First line(s): human-readable description — do **not** open with the Go identifier name (`EnvFoo is …`); the variable name is already in the generated table's first column
- `Default:` line: the default value, or `none` if the variable is required

All `os.Getenv(...)` call sites must reference the constant; magic string literals for `STRATUM_*` names are not allowed.

**Code generator:**
`cmd/configdocs` walks all non-test Go source files, extracts constants matching this pattern, and writes a sorted reference table to `docs/configuration.md`. Regenerate from the module root with:

```
go generate ./...          # or: go run ./cmd/configdocs
```

No changes to `internal/config` are required when adding a new env var — declare the constant in your package's `env.go` and the generator picks it up automatically.
