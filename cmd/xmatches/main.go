package main

import (
    "context"
    "database/sql"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"
    "time"

    _ "modernc.org/sqlite"

    "golang.org/x/crypto/bcrypt"

    dbpkg "github.com/xaitan80/X-Matches/internal/db"
    "github.com/xaitan80/X-Matches/internal/matches"
    "github.com/xaitan80/X-Matches/internal/auth"
)

func main() {
    if len(os.Args) < 2 {
        usage()
        return
    }

    cmd := os.Args[1]
    switch cmd {
    case "help", "-h", "--help":
        usage()
    case "init":
        withDB(func(sqlDB *sql.DB) error {
            fmt.Println("Database initialized and migrated.")
            return nil
        })
    case "register":
        fs := flag.NewFlagSet("register", flag.ExitOnError)
        email := fs.String("email", "", "Email address")
        pass := fs.String("password", "", "Password (min 12 chars)")
        _ = fs.Parse(os.Args[2:])
        if strings.TrimSpace(*email) == "" || *pass == "" {
            returnOrExit(errors.New("usage: xmatches register --email <email> --password <password>"))
        }
        if len(*pass) < 12 { returnOrExit(errors.New("password too short (min 12)")) }
        withDB(func(sqlDB *sql.DB) error {
            repo := auth.NewRepository(sqlDB)
            hash, err := bcrypt.GenerateFromPassword([]byte(*pass), bcrypt.DefaultCost)
            if err != nil { return err }
            u, err := repo.CreateUser(defaultCtx(), strings.ToLower(strings.TrimSpace(*email)), string(hash))
            if err != nil { return err }
            fmt.Printf("Registered user: %s (id=%d)\n", u.Email, u.ID)
            return nil
        })
    case "login":
        fs := flag.NewFlagSet("login", flag.ExitOnError)
        email := fs.String("email", "", "Email address")
        pass := fs.String("password", "", "Password")
        _ = fs.Parse(os.Args[2:])
        if strings.TrimSpace(*email) == "" || *pass == "" { returnOrExit(errors.New("usage: xmatches login --email <email> --password <password>")) }
        withDB(func(sqlDB *sql.DB) error {
            repo := auth.NewRepository(sqlDB)
            u, err := repo.GetUserByEmail(defaultCtx(), strings.ToLower(strings.TrimSpace(*email)))
            if err != nil { return errors.New("invalid credentials") }
            if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(*pass)); err != nil { return errors.New("invalid credentials") }
            ttl := sessionTTL()
            s, err := repo.CreateSession(defaultCtx(), u.ID, ttl)
            if err != nil { return err }
            if err := saveSessionToken(s.Token); err != nil { return err }
            fmt.Printf("Logged in as %s. Session valid for %s.\n", u.Email, ttl)
            return nil
        })
    case "logout":
        withDB(func(sqlDB *sql.DB) error {
            repo := auth.NewRepository(sqlDB)
            if tok, err := readSessionToken(); err == nil && tok != "" {
                _ = repo.DeleteSession(defaultCtx(), tok)
            }
            _ = deleteSessionToken()
            fmt.Println("Logged out.")
            return nil
        })
    case "me":
        withDB(func(sqlDB *sql.DB) error {
            ensureLoggedIn(sqlDB)
            repo := auth.NewRepository(sqlDB)
            tok, _ := readSessionToken()
            u, err := repo.GetUserBySession(defaultCtx(), tok)
            if err != nil { return errors.New("not logged in") }
            fmt.Printf("User: %s (id=%d) admin=%v\n", u.Email, u.ID, (u.IsAdmin || isAdminEmail(u.Email)))
            return nil
        })
    case "list":
        fs := flag.NewFlagSet("list", flag.ExitOnError)
        _ = fs.Parse(os.Args[2:])
        withDB(func(sqlDB *sql.DB) error {
            ensureLoggedIn(sqlDB)
            q := dbpkg.New(sqlDB)
            repo := matches.NewRepository(q)
            items, err := repo.List(defaultCtx())
            if err != nil { return err }
            if len(items) == 0 { fmt.Println("No matches."); return nil }
            for _, m := range items {
                date := sval(m.DateRaw)
                tm := sval(m.TimeRaw)
                fmt.Printf("#%d  %s %s  %s vs %s  @ %s %s\n", m.ID, date, tm, sval(m.Team), sval(m.Opponent), sval(m.Venue), sval(m.City))
            }
            return nil
        })
    case "add":
        fs := flag.NewFlagSet("add", flag.ExitOnError)
        date := fs.String("date", "", "Date (YYYY-MM-DD)")
        timeRaw := fs.String("time", "", "Start time (HH:MM)")
        league := fs.String("league", "", "League")
        team := fs.String("team", "", "Our team")
        opp := fs.String("opponent", "", "Opponent")
        venue := fs.String("venue", "", "Venue/Hall")
        city := fs.String("city", "", "City")
        _ = fs.Parse(os.Args[2:])
        withDB(func(sqlDB *sql.DB) error {
            ensureLoggedIn(sqlDB)
            q := dbpkg.New(sqlDB)
            repo := matches.NewRepository(q)
            m := matches.Match{ DateRaw: *date, TimeRaw: *timeRaw, League: *league, Team: *team, Opponent: *opp, Venue: *venue, City: *city }
            row, err := repo.Create(defaultCtx(), m)
            if err != nil { return err }
            fmt.Printf("Created match #%d\n", row.ID)
            return nil
        })
    case "import":
        fs := flag.NewFlagSet("import", flag.ExitOnError)
        file := fs.String("file", "", "Path to CSV/XLSX")
        our := fs.String("our-team", "", "Our team name (optional)")
        _ = fs.Parse(os.Args[2:])
        if strings.TrimSpace(*file) == "" { returnOrExit(errors.New("usage: xmatches import --file <path> [--our-team <name>]")) }
        withDB(func(sqlDB *sql.DB) error {
            ensureLoggedIn(sqlDB)
            list, err := matches.ParseFile(*file, *our)
            if err != nil { return err }
            q := dbpkg.New(sqlDB)
            repo := matches.NewRepository(q)
            n := 0
            for _, m := range list {
                if _, err := repo.Create(defaultCtx(), m); err == nil { n++ } else { return fmt.Errorf("failed at row %d: %w", n+1, err) }
            }
            fmt.Printf("Imported %d matches.\n", n)
            return nil
        })
    default:
        fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
        usage()
    }
}

