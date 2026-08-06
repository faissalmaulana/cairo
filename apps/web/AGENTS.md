# cairo — Web Client Guide

Frontend for the cairo object-storage API (Go/Gin backend at the repo root — the root `AGENTS.md` is inherited and fully applies, including the **never install dependencies** rule).

**The dependencies are already fixed**. Do not install new dependency /upgrade any dependencies.  

Vite 8 + React 19 + TypeScript 6 SPA, Tailwind CSS 4, oxlint. Auth shell is in place: routing (`react-router` v8), in-memory auth state, and a data fetching layer (`@tanstack/react-query`) — see `src/App.tsx`, `src/auth/`, `src/api/`, `src/pages/`.

## Commands

```bash
pnpm install   # user-managed only — never install/update deps yourself
pnpm dev       # vite dev server (default http://localhost:5173)
pnpm build     # tsc -b && vite build
pnpm lint      # oxlint (NOT eslint)
```

- Linter is **oxlint** (`.oxlintrc.json`); there is no ESLint config — don't add or run eslint.
- No test framework configured — the web app has no unit tests yet.

## Stack quirks

- **Tailwind 4 is CSS-first**: no `tailwind.config.js`/PostCSS config; `@tailwindcss/vite` plugin + `@import "tailwindcss"` in `src/index.css`.
- **Strict TS flags** (`tsconfig.app.json`): `verbatimModuleSyntax` → type-only imports must use `import type`; `erasableSyntaxOnly` → no enums, namespaces, or parameter properties; `allowImportingTsExtensions` → relative imports keep the `.tsx` extension (`import App from './App.tsx'`).
- Build uses `tsc -b` over project references (`tsconfig.app.json` for `src/`, `tsconfig.node.json` for `vite.config.ts`).
- `pnpm-workspace.yaml` is **not** a monorepo workspace — it only excludes `postcss@8.5.23` and `vite@8.2.0` from pnpm's minimum-release-age check.
- Intended stack already in `package.json`: `@tanstack/react-query`, `@tanstack/react-form`, `react-router` (v8 — the package is `react-router`, not `react-router-dom`).
- **Auth flow**: tokens live in a module singleton (`src/api/tokens.ts`) persisted to localStorage under `cairo.auth`; the fetch wrapper `src/api/client.ts` attaches `Authorization: Bearer` automatically and rotates refresh tokens single-flight on 401 (POST `/refresh`). `src/auth/AuthProvider.tsx` exposes `useAuth()` (`src/auth/useAuth.ts`). Signin/signup use `@tanstack/react-form` in `src/pages/SignInPage.tsx` / `SignUpPage.tsx`.
- Error envelope: `{ success, data?, error?: { code, message } }` — the client throws `ApiError` (`status`, `code`, `message`); auth-relevant codes are `BAD_REQUEST`, `EMAIL_EXISTS` (409), `INVALID` (401).

## Backend integration

- API: `http://localhost:8080/api/v1` — JWT-protected account routes + API-key-protected object-storage routes. Full surface in the root `AGENTS.md`; OpenAPI spec in `docs/api/`.
- **No dev proxy and none needed** — the backend has CORS enabled; `src/api/client.ts` calls `API_BASE` directly (override via `VITE_API_BASE`). The separate health server on `:8081` must never be integrated into the client.
