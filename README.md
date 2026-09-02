# Citra Negara LMS API

Backend REST API untuk Citra Negara LMS.

Stack utama: Go, Gin, GORM, MySQL, JWT, bcrypt, dan validator.

## Architecture

The API is a modular monolith. HTTP routing and middleware stay in `internal`,
business domains live under `internal/modules`, infrastructure stays in
dedicated packages, and `pkg` contains domain-agnostic reusable helpers.

Roadmap phases 0–4 cover authentication, users, academics, question authoring,
exam authoring, secure attempts, grading, and result publication. See
`docs/ARCHITECTURE.md` and the workspace `docs/IMPLEMENTATION_STATUS.md`.