// Helpers

func usage() {
    fmt.Println("X-Matches CLI")
    fmt.Println()
    fmt.Println("Usage:")
    fmt.Println("  xmatches <command> [flags]")
    fmt.Println()
    fmt.Println("Commands:")
    fmt.Println("  init                      Initialize DB (migrate)")
    fmt.Println("  register --email --password   Create user (first becomes admin)")
    fmt.Println("  login --email --password      Login and save session locally")
    fmt.Println("  logout                    Logout current session")
    fmt.Println("  me                        Show current user")
    fmt.Println("  list                      List matches (requires login)")
    fmt.Println("  add --date --time --league --team --opponent [--venue --city]")
    fmt.Println("  import --file <csv/xlsx> [--our-team <name>]")
    fmt.Println()
    fmt.Println("Environment:")
    fmt.Println("  DB_PATH           SQLite path (default: xmatches.db)")
    fmt.Println("  SESSION_TTL       e.g. 720h, default 720h (30 days)")
    fmt.Println("  ADMIN_EMAILS      Comma-separated admin emails (optional)")
}

func withDB(fn func(*sql.DB) error) {
    sqlDB, err := openDB()
    if err != nil { returnOrExit(err) }
    defer sqlDB.Close()
    if err := dbpkg.Migrate(sqlDB); err != nil { returnOrExit(fmt.Errorf("migrate: %w", err)) }
    if err := fn(sqlDB); err != nil { returnOrExit(err) }
}

func openDB() (*sql.DB, error) {
    dsn := env("DB_PATH", "xmatches.db")
    if !strings.Contains(strings.ToLower(dsn), "_pragma=foreign_keys(1)") {
        if strings.Contains(dsn, "?") { dsn += "&_pragma=foreign_keys(1)" } else { dsn += "?_pragma=foreign_keys(1)" }
    }
    sqlDB, err := sql.Open("sqlite", dsn)
    if err != nil { return nil, err }
    if err := sqlDB.Ping(); err != nil { _ = sqlDB.Close(); return nil, err }
    return sqlDB, nil
}

func env(k, def string) string {
    if v := os.Getenv(k); v != "" { return v }
    return def
}

func sessionTTL() time.Duration {
    if v := os.Getenv("SESSION_TTL"); v != "" {
        if d, err := time.ParseDuration(v); err == nil && d > 0 { return d }
    }
    return 30 * 24 * time.Hour
}

func configDir() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil { return "", err }
    dir := filepath.Join(home, ".xmatches")
    if err := os.MkdirAll(dir, 0o700); err != nil { return "", err }
    return dir, nil
}

func sessionPath() (string, error) {
    dir, err := configDir()
    if err != nil { return "", err }
    return filepath.Join(dir, "session"), nil
}

func saveSessionToken(tok string) error {
    p, err := sessionPath()
    if err != nil { return err }
    return os.WriteFile(p, []byte(tok), 0o600)
}

func readSessionToken() (string, error) {
    p, err := sessionPath()
    if err != nil { return "", err }
    b, err := os.ReadFile(p)
    if err != nil { return "", err }
    return strings.TrimSpace(string(b)), nil
}

func deleteSessionToken() error {
    p, err := sessionPath()
    if err != nil { return err }
    if err := os.Remove(p); err != nil && !os.IsNotExist(err) { return err }
    return nil
}

func ensureLoggedIn(sqlDB *sql.DB) {
    tok, err := readSessionToken()
    if err != nil || tok == "" { returnOrExit(errors.New("not logged in (run: xmatches login)")) }
    repo := auth.NewRepository(sqlDB)
    if _, err := repo.GetUserBySession(defaultCtx(), tok); err != nil {
        returnOrExit(errors.New("session invalid; please login again"))
    }
}

func sval(p *string) string { if p == nil { return "" }; return *p }

func isAdminEmail(email string) bool {
    list := strings.Split(os.Getenv("ADMIN_EMAILS"), ",")
    e := strings.ToLower(strings.TrimSpace(email))
    for _, item := range list {
        if strings.ToLower(strings.TrimSpace(item)) == e && e != "" { return true }
    }
    return false
}

func defaultCtx() context.Context { return context.Background() }

func returnOrExit(err error) {
    if err == nil { return }
    log.Fatalf("Error: %v", err)
}
