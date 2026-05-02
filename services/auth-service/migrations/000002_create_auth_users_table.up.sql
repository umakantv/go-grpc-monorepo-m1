CREATE TABLE IF NOT EXISTS auth_users (
    id TEXT PRIMARY KEY,
    firebase_uid TEXT UNIQUE,
    email TEXT,
    phone_number TEXT,
    display_name TEXT,
    photo_url TEXT,
    auth_provider TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);