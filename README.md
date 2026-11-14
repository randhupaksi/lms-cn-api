# Citra Negara API

The Go/Gin REST API for Citra Negara's single-school learning and examination
platform. It owns authentication, authorization, academic and exam rules,
grading, publication, school governance, and persistence.

## Stack and prerequisites

- Go 1.25.5
- MySQL and a local database configured through environment variables
- Git for source checkout

The API uses GORM, JWT access tokens, opaque rotating refresh sessions,
bcrypt, and request validation. See the [documentation index](docs/INDEX.md)
for architecture, security, governance, operations, and contract details.

## Local development

1. Copy `.env.example` to `.env` and set local database and application values
   in that ignored file. Do not copy real secrets into documentation or Git.
2. Start the configured MySQL service and create the local database.
3. From this repository, run:

   ```bash
   go run ./cmd/api
   ```

   The API applies pending versioned migrations during startup. Demo data is
   controlled by `SEED_DEMO_DATA` and is allowed only for development.
4. With the default API prefix, process liveness is available at
   `/api/v1/health/live`; readiness at `/api/v1/health` also checks the
   database.

For a controlled first administrative account, after the schema exists,
`cmd/bootstrap-admin` reads
`BOOTSTRAP_ADMIN_IDENTIFIER`, `BOOTSTRAP_ADMIN_NAME`, `BOOTSTRAP_ADMIN_PASSWORD`,
and the optional `BOOTSTRAP_ADMIN_ROLE` from the process environment. Supply
those values through a secure local environment or secret manager; the password
must not be placed in command history or committed files. The new account must
change its password at first sign-in.

## Checks

Run from this repository root:

```bash
go vet ./...
go test ./...
go build ./cmd/api
```

Database integration checks are opt-in and require the isolated test setup
described in the relevant API documents. Backup, restore, deployment, and
pre-exam procedures are in [Operations](docs/OPERATIONS.md).

## Repository boundaries

HTTP routes and middleware compose the application. Domain modules own their
business rules and persistence coordination; reusable packages remain
domain-agnostic. The web client is maintained separately in
[`../lms-cn-web`](../lms-cn-web/README.md), and workspace-wide product status is
recorded in [`../docs/IMPLEMENTATION_STATUS.md`](../docs/IMPLEMENTATION_STATUS.md).
