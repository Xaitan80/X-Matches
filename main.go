package main

import (
    "database/sql"
    "embed"
    "log"
    "net/http"
    "os"
    "strings"

    _ "time/tzdata"

    "github.com/gin-gonic/gin"
    _ "modernc.org/sqlite"

	dbpkg "github.com/xaitan80/X-Matches/internal/db"
	"github.com/xaitan80/X-Matches/internal/matches"
    "github.com/xaitan80/X-Matches/internal/auth"
)

//go:embed web/*
var webFS embed.FS

func main() {
	dsn := env("DB_PATH", "xmatches.db")

    // Öppna DB (modernc driver name: "sqlite")
    sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer sqlDB.Close()

	// Migrera (goose via embed)
	if err := dbpkg.Migrate(sqlDB); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// Init sqlc-queries
	q := dbpkg.New(sqlDB)
	repo := matches.NewRepository(q)

    // HTTP
    r := gin.Default()
    // Configure explicit trusted proxies to avoid gin's trust-all warning
    // Default trusts only loopback addresses; override via TRUSTED_PROXIES env (comma-separated CIDRs/IPs)
    tp := strings.Split(env("TRUSTED_PROXIES", "127.0.0.1,::1"), ",")
    for i := range tp {
        tp[i] = strings.TrimSpace(tp[i])
    }
    if err := r.SetTrustedProxies(tp); err != nil {
        log.Fatalf("trusted proxies: %v", err)
    }

    // API
    auth.RegisterRoutes(r, sqlDB)
    // Auth-aware frontend routing
    authRepo := auth.NewRepository(sqlDB)
    matches.RegisterRoutes(r, repo, auth.AuthRequired(authRepo))

    // Auth-aware frontend routing

    // Redirect root based on session
    r.GET("/", func(c *gin.Context) {
        if _, ok := auth.CurrentUser(c, authRepo); ok {
            c.Redirect(http.StatusFound, "/app")
            return
        }
        c.Redirect(http.StatusFound, "/login")
    })

    // Public login page
    r.GET("/login", func(c *gin.Context) {
        f, err := webFS.ReadFile("web/login.html")
        if err != nil { c.String(http.StatusInternalServerError, "missing login"); return }
        c.Data(http.StatusOK, "text/html; charset=utf-8", f)
    })

    // Protected app page
    r.GET("/app", auth.AuthRequired(authRepo), func(c *gin.Context) {
        f, err := webFS.ReadFile("web/app.html")
        if err != nil { c.String(http.StatusInternalServerError, "missing app"); return }
        c.Data(http.StatusOK, "text/html; charset=utf-8", f)
    })

	addr := env("ADDR", ":8080")
    // Liveness/Readiness
    r.GET("/healthz", func(c *gin.Context) {
        if err := sqlDB.Ping(); err != nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"ok": true})
    })

    log.Printf("Lyssnar på %s", addr)
    if err := r.Run(addr); err != nil {
        log.Fatal(err)
    }
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
