# Security Baseline

## Implemented controls

- password hashing with bcrypt;
- short-lived JWT access tokens and rotated, hashed refresh tokens;
- server-side session revocation and disabled-account checks;
- role and course ownership checks on sensitive endpoints;
- DTO whitelisting and generic production-safe error responses;
- question snapshots and student payloads without answer keys;
- idempotent attempt start, answer save, and final submission;
- rate limiting for public authentication endpoints;
- structured request logging without request bodies or tokens;
- read-only administrative audit access;
- explicit CORS and environment-driven secrets.

## Required infrastructure controls

- TLS termination and secure cookies in production;
- network restrictions around MySQL;
- secret rotation and restricted operator access;
- encrypted backups and tested recovery;
- centralized log retention with access control and alerting;
- operating-system and dependency patch management;
- a reverse-proxy or distributed rate limiter when multiple API replicas are
  deployed.

## Review points

- Never log passwords, tokens, question keys, student answers, or full export
  contents.
- Treat exported results as confidential files.
- Re-run authorization and exam-integrity tests after route or schema changes.
- Review any attempt reset or score-change feature before implementation; both
  require explicit audit events and product policy.

