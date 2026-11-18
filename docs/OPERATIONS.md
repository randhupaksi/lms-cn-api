# Citra Negara API Operations Runbook

This runbook describes the minimum controls for deployment and an important
school examination. It complements the workspace [release gates](../../docs/IMPLEMENTATION_STATUS.md)
and the [API documentation index](INDEX.md); it does not replace
infrastructure-specific procedures.

## Deployment prerequisites

1. Use a dedicated MySQL database and restricted application account.
2. The API applies pending versioned migrations in filename order and records
   each applied version in `schema_migrations`. The current source range is
   `000001`–`000009`. Review the deployment log and resulting schema before
   continuing.
3. Create a verified database backup before applying a schema change.
4. Configure production secrets outside the repository.
5. Set `COOKIE_SECURE=true`, an explicit `ALLOWED_ORIGINS`, a strong
   `JWT_SECRET` of at least 32 random characters, `ACCESS_TOKEN_TTL=15m`, and
   `REFRESH_TOKEN_TTL=24h` for a 24-hour absolute login session.
6. Run the API and web checks documented in their repository READMEs.
7. Verify `/api/v1/health/live` and `/api/v1/health` after deployment.

Course material attachment bytes are stored in MySQL and are included in the
database backup. Uploads are capped at 8 MiB per file and 20 MiB per material;
plan backup capacity and retention for this additional database content. Deploy
the API migration before the web version that offers attachment uploads.

Migration runners on the same database server hold a lock scoped to the
database while applying pending files. A replica that cannot acquire the lock
within 30 seconds fails startup; retry it once the migration runner has
finished. MySQL and MariaDB can commit DDL implicitly. If a file fails
partway through, inspect the applied statements and version record before
retrying. Rehearse recovery on an isolated copy before modifying the target
schema.

Production startup fails fast when secure cookies are disabled, the example JWT
secret is configured, an allowed origin is not an HTTPS non-local origin, the
refresh-token lifetime is not longer than the access-token lifetime, or the
idle database pool exceeds the open-connection limit.

## Container images

Both repositories include multi-stage Dockerfiles and run as non-root users.
Build from each repository root:

```bash
docker build -t citra-negara-lms-api:VERSION .
docker build \
  --build-arg NEXT_PUBLIC_API_BASE_URL=https://api.example.sch.id/api/v1 \
  --build-arg NEXT_PUBLIC_APP_NAME="Citra Negara LMS" \
  -t citra-negara-lms-web:VERSION .
```

`NEXT_PUBLIC_API_BASE_URL` is embedded in the browser bundle and is required
when building the web image. It is not a secret. Never place JWT, database, or
private infrastructure credentials in a `NEXT_PUBLIC_*` value.

Supply API secrets and environment-specific configuration at runtime through
the deployment platform. Production requires `DATABASE_URL`, a unique
`JWT_SECRET`, `ALLOWED_ORIGINS`, and `COOKIE_SECURE=true`. When the API is
behind a reverse proxy, set `TRUSTED_PROXIES` to that proxy's exact IP addresses
or CIDRs; an empty value ignores forwarded client-IP headers. Keep
`SEED_DEMO_DATA=false`. The API image health check covers process liveness; the
orchestrator readiness probe should use `/api/v1/health` so database failures
stop new traffic.

## Deployment and rollback sequence

1. Record image versions and create a verified database backup.
2. Deploy the API image. Startup applies pending forward migrations before
   accepting traffic; stop the rollout if migration or readiness fails.
3. Verify liveness, readiness, structured logs, and one authenticated smoke
   request without logging credentials or response data.
4. Deploy the web image built for the intended public API origin.
5. Complete the role-based smoke path before opening an examination window.

For an application rollback, redeploy the previously recorded image only after
verifying compatibility with the current schema. Database down migrations are
never run automatically. If schema rollback is required, use a maintenance
window and follow the reviewed migration-specific recovery plan. Restoring a
backup is a last-resort data operation and must target a separately verified
database before traffic resumes.

## Local development data

The API can seed a connected, fictional dataset when `SEED_DEMO_DATA` is
enabled in a development environment. Configuration rejects demo seeding in
production. Keep seed account identifiers and passwords in the approved local
setup flow; do not copy them into documentation, source control, screenshots,
or shared logs. The seeder is not a production data migration and must not be
used to create real student records.

For first-account bootstrap, use `go run ./cmd/bootstrap-admin` with the
required `BOOTSTRAP_ADMIN_*` values supplied through a secure local environment
or secret manager. Do not pass passwords through shell history or write them
into a tracked file. See the [API README](../README.md) for the required
variable names and behavior.

## Pre-exam checklist

- Confirm the server, database, and school devices use the expected timezone;
  application timestamps are stored and compared in UTC.
- Review the final exam schedule, duration, participants, questions, points,
  navigation, and randomization policy before publication.
- Run the role-based smoke path: sign in, configure academics, author and
  publish an exam, start, answer, refresh, resume, submit, review the receipt,
  grade, publish the result, and verify student access.
- Check the course material and assignment flows used by the examination cohort.
- Run the representative k6 scenario with non-production accounts and a
  database volume close to the expected examination.
- For a 2,000-account single-site test, set `LOGIN_RATE_LIMIT` above the planned
  one-minute login burst or distribute load generators across representative
  source networks. Keep `LOGIN_ACCOUNT_RATE_LIMIT` enabled; never reuse one
  student identifier across virtual users.
- Create a fresh encrypted backup and verify its checksum.
- Perform a restore rehearsal in an isolated database before the examination
  window.
- Confirm operators can inspect structured API logs by `request_id` and reach
  the administrative audit view.

## Backup

Use `scripts/backup-mysql.ps1` from a secured operator shell. Supply connection
values through `MYSQL_HOST`, `MYSQL_PORT`, `MYSQL_USER`, `MYSQL_PASSWORD`, and
`MYSQL_DATABASE`. Set `BACKUP_DIRECTORY` to an encrypted location outside the
repository. The script does not print the database password and writes a
SHA-256 checksum next to the SQL file.

Recommended policy before production use:

- daily backup during normal operation;
- an additional backup immediately before and after an important exam;
- encrypted off-host retention;
- retention duration approved by the school;
- a documented person responsible for backup verification.

## Restore

Restoration overwrites the target database state. Use
`scripts/restore-mysql.ps1 -BackupFile <absolute-path> -Force` only against an
explicitly verified target. The script validates the checksum when a sibling
`.sha256` file exists. Restore rehearsals must use an isolated database.

## Incident response

1. Record incident time, exam ID, affected users, request IDs, and visible
   symptoms without copying answers or credentials into chat or log notes.
2. Check readiness, database connections, saturation, and error-rate logs.
3. Do not modify attempts or grades directly in SQL during an active exam.
4. Pause operational decisions, preserve logs, and use audited application
   actions when remediation is available.
5. After recovery, reconcile attempt status, receipts, results, and audit events
   before communicating final outcomes.

## Capacity evidence

The repository includes a configurable k6 exam-flow scenario. A production
capacity claim is valid only after recording environment, data volume,
concurrent users, latency percentiles, error rate, database pool metrics, and
test result. Source code readiness alone is not evidence of 2,000 concurrent
users.
