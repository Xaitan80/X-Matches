-- +goose Up
ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite cannot drop columns easily; recreate would be needed. No-op.

