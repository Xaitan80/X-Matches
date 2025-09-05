-- +goose Up
ALTER TABLE matches ADD COLUMN top_scorer_team TEXT;
ALTER TABLE matches ADD COLUMN top_scorer_opponent TEXT;

-- +goose Down
-- SQLite DROP COLUMN is not universally supported in older versions.
-- No-op down migration.
SELECT 1;

