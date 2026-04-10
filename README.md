# Quickpid

A quick implementation of the PID Resolver API ([`server/openapi.yaml`](server/openapi.yaml)).

## Commands

**Persistent SQLite (default)** — stores data in a SQLite file using [GORM](https://gorm.io) and a pure-Go driver ([glebarez/sqlite](https://github.com/glebarez/sqlite), no CGO):

```bash
go run ./cmd/quickpid
```

Database DSN (optional), checked in order:

- `QUICKPID_DSN`
- `DATABASE_URL`
- default: `quickpid.db?_pragma=foreign_keys(1)` (file `quickpid.db` in the current directory)

**In-memory only** — no persistence, useful for local testing:

```bash
go run ./cmd/quickpid-mem
```

Both serve the HTTP API and embedded Swagger UI under `/api/v2/`. Port defaults to `8080`; override with `PORT`.

## Layout

- [`api/`](api/) — `Resolver` interface and request/response types
- [`server/`](server/) — HTTP handler and embedded OpenAPI spec
- [`mem/`](mem/) — in-memory `Resolver`
- [`gormstore/`](gormstore/) — GORM-backed `Resolver` (any dialector; SQLite used by `cmd/quickpid`)
