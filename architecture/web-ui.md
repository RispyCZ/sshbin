# Web UI

React single-page app for configuring and managing file shares, backed by a
JSON API. The Go server owns auth, storage, and access control; the browser owns
rendering and routing.

## Tech stack

- **React 19 + TypeScript** — SPA in `web/src`, entry `src/main.tsx`
- **MUI (Material UI) v9** with Emotion — component library and theming
- **react-router v7** — client-side routing
- **Vite+** (`vp`) — dev server (HMR) and production build
- **Go** — JSON API, session/auth, file streaming; serves the SPA shell

The built assets (`dist/`) are embedded into the binary via `//go:embed all:dist`
(`spa.go`), so a single `sshbin` binary ships the whole UI.

## Serving model

- **`spa.go`** renders the SPA shell (`templates/spa.html`). In production it
  reads the Vite manifest and injects hashed `dist/` asset URLs; in dev
  (`Config.Dev`) it points at the Vite dev server (`ViteOrigin`) for HMR.
- Any non-API path falls through to the shell, so client-side deep links
  (e.g. `/shares`, `/s/{id}`) resolve.
- **`error.html`** is the only server-rendered page — a standalone document for
  binary download errors (share missing / expired / forbidden), where no SPA is
  loaded.

## Client routes

| Path | Auth | Component | Description |
|------|------|-----------|-------------|
| `/` | — | `Landing` | Landing page with upload instructions |
| `/login` | — | `Login` | Email + OTP login |
| `/s/:id` | varies | `Setup` | Public share view / owner configuration |
| `/shares` | session | `Shares` | My Shares dashboard |
| `/profile` | session | `Profile` | Profile settings and data deletion |
| `*` | — | — | Redirect to `/` |

## JSON API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/session` | — | Current session (`401` when signed out) |
| `POST` | `/api/login` | — | Send OTP code |
| `POST` | `/api/verify` | — | Verify OTP, set session cookie |
| `POST` | `/api/logout` | session | Clear session |
| `GET` | `/api/shares` | session | List the caller's shares |
| `DELETE` | `/api/shares/{id}` | session + owner | Delete a share |
| `POST` | `/api/setup/{id}` | session + owner | Save share settings |
| `GET` | `/api/profile` | session | Read profile |
| `PUT` | `/api/profile` | session | Save profile |
| `DELETE` | `/api/profile` | session | Delete account and all data |
| `GET` | `/api/s/{id}` | varies | Public share view state |
| `POST` | `/api/s/{id}` | varies | Submit share password |
| `GET` | `/shares/{id}/qr` | session | QR code PNG for a share |
| `GET` | `/s/{id}/download` | varies | Stream file download |
| `GET` | `/static/` | — | Embedded favicons |

The typed client lives in `src/api/client.ts`; error responses carry
`{ error, code }` and surface as `ApiError`.

## Key components

| Component | Purpose |
|-----------|---------|
| `App` | Router, auth bootstrap, header/footer shell |
| `ColorModeProvider` | Light/dark theme state, persisted in `localStorage` |
| `ThemeToggle` | Light/dark switch |
| `NotifyProvider` / `useNotify` | Global snackbar notifications |
| `useConfirm` | Promise-based confirmation dialog |
| `SetupDialog` | Configure visibility, password, emails, expiry |
| `ShareDialog` | Share link with QR code and copy-to-clipboard |
| `UserMenu` | Avatar dropdown: My Shares, profile, sign out |
| `Logo` | Theme-aware wordmark |

## Access control

- **Public shares** — no session required; optional password checked via a
  stateless HMAC cookie (`fd_pw_{id}`)
- **Private shares** — session required; the viewer's email must be in the
  owner's allowlist
- **Setup / delete** — session required and email must match `owner_email`
  (first visitor claims an unconfigured share)

## Development

```
vp install    # install deps
vp dev        # Vite dev server (HMR); run sshbin with --dev to proxy to it
vp build      # produce embedded dist/
vp check      # format, lint, type check
```

## Testing

Two layers, run from `web`:

```
vp run test       # unit + component (Vitest + jsdom)
vp run test:watch # ...in watch mode
vp run e2e        # end-to-end (Playwright + Chromium)
vp run e2e:ui      # ...with the Playwright UI
```

**Unit + component** (`src/**/*.test.{ts,tsx}`): Vitest under a jsdom
environment (`vite.config.ts` `test` block, setup in `src/test/setup.ts`).
`src/api/client.test.ts` mocks `fetch`; `src/components/OtpInput.test.tsx`
drives the component with `@testing-library/react` + `user-event`.

`vp test` is currently broken — the published `vite-plus-test@0.1.24` ships no
`vitest` bin for `vite-plus-core@0.2.1` to launch. The `test` scripts invoke the
bundled runner directly via `node node_modules/vitest/vitest.mjs` (the stable
path exists because `vitest` is a catalog devDependency). Revisit once vite-plus
ships a matching test package.

**E2E** (`e2e/*.spec.ts`, `playwright.config.ts`): the real SPA in Chromium with
`/api/*` mocked by `e2e/api-mock.ts` (Playwright route interception) — no Go
backend needed. Playwright's `webServer` runs `vp dev` on port 5199, served from
the dev/test-only `index.html` (production still renders `templates/spa.html`;
`vp build` inputs `src/main.tsx`, so `index.html` never reaches `dist/`).
