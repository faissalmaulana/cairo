-- +goose Up
ALTER TABLE objects ADD COLUMN content_type TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE objects DROP COLUMN content_type;
