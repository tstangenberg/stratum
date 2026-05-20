# ADR-1010: Double Loop TDD with real PostgreSQL — no mocks in E2E

**Status:** Accepted

## Context and Problem Statement

Stratum is a data middleware — its correctness is entirely defined by what it does to a real database. The test strategy must catch regressions in the full stack (HTTP → schema parsing → migration → GraphQL resolver → SQL → PostgreSQL) without sacrificing development speed.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **Double Loop TDD, E2E against real PostgreSQL** | Full stack coverage, no mock drift, catches migration bugs, matches production behaviour exactly | Requires PostgreSQL for tests; slightly slower than in-memory |
| **Unit-first with mocks** | Fast, isolated | Mock/real divergence is the primary source of prod bugs in data middleware; mocked SQL behaviour rarely matches PostgreSQL exactly |
| **Integration tests with in-memory SQLite** | No PostgreSQL dependency | Different SQL dialect, different type system, different JSON aggregation — tests would not reflect PostgreSQL behaviour |

## Decision Outcome

Chosen: **Double Loop TDD — E2E tests drive the outer loop, unit tests drive the inner loop, all E2E tests run against a real PostgreSQL instance.**

```
🔴 E2E test fails
  → 🔴 unit test fails
  → 🟢 unit test passes → refactor
  → (repeat until E2E passes)
🟢 E2E test passes → next E2E test
```

**No mocks in E2E.** Every E2E test starts with a clean PostgreSQL schema, seeds fixtures, runs the full HTTP stack, and asserts on the response. There is no mock database, no mock HTTP client.

**Why no mocks in E2E specifically:**

- Stratum generates SQL from GraphQL SDL at runtime. The correctness of that SQL can only be verified against a real database. An in-memory mock would accept SQL that PostgreSQL rejects, and vice versa.
- Atlas migrations run against real PostgreSQL. Testing migrations against a mock is testing the mock, not Atlas.
- The E2E test fixture (3 Kantone, 6 Ortschaften, 9 PLZ from MVP-1) is small enough that a full test cycle on a local PostgreSQL instance completes in seconds.

Unit tests (inner loop) may use interfaces and test doubles freely — they cover logic that does not touch the database (SDL parsing, schema diffing, plugin registration).
