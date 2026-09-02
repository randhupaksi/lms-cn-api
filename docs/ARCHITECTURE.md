# Citra Negara LMS API Architecture

Citra Negara LMS API uses a pragmatic modular monolith. It keeps deployment simple while
giving each future business domain a clear ownership boundary.

```text
cmd/api/             # executable composition root
internal/app/        # application bootstrap and server lifecycle
internal/config/     # environment configuration
internal/router/     # HTTP route composition
internal/middleware/ # cross-cutting HTTP concerns
internal/modules/    # business domains
internal/database/   # database and migration infrastructure
pkg/                 # domain-agnostic reusable packages
```

## Module rules

Each domain module may own its model, DTO, validation, service, and handler.
Handlers translate HTTP concerns; services own business decisions; database
access stays behind the module or an explicit infrastructure boundary.
Modules must not reach into another module's private implementation.

## Dependency direction

```text
cmd -> app -> router/middleware -> module handlers -> module services -> infrastructure
pkg is reusable and domain-agnostic
```

Implemented modules are `auth`, `users`, `academics`, `questions`, `exams`,
`attempts`, `grading`, `results`, `audit`, `monitoring`, `materials`,
`assignments`, and `analytics`. Cross-module access uses explicit service
boundaries for course ownership and enrollment checks. Query composition stays
inside each repository; services coordinate authorization and business rules.

## Security and exam integrity

- Short-lived JWT access tokens identify a revocable server-side session.
- Refresh tokens are opaque, hashed at rest, rotated, and delivered through an
  HttpOnly cookie. Disabled users and revoked sessions are rejected server-side.
- Temporary credentials carry a signed password-change requirement and cannot
  access business endpoints until changed.
- Exam questions and answer options are snapshotted before publication.
- Attempt start and every answer save use durable idempotency records.
- Deadlines, finalization, objective grading, result visibility, and ownership
  checks are enforced in database-backed API transactions.
- Student contracts omit correct-answer fields.

## Operations and observability

- `/health/live` reports process liveness; `/health` verifies database
  readiness before reporting the service ready.
- Structured request logs include request ID, route, status, latency, and
  client address without request bodies, credentials, answers, or tokens.
- Login protection combines a network-level request ceiling with a stricter
  per-identifier throttle whose in-memory key is SHA-256 hashed.
- Audit reads are admin-only and operational monitoring remains course-scoped.
- Versioned migrations are the only production schema path. Backup, restore,
  load-test, and incident procedures are documented in `docs/OPERATIONS.md`.

## Conventions

- Use lowercase Go package names and descriptive file names in snake_case.
- Return the shared response envelope from HTTP handlers.
- Pass `context.Context` through I/O and service boundaries.
- Wrap errors with operation context and map expected errors at the HTTP edge.
- Keep transactions close to the service use case that requires them.
- Never commit secrets; environment configuration belongs in `.env.example`.
