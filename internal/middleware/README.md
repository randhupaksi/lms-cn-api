# HTTP Middleware

This package implements cross-cutting HTTP behavior around the route and
handler layers. Middleware must remain transport-focused and must not contain
domain workflows or repository queries.

Current concerns include request IDs, authentication/authorization context,
CORS, security headers, public-auth rate limits, panic recovery, and structured
request logging. Keep response envelopes and safe error mapping consistent
with the shared API contract.

Authentication middleware establishes trusted identity from server-validated
session state. Handlers/services still enforce the domain action and ownership
rules. Do not trust client-supplied roles, log request bodies or secrets, or
add client-IP forwarding trust without an explicit `TRUSTED_PROXIES`
configuration.

See the [security baseline](../../docs/SECURITY.md) and
[API architecture](../../docs/ARCHITECTURE.md) before changing these
cross-cutting controls.
