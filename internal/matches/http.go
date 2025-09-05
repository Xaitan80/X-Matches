package matches

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type createOrUpdateReq struct {
	DateRaw     *string `json:"date_raw"` // "YYYY-MM-DD"
	TimeRaw     *string `json:"time_raw"` // "HH:MM"
	EndTimeRaw  *string `json:"end_time_raw"`
	Weekday     *string `json:"weekday"`
	League      *string `json:"league"`
	Team        *string `json:"team"`
	Opponent    *string `json:"opponent"`
	HomeTeam    *string `json:"home_team"`
	AwayTeam    *string `json:"away_team"`
	Venue       *string `json:"venue"`
	Court       *string `json:"court"`
	City        *string `json:"city"`
	GatherTime  *string `json:"gather_time"`
	GatherPlace *string `json:"gather_place"`
	MatchNumber *string `json:"match_number"`
	Referees    *string `json:"referees"`
	Notes       *string `json:"notes"`

	Played       *bool   `json:"played"`
	GoalsFor     *int    `json:"goals_for"`
	GoalsAgainst *int    `json:"goals_against"`
	PlayerNotes  *string `json:"player_notes"`
}

func applyStrings(m *Match, r createOrUpdateReq) {
	set := func(dst *string, src *string) {
		if src != nil {
			*dst = *src
		}
	}
	set(&m.DateRaw, r.DateRaw)
	set(&m.TimeRaw, r.TimeRaw)
	set(&m.EndTimeRaw, r.EndTimeRaw)
	set(&m.Weekday, r.Weekday)
	set(&m.League, r.League)
	set(&m.Team, r.Team)
	set(&m.Opponent, r.Opponent)
	set(&m.HomeTeam, r.HomeTeam)
	set(&m.AwayTeam, r.AwayTeam)
	set(&m.Venue, r.Venue)
	set(&m.Court, r.Court)
	set(&m.City, r.City)
	set(&m.GatherTime, r.GatherTime)
	set(&m.GatherPlace, r.GatherPlace)
	set(&m.MatchNumber, r.MatchNumber)
	set(&m.Referees, r.Referees)
	set(&m.Notes, r.Notes)
	set(&m.PlayerNotes, r.PlayerNotes)
	if r.Played != nil {
		m.Played = *r.Played
	}
	if r.GoalsFor != nil {
		m.GoalsFor = *r.GoalsFor
	}
	if r.GoalsAgainst != nil {
		m.GoalsAgainst = *r.GoalsAgainst
	}
}

func RegisterRoutes(r *gin.Engine, repo *Repo) {
	api := r.Group("/api")
	{
		api.GET("/matches", func(c *gin.Context) {
			list, err := repo.List()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, list)
		})

		api.GET("/matches/:id", func(c *gin.Context) {
			id, _ := strconv.Atoi(c.Param("id"))
			m, err := repo.Get(uint(id))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusOK, m)
		})

		api.POST("/matches", func(c *gin.Context) {
			var req createOrUpdateReq
			if err := c.BindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
				return
			}
			m := &Match{}
			applyStrings(m, req)

			if m.DateRaw != "" || m.TimeRaw != "" {
				if t, err := ParseLocal(m.DateRaw, m.TimeRaw); err == nil {
					m.StartISO = t
				}
			}
			if m.DateRaw != "" && m.EndTimeRaw != "" {
				if t, err := ParseLocal(m.DateRaw, m.EndTimeRaw); err == nil {
					m.EndISO = t
				}
			}

			if err := repo.Create(m); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, m)
		})

		api.PATCH("/matches/:id", func(c *gin.Context) {
			id, _ := strconv.Atoi(c.Param("id"))
			m, err := repo.Get(uint(id))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			var req createOrUpdateReq
			if err := c.BindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
				return
			}

			applyStrings(m, req)
			// Uppdatera tider om date/time ändrats
			if req.DateRaw != nil || req.TimeRaw != nil {
				if t, err := ParseLocal(m.DateRaw, m.TimeRaw); err == nil {
					m.StartISO = t
				}
			}
			if req.DateRaw != nil || req.EndTimeRaw != nil {
				if t, err := ParseLocal(m.DateRaw, m.EndTimeRaw); err == nil {
					m.EndISO = t
				}
			}

			if err := repo.Upsert(m); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, m)
		})

		api.DELETE("/matches/:id", func(c *gin.Context) {
			id, _ := strconv.Atoi(c.Param("id"))
			if err := repo.Delete(uint(id)); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.Status(http.StatusNoContent)
		})
	}
}
