CREATE TABLE IF NOT EXISTS notifications (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,
    title           TEXT NOT NULL,
    body            TEXT NOT NULL,
    data            TEXT,
    image_url       TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',
    failure_reason  TEXT,
    sent_at         TIMESTAMP,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);