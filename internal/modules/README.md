# Domain modules

Business domains belong here. Each module owns its models, DTOs, validation,
service logic, and handlers. Modules must not reach into another module's
private implementation; shared concerns belong in `pkg` or dedicated internal
infrastructure packages.
