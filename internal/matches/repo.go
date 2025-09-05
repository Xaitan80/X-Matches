package matches

import (
	"time"

	"gorm.io/gorm"
)

type Repo struct{ db *gorm.DB }

func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) Create(m *Match) error { return r.db.Create(m).Error }
func (r *Repo) Upsert(m *Match) error { return r.db.Save(m).Error }
func (r *Repo) Delete(id uint) error  { return r.db.Delete(&Match{}, id).Error }
func (r *Repo) Get(id uint) (*Match, error) {
	var m Match
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}
func (r *Repo) List() ([]Match, error) {
	var out []Match
	err := r.db.Order("start_iso NULLS LAST, id").Find(&out).Error
	return out, err
}

// ParseLocal bygger en tid i Europe/Stockholm från date_raw + time_raw
func ParseLocal(dt, tm string) (*time.Time, error) {
	if dt == "" && tm == "" {
		return nil, nil
	}
	loc, _ := time.LoadLocation("Europe/Stockholm")
	layout := "2006-01-02 15:04"
	if tm == "" {
		tm = "00:00"
	}
	t, err := time.ParseInLocation(layout, dt+" "+tm, loc)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
