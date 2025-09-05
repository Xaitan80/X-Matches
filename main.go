package main

import (
    "database/sql"
    "embed"
    "log"
    "net/http"
    "os"

    _ "time/tzdata"

    "github.com/gin-gonic/gin"
    _ "modernc.org/sqlite"

	dbpkg "github.com/xaitan80/X-Matches/internal/db"
	"github.com/xaitan80/X-Matches/internal/matches"
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

	// API
	matches.RegisterRoutes(r, repo)

	// Enkel frontend
	r.GET("/", func(c *gin.Context) {
		f, err := webFS.ReadFile("web/index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "missing index")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", f)
	})

	addr := env("ADDR", ":8080")
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
