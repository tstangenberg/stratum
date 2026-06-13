---
id: US-0052
tags: [community, launch]
status: ready
---

# US-0052: Set repository metadata

**As a** developer discovering Stratum via GitHub search or trending  
**I want** to immediately understand what the project does  
**So that** I can decide in seconds whether it's relevant to me

## Context

GitHub surfaces the repository description, topics, and website URL on the repo card in search results, the explore page, and trending lists. This is the project's first impression for most visitors and costs nothing to configure.

## Acceptance Criteria

- [ ] Repository description is set: "Schema-first data middleware. Define your domain model in GraphQL SDL — Stratum handles persistence and exposes a GraphQL API automatically."
- [ ] Repository website URL is set (leave blank until a project site exists; update when available)
- [ ] The following topics are added: `graphql`, `postgresql`, `go`, `middleware`, `api`, `open-source`, `data-layer`, `schema-first`
- [ ] Repository social preview image is set (can be a simple text-based image with the Stratum name and tagline)

## E2E Tests

Manual verification: view the repository page as an unauthenticated user and confirm description, topics, and social preview are visible.
