# cairo — Web Client Guide

Frontend for the cairo object-storage API (Go/Gin backend at the repo root — the root `AGENTS.md` is inherited and fully applies, including the **never install dependencies** rule).

**The dependencies are already fixed**. Do not install new dependency /upgrade any dependencies.  

Vite 8 + React 19 + TypeScript 6 SPA, Tailwind CSS 4, oxlint. Fresh Vite template scaffold — `src/App.tsx` is still the template counter demo, not the real client yet.

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

## Backend integration

- API: `http://localhost:8080/api/v1` — JWT-protected account routes + API-key-protected object-storage routes. Full surface in the root `AGENTS.md`; OpenAPI spec in `docs/api/`.
- `vite.config.ts` has **no dev proxy** yet — the API is not wired up. If you add one, remember the backend also runs a separate health server on `:8081` (health server should not to integrate to any client).
