# AGENTS.md

## Project Overview

QuickPID is a Go implementation of a persistent identifier resolver and management API.
This repository contains both the implementation and the API specification:

- The normative HTTP API specification is [`spec/openapi.yaml`](spec/openapi.yaml).
- Supporting specification notes and the JSON flow-test format live in [`spec/README.md`](spec/README.md).
- The Go code in the repository root implements the API, storage backends, service layer, HTTP server, and CLI entrypoints.

Treat `spec/openapi.yaml` as the source of truth for externally visible API behavior.
If you change routes, request or response shapes, validation rules, or error behavior, review the matching spec files and tests in the same change.

## Repository Layout

- `api/`: API-facing types, JSON models, error strings, validation helpers, and typed validated values.
- `backend/`: Storage interfaces and backend contracts, with implementations in `backend/memory` and `backend/gorm`.
- `service/`: Business logic over a `backend.Store`, including auth mode behavior, limits, and higher-level operations.
- `server/`: HTTP routing, request parsing, auth wrappers, Swagger/OpenAPI serving, and HTTP-to-service translation.
- `cmd/`: Runnable binaries and shared CLI/bootstrap logic.
- `spec/`: OpenAPI spec, narrative documentation, and JSON flow tests.
- `internal/`: Internal helpers and shared test support such as `internal/pidtest`.

## API Type Conventions

The `api` package uses `Valid*` wrapper types instead of passing raw strings through the system.
Important examples include:

- `api.ValidNamespaceID`
- `api.ValidPID`
- `api.ValidUsername`
- `api.ValidPassword`
- `api.ValidBaseURI`

Construct these via their `New*` helpers, such as `api.NewNamespaceID(...)` and `api.NewBaseURI(...)`.
Do not bypass this pattern by threading unvalidated strings through `service`, `server`, or `backend` code.
Backend interfaces generally take validated values as input and return simple, non-validated, storage-friendly or JSON-ready values as output.
Service-layer code should also accept validated inputs, call the backend interface methods, and perform the appropriate validation and business logic, and return API-facing values that are directly JSON-marshalable or the appropriate wrapped errors for the server layer to translate.
The Server layer is responsible for validating input logic. 

## Setup And Dependencies

- Required Go version is declared in [`go.mod`](go.mod), currently `1.26.5` but might have changed since the last update of this document. 
- This repository is a single Go module; use standard Go tooling from the repo root.
- Main external tooling referenced by the repo:
  - `golangci-lint`
  - `go-check-spellchecker`
  - `gogenlicense`
  - `cspell` in CI

If a tool is missing locally, prefer `go tool ...` for Go-based tools that are already declared in `go.mod`.

## Testing And Validation

Run the full Go test suite:

```sh
go test ./...
```

CI runs the verbose form:

```sh
go test -v ./...
```

Run lint:

```sh
go tool golangci-lint run ./...
```

Update Go source spellchecker annotations:

```sh
go tool go-check-spellchecker -fix ./...
```

Run code generation, including license notice updates:

```sh
go generate ./...
```

Focused validation patterns:

- Run a subset of packages, for example: `go test ./service ./server/...`
- Run a specific test by name: `go test ./... -run '<TestName>'`

Testing conventions:

- Standard Go `*_test.go` files are used throughout the repo.
- Shared backend contract tests live in [`internal/pidtest/store.go`](internal/pidtest/store.go). Backend changes should stay compatible with these shared suites.
- HTTP/API behavior is also exercised through JSON flow tests in `spec/tests/*.json`. If you change externally visible behavior, review whether matching spec tests need updates or new coverage.

Spell-checking:

- CI runs `cspell` via [`.github/workflows/cspell.yaml`](.github/workflows/cspell.yaml).
- Repository spelling exceptions are configured in [`cspell.json`](cspell.json).
- When fixing cspell errors, first run `go tool go-check-spellchecker -fix ./...`. That command regenerates import- and package-name `//spellchecker:words` comments; do not edit those comments by hand, and do not add custom words into them. Only add custom `//spellchecker:words` comments outside of the auto-managed ones.
- Decide where an intentional project-specific term belongs: put it in [`cspell.json`](cspell.json) when it is used widely across the repo; otherwise keep it in a file-local `//spellchecker:words` comment.
- When editing Markdown, YAML, OpenAPI, comments, or error strings, keep spelling clean and update `cspell.json` or a custom `//spellchecker:words` comment if a project-specific term is intentionally introduced.
- If `cspell` is installed locally, run it against the files you changed using [`cspell.json`](cspell.json).

Before finishing a change, prefer the narrowest relevant validation first, then broader checks if the change crosses package or API boundaries.

