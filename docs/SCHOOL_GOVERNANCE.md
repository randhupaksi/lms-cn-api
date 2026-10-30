# School governance API

Scope: one school, six fixed roles (`superadmin`, `principal`, `curriculum`, `admin`, `teacher`, `student`).
Principal shares school governance, academic administration, monitoring, reports,
and read-only audit access. Principal can manage admin, teacher, and student
accounts. Only superadmin can create or manage principal/superadmin accounts,
change roles, inspect sessions, and revoke sessions. Teacher ownership of
authoring, grading, and result publication remains unchanged.
Curriculum is a teaching-staff role with school-wide academic coordination. It
manages academic structure and can review readiness, quality data, academic
agenda, and reports. It can author, grade, and publish only in courses where it
is explicitly assigned. It cannot manage accounts, audit logs, school settings,
or approve schedule requests.
Superadmin and principal share school governance. Assigned teachers continue to own
content, grading, and result publication. Superadmin does not bypass those rules.

## Endpoints (prefix `/api/v1`)

| Method / path | Actor | Behavior |
|---|---|---|
| GET `/school-profile` | Public | Whitelisted school name, motto, support email/message only |
| GET `/governance/overview` | Superadmin, principal, curriculum | Whole-school counters, role distribution, global exam lifecycle, six upcoming/running exams, quality checks, 30-day agenda |
| GET `/governance/exams` | Superadmin, principal, curriculum | Paginated readiness, search by title/course |
| GET `/governance/request-exams` | Staff | Same safe exam metadata, assigned-course scope for teachers |
| GET `/governance/quality` | Superadmin, principal, curriculum | Six complete-data counts and at most eight samples per category |
| GET/POST `/governance/requests` | Staff read; teacher/curriculum/admin create | Teachers create for assigned courses; curriculum/admin can coordinate school-wide; whole-school inbox for superadmin and principal; optional status filter and pagination |
| POST `/governance/requests/:id/decision` | Superadmin, principal | `decision: approved/rejected`, `note` (10–1000 characters) |
| POST `/governance/requests/:id/cancel` | Request owner | Cancel pending request |
| GET `/governance/events` | Superadmin, principal, curriculum | School events + published exam agenda; `from`/`to` date-only WIB, end exclusive, maximum 366 days; each source capped at 500 |
| POST `/governance/events` | Superadmin, principal, curriculum | Create school academic event |
| PUT `/governance/events/:id` | Superadmin, principal, curriculum | Update active event, requires current `version` |
| POST `/governance/events/:id/archive` | Superadmin, principal, curriculum | Soft archive, requires current `version` |
| GET/PUT `/governance/settings` | Superadmin, principal | Read/save school profile and reporting reference; save requires version + reason |
| GET `/governance/settings/history` | Superadmin, principal | Last 50 version snapshots, actor name, reason and date |
| GET `/governance/reports` | Superadmin, principal, curriculum | Paginated per-course aggregates; optional `year_id`, `class_id` |
| GET `/governance/reports/export` | Superadmin, principal, curriculum | Same filters, CSV maximum 10,000 courses, spreadsheet formula protection |
| GET/POST `/governance/actions` | Staff read; leaders create | Leaders see all; staff see assigned actions |
| GET `/governance/actions/summary` | Staff | Open and overdue counts scoped to actor |
| GET `/governance/actions/assignees` | Superadmin, principal | Active teachers, curriculum staff, and administrators |
| GET `/governance/actions/:id/updates` | Leader or assignee | Action history |
| PATCH `/governance/actions/:id/status` | Leader or assignee | Versioned status change with note |
| PATCH `/users/:id/role` | Superadmin | New `role`, `reason`, `current_password`; revokes target sessions |
| GET `/users/:id/sessions` | Superadmin | At most 100 active session IDs and timestamps, no hashes/tokens |
| POST `/users/:id/revoke-sessions` | Superadmin | Revoke target sessions and record audit atomically |

