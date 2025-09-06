package matches

import (
    "bufio"
    "bytes"
    "encoding/csv"
    "fmt"
    "io"
    "mime/multipart"
    "path/filepath"
    "strconv"
    "strings"
    "unicode"

    "github.com/xuri/excelize/v2"
)

// parseImport reads a CSV or XLSX file from a multipart form file and returns a slice of Match.
func parseImport(fh *multipart.FileHeader, ourTeam string) ([]Match, error) {
    ext := strings.ToLower(filepath.Ext(fh.Filename))
    file, err := fh.Open()
    if err != nil { return nil, err }
    defer file.Close()

    switch ext {
    case ".csv":
        return parseCSV(file, ourTeam)
    case ".xlsx":
        // excelize needs a path or bytes; we can read into memory (reasonable for small files).
        // To avoid large memory, cap at ~10MB by streaming into a buffer.
        b, err := io.ReadAll(io.LimitReader(file, 10<<20))
        if err != nil { return nil, err }
        return parseXLSX(b, ourTeam)
    default:
        return nil, fmt.Errorf("unsupported file type: %s", ext)
    }
}

func parseCSV(r io.Reader, ourTeam string) ([]Match, error) {
    br := bufio.NewReader(r)
    // Peek first line to guess delimiter
    line, _ := br.ReadString('\n')
    // Put it back into the stream
    rest := io.MultiReader(strings.NewReader(line), br)
    reader := csv.NewReader(rest)
    reader.FieldsPerRecord = -1
    if strings.Count(line, ";") > strings.Count(line, ",") {
        reader.Comma = ';'
    }
    rows, err := reader.ReadAll()
    if err != nil { return nil, err }
    if len(rows) == 0 { return nil, fmt.Errorf("empty csv") }
    headers := normHeaders(rows[0])
    var out []Match
    for i := 1; i < len(rows); i++ {
        if len(strings.TrimSpace(strings.Join(rows[i], ""))) == 0 { continue }
        out = append(out, rowToMatch(headers, rows[i], ourTeam))
    }
    return out, nil
}

func parseXLSX(b []byte, ourTeam string) ([]Match, error) {
    // Use bytes.Reader to provide Reader, ReaderAt, and Seeker which excelize can leverage
    f, err := excelize.OpenReader(bytes.NewReader(b))
    if err != nil { return nil, err }
    defer f.Close()
    sheet := f.GetSheetName(0)
    if sheet == "" { return nil, fmt.Errorf("no sheet") }
    rows, err := f.GetRows(sheet)
    if err != nil { return nil, err }
    if len(rows) == 0 { return nil, fmt.Errorf("empty sheet") }
    headers := normHeaders(rows[0])
    var out []Match
    for i := 1; i < len(rows); i++ {
        out = append(out, rowToMatch(headers, rows[i], ourTeam))
    }
    return out, nil
}

// (no custom reader needed; using bytes.NewReader above)

// normalize headers: lower, remove spaces/underscores, swedish variants
func normHeaders(hdr []string) map[int]string {
    m := make(map[int]string, len(hdr))
    for i, h := range hdr {
        k := strings.ToLower(strings.TrimSpace(h))
        // keep only letters/digits for robustness (drops pipes, commas, etc.)
        b := strings.Builder{}
        for _, r := range k {
            if unicode.IsLetter(r) || unicode.IsDigit(r) {
                // fold Swedish diacritics to simple letters
                switch r {
                case 'å', 'ä': r = 'a'
                case 'ö': r = 'o'
                }
                b.WriteRune(r)
            }
        }
        k = b.String()
        // Swedish aliases
        switch k {
        case "datum": k = "dateraw"
        case "starttid", "tid": k = "timeraw"
        case "sluttid": k = "endtimeraw"
        case "hall", "hallplats", "plats", "spelplats": k = "venue"
        case "stad": k = "city"
        case "serie", "tavling": k = "league"
        case "plan": k = "court"
        case "samlingstid": k = "gathertime"
        case "samlingplats": k = "gatherplace"
        case "noteringar", "notis": k = "notes"
        case "spelad": k = "played"
        case "malfor", "malhemma": k = "goalsfor"
        case "malmot", "malborta": k = "goalsagainst"
        case "toppvart", "toppskyttlag": k = "topscorerteam"
        case "toppmot", "toppskyttmot": k = "topscoreropponent"
        case "hemmalag": k = "hometeam"
        case "bortalag": k = "awayteam"
        case "matchnr": k = "matchnumber"
        case "dag": k = "weekday"
        case "resultat": k = "result"
        }
        m[i] = k
    }
    return m
}

