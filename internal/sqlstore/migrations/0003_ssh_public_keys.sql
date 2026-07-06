CREATE TABLE IF NOT EXISTS ssh_public_keys (
    id             TEXT PRIMARY KEY,
    email          TEXT NOT NULL,
    title          TEXT NOT NULL DEFAULT '',
    fingerprint    TEXT NOT NULL UNIQUE,
    authorized_key TEXT NOT NULL,
    created_at     INTEGER NOT NULL,
    last_used_at   INTEGER
);
CREATE INDEX IF NOT EXISTS idx_ssh_public_keys_email ON ssh_public_keys (email);
