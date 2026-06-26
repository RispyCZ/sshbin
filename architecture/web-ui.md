# Web UI

Simple, minimal interface for configuring and managing file shares.

## Tech stack

- **Go templates** (`html/template`) — server-rendered pages, embedded via `embed.FS`
- **Lit** — lightweight web components for interactive elements (OTP input, user menu, toast, share modal)
- **Vanilla CSS** — modern CSS (oklch colors, `color-mix`, nesting, `container-query`-ready); no framework

## Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/` | — | Landing page with upload instructions |
| `GET` | `/login` | — | Email input for OTP login |
| `POST` | `/login` | — | Send OTP code |
| `POST` | `/verify` | — | Verify OTP, set session cookie |
| `POST` | `/logout` | session | Clear session |
| `GET` | `/shares` | session | My Shares dashboard |
| `GET` | `/shares/{id}/qr` | session | QR code PNG for a share |
| `POST` | `/shares/{id}/delete` | session + owner | Delete a share |
| `GET` | `/setup/{id}` | session + owner | Configure share settings |
| `POST` | `/setup/{id}` | session + owner | Save share settings |
| `GET` | `/s/{id}` | varies | View / download a share |
| `POST` | `/s/{id}` | varies | Submit share password |
| `GET` | `/s/{id}/download` | varies | Stream file download |

## Web components

| Component | File | Purpose |
|-----------|------|---------|
| `sb-otp` | `components/sb-otp.js` | Split-digit OTP input with auto-advance |
| `sb-user-menu` | `components/sb-user-menu.js` | Avatar dropdown with My Shares link and sign out |
| `sb-share-link` | `components/sb-share-link.js` | URL input with copy-to-clipboard |
| `sb-share-modal` | `components/sb-share-modal.js` | Modal with QR code image and copy URL |
| `sb-notice` | `components/sb-notice.js` | Inline success / error notice |
| `sb-toast` | `components/sb-toast.js` | Ephemeral toast notifications |

## Access control

- **Public shares** — no session required; optional password checked via stateless HMAC cookie (`fd_pw_{id}`)
- **Private shares** — session required; viewer's email must be in the owner's allowlist
- **Setup / delete** — session required and email must match `owner_email` (first visitor claims an unconfigured share)
