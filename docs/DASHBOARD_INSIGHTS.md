# Dashboard insights contract

`GET /api/v1/analytics/dashboard/insights` uses the existing authenticated,
password-change-complete analytics route group. It accepts no client role,
user, course, or date overrides. The existing `/analytics/dashboard` response
is unchanged apart from clearer display labels.

The standard success envelope contains:

- `generated_at`: UTC response timestamp.
- `timezone`: `Asia/Jakarta` (WIB, UTC+7).
- `starts_on`, `ends_on`: school-calendar dates covering today and the previous
  13 days, inclusive.
- `metrics`: three additional role-specific counts, each with `key`, `label`,
  and `value`.
- `activity`: exactly 14 chronological rows with `date`, `exam_submissions`,
  and `assignment_submissions`. Missing days are zero. Events after the
  response timestamp are excluded. Exams count finalized submitted/expired
  attempts with a submission timestamp; assignments count submission records,
  regardless of their later grading status.
- `distribution_title`, `distribution`: exhaustive labeled category counts.
  Staff receive all course-scoped exams grouped into draft, scheduled,
  ongoing, and ended. Students receive currently published, enrolled-course
  assignments grouped into pending, overdue, submitted, graded, and returned.
- `learning_progress` (student only): completed/total currently published
  materials in enrolled courses. Archived materials do not inflate completion.
- `recent_results`: students receive at most five of their own published
  results, newest publication first, with ID, exam title, course name,
  percentage, and publication timestamp. Results with a nonpositive maximum
  score are omitted from this chart. Staff receive an empty array.

Admin counts cover the school. Teacher aggregates use current course-manager
membership through `EXISTS`, preventing duplicate counts for co-managed courses.
Student historical activity is restricted to their own records; material and
active-assignment aggregates additionally require current enrollment. No
student identities, answer content, or answer keys are returned.

All counts aggregate the complete matching set, not a paginated sample.
Database errors use the existing safe `ANALYTICS_READ_FAILED` response; the
frontend keeps the original metrics/actions available when insights fail.
No migrations, schema changes, or new dependencies are required.

Unit tests cover the WIB year boundary and zero-filled dates, membership SQL
scope, unknown roles, and rejection of role spoofing via query parameters.

`TestInsightsLocalMySQLReadOnly` additionally checks all role query branches
against the existing local schema and verifies empty results for identities
without membership. It creates no fixtures and uses a MySQL read-only
transaction that is rolled back. It requires the local development/test
configuration and a loopback database host. Opt in by setting
`LMS_ANALYTICS_READONLY_CHECK=1`, then run
`go test ./internal/modules/analytics -run TestInsightsLocalMySQLReadOnly -v`.
Credentials and query results are never printed.
