-- +goose Up
CREATE TABLE IF NOT EXISTS buckets(
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    owner_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    visibility  INTEGER NOT NULL,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX idx_buckets_owner_id ON buckets(owner_id);
CREATE INDEX idx_buckets_name ON buckets(name);

CREATE TABLE IF NOT EXISTS objects(
    id          TEXT PRIMARY KEY,
    bucket_id   TEXT NOT NULL REFERENCES buckets(id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    path        TEXT NOT NULL,
    size        INTEGER NOT NULL,
    sha256sum   TEXT NOT NULL
);
CREATE INDEX idx_objects_bucket_id ON objects(bucket_id);

-- +goose Down
DROP TABLE IF EXISTS objects;
DROP TABLE IF EXISTS buckets;
