# Stratum

[![CI](https://github.com/tstangenberg/stratum/actions/workflows/ci.yml/badge.svg)](https://github.com/tstangenberg/stratum/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/tstangenberg/stratum/graph/badge.svg)](https://codecov.io/gh/tstangenberg/stratum)
[![Go Report Card](https://goreportcard.com/badge/github.com/tstangenberg/stratum)](https://goreportcard.com/report/github.com/tstangenberg/stratum)

Stratum is a schema-first data middleware. Define your data model once as a GraphQL SDL schema; Stratum exposes a REST API and manages the underlying PostgreSQL storage automatically. It handles schema uploads, record creation, querying, filtering, and relation traversal — so you can focus on your data model rather than boilerplate persistence code.

## Getting Started

**Prerequisites**

- [Go](https://go.dev/dl/) 1.22 or later
- [PostgreSQL](https://www.postgresql.org/) (for persistence plugins)

**Build**

```bash
go build ./...
```

**Run**

```bash
go run ./cmd/stratum
```

By default the server listens on `:8080`. Set `STRATUM_SERVER_ADDR` to override:

```bash
STRATUM_SERVER_ADDR=:9090 go run ./cmd/stratum
```

Copy `stratum.yaml.example` to `stratum.yaml` for file-based configuration. See `docs/decisions/ADR-1014-configuration-system.md` for details.

## Rebuilding the CodeMirror bundle

The pre-built bundle is at `internal/ui/static/codemirror.js`. To rebuild it after changing dependencies:

```bash
cd internal/ui/codemirror
npm install
npm run build
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, branch conventions, and the pull request workflow.

## License

This project is licensed under the [GNU Affero General Public License v3.0 or later](LICENSE) (AGPL-3.0-or-later).
