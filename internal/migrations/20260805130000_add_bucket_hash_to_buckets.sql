-- +goose Up
ALTER TABLE buckets ADD COLUMN bucket_hash TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE buckets DROP COLUMN bucket_hash;
