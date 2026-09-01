# Citra Negara LMS API

Backend REST API untuk Citra Negara LMS.

Baseline stack mengikuti Absensi CN: Go, Gin, GORM, MySQL, JWT, bcrypt,
golang-migrate, validator, Excelize, dan Cloudinary opsional.

## Architecture

The API is a modular monolith. HTTP routing and middleware stay in `internal`,
business domains will live under `internal/modules`, infrastructure stays in
dedicated packages, and `pkg` contains domain-agnostic reusable helpers.

The current implementation intentionally exposes only the platform health
endpoint. Business modules are added incrementally after the foundation is
reviewed.
