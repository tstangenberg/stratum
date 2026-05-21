---
id: US-0035
tags: [config, api]
status: ready
---

# US-0035: Configurable HTTP listen address

**As an** operator  
**I want** the HTTP listen address to be configurable at startup  
**So that** I can run Stratum on a non-default port or bind to a specific interface without recompiling

## Context

The listen address is currently hardcoded to `:8080` in `cmd/stratum/main.go` via the `run()` function. Following the 12-factor app convention, it should be configurable via an environment variable (`STRATUM_ADDR`) with `:8080` as the default.

## Acceptance Criteria

- [ ] The listen address is read from the `STRATUM_ADDR` environment variable
- [ ] If `STRATUM_ADDR` is not set, the server defaults to `:8080`
- [ ] The resolved address is logged at startup before the server begins listening
- [ ] `run()` continues to accept the address as a parameter so it remains unit-testable

## E2E Tests

None — verified by setting `STRATUM_ADDR=:9090` and confirming the server binds to port 9090.
