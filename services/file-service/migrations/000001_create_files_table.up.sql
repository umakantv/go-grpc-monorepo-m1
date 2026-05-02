CREATE TABLE IF NOT EXISTS files (
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL,
    mimetype TEXT NOT NULL,
    location TEXT NOT NULL,
    size_kb BIGINT NOT NULL DEFAULT 0,
    owner_entity TEXT NOT NULL,
    owner_entity_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    s3_key TEXT NOT NULL,
    url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);