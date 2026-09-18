# Citra Negara LMS API

The complete backend guidance is in [AGENTS.md](./AGENTS.md).

@AGENTS.md

Important references:

- Workspace rules: [../AGENTS.md](../AGENTS.md)
- Architecture: [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md)
- Workflow: [../docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md)

Summary:

- Go 1.25.5 + Gin + GORM/MySQL.
- Pragmatic modular monolith: handler → service → repository.
- Implemented modules cover auth, users, academics, examinations, grading,
  results, monitoring/audit, materials, assignments, and analytics.
- Student and examination data are confidential.
- Validate ownership and authorization in the API.
- Before handoff: vet, test, build. Never commit or push.
