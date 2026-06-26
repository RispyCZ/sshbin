# SFTP Server

Built on top of [charmbracelet/wish](https://github.com/charmbracelet/wish) and [pkg/sftp](https://github.com/pkg/sftp).

## How it works

The SFTP server accepts anonymous connections on `:2022` and is intentionally **upload-only** — reads, deletes, and directory mutations are rejected with `EPERM`.

On `scp` / `sftp put`:
1. A UUID is generated for the upload session.
2. The file is written to `{storage}/{uuid}/{filename}` as the SFTP client streams it.
3. On file close, a `shares` record is created in the database and the setup URL is printed to the client's terminal.

## Upload handler

`internal/sftp/handler.go` — `uploadOnlyHandler` implements the `sftp.Handlers` interface:

| Method | Behaviour |
|--------|-----------|
| `Filewrite` | Creates storage file, returns write handle |
| `WriteAt` | Streams bytes in order (sequential writes only) |
| `Close` | Finalises storage, creates share record, prints setup URL |
| `Fileread` | Returns `EPERM` |
| `Filecmd` | Accepts `Setstat` (mtime preservation), rejects mutations |
| `Filelist` | Returns synthetic root dir or file stat for SFTP protocol compliance |

## Host key

Path set by `--host-key` (default: `host_key`). Generated automatically on first run if the file does not exist.