func rowToMatch(h map[int]string, row []string, ourTeam string) Match {
    get := func(key string) string {
        for i, k := range h { if k == key && i < len(row) { return strings.TrimSpace(row[i]) } }
        return ""
    }
    atoi := func(s string) int64 { if s=="" { return 0 }; v, _ := strconv.ParseInt(s, 10, 64); return v }
    atob := func(s string) bool {
        s = strings.ToLower(strings.TrimSpace(s))
        return s=="1" || s=="true" || s=="ja" || s=="yes" || s=="y"
    }
    venue := get("venue")
    // Split venue into venue/city if comma-separated
    if parts := strings.SplitN(venue, ",", 2); len(parts)==2 {
        venue = strings.TrimSpace(parts[0])
        if c := strings.TrimSpace(parts[1]); c != "" { /* city handled below */ }
    }

    m := Match{
        DateRaw:      get("dateraw"),
        TimeRaw:      get("timeraw"),
        EndTimeRaw:   get("endtimeraw"),
        Weekday:      get("weekday"),
        League:       get("league"),
        Team:         get("team"),
        Opponent:     get("opponent"),
        HomeTeam:     get("hometeam"),
        AwayTeam:     get("awayteam"),
        Venue:        venue,
        Court:        get("court"),
        City:         get("city"),
        MatchNumber:  get("matchnumber"),
        Referees:     get("referees"),
        Notes:        get("notes"),
        Played:       atob(get("played")),
        GoalsFor:     atoi(get("goalsfor")),
        GoalsAgainst: atoi(get("goalsagainst")),
        PlayerNotes:  get("playernotes"),
        TopScorerTeam: get("topscorerteam"),
        TopScorerOpponent: get("topscoreropponent"),
    }
    // If venue had embedded city (after comma) and no explicit city provided
    if m.City == "" {
        if parts := strings.SplitN(get("venue"), ",", 2); len(parts)==2 {
            c := strings.TrimSpace(parts[1])
            if c != "" { m.City = c }
        }
    }
    // Populate team/opponent based on ourTeam if provided
    ht := strings.TrimSpace(m.HomeTeam)
    at := strings.TrimSpace(m.AwayTeam)
    ot := strings.TrimSpace(ourTeam)
    eq := func(a, b string) bool { return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) }
    if ot != "" {
        if eq(ht, ot) {
            m.Team, m.Opponent = ht, at
        } else if eq(at, ot) {
            m.Team, m.Opponent = at, ht
        } else {
            // fallback to home/away
            if m.Team == "" { m.Team = ht }
            if m.Opponent == "" { m.Opponent = at }
        }
    } else {
        if m.Team == "" { m.Team = ht }
        if m.Opponent == "" { m.Opponent = at }
    }
    // Parse simple result like "3-1" into goals if present
    if r := get("result"); r != "" {
        r = strings.TrimSpace(r)
        if strings.Contains(r, "-") {
            parts := strings.SplitN(r, "-", 2)
            gf := atoi(strings.TrimSpace(parts[0])); ga := int64(0)
            if len(parts)==2 { ga = atoi(strings.TrimSpace(parts[1])) }
            if gf>0 || ga>0 { m.GoalsFor, m.GoalsAgainst, m.Played = gf, ga, true }
        }
    }
    // Optional ISO columns
    if s := get("startiso"); s != "" { m.StartISO = &s }
    if s := get("endiso"); s != "" { m.EndISO = &s }
    return m
}
