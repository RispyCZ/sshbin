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

| Flag | Default | Description |
|------|---------|-------------|
| `--sftp-listen` | `:2022` | SFTP server listen address |
| `--web-listen` | `:8080` | Web UI listen address |
| `--base-url` | `http://localhost:8080` | Public URL used in share links and QR codes |
| `--host-key` | `host_key` | Path to SSH host key (generated on first run if missing) |
| `--storage` | `uploads` | Directory for uploaded files |
| `--db` | `sqlite://sshbin.db` | Database DSN |

> **OTP email:** the default build logs OTP codes to stdout. Wire up an SMTP sender by replacing `auth.LogSender` in `cmd/sshbin/main.go`.

## Architecture

Two servers run side by side:

- **SFTP server** (`:2022`) — anonymous upload-only. On close, creates a share record and prints the setup URL to the terminal via SSH stderr.
- **Web UI** (`:8080`) — share configuration, access control, download, and the My Shares dashboard.

See [`architecture/`](architecture/) for detailed design notes.
