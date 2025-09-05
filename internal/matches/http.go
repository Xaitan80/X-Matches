package matches

import (
	"net/http"
	"strconv"

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
		GatherTime:   sval(m.GatherTime),
		GatherPlace:  sval(m.GatherPlace),
		MatchNumber:  sval(m.MatchNumber),
		Referees:     sval(m.Referees),
		Notes:        sval(m.Notes),
		Played:       bval(m.Played),
		GoalsFor:     ival(m.GoalsFor),
		GoalsAgainst: ival(m.GoalsAgainst),
		PlayerNotes:  sval(m.PlayerNotes),
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
	GatherTime   *string `json:"gather_time"`
	GatherPlace  *string `json:"gather_place"`
	MatchNumber  *string `json:"match_number"`
	Referees     *string `json:"referees"`
	Notes        *string `json:"notes"`
	Played       *bool   `json:"played"`
	GoalsFor     *int64  `json:"goals_for"`
	GoalsAgainst *int64  `json:"goals_against"`
	PlayerNotes  *string `json:"player_notes"`
	StartISO     *string `json:"start_iso"`
	EndISO       *string `json:"end_iso"`
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
		GatherTime:   val(req.GatherTime),
		GatherPlace:  val(req.GatherPlace),
		MatchNumber:  val(req.MatchNumber),
		Referees:     val(req.Referees),
		Notes:        val(req.Notes),
		Played:       pb(req.Played),
		GoalsFor:     pi(req.GoalsFor),
		GoalsAgainst: pi(req.GoalsAgainst),
		PlayerNotes:  val(req.PlayerNotes),
	}
}

// ----- Routes -----

func RegisterRoutes(r *gin.Engine, repo *Repository) {
	api := r.Group("/api")
	{
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

		api.POST("/matches", func(c *gin.Context) {
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
		})

		api.PATCH("/matches/:id", func(c *gin.Context) {
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
		})

		api.DELETE("/matches/:id", func(c *gin.Context) {
			id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
			if err := repo.Delete(c.Request.Context(), id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.Status(http.StatusNoContent)
		})
	}
}
