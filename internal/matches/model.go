package matches

import "time"

type Match struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	StartISO    *time.Time `json:"start_iso"`
	EndISO      *time.Time `json:"end_iso"`
	DateRaw     string     `json:"date_raw"`
	TimeRaw     string     `json:"time_raw"`
	EndTimeRaw  string     `json:"end_time_raw"`
	Weekday     string     `json:"weekday"`
	League      string     `json:"league"`
	Team        string     `json:"team"`
	Opponent    string     `json:"opponent"`
	HomeTeam    string     `json:"home_team"`
	AwayTeam    string     `json:"away_team"`
	Venue       string     `json:"venue"`
	Court       string     `json:"court"`
	City        string     `json:"city"`
	GatherTime  string     `json:"gather_time"`
	GatherPlace string     `json:"gather_place"`
	MatchNumber string     `json:"match_number"`
	Referees    string     `json:"referees"`
	Notes       string     `json:"notes"`

	Played       bool   `json:"played"`
	GoalsFor     int    `json:"goals_for"`
	GoalsAgainst int    `json:"goals_against"`
	PlayerNotes  string `json:"player_notes"`
}
