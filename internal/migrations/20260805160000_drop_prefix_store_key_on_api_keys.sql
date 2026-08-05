-- +goose Up
ALTER TABLE api_keys DROP COLUMN prefix;
ALTER TABLE api_keys ADD COLUMN key TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE api_keys DROP COLUMN key;
ALTER TABLE api_keys ADD COLUMN prefix TEXT NOT NULL DEFAULT 'cairo_';
