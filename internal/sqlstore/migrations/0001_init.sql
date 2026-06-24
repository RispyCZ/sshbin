CREATE TABLE shares (
    id            TEXT PRIMARY KEY,
    file_id       TEXT NOT NULL,
    file_name     TEXT NOT NULL,
    created_at    INTEGER NOT NULL,
    configured    INTEGER NOT NULL DEFAULT 0,
    owner_email   TEXT NOT NULL DEFAULT '',
    expires_at    INTEGER,
    password_hash TEXT NOT NULL DEFAULT '',
    public        INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE share_allowed_emails (
    share_id TEXT NOT NULL,
    email    TEXT NOT NULL,
    PRIMARY KEY (share_id, email)
);

CREATE TABLE sessions (
    token      TEXT PRIMARY KEY,
    email      TEXT NOT NULL,
    expires_at INTEGER NOT NULL
);

CREATE TABLE app_meta (
    key   TEXT PRIMARY KEY,
    value BLOB NOT NULL
);