Requests use the existing response envelope and pagination metadata. Dates in
JSON are RFC3339; user-facing dates are WIB. Event categories are `academic`,
`holiday`, `meeting`, `announcement`; exam agenda entries are derived/read-only.

## Invariants

- Leadership actions are created by principal or superadmin and may be assigned
  to an active teacher or administrator. Assigned staff can read their own
  actions and record progress; leaders can see all actions and reopen or cancel
  them. Each status change requires a note and current version, and is audited.
- Migration `000007_principal_leadership_actions` adds the role and action
  history. Before rollback, export action history and remove or change every
  principal role; rollback drops the action tables.

- Normal admin can list/mutate teacher/student accounts only. It cannot create
  administrators or manage privileged targets by guessing IDs.
- Superadmin can create admin/teacher/student accounts. Granting superadmin
  requires the dedicated role-change action and current-password confirmation.
- Self-demotion/deactivation is blocked. Privileged mutations lock existing
  superadmin records; at least one active superadmin must remain.
- Teacher/student role changes are blocked while academic relations/history
  exist. Preserve identity attribution with a separate account when necessary.
- Password reset, deactivation, role change and session revocation invalidate
  server sessions. Authentication also compares token role/password-change flag
  with current database state on every authenticated request.
- Session `last_used_at` means last refresh-session rotation, not last request.
- Schedule requests contain exam ID, reason, proposed start/end and a server-owned
  snapshot of the original schedule. Only one pending request per exam.
- Approval locks request + exam, rejects self-approval/stale original schedule,
  rechecks future window/duration and rejects closed/archived exams or any exam
  with attempts. Exam start uses a compatible shared lock. A repeated decision
  conflicts. Rejection/cancellation does not alter the exam.
- School settings use optimistic version checks and transactional history/audit.
  The passing reference affects report interpretation, not grades/publication.
- Role/status/credential/schedule/settings/event changes are audited in the same
  transaction as the write. Audit logs have no mutation endpoint.

## Schema and local demo

Migration `000006_school_governance` adds the new role and governance tables.
The runner adapts the role-check removal syntax for MariaDB; other SQL retains
the MySQL syntax. Never edit an already-applied migration to retrofit a change.
The down migration first refuses rollback while any `superadmin` rows remain;
it does not silently demote users. Remaining down steps remove governance data,
so export required records and verify the target before a manual rollback.

The development seed includes `kepala.sekolah` (principal), `kurikulum` (curriculum), `arif.wicaksana` (superadmin), `dewi.kusuma`
(additional admin), `nanda.prasetyo` (unassigned teacher) and `alya.ramadhani`
(unenrolled student), plus school events, schedule requests and initial settings
history. Existing demo credential is documented in OPERATIONS. Account role,
status, identity edits and changed credentials persist across restarts. Untouched
version-1 seeded agendas refresh relative dates; edited/archived agendas remain.
Resolved requests and approved schedules persist.

For a fresh non-demo environment, first apply migrations, then run the existing
bootstrap command with `BOOTSTRAP_ADMIN_ROLE=superadmin` plus its identifier,
name and password environment variables. It refuses a second bootstrap
superadmin; use access governance to grant additional superadmins. The new
account must change its password. Do not put credentials in shell history/docs.

## Verification

Run `go test ./...`, `go vet ./...`, and `go build ./cmd/api`.
Optional transaction integration test: set `GOVERNANCE_MYSQL_TEST=1` and
`GOVERNANCE_TEST_DATABASE_URL` to a migrated, seeded, disposable localhost database
named `*_test` or `lms_governance_demo`, then run
`go test ./internal/modules/governance -count=1`.
Integration fixture mutations roll back; tests never apply migrations themselves.
Seeder restart preservation can be checked separately with
`GOVERNANCE_SEED_TEST=1` and the same disposable database URL, using
`go test ./internal/database -run TestDemoRestartPreservesGovernance -count=1`.

MFA/recovery, bulk imports and external message delivery remain separate future
features. There are no placeholder controls claiming those capabilities.