## Code Style And Conventions

- Follow idiomatic Go.
- Keep changes small and package-local when possible.
- Respect the strict lint configuration in [`.golangci.yml`](.golangci.yml).
- Preserve existing error-handling patterns, especially API error mapping in `api`, request parsing in `server`, and business-rule enforcement in `service`.
- Keep pagination, validation, and authorization behavior aligned with existing helper patterns rather than reimplementing them ad hoc.
- Use the `api.Valid*` types and constructors for validated identifiers and credentials.
- Keep backend interfaces in the established style: validated values in, simple values out.
- Keep service interfaces in the established style: validated values in, validated behavior and JSON-ready API outputs out.

Error mapping conventions:

- `api.ErrorString` in [`api/errors.go`](api/errors.go) is the stable API error identifier exposed to clients.
- When returning an API-level failure, preserve the appropriate `api.ErrorString` using `api.WithErrorString(...)`.
- Wrapped underlying errors are for logging and diagnostics; the stable `ErrorString` is what the API exposes and what HTTP status mapping is based on.
- Server code should translate annotated errors using the existing `api.ErrorString` machinery instead of inventing ad hoc response formats.

Spelling conventions matter in this repo:

- Go files often use `//spellchecker:words ...` comments for unusual identifiers and domain terms.
- Import- and package-name spellchecker comments are maintained by `go tool go-check-spellchecker -fix ./...`; leave those alone and only add custom words in separate comments.
- Documentation and spec files rely on [`cspell.json`](cspell.json).
- Prefer `cspell.json` for terms used in many places; prefer a file-local comment for one-off terms.
- If you add a new uncommon term, update the relevant spellchecker mechanism in the same change.

## Build And Deployment Notes

CI workflows are in [`.github/workflows/`](.github/workflows/):

- `go.yaml`: tests and lint
- `cspell.yaml`: spelling checks
- `docker.yaml`: publishes container images on pushes to `main`

Container build files:

- `Dockerfile.mem`
- `Dockerfile.sqlite`
- `Dockerfile.postgres`

The Docker publish workflow builds and pushes images for all three entrypoints.

## Security And Operational Notes

- `-anon` disables authentication and authorization checks for resolver operations. Be explicit about whether a change is intended for anonymous mode, authenticated mode, or both.
- The default startup path bootstraps a root superuser if no accounts exist. Avoid breaking this flow unintentionally.
- Do not commit real secrets, production DSNs, or private credentials.
- The repository code is intentionally unlicensed for reuse, while `spec/` is licensed separately. Preserve that distinction when editing docs.

## Change Guidance For Agents

- Read this file before working in the repo; if nested `AGENTS.md` files are added later, the nearest one should take precedence.
- Before changing API routes or wire formats, inspect:
  - [`spec/openapi.yaml`](spec/openapi.yaml)
  - [`spec/README.md`](spec/README.md)
  - relevant handlers in `server/`
  - relevant logic in `service/`
  - relevant request/response types in `api/`
- When editing the documented resolver routes in [`server/server.go`](server/server.go), follow the route-sync guidance in that file's comment block near the route declarations. 
- Before changing validation or identifiers, inspect the relevant `api.Valid*` type and keep parser, handler, service, and backend behavior consistent with it.
- Backend changes should preserve the `backend.Store` contract and shared behavior expected by `internal/pidtest`.
- If you change user-visible API behavior, update code, tests, and spec artifacts together.

## OpenAPI Description Conventions

When editing [`spec/openapi.yaml`](spec/openapi.yaml):

- Every `description` on a path parameter, `requestBody`, response, response header, schema component, or schema property must be unique across the whole document (vacuum `description-duplication`).
- Phrase path-level descriptions in terms of the current operation, not a shared generic string.
  Prefer: "Authentication is required … when listing mounts."
  Avoid: "Authentication is required or the credentials are incorrect."
- Keep status-code meaning consistent (401 = auth missing/wrong, 403 = authenticated but forbidden, 404 = missing or unavailable in anonymous mode, 413 = body too large, 500 = internal failure), but always include the operation context.
- Pagination `limit` / `offset` and path params like `namespace`, `baseUri`, `pid`, and `username` should say what they paginate or identify for that route.
- Schema and property descriptions must likewise be unique and name their schema context (for example, "Total number of mounts available." vs "Total number of users available.").
- Do not copy path/response wording into schema (or property) descriptions, and do not reuse schema wording on paths; vacuum treats them as one global pool of description strings.
- Each `components.schemas` entry should have a top-level `description` (vacuum `component-description`).
- After description edits, re-run vacuum and clear remaining description findings before considering the change done.
