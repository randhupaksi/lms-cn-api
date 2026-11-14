# API Domain Modules

Business domains belong here. Current modules cover authentication, users,
academics, questions, exams, attempts, grading, results, audit, monitoring,
materials, assignments, analytics, and school governance.

Each module owns its domain models, DTOs, validation, service logic, handlers,
and repository/query details as needed. Handlers translate HTTP input and
output; services enforce use cases, authorization policy, and transaction
boundaries; repositories own persistence queries.

Modules must not reach into another module's private implementation. Use
explicit narrow service interfaces for cross-domain workflows, and put only
genuinely domain-agnostic helpers in `pkg` or infrastructure packages. The
`curriculum` role is implemented through existing academic/governance
capabilities, not a standalone module.

See [API architecture](../../docs/ARCHITECTURE.md) and the
[school governance contract](../../docs/SCHOOL_GOVERNANCE.md) for current
boundaries and role policy.
