# School Governance API

See the [API documentation index](INDEX.md) for security and operational
guidance.

Scope: one school and six fixed roles (`superadmin`, `principal`,
`curriculum`, `admin`, `teacher`, `student`). Principal shares the documented
school governance, academic administration, monitoring, reporting, and
read-only audit access. Principal can manage admin, teacher, and student
accounts. Only superadmin can create/manage principal or superadmin accounts,
change roles, inspect sessions, and revoke sessions. Teacher ownership of
authoring, grading, and result publication remains unchanged.

Curriculum is a teaching-staff role with school-wide academic coordination. It
manages academic structure and can review readiness, quality data, academic
agenda, and reports. It can author, grade, and publish only in courses where
it is explicitly assigned. It cannot manage accounts, audit logs, school
settings, or approve schedule requests. Superadmin and principal share school
governance; neither bypasses assigned-teacher ownership of learning content,
grading, or result publication.

## Endpoints (prefix `/api/v1`)

| Method / path | Actor | Behavior |
|---|---|---|
| GET `/school-profile` | Public | Whitelisted school name, motto, support email/message only |
| GET `/governance/overview` | Superadmin, principal, curriculum | Whole-school counters, role distribution, global exam lifecycle, six upcoming/running exams, quality checks, and 30-day agenda |
| GET `/governance/exams` | Superadmin, principal, curriculum | Paginated readiness, searchable by title/course |
| GET `/governance/request-exams` | Staff | Safe exam metadata; teachers are scoped to assigned courses |
| GET `/governance/quality` | Superadmin, principal, curriculum | Six complete-data counts and at most eight samples per category |
| GET/POST `/governance/requests` | Staff read; teacher/curriculum/admin create | Teachers create for assigned courses; curriculum/admin can coordinate school-wide; leaders see the whole-school inbox; optional status filter and pagination |
| POST `/governance/requests/:id/decision` | Superadmin, principal | `decision: approved/rejected`, `note` (10–1000 characters) |
| POST `/governance/requests/:id/cancel` | Request owner | Cancel a pending request |
| GET `/governance/events` | Superadmin, principal, curriculum | School events plus published exam agenda; date-only WIB range with exclusive end, maximum 366 days, each source capped at 500 |
| POST `/governance/events` | Superadmin, principal, curriculum | Create a school academic event |
| PUT `/governance/events/:id` | Superadmin, principal, curriculum | Update an active event with its current `version` |
| POST `/governance/events/:id/archive` | Superadmin, principal, curriculum | Soft-archive an event with its current `version` |
| GET/PUT `/governance/settings` | Superadmin, principal | Read/save school profile and reporting reference; save requires version and reason |
| GET `/governance/settings/history` | Superadmin, principal | Last 50 version snapshots with actor name, reason, and date |
| GET `/governance/reports` | Superadmin, principal, curriculum | Paginated per-course aggregates; optional `year_id` and `class_id` |
| GET `/governance/reports/export` | Superadmin, principal, curriculum | Same filters, CSV capped at 10,000 courses, spreadsheet-formula protection |
| GET/POST `/governance/actions` | Staff read; leaders create | Leaders see all actions; staff see assigned actions |
| GET `/governance/actions/summary` | Staff | Actor-scoped open and overdue counts |
| GET `/governance/actions/assignees` | Superadmin, principal | Active teachers, curriculum staff, and administrators |
| GET `/governance/actions/:id/updates` | Leader or assignee | Action history |
| PATCH `/governance/actions/:id/status` | Leader or assignee | Versioned status change with a note |
| PATCH `/users/:id/role` | Superadmin | New `role`, `reason`, and `current_password`; revokes target sessions |
| GET `/users/:id/sessions` | Superadmin | At most 100 active session IDs and timestamps; no hashes/tokens |
| POST `/users/:id/revoke-sessions` | Superadmin | Revoke target sessions and record audit atomically |

Requests use the shared response envelope and pagination metadata. JSON dates
are RFC3339; user-facing dates are WIB. Event categories are `academic`,
`holiday`, `meeting`, and `announcement`; exam agenda entries are derived and
read-only.

## Invariants

- Leadership actions are created by principal or superadmin and may be
  assigned to an active teacher or administrator. Assigned staff can read
  their own actions and record progress; leaders can see all actions and
  reopen or cancel them. Status changes require a note and current version,
  and are audited.
- Normal admin can list or mutate teacher/student accounts only. It cannot
  create administrators or manage privileged targets by guessing IDs.
- Superadmin can create admin/teacher/student accounts. Granting superadmin
  requires the dedicated role-change action and current-password confirmation.
- Self-demotion/deactivation is blocked. Privileged mutations lock existing
  superadmin records; at least one active superadmin must remain.
- Teacher/student role changes are blocked while academic relations/history
  exist. Preserve identity attribution with a separate account when necessary.
- Password reset, deactivation, role change, and session revocation invalidate
  server sessions. Authentication compares token role/password-change state
  with current database state on authenticated requests.
- Session `last_used_at` means last refresh-session rotation, not last request.
- Schedule requests contain exam ID, reason, proposed start/end, and a
  server-owned snapshot of the original schedule. Only one request per exam
  may be pending.
- Approval locks the request and exam, rejects self-approval/stale schedules,
  rechecks the future window and duration, and rejects closed/archived exams
  or exams with attempts. Exam start uses a compatible shared lock. Repeated
  decisions conflict; rejection/cancellation does not alter the exam.
- School settings use optimistic version checks and transactional history/audit.
  The passing reference affects report interpretation, not grades/publication.
- Role, status, credential, schedule, settings, and event changes are audited
  in the same transaction as the write. Audit logs have no mutation endpoint.

## Schema and local development data

The governance schema is introduced and extended by these migrations:

1. `000006_school_governance` — school governance data and access foundations.
2. `000007_principal_leadership_actions` — principal leadership actions and
   action history.
3. `000008_curriculum_role` — curriculum role support and related constraints.
4. `000009_material_attachments` — protected file metadata and content for
   course material attachments.

The complete current migration range is `000001` through `000009`. Review the
specific down migration and preserve a verified backup before any rollback;
rollback may remove governance records, role assignments, or material files.

The API seeder creates fictional development data and runs only when
`SEED_DEMO_DATA` is enabled in a development environment. It is disabled in
production. Keep account identifiers and passwords out of documentation,
source control, and shared logs. For a controlled initial administrative
account, use the environment-based `cmd/bootstrap-admin` flow described in the
[API README](../README.md).

## Verification

Run from the API repository root:

```bash
go test ./...
go vet ./...
go build ./cmd/api
```

Optional transaction integration test: set `GOVERNANCE_MYSQL_TEST=1` and
`GOVERNANCE_TEST_DATABASE_URL` to a migrated, seeded, disposable localhost
database named `*_test` or `lms_governance_demo`, then run
`go test ./internal/modules/governance -count=1`. Integration fixture
mutations roll back; the test does not apply migrations.

Seeder restart preservation can be checked separately with
`GOVERNANCE_SEED_TEST=1` and the same disposable database URL, using
`go test ./internal/database -run TestDemoRestartPreservesGovernance -count=1`.

MFA/recovery, bulk imports, and external message delivery remain separate
future features. There are no placeholder controls claiming these capabilities.
