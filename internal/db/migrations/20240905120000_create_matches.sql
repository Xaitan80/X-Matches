-- +goose Up
CREATE TABLE matches (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    start_iso      TEXT,
    end_iso        TEXT,
    date_raw       TEXT,
    time_raw       TEXT,
    end_time_raw   TEXT,
    weekday        TEXT,
    league         TEXT,
    team           TEXT,
    opponent       TEXT,
    home_team      TEXT,
    away_team      TEXT,
    venue          TEXT,
    court          TEXT,
    city           TEXT,
    gather_time    TEXT,
    gather_place   TEXT,
    match_number   TEXT,
    referees       TEXT,
    notes          TEXT,
    played         INTEGER DEFAULT 0,
    goals_for      INTEGER,
    goals_against  INTEGER,
    player_notes   TEXT
);

-- +goose Down
DROP TABLE matches;
