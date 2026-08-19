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

- `api/`: API-facing types, JSON models, error codes, validation helpers, typed validated values, and the declarative scope model (`scopes.go`, `scopes_user.go`, `scopes_namespace.go`) plus pure evaluation helpers. Domain files follow OpenAPI tags (`user.go`, `credentials.go`, `namespaces.go`, `resources.go`, `mounts.go`, `roles.go`, `quickpid.go`).
- `backend/`: Storage interfaces and backend contracts, split the same way (`user.go`, `credentials.go`, `namespaces.go`, `resources.go`, `mounts.go`, `roles.go`, plus `backend.go`). Implementations live in `backend/memory` and `backend/gorm` as matching `memory_*` / `gorm_*` files. `Backend` embeds the domain backends (`UserBackend`, `CredentialsBackend`, and so on).
- `service/`: Business logic over a `backend.Backend`, including limits and higher-level operations, also split by OpenAPI tag (`user.go`, `credentials.go`, `namespaces.go`, `resources.go`, `mounts.go`, `roles.go`, `quickpid.go`) with shared pieces in `service.go`, `options.go`, `runtime.go`, and `errors.go`. Scope checks for HTTP routes are not performed here (see Permission Handling below).
- `server/`: HTTP routing, request parsing, Swagger/OpenAPI serving, and HTTP-to-service translation. Route wiring lives in `server/server.go`; handlers are split by OpenAPI tag (`user.go`, `credentials.go`, `namespaces.go`, `resources.go`, `mounts.go`, `roles.go`, `quickpid.go`) with shared parsing in `parse.go`. Auth, logging, serialization, and permission checks live in `server/internal/lowlevel`.
- `cmd/`: Runnable binaries and shared CLI/bootstrap logic.
- `spec/`: OpenAPI spec, narrative documentation, and JSON flow tests.
- `internal/`: Internal helpers and shared test support such as `internal/pidtest`.

## GORM Backend Conventions

The SQLite/Postgres storage implementation lives in [`backend/gorm`](backend/gorm). Keep it aligned with these rules when changing models or CRUD:

### Tables

- Plural entity names for top-level tables (`users`, `namespaces`, `resources`, `mounts`).
- Dependent tables use `{parent}_{child}` (`user_passwords`, `user_api_keys`, `user_api_key_user_scopes`, `namespace_tags`, `namespace_roles`, `resource_tags`).

### Columns

- User foreign keys are always `username`.
- Namespace foreign keys are always `namespace_id`.
- API key identifiers are always `key_id` (parent PK and child FK).
- Timestamps are `created_at`, `updated_at`, and `expires_at` (store UTC). Use Go field names other than `CreatedAt` / `UpdatedAt` (for example `DateCreated` / `DateUpdated`) so GORM does not overwrite them with wall-clock time; the service-injected `now` owns those values.
- Ordered children include a `pos` integer in the composite primary key.

### Storage shape

- Store only plain column types (`string`, `bool`, `[]byte`, `time.Time`, and pointers thereof). Do not JSON/gob/serializer-encode values into a column, and do not use custom `driver.Valuer` / `sql.Scanner` wrappers to pack data.
- Collections live in junction tables (one value per row). Optional 1:1 data (passwords) uses a separate table; absence of a row means unset.
- Scalar fields stay on the parent row. Row ↔ API mapping is field assignment only (`toSpec` / building rows from request fields).

### Foreign keys and deletes

- Declare associations with `foreignKey` / `references` and `constraint:OnDelete:CASCADE`.
- Parent models expose association fields for their children.
- Cascading parent deletes use the shared `cascadingDelete` helper (`Select(clause.Associations).Delete`); do not hand-roll per-table delete lists for association-backed children. Exception: `namespace_roles` are deleted by `username` on user delete without a DB FK to `users`, because roles may reference usernames that do not (yet) have a user row.
- Replace ordered children with `Delete` of existing child rows then `Create` of the new set. Do not use `Association.Replace` for NOT NULL foreign keys — GORM nulls removed FKs instead of deleting rows.
- Prefer nested `Create` (or `CreateInBatches`) so associations are written with the parent. When updating a parent that was loaded with `Preload`, `Omit` association fields on `Save` so preloaded children are not rewritten.

