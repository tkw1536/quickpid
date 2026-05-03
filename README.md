# Quickpid

[![CI](https://github.com/tkw1536/quickpid/actions/workflows/test.yaml/badge.svg)](https://github.com/tkw1536/quickpid/actions/workflows/test.yaml)

> [!WARNING]
> This README is still a work in progress and incomplete.

This repository holds a specification and implementation of the backend of a PID system. 
PID stands for Persistent Identifier - an identifier for an object that does not change.

## Commands

**Persistent SQLite (default)** — stores data in a SQLite file using [GORM](https://gorm.io) and a pure-Go driver ([glebarez/sqlite](https://github.com/glebarez/sqlite):

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

## LICENSE

The contents of this repository are available under the terms of the [GNU Affero General Public License 3.0](https://www.gnu.org/licenses/agpl-3.0.en.html) license, see [the LICENSE file](./LICENSE).

To enable re-use, the specification and associated test data in the [`spec`](./spec/README.md) directory are *additionally* available under the terms of the [Creative Commons Attribution-ShareAlike 4.0 International](https://creativecommons.org/licenses/by-sa/4.0/), see [The Spec README](./spec/README.md) for details.