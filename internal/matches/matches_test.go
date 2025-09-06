package matches

import (
	"strings"
	"testing"
)

func TestNormHeaders_SwedishAliases(t *testing.T) {
	hdr := []string{"Matchnr", "Dag", "Datum |", "Tid", "Tävling", "Hemmalag", "Bortalag", "Spelplats", "Resultat", "Mål borta", "Mål hemma"}
	m := normHeaders(hdr)
	assertEq(t, m[0], "matchnumber")
	assertEq(t, m[1], "weekday")
	assertEq(t, m[2], "dateraw") // strip pipe and space
	assertEq(t, m[3], "timeraw")
	assertEq(t, m[4], "league") // folds ä -> a in Tävling
	assertEq(t, m[5], "hometeam")
	assertEq(t, m[6], "awayteam")
	assertEq(t, m[7], "venue")
	assertEq(t, m[8], "result")
	assertEq(t, m[9], "goalsagainst")
	assertEq(t, m[10], "goalsfor")
}

func TestRowToMatch_FallbackHomeAway_NoOurTeam(t *testing.T) {
	hdr := []string{"Hemmalag", "Bortalag", "Spelplats"}
	h := normHeaders(hdr)
	row := []string{"Home FC", "Away FC", "Arena, City"}
	m := rowToMatch(h, row, "")
	assertEq(t, m.Team, "Home FC")
	assertEq(t, m.Opponent, "Away FC")
	assertEq(t, m.Venue, "Arena")
	assertEq(t, m.City, "City")
}

func TestParseCSV_CommaDelimiter(t *testing.T) {
	csv := "Matchnr,Dag,Datum |,Tid,Tävling,Hemmalag,Resultat,Bortalag,Spelplats\n" +
		"1,Lördag,2025-10-10,12:00,F16,H43 Lund HF,1-0,XYZ,Hall, Lund\n"
	rows, err := parseCSV(strings.NewReader(csv), "H43 Lund HF")
	if err != nil {
		t.Fatalf("parseCSV error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if !rows[0].Played || rows[0].GoalsFor != 1 || rows[0].GoalsAgainst != 0 {
		t.Fatalf("result parse failed: %+v", rows[0])
	}
}

func TestRowToMatch_ResultWhitespace(t *testing.T) {
	hdr := []string{"Resultat"}
	h := normHeaders(hdr)
	row := []string{" 4 - 2 "}
	m := rowToMatch(h, row, "")
	if !m.Played || m.GoalsFor != 4 || m.GoalsAgainst != 2 {
		t.Fatalf("bad parse: played=%v gf=%d ga=%d", m.Played, m.GoalsFor, m.GoalsAgainst)
	}
}

// --- small helpers ---
func assertEq[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}
