# cairo — Repo Guide

Two independent packages, each with its own detailed guide. Read the relevant one before working there:

- `service/` — Go (1.26.5) + Gin object-storage API with JWT/API-key auth, SQLite, Redis. Commands, architecture, gotchas: **`service/AGENTS.md`**.
- `apps/web/` — Vite 8 + React 19 SPA consuming that API. Commands, TS/Tailwind quirks: **`apps/web/AGENTS.md`**.

> ⚠️ **NEVER download or install any library, package, Go module, or plugin** — the user manages dependencies manually. This applies in both packages and is unconditional.

## Cross-cutting facts

- Backend API: `http://localhost:8080/api/v1` (CORS enabled, no dev proxy needed). Separate health server on `:8081` — never integrate it into the client.
- `go test ./...` in `service/` also runs e2e tests that require a running Docker daemon; `go test ./internal/...` does not. No build tag to skip them.
- Backend config comes from env (`.env`, gitignored); web client overrides the API base via `VITE_API_BASE` (default `http://localhost:8080`).
- `service/docs/api/` holds the OpenAPI spec;
