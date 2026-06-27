CREATE TABLE IF NOT EXISTS user_preferences (
    email TEXT PRIMARY KEY,
    default_public INTEGER NOT NULL DEFAULT 0
);
