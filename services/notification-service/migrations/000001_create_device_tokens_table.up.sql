CREATE TABLE IF NOT EXISTS device_tokens (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    token       TEXT NOT NULL UNIQUE,
    platform    TEXT NOT NULL,
    device_name TEXT,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_user_id ON device_tokens(user_id);