package matches

import (
    "strings"
    "testing"

    "github.com/xuri/excelize/v2"
)

func TestParseCSV_WithSwedishHeaders_MapsFields(t *testing.T) {
    csv := "Matchnr;Dag;Datum |;Tid;Tävling;Hemmalag;Resultat;Bortalag;Spelplats;Match\r\n" +
        "25 4 1461 041;Lördag;2025-11-08;14:30;Flickor - F16 Syd;IK Sund;3-1;H43 Lund HF;Norrehedshallen, Helsingborg;\r\n"

    rows, err := parseCSV(strings.NewReader(csv), "H43 Lund HF")
    if err != nil { t.Fatalf("parseCSV error: %v", err) }
    if len(rows) != 1 { t.Fatalf("expected 1 row, got %d", len(rows)) }
    m := rows[0]

    if m.MatchNumber == "" { t.Errorf("match number not mapped") }
    if m.Weekday == "" { t.Errorf("weekday not mapped") }
    if m.DateRaw != "2025-11-08" { t.Errorf("date_raw = %q", m.DateRaw) }
    if m.TimeRaw != "14:30" { t.Errorf("time_raw = %q", m.TimeRaw) }
    if m.League == "" { t.Errorf("league not mapped") }
    if m.Team != "H43 Lund HF" || m.Opponent != "IK Sund" {
        t.Errorf("team/opponent mapping failed: team=%q opp=%q", m.Team, m.Opponent)
    }
    if m.Venue != "Norrehedshallen" || m.City != "Helsingborg" {
        t.Errorf("venue/city split failed: venue=%q city=%q", m.Venue, m.City)
    }
    if !m.Played || m.GoalsFor != 3 || m.GoalsAgainst != 1 {
        t.Errorf("result parse failed: played=%v gf=%d ga=%d", m.Played, m.GoalsFor, m.GoalsAgainst)
    }
}

func TestParseXLSX_Basic(t *testing.T) {
    f := excelize.NewFile()
    sh := f.GetSheetName(0)
    header := []string{"Matchnr", "Dag", "Datum |", "Tid", "Tävling", "Hemmalag", "Resultat", "Bortalag", "Spelplats", "Match"}
    data := []string{"A1", "Lördag", "2025-10-18", "09:00", "Flickor - F16 Syd", "H43 Lund HF", "2-2", "IFK Kristianstad", "Sparbanken Skåne Arena A, Lund", ""}
    if err := f.SetSheetRow(sh, "A1", &header); err != nil { t.Fatal(err) }
    if err := f.SetSheetRow(sh, "A2", &data); err != nil { t.Fatal(err) }
    buf, err := f.WriteToBuffer()
    if err != nil { t.Fatal(err) }

    rows, err := parseXLSX(buf.Bytes(), "H43 Lund HF")
    if err != nil { t.Fatalf("parseXLSX error: %v", err) }
    if len(rows) != 1 { t.Fatalf("expected 1 row, got %d", len(rows)) }
    m := rows[0]
    if m.Team != "H43 Lund HF" || m.Opponent != "IFK Kristianstad" {
        t.Errorf("team/opponent mapping failed: %q vs %q", m.Team, m.Opponent)
    }
    if m.Venue != "Sparbanken Skåne Arena A" || m.City != "Lund" {
        t.Errorf("venue split failed: venue=%q city=%q", m.Venue, m.City)
    }
    if !m.Played || m.GoalsFor != 2 || m.GoalsAgainst != 2 {
        t.Errorf("result parse failed: played=%v gf=%d ga=%d", m.Played, m.GoalsFor, m.GoalsAgainst)
    }
}

func TestParseLocalISO(t *testing.T) {
    // September should be CEST (+02:00) in Europe/Stockholm
    iso := ParseLocalISO("2025-09-20", "14:30")
    if iso == nil { t.Fatalf("expected iso not nil") }
    if !strings.Contains(*iso, "T14:30:00") {
        t.Errorf("unexpected time in iso: %s", *iso)
    }
    if !strings.HasSuffix(*iso, "+02:00") && !strings.HasSuffix(*iso, "+0200") {
        t.Errorf("unexpected offset in iso: %s", *iso)
    }
}
