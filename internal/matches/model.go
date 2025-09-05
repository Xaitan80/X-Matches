package matches

type Match struct {
    ID           int64   `json:"id"`
    StartISO     *string `json:"start_iso"`
    EndISO       *string `json:"end_iso"`
    DateRaw      string  `json:"date_raw"`
    TimeRaw      string  `json:"time_raw"`
    EndTimeRaw   string  `json:"end_time_raw"`
    Weekday      string  `json:"weekday"`
    League       string  `json:"league"`
    Team         string  `json:"team"`
    Opponent     string  `json:"opponent"`
    HomeTeam     string  `json:"home_team"`
    AwayTeam     string  `json:"away_team"`
    Venue        string  `json:"venue"`
    Court        string  `json:"court"`
    City         string  `json:"city"`
    GatherTime   string  `json:"gather_time"`
    GatherPlace  string  `json:"gather_place"`
    MatchNumber  string  `json:"match_number"`
    Referees     string  `json:"referees"`
    Notes        string  `json:"notes"`
    Played       bool    `json:"played"`
    GoalsFor     int64   `json:"goals_for"`     // <-- int64
    GoalsAgainst int64   `json:"goals_against"` // <-- int64
    PlayerNotes  string  `json:"player_notes"`
    TopScorerTeam string `json:"top_scorer_team"`
    TopScorerOpponent string `json:"top_scorer_opponent"`
}