## Permission Handling

Authorization for HTTP routes is owned by [`server/internal/lowlevel`](server/internal/lowlevel), not by ad-hoc checks in `service`.

### Scope model

There are two kinds of scopes, both defined in the `api` package:

- [`api.UserScope`](api/scopes_user.go): global actions (not tied to a `{namespace}` path). Examples: list mounts, create users, issue keys.
- [`api.NamespaceScope`](api/scopes_namespace.go): actions on a specific namespace. Examples: get/update namespace, create resources, manage roles.

Each scope constant has a matching definition (`UserScopeDefinition` / `NamespaceScopeDefinition`) that declares:

| Flag | Meaning |
| --- | --- |
| `AnonymousMode` | Whether the action is available when the server runs with `-anon`. |
| `AllowUnauthenticated` | Whether a missing caller is allowed (authenticated mode only). |
| `RequireSuperuser` | Whether only superusers may perform the action. |
| `MinRole` (namespace only) | Minimum explicit namespace role that grants access (`none` < `contributor` < `editor` < `manager`). `RoleNone` means no role alone is enough (superuser or a dedicated definition path is required). |

Special scopes that do not map 1:1 to a single OpenAPI operation:

- `api.ScopeImpersonate` (user): may impersonate another user via `X-Impersonate-User`; the call then inherits that user's permissions.
- `api.ScopeSeeDeletedResource` (namespace): gates whether a deleted resource is returned un-redacted.

Pure evaluators live in [`api/scopes.go`](api/scopes.go):

- `EvaluateAnonymousModeUserScope` / `EvaluateAnonymousModeNamespaceScope`
- `EvaluateUserScope` / `EvaluateNamespaceScope`

Evaluation also consults `caller.Method().AllowsUserScope` / `AllowsNamespaceScope` so an authentication method can impose extra restrictions (for example a future namespace-limited token). Today the built-in methods allow all scopes.

### HTTP wiring

[`server/internal/lowlevel`](server/internal/lowlevel) authenticates the caller, evaluates the required scope, and only then invokes the server handler. Wire routes in [`server/server.go`](server/server.go) with:

- `UserScope` for actions that are not namespace-specific
- `NamespaceScope` for a fixed scope on a `{namespace}` path (also parses and passes `api.ValidNamespaceID`)
- `DynamicNamespaceScope` when the scope depends on request data (for example own vs other role endpoints)
- `Public` only for endpoints that intentionally ignore authentication

In anonymous mode, lowlevel short-circuits to the anonymous-mode evaluators and does not load namespace roles. In authenticated mode, namespace checks load the caller's explicit role for that namespace, then call `EvaluateNamespaceScope` (superusers bypass role requirements).

`service` implements business logic and assumes the HTTP layer already enforced the route scope. Do not re-check the same scope in service methods.

Exceptions where a second, conditional scope check is needed after the route scope:

- Listing namespaces: the handler filters each item with `ScopeGetNamespace` via a callback into `Service.ListNamespaces`.
- Reading a deleted resource: the handler passes a `shouldRedact` callback into `Service.GetResource` that evaluates `ScopeSeeDeletedResource` through lowlevel. Service does not evaluate the scope itself.

When adding or changing a protected route:

1. Add or update the scope constant and definition in [`api/scopes_user.go`](api/scopes_user.go) or [`api/scopes_namespace.go`](api/scopes_namespace.go) (keep order aligned with `server/server.go` and the OpenAPI operation list).
2. Wire the route with the matching lowlevel helper in [`server/server.go`](server/server.go).
3. Keep service free of duplicate scope checks for that route.
4. List every error code the lowlevel helper or handler can return in the route's `allowedErrors` slice.
5. Update [`spec/tests/298_resolver_scopes_list.json`](spec/tests/298_resolver_scopes_list.json) to reflect any added, removed, or modified scope entries (the test asserts the exact JSON response of the `/resolver/scopes` endpoint, which lists every defined scope and its definition).

