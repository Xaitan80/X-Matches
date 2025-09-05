-- name: CreateMatch :one
INSERT INTO matches (
  start_iso, end_iso, date_raw, time_raw, end_time_raw, weekday,
  league, team, opponent, home_team, away_team, venue, court, city,
  gather_time, gather_place, match_number, referees, notes,
  played, goals_for, goals_against, player_notes,
  top_scorer_team, top_scorer_opponent
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetMatch :one
SELECT * FROM matches WHERE id = ?;

-- name: ListMatches :many
SELECT * FROM matches
ORDER BY (start_iso IS NULL), start_iso, id;

-- name: UpdateMatch :one
UPDATE matches
SET
  start_iso = ?,
  end_iso = ?,
  date_raw = ?,
  time_raw = ?,
  end_time_raw = ?,
  weekday = ?,
  league = ?,
  team = ?,
  opponent = ?,
  home_team = ?,
  away_team = ?,
  venue = ?,
  court = ?,
  city = ?,
  gather_time = ?,
  gather_place = ?,
  match_number = ?,
  referees = ?,
  notes = ?,
  played = ?,
  goals_for = ?,
  goals_against = ?,
  player_notes = ?,
  top_scorer_team = ?,
  top_scorer_opponent = ?
WHERE id = ?
RETURNING *;

-- name: DeleteMatch :exec
DELETE FROM matches WHERE id = ?;
