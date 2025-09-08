package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"

    _ "modernc.org/sqlite"

    "golang.org/x/crypto/bcrypt"

    dbpkg "github.com/xaitan80/X-Matches/internal/db"
    "github.com/xaitan80/X-Matches/internal/matches"
    "github.com/xaitan80/X-Matches/internal/auth"

    fyne "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/data/binding"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/layout"
    "fyne.io/fyne/v2/widget"
)

func main() {
    sqlDB, err := openDB()
    if err != nil { log.Fatalf("DB: %v", err) }
    defer sqlDB.Close()
    if err := dbpkg.Migrate(sqlDB); err != nil { log.Fatalf("migrate: %v", err) }

    a := app.New()
    w := a.NewWindow("X‑Matches")
    w.Resize(fyne.NewSize(1000, 700))

    // State
    repoAuth := auth.NewRepository(sqlDB)
    q := dbpkg.New(sqlDB)
    repo := matches.NewRepository(q)

    // Data binding for table
    listData := binding.NewUntypedList()

    refresh := func() {
        ctx := context.Background()
        rows, err := repo.List(ctx)
        if err != nil {
            dialog.ShowError(err, w)
            return
        }
        // convert to []any for binding
        anyRows := make([]any, len(rows))
        for i := range rows { anyRows[i] = rows[i] }
        _ = listData.Set(anyRows)
    }

    // Table (List) view
    list := widget.NewListWithData(listData,
        func() fyne.CanvasObject {
            // row template
            id := widget.NewLabel("#")
            date := widget.NewLabel("")
            tm := widget.NewLabel("")
            teams := widget.NewLabel("")
            venue := widget.NewLabel("")
            score := widget.NewLabel("")
            id.TextStyle.Monospace = true
            return container.NewGridWithColumns(6, id, date, tm, teams, venue, score)
        },
        func(item binding.DataItem, obj fyne.CanvasObject) {
            // bind values
            row := item.(binding.Untyped)
            v, _ := row.Get()
            m := v.(dbpkg.Match)
            cols := obj.(*fyne.Container).Objects
            cols[0].(*widget.Label).SetText(fmt.Sprintf("#%d", m.ID))
            cols[1].(*widget.Label).SetText(sval(m.DateRaw))
            cols[2].(*widget.Label).SetText(sval(m.TimeRaw))
            cols[3].(*widget.Label).SetText(fmt.Sprintf("%s vs %s", sval(m.Team), sval(m.Opponent)))
            cols[4].(*widget.Label).SetText(strings.TrimSpace(sval(m.Venue)+" "+sval(m.City)))
            played := ""
            if m.Played != nil && *m.Played == 1 {
                played = fmt.Sprintf("%d-%d", i64(m.GoalsFor), i64(m.GoalsAgainst))
            }
            cols[5].(*widget.Label).SetText(played)
        },
    )

    // Toolbar actions
    btnAdd := widget.NewButton("Lägg till", func() { showAddDialog(w, repo, refresh) })
    btnImport := widget.NewButton("Importera", func() { showImportDialog(w, repo, refresh) })
    btnRefresh := widget.NewButton("Uppdatera", func() { refresh() })
    btnLogout := widget.NewButton("Logga ut", func() { showLoginDialog(w, repoAuth, func(u auth.User) { refresh() }) })

    top := container.NewHBox(btnAdd, btnImport, layout.NewSpacer(), btnRefresh, btnLogout)
    content := container.NewBorder(top, nil, nil, nil, list)

    // Require login first
    showLoginDialog(w, repoAuth, func(u auth.User) { refresh() })

    w.SetContent(content)
    w.ShowAndRun()
}

func showLoginDialog(w fyne.Window, repo *auth.Repository, onOK func(auth.User)) {
    email := widget.NewEntry(); email.SetPlaceHolder("E‑post")
    pass := widget.NewPasswordEntry(); pass.SetPlaceHolder("Lösenord")
    status := widget.NewLabel("")
    form := &widget.Form{Items: []*widget.FormItem{
        {Text: "E‑post", Widget: email},
        {Text: "Lösenord", Widget: pass},
    }}
    regBtn := widget.NewButton("Ny användare", func() { showRegisterDialog(w, repo) })
    dialog.ShowCustomConfirm("Logga in", "Logga in", "Avbryt",
        container.NewVBox(form, status, regBtn), func(ok bool) {
            if !ok { return }
            e := strings.ToLower(strings.TrimSpace(email.Text))
            p := pass.Text
            if e == "" || p == "" { status.SetText("Fyll i e‑post och lösenord"); showLoginDialog(w, repo, onOK); return }
            u, err := repo.GetUserByEmail(context.Background(), e)
            if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(p)) != nil {
                dialog.ShowError(fmt.Errorf("Ogiltiga uppgifter"), w)
                showLoginDialog(w, repo, onOK)
                return
            }
            onOK(u)
        }, w)
}

