# Ranvex API Architecture

Ranvex API uses a pragmatic modular monolith. It keeps deployment simple while
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

The router currently exposes only a health endpoint. Authentication,
authorization, exam integrity, and business modules are intentionally deferred
until product requirements are reviewed.

## Conventions

- Use lowercase Go package names and descriptive file names in snake_case.
- Return the shared response envelope from HTTP handlers.
- Pass `context.Context` through I/O and service boundaries.
- Wrap errors with operation context and map expected errors at the HTTP edge.
- Keep transactions close to the service use case that requires them.
- Never commit secrets; environment configuration belongs in `.env.example`.