`-anon` (anonymous mode) is modeled per scope via each action's `AnonymousMode` flag; lowlevel evaluates that before calling the handler.

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

- Required Go version is declared in [`go.mod`](go.mod), currently `1.27rc3` but might have changed since the last update of this document.
- This repository is a single Go module; use standard Go tooling from the repo root.
- Main external tooling referenced by the repo:
  - `golangci-lint`
  - `go-check-spellchecker`
  - `gogenlicense`
  - `cspell` in CI
  - `vacuum` in CI (OpenAPI lint)

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
- Keep pagination and validation aligned with existing helper patterns rather than reimplementing them ad hoc.
- Keep authorization aligned with lowlevel scope helpers and `api` scope definitions; do not add ad-hoc permission checks in handlers or service for routes already covered by `UserScope` / `NamespaceScope`.
- Use the `api.Valid*` types and constructors for validated identifiers and credentials.
- Keep backend interfaces in the established style: validated values in, simple values out.
- Keep service interfaces in the established style: validated values in, validated behavior and JSON-ready API outputs out.

Error mapping conventions:

- `api.ErrorCode` in [`api/code.go`](api/code.go) is the stable API error identifier exposed to clients.
- When returning an API-level failure, preserve the appropriate `api.ErrorCode` using `api.WithErrorCode(...)`.
- Wrapped underlying errors are for logging and diagnostics; the stable `ErrorCode` is what the API exposes and what HTTP status mapping is based on.
- Server code should translate annotated errors using the existing `api.ErrorCode` machinery instead of inventing ad hoc response formats.

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
- `vacuum.yaml`: OpenAPI lint via vacuum
- `docker.yaml`: publishes container images on pushes to `main`

Container build files:

- `Dockerfile.mem`
- `Dockerfile.sqlite`
- `Dockerfile.postgres`

The Docker publish workflow builds and pushes images for all three entrypoints.

## Security And Operational Notes

- `-anon` disables authentication and authorization checks for resolver operations according to each scope's `AnonymousMode` flag (evaluated in lowlevel). Be explicit about whether a change is intended for anonymous mode, authenticated mode, or both.
- The default startup path bootstraps a root superuser if no accounts exist. Avoid breaking this flow unintentionally.
- Do not commit real secrets, production DSNs, or private credentials.
- Repository code is licensed under AGPL-3.0 ([`LICENSE`](LICENSE)). Content under `spec/` is dual-licensed AGPL-3.0 and CC BY-SA 4.0 ([`spec/LICENSE`](spec/LICENSE)). Preserve that distinction when editing docs.

## Change Guidance For Agents

- Read this file before working in the repo; if nested `AGENTS.md` files are added later, the nearest one should take precedence.
- Before changing API routes or wire formats, inspect:
  - [`spec/openapi.yaml`](spec/openapi.yaml)
  - [`spec/README.md`](spec/README.md)
  - relevant handlers in `server/`
  - relevant logic in `service/`
  - relevant request/response types in `api/`
  - relevant scope definitions in `api/scopes_*.go` when permissions change
- When editing the documented resolver routes in [`server/server.go`](server/server.go), follow the route-sync guidance in that file's comment block near the route declarations.
- When changing who may call an endpoint, update the scope in `api/scopes_*.go` and the lowlevel wiring in `server/server.go`; do not push permission checks back into `service`.
- Before changing validation or identifiers, inspect the relevant `api.Valid*` type and keep parser, handler, service, and backend behavior consistent with it.
- Backend changes should preserve the `backend.Backend` contract and shared behavior expected by `internal/pidtest`.
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
