# Runtime Operations

- After changes that affect runtime behavior, Codex owns local environment preparation before handoff.
- Start or restart the Go API when code changes require it; do not leave routine restarts to the user.
- Run `go run ./cmd/migrate up` for new migrations and verify the resulting version before asking the user to test.
- Start required local dependencies (MySQL/Redis) and verify health/readiness.
- The user performs final product testing and acceptance; Codex should hand off a ready-to-test environment.
- `server/.env` contains values with shell metacharacters and must not be loaded with plain `source`; load key/value lines safely without printing secrets.