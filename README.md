# sshbin

Share logs, configs, or any file from a Linux server with colleagues, vendors, or friends — using plain SSH you already have.

## How it works

```
scp my-log-file.log sshbin.com
```

1. Upload via `scp` or any SFTP client — no new tools to install.
2. Get a setup URL printed to your terminal. Open it to configure the share.
3. Set visibility (public or private), optional password, allowed emails, and expiry.
4. Send the link or scan the QR code.

## Features

- **Upload over SSH** — standard `scp` / `sftp`, no client-side agent needed
- **Access control** — public link, private (email allowlist), or password-protected
- **Expiry** — 1 hour, 24 hours, 7 days, or never
- **My Shares dashboard** — list, edit, and delete your shares
- **QR code** — every share link has a scannable QR code
- **OTP auth** — passwordless login via one-time code (email)
- **Light & dark mode**

## Self-hosting

```
go install github.com/rispycz/sshbin/cmd/sshbin@latest
sshbin --base-url https://sshbin.example.com
```

### Configuration

Every flag can also be set via an environment variable named `SSHBIN_<FLAG>`,
where the flag name is uppercased and `-` becomes `_` (e.g. `--sftp-listen` →
`SSHBIN_SFTP_LISTEN`). Precedence: **flag > env var > default**. Env vars make
container deployments easy; flags stay handy for the CLI.

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `--sftp-listen` | `SSHBIN_SFTP_LISTEN` | `:2022` | SFTP server listen address |
| `--web-listen` | `SSHBIN_WEB_LISTEN` | `:8080` | Web UI listen address |
| `--base-url` | `SSHBIN_BASE_URL` | `http://localhost:8080` | Public URL used in share links and QR codes |
| `--host-key` | `SSHBIN_HOST_KEY` | `host_key` | Path to SSH host key (generated on first run if missing) |
| `--storage` | `SSHBIN_STORAGE` | `local://uploads` | Storage backend DSN (`local://path` or `s3://bucket/prefix`) |
| `--db` | `SSHBIN_DB` | `sqlite://sshbin.db` | Database DSN |
| `--dev` | `SSHBIN_DEV` | `false` | Serve the SPA from the Vite dev server for HMR |
| `--vite-origin` | `SSHBIN_VITE_ORIGIN` | `http://localhost:5173` | Vite dev server URL (used with `--dev`) |
| `--smtp-host` | `SSHBIN_SMTP_HOST` | _(empty)_ | SMTP host for login-code emails (empty → codes logged to stdout) |
| `--smtp-port` | `SSHBIN_SMTP_PORT` | `587` | SMTP port (465 = implicit TLS, otherwise STARTTLS) |
| `--smtp-user` | `SSHBIN_SMTP_USER` | _(empty)_ | SMTP username |
| `--smtp-from` | `SSHBIN_SMTP_FROM` | _(empty)_ | From address for login-code emails |
| `--smtp-insecure` | `SSHBIN_SMTP_INSECURE` | `false` | Skip SMTP TLS certificate verification (dev only) |

**Secrets (env-only, no flag):**

| Env var | Description |
|---------|-------------|
| `SSHBIN_SMTP_PASSWORD` | SMTP password for login-code delivery |
| `AWS_ENDPOINT_URL` | Custom S3 endpoint (enables path-style addressing) for the `s3://` backend |
| `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION` | AWS credentials for the `s3://` backend (standard AWS SDK chain) |

> **OTP email:** without `--smtp-host` set, OTP login codes are printed to the
> log (dev only). Set the SMTP options above to deliver them by email.

## Architecture

Two servers run side by side:

- **SFTP server** (`:2022`) — anonymous upload-only. On close, creates a share record and prints the setup URL to the terminal via SSH stderr.
- **Web UI** (`:8080`) — share configuration, access control, download, and the My Shares dashboard.

See [`architecture/`](architecture/) for detailed design notes.
