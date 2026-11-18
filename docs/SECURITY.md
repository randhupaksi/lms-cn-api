# Security Baseline

See the [API documentation index](INDEX.md) for architecture, governance,
operations, and contract documentation.

## Implemented controls

- password hashing with bcrypt;
- 15-minute JWT access tokens and rotated, hashed refresh tokens with a
  24-hour absolute session expiry from login; refresh rotation does not extend
  the server-side expiry;
- server-side session revocation and disabled-account checks;
- role and course ownership checks on sensitive endpoints;
- DTO whitelisting and generic production-safe error responses;
- question snapshots and student payloads without answer keys;
- idempotent attempt start, answer save, and final submission;
- rate limiting for public authentication endpoints;
- structured request logging and handler panic recovery using route templates
  and request IDs, without dumping headers, cookies, request bodies, or panic
  values;
- read-only administrative audit access;
- explicit CORS and environment-driven secrets;
- forwarded client IP headers are trusted only from explicitly configured
  `TRUSTED_PROXIES` addresses; an empty value disables proxy-header trust;
- JSON request bodies are limited to 4 MiB and oversized requests receive 413;
- course material attachments accept verified JPEG, PNG, PDF, and DOCX files;
  each file is limited to 8 MiB, with at most five files and 20 MiB per
  material. Images are dimension-checked; documents are served as downloads.
  Attachment reads require course enrollment for students, assigned-course
  access for teaching staff, or an explicitly authorized school-admin role;
- attachment bytes are stored in MySQL, never in a public web directory. The
  API returns attachment metadata separately from bytes and does not log file
  content or download URLs;
- password creation and change reject values beyond bcrypt's 72-byte limit.

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
- Treat exported results as confidential files. Result exports ask clients and
  intermediaries to avoid storing the HTTP response. CSV exporters keep user
  text beginning with spreadsheet formula markers in text form.
- Re-run authorization and exam-integrity tests after route or schema changes.
- Review any attempt reset or score-change feature before implementation; both
  require explicit audit events and product policy.

