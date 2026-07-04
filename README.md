# Bicpid

This is a fork of [github.com/tkw1536/quickpid](https://github.com/tkw1536/quickpid).
Because it is my personal copyright, the original license is removed, and you're not allowed to use this.

<!--
[![CI](https://github.com/tkw1536/bicpid/actions/workflows/go.yaml/badge.svg)](https://github.com/tkw1536/bicpid/actions/workflows/go.yaml)
-->

In the scientific community it is common to issue [persistent identifiers](https://en.wikipedia.org/wiki/Persistent_identifier) -- or PIDs for short -- to objects to be able to identify and refer to them unambiguously.
The term object can include papers, presentations, other publications as well as files, web pages or any kind of object.

One type of persistent identifier is the [Digital Object Identifier](https://en.wikipedia.org/wiki/Digital_object_identifier) commonly referred to as a DOI.
DOIs are centrally administered by the [International DOI Foundation](https://en.wikipedia.org/wiki/International_DOI_Foundation), and it is common for universities to issue a DOI for each publication.

DOIs incur licensing fees for each identifier issued making them unsuitable for use cases where large sets of objects require identifiers.
They also introduce a dependency on an external organization.

This repository instead contains a specification and implementation for an alternate system capable of issuing persistent identifiers.
It roughly consists of two parts:

- An API specification and associated test cases in the [spec](./spec/) directory
- An implementation of this API in the root directory of this repository

The rationale behind the design of the API, and how it fits into a larger PID System, are described in the spec directory.
This README only describes the implementation.

## API Implementation

The API is implemented in modern idiomatic [Go](https://go.dev).
It can be installed and run like any other Go program.

The code has several entry points, each using a different backend for storage:

- [quickpid-mem](./cmd/quickpid-mem/main.go), an in-memory backend.
  It is intended to demonstrate the functionality of the API, and not intended as a production system.
- [quickpid-sqlite](./cmd/quickpid-sqlite/main.go) a backend using an SQLite database for storage.
- [quickpid-postgres](./cmd/quickpid-postgres/main.go) a backend using an external Postgres database for storage.

The commands produce informational output on STDOUT, and produce logs on STDERR.
By default, each command runs the API at:

```http://localhost:8080/api/v2```

Each command can be run like:

```
go run ./cmd/quickpid-[IMPLEMENTATION] [...FLAGS]
```
Each command can be invoked with a `-help` flag to list available options.

With the exception of the storage backend, all other code is shared between the implementations.

The two database implementations are based on [GORM](https://gorm.io) and appropriate pure go database drivers.
Beyond the standard library, dependencies are otherwise kept to a minimum.
All parts of the code are well-documented and include tests, which can be run with `go test`, and are checked by CI.

To implement license notices, [gogenlicense](https://github.com/tkw1536/gogenlicense) is used.
These should be updated using `go generate ./...` whenever the dependencies change.

<!--
## Docker images

Multi-arch images are published to GitHub Container Registry:

- `ghcr.io/tkw1536/quickpid-mem:latest`
- `ghcr.io/tkw1536/quickpid-sqlite:latest`
- `ghcr.io/tkw1536/quickpid-postgres:latest`

Examples:

- **In-memory backend**:

  `docker run --rm -p 8080:8080 ghcr.io/tkw1536/quickpid-mem:latest`

- **SQLite backend** (persist DB in a volume at `/data`):

  `docker run --rm -p 8080:8080 -v quickpid_data:/data ghcr.io/tkw1536/quickpid-sqlite:latest`

- **Postgres backend** (set `DSN` to point at your Postgres):

  `docker run --rm -p 8080:8080 -e DSN='host=postgres user=postgres password=postgres dbname=quickpid port=5432 sslmode=disable' ghcr.io/tkw1536/quickpid-postgres:latest`
-->

## Authentication & Authorization

Quickpid can protect write and list operations with HTTP Basic authentication.
Authentication and authorization are loaded from a JSON file passed via
`-basic-auth-file /path/to/auth.json`.

If authentication is enabled, protected endpoints respond with an HTTP Basic
challenge, so browsers should prompt for a username and password automatically.

Permission levels are:

- `none`: may read individual PID records, but may not list namespace contents or create/update resources (default)
- `contributor`: may do everything from `none`, and may create resources and submit batch creates
- `editor`: may do everything from `contributor`, and may also list and update resources

Authorization is determined in two steps:

- `superusers`: usernames listed here may create new namespaces and automatically have `editor` access to every namespace
- `permissions`: per-namespace permission levels for individual users; users not listed for a namespace default to `none`

The authentication file stores HTTP Basic credentials in `htpasswd` format under
the `users` key. Each entry is a `username:hash` string. Hashes should be
generated with a tool such as `htpasswd`; for example:

```sh
htpasswd -nB admin
```

An example authentication file looks like:

<!-- spellchecker:disable -->
```json
{
  "users": [
    "admin:$apr1$b6C7u008$LIOlAY4ltdsPD.mbKPRjc.",
    "editor:$2y$05$y6RJtugGZOvBPuDg/f8.7O8GqTZAvUD8nNm1ipNrS8LiAm28fbdk2"
  ],
  "superusers": ["admin"],
  "permissions": {
    "my-namespace": {
      "editor": "editor",
      "contributor": "contributor"
    }
  }
}
```
<!-- spellchecker:enable -->

## Future Technical Work

- add [Golangci-lint](https://golangci-lint.run)
- use something other than uuid for namespace generation to drop dependency
- consider adding a minimal public frontend
- consider removing the `/resolver/` prefix

## LICENSE

The code in this repository is unlicensed.

The [`spec` directory](./spec/README.md), which contains the API specification and test cases, is licensed separately to enable re-use.
In addition to being available under AGPL, it is also available under the terms of the [Creative Commons Attribution-ShareAlike 4.0 International](https://creativecommons.org/licenses/by-sa/4.0/) license.
See [the `spec` README](./spec/README.md) for details.