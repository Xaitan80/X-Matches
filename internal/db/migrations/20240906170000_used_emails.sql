-- +goose Up
CREATE TABLE IF NOT EXISTS used_emails (
    email       TEXT PRIMARY KEY,
    reserved_by INTEGER, -- optional user id (no FK to allow persistence after user deletion)
    created_at  TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

CREATE INDEX IF NOT EXISTS idx_used_emails_reserved_by ON used_emails(reserved_by);

-- +goose Down
DROP TABLE IF EXISTS used_emails;

