CREATE TABLE IF NOT EXISTS matches (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    start_iso      TEXT,      -- ISO8601, t.ex. "2025-09-20T14:30:00+02:00"
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
    played         INTEGER DEFAULT 0, -- 0/1
    goals_for      INTEGER,
    goals_against  INTEGER,
    player_notes   TEXT
);
