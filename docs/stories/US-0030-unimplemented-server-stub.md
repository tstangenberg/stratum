---
id: US-0030
tags: [api, dev-tooling]
status: blocked
depends_on: [US-0029]
---

# US-0030: Embed UnimplementedStrictServerInterface for partial API rollout

**As a** developer  
**I want** unimplemented API endpoints to return 501 rather than fail to compile  
**So that** I can implement the spec one story at a time without blocking the build

## Context

oapi-codegen generates a `ServerInterface` with one method per operation. Go requires every interface method to be implemented — without a stub, adding a new endpoint to the spec breaks the build until it is fully implemented.

oapi-codegen v2 generates `UnimplementedStrictServerInterface`, a struct that implements every method with `501 Not Implemented`. Embedding it in the server struct makes unimplemented endpoints compile and respond honestly. Each story replaces the 501 with a real implementation by overriding the relevant method.

501 responses may reach production during active development. This is acceptable — 501 is an honest signal that the endpoint exists in the spec but is not yet ready.

## Acceptance Criteria

- [ ] `StratumServer` embeds `api.UnimplementedStrictServerInterface`
- [ ] The server compiles with no method implementations beyond the embedding
- [ ] Any endpoint not yet implemented returns HTTP 501 with a consistent error body
- [ ] Adding a new operation to `api/openapi.yaml` and regenerating does not break the build
- [ ] Implemented methods override the stub cleanly — no changes to the embedding required

## E2E Tests

None — verified by calling an unimplemented endpoint and confirming HTTP 501 is returned.
