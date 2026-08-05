-- +goose Up
CREATE TABLE api_keys_new(
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key          TEXT NOT NULL UNIQUE,
    last_used_at int,
    createdAt    int
);
INSERT INTO api_keys_new(id,user_id,key,last_used_at,createdAt)
    SELECT id,user_id,key,last_used_at,createdAt FROM api_keys;
DROP TABLE api_keys;
ALTER TABLE api_keys_new RENAME TO api_keys;

-- +goose Down
CREATE TABLE api_keys_old(
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash     TEXT NOT NULL UNIQUE,
    key          TEXT NOT NULL,
    last_used_at int,
    createdAt    int
);
INSERT INTO api_keys_old(id,user_id,key_hash,key,last_used_at,createdAt)
    SELECT id,user_id,key,key,last_used_at,createdAt FROM api_keys;
DROP TABLE api_keys;
ALTER TABLE api_keys_old RENAME TO api_keys;
