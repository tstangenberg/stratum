# ADR-1015: Generic async job system with JobPlugin and StoragePlugin

**Status:** Accepted

## Context and Problem Statement

Stratum needs bulk data import and export. These operations can take seconds to minutes on large datasets — too long for a synchronous HTTP request. A job system is needed. The question is whether to build an import/export-specific mechanism or a generic extensible one that import/export merely validates.

This ADR also adds two new plugin types to the architecture established in ADR-1008.

## Considered Options

| Option | Pros | Cons |
|--------|------|------|
| **Synchronous import/export** | No job infrastructure needed | Times out on large datasets; not extensible |
| **Async jobs, import/export-specific** | Simpler scope | Dead end — next long-running operation needs a new mechanism |
| **Generic async job system with JobPlugin** | Import/export validate the pattern; any future long-running operation plugs in without core changes | More upfront design |

## Decision Outcome

Chosen: **generic async job system with `JobPlugin` and `StoragePlugin`**.

**Job lifecycle:** `pending` → `running` → `done` | `failed`. Jobs are stored in PostgreSQL (`stratum_jobs` table in `stratum_system` schema). Workers claim jobs atomically using `SELECT ... FOR UPDATE SKIP LOCKED` — no external queue required.

**API:** single top-level resource `/api/jobs`. Job type is expressed in the body (`"operation": "export" | "import" | ...`), not the URL. This keeps the URL stable as new job types are added.

**`JobPlugin` interface** (extends ADR-1008's plugin catalogue):
```go
type JobPlugin interface {
    Operation() string
    Execute(ctx context.Context, job Job, storage StoragePlugin) error
}
```
Adding a new long-running operation = writing a new `JobPlugin`. No core changes.

**`StoragePlugin` interface** (extends ADR-1008's plugin catalogue):
```go
type StoragePlugin interface {
    Write(ctx context.Context, jobID string, r io.Reader) error
    Read(ctx context.Context, jobID string) (io.ReadCloser, error)
    Delete(ctx context.Context, jobID string) error
}
```
Storage backends are pluggable. Compression is an implementation detail of each backend. MVP ships PostgreSQL + gzip. S3 and filesystem follow as plugins when needed.

**Import data** is sent inline in the POST body (`application/json` or `Content-Encoding: gzip`). The server writes it to storage before returning 202, so the async worker never reads from the original request.

**Post-MVP extensions** that fit this model without schema or API changes: recurring jobs (`cron` column), planned jobs (`scheduled_at` column), schema-level operations, cross-type operations. These columns are intentionally deferred and added via Atlas migration when needed.

**Consequences:**
- Import and export are `JobPlugin` implementations, not special-cased in core
- Any future long-running operation (index generation, data transformation, validation) follows the same pattern
- The `StoragePlugin` interface makes result storage swappable — PostgreSQL for MVP, S3 for production-scale deployments
