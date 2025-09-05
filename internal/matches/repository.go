package matches

import (
	"context"
	"fmt"
	"time"

	dbpkg "github.com/xaitan80/X-Matches/internal/db"
)

type Repository struct {
	q *dbpkg.Queries
}

func NewRepository(q *dbpkg.Queries) *Repository { return &Repository{q: q} }

// -------- Helpers --------

func ParseLocalISO(dateRaw, timeRaw string) *string {
	if dateRaw == "" && timeRaw == "" {
		return nil
	}
	if timeRaw == "" {
		timeRaw = "00:00"
	}
	loc, _ := time.LoadLocation("Europe/Stockholm")
	t, err := time.ParseInLocation("2006-01-02 15:04", dateRaw+" "+timeRaw, loc)
	if err != nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func pstr(s string) *string {
	if s == "" {
		return nil
	}
	v := s
	return &v
}

func pstrKeep(newVal string, cur *string) *string {
	if newVal != "" {
		return pstr(newVal)
	}
	return cur
}

func pPlayed(b bool) *int64 {
	var v int64 = 0
	if b {
		v = 1
	}
	return &v
}

func pI64ZeroNil(v int64) *int64 {
	if v == 0 {
		return nil
	}
	x := v
	return &x
}

// -------- CRUD --------

func (r *Repository) List(ctx context.Context) ([]dbpkg.Match, error) {
	return r.q.ListMatches(ctx)
}

func (r *Repository) Get(ctx context.Context, id int64) (dbpkg.Match, error) {
	return r.q.GetMatch(ctx, id)
}

func (r *Repository) Create(ctx context.Context, m Match) (dbpkg.Match, error) {
	// Beräkna ISO-tider om inte satta
	startISO := m.StartISO
	if startISO == nil && (m.DateRaw != "" || m.TimeRaw != "") {
		startISO = ParseLocalISO(m.DateRaw, m.TimeRaw)
	}
	endISO := m.EndISO
	if endISO == nil && m.DateRaw != "" && m.EndTimeRaw != "" {
		endISO = ParseLocalISO(m.DateRaw, m.EndTimeRaw)
	}

	row, err := r.q.CreateMatch(ctx, dbpkg.CreateMatchParams{
		StartIso:     startISO,           // *string
		EndIso:       endISO,             // *string
		DateRaw:      pstr(m.DateRaw),    // *string
		TimeRaw:      pstr(m.TimeRaw),    // *string
		EndTimeRaw:   pstr(m.EndTimeRaw), // *string
		Weekday:      pstr(m.Weekday),
		League:       pstr(m.League),
		Team:         pstr(m.Team),
		Opponent:     pstr(m.Opponent),
		HomeTeam:     pstr(m.HomeTeam),
		AwayTeam:     pstr(m.AwayTeam),
		Venue:        pstr(m.Venue),
		Court:        pstr(m.Court),
		City:         pstr(m.City),
		GatherTime:   pstr(m.GatherTime),
		GatherPlace:  pstr(m.GatherPlace),
		MatchNumber:  pstr(m.MatchNumber),
		Referees:     pstr(m.Referees),
		Notes:        pstr(m.Notes),
		Played:       pPlayed(m.Played),       // *int64 (0/1)
		GoalsFor:     pI64ZeroNil(m.GoalsFor), // *int64
		GoalsAgainst: pI64ZeroNil(m.GoalsAgainst),
		PlayerNotes:  pstr(m.PlayerNotes),
	})
	return row, err
}

func (r *Repository) Update(ctx context.Context, id int64, m Match) (dbpkg.Match, error) {
	cur, err := r.q.GetMatch(ctx, id)
	if err != nil {
		return dbpkg.Match{}, fmt.Errorf("get: %w", err)
	}

	// Mergning: tom sträng => behåll, annars sätt nytt
	out := cur
	out.DateRaw = pstrKeep(m.DateRaw, cur.DateRaw)
	out.TimeRaw = pstrKeep(m.TimeRaw, cur.TimeRaw)
	out.EndTimeRaw = pstrKeep(m.EndTimeRaw, cur.EndTimeRaw)
	out.Weekday = pstrKeep(m.Weekday, cur.Weekday)
	out.League = pstrKeep(m.League, cur.League)
	out.Team = pstrKeep(m.Team, cur.Team)
	out.Opponent = pstrKeep(m.Opponent, cur.Opponent)
	out.HomeTeam = pstrKeep(m.HomeTeam, cur.HomeTeam)
	out.AwayTeam = pstrKeep(m.AwayTeam, cur.AwayTeam)
	out.Venue = pstrKeep(m.Venue, cur.Venue)
	out.Court = pstrKeep(m.Court, cur.Court)
	out.City = pstrKeep(m.City, cur.City)
	out.GatherTime = pstrKeep(m.GatherTime, cur.GatherTime)
	out.GatherPlace = pstrKeep(m.GatherPlace, cur.GatherPlace)
	out.MatchNumber = pstrKeep(m.MatchNumber, cur.MatchNumber)
	out.Referees = pstrKeep(m.Referees, cur.Referees)
	out.Notes = pstrKeep(m.Notes, cur.Notes)
	out.PlayerNotes = pstrKeep(m.PlayerNotes, cur.PlayerNotes)

	// Played/mål – sätt om inkommande värden är "meningsfulla"
	// (Vi tolkar Goals* = 0 som "lämna som är")
	if m.Played {
		out.Played = pPlayed(m.Played)
	}
	if m.GoalsFor != 0 {
		out.GoalsFor = pI64ZeroNil(m.GoalsFor)
	}
	if m.GoalsAgainst != 0 {
		out.GoalsAgainst = pI64ZeroNil(m.GoalsAgainst)
	}

	// Recompute ISO-tider om date/time ändrats
	startISO := out.StartIso
	endISO := out.EndIso
	if m.DateRaw != "" || m.TimeRaw != "" {
		startISO = ParseLocalISO(sval(out.DateRaw), sval(out.TimeRaw))
	}
	if m.DateRaw != "" || m.EndTimeRaw != "" {
		endISO = ParseLocalISO(sval(out.DateRaw), sval(out.EndTimeRaw))
	}

	return r.q.UpdateMatch(ctx, dbpkg.UpdateMatchParams{
		StartIso:     startISO,
		EndIso:       endISO,
		DateRaw:      out.DateRaw,
		TimeRaw:      out.TimeRaw,
		EndTimeRaw:   out.EndTimeRaw,
		Weekday:      out.Weekday,
		League:       out.League,
		Team:         out.Team,
		Opponent:     out.Opponent,
		HomeTeam:     out.HomeTeam,
		AwayTeam:     out.AwayTeam,
		Venue:        out.Venue,
		Court:        out.Court,
		City:         out.City,
		GatherTime:   out.GatherTime,
		GatherPlace:  out.GatherPlace,
		MatchNumber:  out.MatchNumber,
		Referees:     out.Referees,
		Notes:        out.Notes,
		Played:       out.Played,
		GoalsFor:     out.GoalsFor,
		GoalsAgainst: out.GoalsAgainst,
		PlayerNotes:  out.PlayerNotes,
		ID:           id,
	})
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	return r.q.DeleteMatch(ctx, id)
}
