package matches

import (
    "encoding/csv"
    "fmt"
    "net/http"
    "strconv"
    "time"
    "strings"

    "github.com/gin-gonic/gin"
    dbpkg "github.com/xaitan80/X-Matches/internal/db"
)

// ----- Helpers för mapping -----

func sval(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ival(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func bval(p *int64) bool {
	return p != nil && *p != 0
}

func toAPI(m dbpkg.Match) Match {
    return Match{
        ID:           m.ID,
        StartISO:     m.StartIso,
        EndISO:       m.EndIso,
        DateRaw:      sval(m.DateRaw),
        TimeRaw:      sval(m.TimeRaw),
        EndTimeRaw:   sval(m.EndTimeRaw),
        Weekday:      sval(m.Weekday),
        League:       sval(m.League),
        Team:         sval(m.Team),
        Opponent:     sval(m.Opponent),
        HomeTeam:     sval(m.HomeTeam),
        AwayTeam:     sval(m.AwayTeam),
        Venue:        sval(m.Venue),
        Court:        sval(m.Court),
        City:         sval(m.City),
        MatchNumber:  sval(m.MatchNumber),
        Referees:     sval(m.Referees),
        Notes:        sval(m.Notes),
        Played:       bval(m.Played),
        GoalsFor:     ival(m.GoalsFor),
        GoalsAgainst: ival(m.GoalsAgainst),
        PlayerNotes:  sval(m.PlayerNotes),
        TopScorerTeam: sval(m.TopScorerTeam),
        TopScorerOpponent: sval(m.TopScorerOpponent),
    }
}

func toAPIList(list []dbpkg.Match) []Match {
	out := make([]Match, 0, len(list))
	for _, m := range list {
		out = append(out, toAPI(m))
	}
	return out
}

// ----- Request payload -----

type createOrUpdateReq struct {
    DateRaw      *string `json:"date_raw"`
    TimeRaw      *string `json:"time_raw"`
    EndTimeRaw   *string `json:"end_time_raw"`
    Weekday      *string `json:"weekday"`
    League       *string `json:"league"`
    Team         *string `json:"team"`
    Opponent     *string `json:"opponent"`
    HomeTeam     *string `json:"home_team"`
    AwayTeam     *string `json:"away_team"`
    Venue        *string `json:"venue"`
    Court        *string `json:"court"`
    City         *string `json:"city"`
    MatchNumber  *string `json:"match_number"`
    Referees     *string `json:"referees"`
    Notes        *string `json:"notes"`
    Played       *bool   `json:"played"`
    GoalsFor     *int64  `json:"goals_for"`
    GoalsAgainst *int64  `json:"goals_against"`
    PlayerNotes  *string `json:"player_notes"`
    StartISO     *string `json:"start_iso"`
    EndISO       *string `json:"end_iso"`
    TopScorerTeam *string `json:"top_scorer_team"`
    TopScorerOpponent *string `json:"top_scorer_opponent"`
}

func toDomain(req createOrUpdateReq) Match {
	val := func(p *string) string {
		if p != nil {
			return *p
		}
		return ""
	}
	pb := func(p *bool) bool {
		if p != nil {
			return *p
		}
		return false
	}
	pi := func(p *int64) int64 {
		if p != nil {
			return *p
		}
		return 0
	}
    return Match{
        StartISO:     req.StartISO,
        EndISO:       req.EndISO,
        DateRaw:      val(req.DateRaw),
        TimeRaw:      val(req.TimeRaw),
        EndTimeRaw:   val(req.EndTimeRaw),
        Weekday:      val(req.Weekday),
        League:       val(req.League),
        Team:         val(req.Team),
        Opponent:     val(req.Opponent),
        HomeTeam:     val(req.HomeTeam),
        AwayTeam:     val(req.AwayTeam),
        Venue:        val(req.Venue),
        Court:        val(req.Court),
        City:         val(req.City),
        MatchNumber:  val(req.MatchNumber),
        Referees:     val(req.Referees),
        Notes:        val(req.Notes),
        Played:       pb(req.Played),
        GoalsFor:     pi(req.GoalsFor),
        GoalsAgainst: pi(req.GoalsAgainst),
        PlayerNotes:  val(req.PlayerNotes),
        TopScorerTeam: val(req.TopScorerTeam),
        TopScorerOpponent: val(req.TopScorerOpponent),
    }
}

// ----- Routes -----

func RegisterRoutes(r *gin.Engine, repo *Repository, protect gin.HandlerFunc) {
    api := r.Group("/api")
    {
        // Import matches from CSV or XLSX (protected)
        api.POST("/matches/import", attachProtect(protect, func(c *gin.Context) {
            if err := c.Request.ParseMultipartForm(12 << 20); err != nil { // 12MB
                c.JSON(http.StatusBadRequest, gin.H{"error":"multipart too large"}); return
            }
            fh, err := c.FormFile("file")
            if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"missing file"}); return }

            ourTeam := c.Query("our_team")

            rows, err := parseImport(fh, ourTeam)
            if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }

            imported := 0
            var errs []string
            for idx, m := range rows {
                if _, err := repo.Create(c.Request.Context(), m); err != nil {
                    errs = append(errs, fmt.Sprintf("row %d: %v", idx+2, err))
                } else {
                    imported++
                }
            }
            c.JSON(http.StatusOK, gin.H{"imported": imported, "failed": len(errs), "errors": errs})
        }))

        // Delete all matches (dangerous)
        api.DELETE("/matches", attachProtect(protect, func(c *gin.Context) {
            n, err := repo.DeleteAll(c.Request.Context())
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }
            c.JSON(http.StatusOK, gin.H{"deleted": n})
        }))

        // iCal export of all matches
        api.GET("/matches.ics", func(c *gin.Context) {
            list, err := repo.List(c.Request.Context())
            if err != nil {
                c.String(http.StatusInternalServerError, err.Error())
                return
            }

            c.Header("Content-Type", "text/calendar; charset=utf-8")
            c.Header("Content-Disposition", "attachment; filename=matches.ics")

            w := c.Writer
            fmt.Fprintln(w, "BEGIN:VCALENDAR")
            fmt.Fprintln(w, "VERSION:2.0")
            fmt.Fprintln(w, "PRODID:-//x-matches//EN")
            fmt.Fprintln(w, "CALSCALE:GREGORIAN")

            now := time.Now().UTC().Format("20060102T150405Z")
            loc, _ := time.LoadLocation("Europe/Stockholm")

            for _, m := range list {
                // Compute start/end times
                var dtStart, dtEnd string
                // Prefer ISO fields
                if m.StartIso != nil && *m.StartIso != "" {
                    if t, err := time.Parse(time.RFC3339, *m.StartIso); err == nil {
                        dtStart = t.UTC().Format("20060102T150405Z")
                    }
                }
                if m.EndIso != nil && *m.EndIso != "" {
                    if t, err := time.Parse(time.RFC3339, *m.EndIso); err == nil {
                        dtEnd = t.UTC().Format("20060102T150405Z")
                    }
                }
                // Fallback from raw date/time in local timezone
                if dtStart == "" && m.DateRaw != nil {
                    tr := "00:00"
                    if m.TimeRaw != nil && *m.TimeRaw != "" { tr = *m.TimeRaw }
                    if t, err := time.ParseInLocation("2006-01-02 15:04", *m.DateRaw+" "+tr, loc); err == nil {
                        dtStart = t.UTC().Format("20060102T150405Z")
                    }
                }
                if dtEnd == "" && m.DateRaw != nil && m.EndTimeRaw != nil && *m.EndTimeRaw != "" {
                    if t, err := time.ParseInLocation("2006-01-02 15:04", *m.DateRaw+" "+*m.EndTimeRaw, loc); err == nil {
                        dtEnd = t.UTC().Format("20060102T150405Z")
                    }
                }

                // Summary and location
                team := sval(m.Team)
                opp := sval(m.Opponent)
                home := sval(m.HomeTeam)
                away := sval(m.AwayTeam)
                var summary string
                if home != "" || away != "" {
                    summary = fmt.Sprintf("%s vs %s", home, away)
                } else if team != "" || opp != "" {
                    summary = fmt.Sprintf("%s – %s", team, opp)
                } else {
                    summary = "Match"
                }
                locStr := sval(m.Venue)
                if city := sval(m.City); city != "" {
                    if locStr != "" { locStr += ", " }
                    locStr += city
                }

                fmt.Fprintln(w, "BEGIN:VEVENT")
                fmt.Fprintf(w, "UID:match-%d@x-matches\n", m.ID)
                fmt.Fprintf(w, "DTSTAMP:%s\n", now)
                if dtStart != "" { fmt.Fprintf(w, "DTSTART:%s\n", dtStart) }
                if dtEnd != "" { fmt.Fprintf(w, "DTEND:%s\n", dtEnd) }
                // Escape commas and semicolons per RFC
                esc := func(s string) string { return strings.NewReplacer(",","\\,",";","\\;","\n","\\n").Replace(s) }
                fmt.Fprintf(w, "SUMMARY:%s\n", esc(summary))
                if locStr != "" { fmt.Fprintf(w, "LOCATION:%s\n", esc(locStr)) }
                if n := sval(m.Notes); n != "" { fmt.Fprintf(w, "DESCRIPTION:%s\n", esc(n)) }
                fmt.Fprintln(w, "END:VEVENT")
            }

            fmt.Fprintln(w, "END:VCALENDAR")
        })

        // CSV export of all matches
        api.GET("/matches.csv", func(c *gin.Context) {
            list, err := repo.List(c.Request.Context())
            if err != nil {
                c.String(http.StatusInternalServerError, err.Error())
                return
            }

            filename := fmt.Sprintf("matches_%s.csv", time.Now().Format("2006-01-02"))
            c.Header("Content-Type", "text/csv; charset=utf-8")
            c.Header("Content-Disposition", "attachment; filename="+filename)

            w := csv.NewWriter(c.Writer)
            // Header
            _ = w.Write([]string{
                "id",
                "date_raw","time_raw","end_time_raw","weekday",
                "league","team","opponent","home_team","away_team",
                "venue","court","city",
                "match_number","referees","notes",
                "played","goals_for","goals_against","player_notes",
                "top_scorer_team","top_scorer_opponent",
                "start_iso","end_iso",
            })
            // Rows
            for _, m := range list {
                _ = w.Write([]string{
                    strconv.FormatInt(m.ID, 10),
                    sval(m.DateRaw), sval(m.TimeRaw), sval(m.EndTimeRaw), sval(m.Weekday),
                    sval(m.League), sval(m.Team), sval(m.Opponent), sval(m.HomeTeam), sval(m.AwayTeam),
                    sval(m.Venue), sval(m.Court), sval(m.City),
                    sval(m.MatchNumber), sval(m.Referees), sval(m.Notes),
                    strconv.FormatBool(bval(m.Played)),
                    strconv.FormatInt(ival(m.GoalsFor), 10),
                    strconv.FormatInt(ival(m.GoalsAgainst), 10),
                    sval(m.PlayerNotes),
                    sval(m.TopScorerTeam), sval(m.TopScorerOpponent),
                    sval(m.StartIso), sval(m.EndIso),
                })
            }
            w.Flush()
            if err := w.Error(); err != nil {
                c.String(http.StatusInternalServerError, err.Error())
                return
            }
        })

        api.GET("/matches", func(c *gin.Context) {
            list, err := repo.List(c.Request.Context())
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }
            c.JSON(http.StatusOK, toAPIList(list))
        })

		api.GET("/matches/:id", func(c *gin.Context) {
			id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			m, err := repo.Get(c.Request.Context(), id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusOK, toAPI(m))
		})

		api.POST("/matches", attachProtect(protect, func(c *gin.Context) {
			var req createOrUpdateReq
			if err := c.BindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
				return
			}
			row, err := repo.Create(c.Request.Context(), toDomain(req))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, toAPI(row))
		}))

		api.PATCH("/matches/:id", attachProtect(protect, func(c *gin.Context) {
			id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			var req createOrUpdateReq
			if err := c.BindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
				return
			}
			row, err := repo.Update(c.Request.Context(), id, toDomain(req))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, toAPI(row))
		}))

		api.DELETE("/matches/:id", attachProtect(protect, func(c *gin.Context) {
			id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if err := repo.Delete(c.Request.Context(), id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.Status(http.StatusNoContent)
		}))
    }
}

// attachProtect conditionally wraps handlers with the given protect middleware for mutating routes.
// We keep read routes public.
func attachProtect(protect gin.HandlerFunc, h gin.HandlerFunc) gin.HandlerFunc {
    if protect == nil { return h }
    return func(c *gin.Context) {
        protect(c)
        if c.IsAborted() { return }
        h(c)
    }
}
