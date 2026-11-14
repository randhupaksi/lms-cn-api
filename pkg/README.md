# Shared API Packages

Packages in this directory are reusable across API modules and remain
domain-agnostic. Current packages provide application errors, pagination,
request validation helpers, and response envelopes.

Do not place exam policy, school-role rules, SQL queries, or module-specific
DTOs here. Prefer an explicit dependency in the owning module over a broad
utility abstraction. Shared response and error changes can affect every
endpoint, so check consumer compatibility and update the API contract docs.
