# Database Infrastructure

This package owns database connection setup, versioned SQL migrations, shared
database helpers, and development-data orchestration. Domain rules and
module-specific queries remain in their owning modules.

## Responsibilities

- `database.go` opens the configured GORM/MySQL connection and exposes the
  connection lifecycle to application bootstrap.
- `migrations.go` applies pending numbered SQL migrations in filename order
  and records applied versions. Migrations are the production schema path;
  do not replace them with automatic model synchronization.
- `seeder*.go` composes fictional local data for development only. The
  application runs it only when configuration allows `SEED_DEMO_DATA`.
- Tests in this package verify migration and seeder behavior; database-backed
  checks require an isolated disposable database.

Review [API architecture](../../docs/ARCHITECTURE.md),
[governance schema notes](../../docs/SCHOOL_GOVERNANCE.md), and the
[operations runbook](../../docs/OPERATIONS.md) before changing migration or
startup behavior. Never put credentials or real student records in a seed
fixture or documentation.
