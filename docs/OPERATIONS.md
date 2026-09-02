# Citra Negara LMS Operations Runbook

This runbook covers the minimum production controls required before an
important school examination. It complements the PRD release gate and does not
replace infrastructure-specific procedures.

## Deployment prerequisites

1. Use a dedicated MySQL database and restricted application account.
2. The API applies pending versioned migrations in numeric order and records
   each applied version in `schema_migrations`. Review the migration result in
   the deployment log before continuing.
3. Configure production secrets outside the repository.
4. Set `COOKIE_SECURE=true`, an explicit `ALLOWED_ORIGINS`, and a strong
   `JWT_SECRET` of at least 32 random characters.
5. Run the API and web validation commands documented in each repository.
6. Verify `/api/v1/health/live` and `/api/v1/health` after deployment.

## Local development seed

`go run ./cmd/api` automatically applies pending versioned migrations, then
seeds the connected development database when `SEED_DEMO_DATA=true` (the default for
non-production environments). The seed is idempotent and includes connected
academic, examination, material, assignment, result, monitoring, and audit
data. Demo credentials are `admin`, `teacher`, and `student`, all using the
development-only password `12121212`.

Set `SEED_DEMO_DATA=false` when a development database should remain empty. The
configuration rejects demo seeding when `APP_ENV=production`.

## Pre-exam checklist

- Confirm the server, database, and school devices use the expected timezone;
  application timestamps are stored and compared in UTC.
- Review the final exam schedule, duration, participants, questions, points,
  navigation, and randomization policy before publication.
- Run the smoke path: login, start, answer, refresh, resume, submit, receipt,
  publish result, and student result access.
- Run the representative k6 scenario with non-production accounts and a
  database volume close to the expected examination.
- For a 2,000-account single-site test, set `LOGIN_RATE_LIMIT` above the planned
  one-minute login burst or distribute load generators across representative
  source networks. Keep `LOGIN_ACCOUNT_RATE_LIMIT` enabled; never reuse one
  student identifier across virtual users.
- Create a fresh encrypted backup and verify its checksum.
- Perform a restore rehearsal in an isolated database before the examination
  window.
- Confirm operators can inspect structured API logs by `request_id` and access
  the admin audit page.

## Backup

Use `scripts/backup-mysql.ps1` from a secured operator shell. Supply connection
values through `MYSQL_HOST`, `MYSQL_PORT`, `MYSQL_USER`, `MYSQL_PASSWORD`, and
`MYSQL_DATABASE`. Set `BACKUP_DIRECTORY` to an encrypted location outside the
repository. The script never prints the database password and produces a
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

1. Record the incident time, exam ID, affected users, request IDs, and visible
   symptoms without copying answers or credentials into chat/log notes.
2. Check readiness, database connections, saturation, and error-rate logs.
3. Do not modify attempts or grades directly in SQL during an active exam.
4. Pause operational decisions, preserve logs, and use audited application
   actions when remediation is available.
5. After recovery, reconcile attempt status, receipts, results, and audit
   events before communicating final outcomes.

## Capacity evidence

The repository includes a configurable k6 exam-flow scenario. A production
capacity claim is valid only after recording the environment, data volume,
concurrent users, latency percentiles, error rate, database pool metrics, and
test result. Source code readiness alone is not evidence of 2,000 concurrent
users.
