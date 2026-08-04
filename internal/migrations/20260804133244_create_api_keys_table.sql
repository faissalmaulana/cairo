-- +goose Up
CREATE TABLE IF NOT EXISTS api_keys(
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash    TEXT NOT NULL UNIQUE,
    prefix      TEXT NOT NULL,
    last_used_at int,
    createdAt   int
);

-- +goose Down
DROP TABLE IF EXISTS api_keys;