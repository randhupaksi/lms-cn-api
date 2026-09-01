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
- Current API is foundation-only with a health endpoint.
- Student and examination data are confidential.
- Validate ownership and authorization in the API.
- Before handoff: vet, test, build. Never commit or push.
