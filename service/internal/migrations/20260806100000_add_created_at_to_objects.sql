-- +goose Up
ALTER TABLE objects ADD COLUMN created_at INTEGER NOT NULL DEFAULT 0;
UPDATE objects SET created_at = unixepoch() WHERE created_at = 0;

-- +goose Down
ALTER TABLE objects DROP COLUMN created_at;