func showRegisterDialog(w fyne.Window, repo *auth.Repository) {
    email := widget.NewEntry(); email.SetPlaceHolder("E‑post")
    pass := widget.NewPasswordEntry(); pass.SetPlaceHolder("Lösenord (min 12 tecken)")
    confirm := widget.NewPasswordEntry(); confirm.SetPlaceHolder("Bekräfta lösenord")

    form := &widget.Form{Items: []*widget.FormItem{
        {Text: "E‑post", Widget: email},
        {Text: "Lösenord", Widget: pass},
        {Text: "Bekräfta", Widget: confirm},
    }}
    dialog.ShowCustomConfirm("Registrera", "Skapa", "Avbryt", form, func(ok bool) {
        if !ok { return }
        e := strings.ToLower(strings.TrimSpace(email.Text))
        p := pass.Text
        c := confirm.Text
        if e == "" || !strings.Contains(e, "@") { dialog.ShowError(fmt.Errorf("Ogiltig e‑post"), w); return }
        if len(p) < 12 { dialog.ShowError(fmt.Errorf("För kort lösenord"), w); return }
        if p != c { dialog.ShowError(fmt.Errorf("Lösenorden matchar inte"), w); return }
        hash, _ := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
        if _, err := repo.CreateUser(context.Background(), e, string(hash)); err != nil {
            dialog.ShowError(err, w); return
        }
        dialog.ShowInformation("Klart", "Användare skapad, logga in nu.", w)
    }, w)
}

func showAddDialog(w fyne.Window, repo *matches.Repository, after func()) {
    date := widget.NewEntry(); date.SetPlaceHolder("YYYY‑MM‑DD")
    tm := widget.NewEntry(); tm.SetPlaceHolder("HH:MM")
    league := widget.NewEntry()
    team := widget.NewEntry()
    opp := widget.NewEntry()
    venue := widget.NewEntry()
    city := widget.NewEntry()
    form := &widget.Form{Items: []*widget.FormItem{
        {Text: "Datum", Widget: date},
        {Text: "Tid", Widget: tm},
        {Text: "Serie", Widget: league},
        {Text: "Lag", Widget: team},
        {Text: "Motstånd", Widget: opp},
        {Text: "Hall", Widget: venue},
        {Text: "Stad", Widget: city},
    }}
    dialog.ShowCustomConfirm("Ny match", "Spara", "Avbryt", form, func(ok bool) {
        if !ok { return }
        m := matches.Match{ DateRaw: date.Text, TimeRaw: tm.Text, League: league.Text, Team: team.Text, Opponent: opp.Text, Venue: venue.Text, City: city.Text }
        if _, err := repo.Create(context.Background(), m); err != nil { dialog.ShowError(err, w); return }
        after()
    }, w)
}

func showImportDialog(w fyne.Window, repo *matches.Repository, after func()) {
    fd := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
        if err != nil || rc == nil { return }
        path := rc.URI().Path()
        _ = rc.Close()
        // Ask for our-team
        our := widget.NewEntry()
        our.SetPlaceHolder("Vårt lag (valfritt)")
        dialog.ShowForm("Importera", "Importera", "Avbryt", []*widget.FormItem{{Text: "Vårt lag", Widget: our}}, func(ok bool) {
            if !ok { return }
            list, err := matches.ParseFile(path, our.Text)
            if err != nil { dialog.ShowError(err, w); return }
            ctx := context.Background()
            n := 0
            for _, m := range list {
                if _, err := repo.Create(ctx, m); err != nil { dialog.ShowError(fmt.Errorf("Rad %d: %v", n+1, err), w); return }
                n++
            }
            dialog.ShowInformation("Klart", fmt.Sprintf("Importerade %d matcher", n), w)
            after()
        }, w)
    }, w)
    fd.SetFilter(storageFilter{exts: []string{".csv", ".xlsx"}})
    fd.Show()
}

// Simple file filter for extensions
type storageFilter struct{ exts []string }

func (f storageFilter) Matches(u fyne.URI) bool {
    ext := strings.ToLower(filepath.Ext(u.Name()))
    for _, e := range f.exts { if e == ext { return true } }
    return false
}

func openDB() (*sql.DB, error) {
    dsn := env("DB_PATH", "xmatches.db")
    if !strings.Contains(strings.ToLower(dsn), "_pragma=foreign_keys(1)") {
        if strings.Contains(dsn, "?") { dsn += "&_pragma=foreign_keys(1)" } else { dsn += "?_pragma=foreign_keys(1)" }
    }
    db, err := sql.Open("sqlite", dsn)
    if err != nil { return nil, err }
    if err := db.Ping(); err != nil { _ = db.Close(); return nil, err }
    return db, nil
}

func env(k, def string) string { if v := os.Getenv(k); v != "" { return v }; return def }
func sval(p *string) string { if p == nil { return "" }; return *p }
func i64(p *int64) int64 { if p == nil { return 0 }; return *p }
