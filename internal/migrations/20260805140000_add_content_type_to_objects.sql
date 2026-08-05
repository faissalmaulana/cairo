-- +goose Up
ALTER TABLE objects ADD COLUMN content_type TEXT NOT NULL DEFAULT '';

-- +goose Down
