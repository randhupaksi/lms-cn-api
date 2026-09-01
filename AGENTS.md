# Citra Negara LMS API — Agent Guide

> Source of truth for AI-assisted work in this repository. Workspace-wide rules
> are in `../AGENTS.md`; read them first. Architecture detail is in
> `docs/ARCHITECTURE.md`.

## 1. Product and current scope

This is the Go/Gin REST API for Citra Negara LMS. The current implementation is
only a foundation with a health endpoint. Do not add authentication, users,
exam management, grading, CRUD, or other business modules unless the task
explicitly requests them and the product requirements are available.

The API may process student identity, question banks, answers, scores, and
results. These are confidential. Security and authorization are enforced by
the API, never by UI assumptions.

This is a Citra Negara application, not a generic SaaS backend. Do not add
multi-tenant, platform-superadmin, or white-label abstractions speculatively.

## 2. Stack and commands

| Area | Standard |
|---|---|
| Language | Go 1.25.5 |
| HTTP framework | Gin |
| Persistence | GORM + MySQL |
| Auth | JWT when the auth module is introduced |
| Validation | go-playground/validator |
| Module | `lms-cn-api` |
| API prefix | `/api/v1` from configuration |

```bash
go vet ./...
go test ./...
go build ./cmd/api
```

Use `gofmt` for changed Go files. Keep `.env.example` current without adding
real credentials. Do not commit generated binaries or secrets.

## 3. Architecture

The API is a pragmatic modular monolith:

```text
cmd/api             composition root and executable entrypoint
internal/app        bootstrap and server lifecycle
internal/config     environment and application configuration
internal/router     route composition
internal/middleware cross-cutting HTTP middleware
internal/modules    business domains
internal/database   database and migration infrastructure
pkg                 reusable, domain-agnostic packages
```

Dependency direction:

```text
cmd → app → router/middleware → handlers → services → repositories/infrastructure
pkg is reusable and must not contain business-specific rules
```

Each domain module should own its model, DTOs, validation, repository, service,
handler, and route registration. A module must not import another module's
private implementation. Cross-domain workflows need an explicit application
boundary or narrow interface.

## 4. Layer responsibilities

- Handler: parse HTTP input, validate, authorize, call a service, and map the
  result to the response. No direct database queries.
- Service: own business decisions, orchestration, transactions, and use-case
  behavior.
- Repository: the only module layer that directly touches GORM/query details.
- DTO: whitelist request and response fields; never expose a sensitive model
  accidentally.
- Router: map paths, middleware, and handlers; keep business logic out.
- `pkg`: generic response, errors, pagination, validation, security, and
  transport helpers only when genuinely reusable.

## 5. API contract

Use the shared response package for every endpoint. Keep one predictable success
and error envelope across modules. Controllers/handlers must not manually
invent response shapes or expose raw database errors.

Errors returned to clients should be safe and actionable. Log technical detail
server-side with appropriate context, but never include passwords, JWTs, SQL,
stack traces, or sensitive student data in the response.

Use explicit DTO validation and whitelist update fields. Never pass a raw
`map[string]any` or untrusted request body into a model update when sensitive
columns exist.

## 6. Security and data safety

- Authentication and authorization belong to the API.
- Verify ownership/scope for every by-ID read, update, and delete.
- Never trust role, owner, user, or scope identifiers from the request body when
  they should come from authenticated identity.
- Do not log or expose student PII, exam answers, answer keys, scores, or tokens.
- Passwords must be hashed and never returned.
- Use secure, explicit CORS and environment configuration when those concerns
  are introduced.
- Prefer versioned migrations for persistent schema changes. Do not enable
  destructive schema synchronization in production.
- Sensitive operations such as final exam submission must be idempotent and
  server-validated when the exam domain is introduced.

## 7. Module workflow

Before adding a module:

1. Confirm the requirement and API contract.
2. Define ownership, authorization, and sensitive fields.
3. Add the module's DTO/model/repository/service/handler/router boundaries.
4. Register it explicitly in the application composition.
5. Add tests for business rules, authorization, validation, and failure paths.
6. Update `docs/ARCHITECTURE.md` when the module changes system boundaries.

Do not create generic base abstractions until at least two real modules share a
stable pattern. Favor explicit code while the domain is still being discovered.

## 8. Naming and code conventions

- Go package names are lowercase and concise.
- Files use descriptive `snake_case` names where multiple words are needed.
- Exported types/functions use `PascalCase`; local variables use `camelCase`.
- Interfaces describe the capability they abstract, not an implementation.
- Context is passed through I/O and service boundaries.
- Errors are wrapped with operation context and compared with `errors.Is` when
  appropriate.
- Configuration keys are uppercase snake case and documented in `.env.example`.
- Keep handlers thin, services explicit, and repository queries readable.

## 9. Do not touch without explicit need

- Do not change the API contract or add business modules during foundation-only
  work.
- Do not bypass the response package or validation boundary.
- Do not expose raw model structs containing secrets or PII.
- Do not introduce `synchronize: true`-style destructive production behavior.
- Do not commit or push.

## 10. Backend handoff checklist

- [ ] Requirement, ownership, and authorization rules are explicit.
- [ ] Handler/service/repository boundaries are respected.
- [ ] DTOs whitelist sensitive fields.
- [ ] Errors do not leak technical or personal data.
- [ ] Migrations/schema changes are safe and documented.
- [ ] `go vet ./...` passes.
- [ ] `go test ./...` passes.
- [ ] `go build ./cmd/api` passes.
- [ ] No commit or push was performed.
