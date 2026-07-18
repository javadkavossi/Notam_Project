// Package reference دسترسی به دادهٔ مرجع هوانوردی (فرودگاه/باند/ناوید/FIR) را فراهم می‌کند (E7).
package reference

import (
	"strings"

	"github.com/hossein-repo/BaseProject/data/db"
	"github.com/hossein-repo/BaseProject/data/db/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store دسترسی به دادهٔ مرجع.
type Store struct {
	db *gorm.DB
}

func NewStore() *Store { return &Store{db: db.GetDb()} }

const upsertBatch = 500

// UpsertAirports فرودگاه‌ها را بر اساس ICAO درج/به‌روزرسانی می‌کند.
// ورودی ابتدا بر اساس ICAO یکتا می‌شود: دادهٔ منبع کد تکراری دارد و Postgres در یک
// دستور ON CONFLICT اجازهٔ اثر دوباره روی یک ردیف را نمی‌دهد (SQLSTATE 21000).
func (s *Store) UpsertAirports(airports []model.Airport) error {
	airports = dedupeAirports(airports)
	if len(airports) == 0 {
		return nil
	}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "icao"}},
		DoUpdates: clause.AssignmentColumns([]string{"iata", "name", "country", "municipality", "type", "lat", "lon", "elevation_ft", "updated_at"}),
	}).CreateInBatches(&airports, upsertBatch).Error
}

// dedupeAirports آخرین رکورد برای هر ICAO را نگه می‌دارد (ترتیب ورودی حفظ می‌شود).
func dedupeAirports(in []model.Airport) []model.Airport {
	seen := make(map[string]int, len(in))
	out := make([]model.Airport, 0, len(in))
	for _, a := range in {
		key := strings.ToUpper(strings.TrimSpace(a.ICAO))
		if key == "" {
			continue
		}
		a.ICAO = key
		if i, ok := seen[key]; ok {
			out[i] = a // جایگزینی با آخرین رکورد
			continue
		}
		seen[key] = len(out)
		out = append(out, a)
	}
	return out
}

// ReplaceRunways باندهای یک مجموعه را به‌سادگی جایگزین می‌کند (حذف قبلی‌ها و درج).
// چون باندها کلید طبیعی پایدار ندارند، جایگزینی کامل ساده‌تر و درست‌تر است.
func (s *Store) ReplaceRunways(runways []model.Runway) error {
	if len(runways) == 0 {
		return nil
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM runways").Error; err != nil {
			return err
		}
		return tx.CreateInBatches(&runways, upsertBatch).Error
	})
}

// ReplaceNavaids ناویدها را جایگزین می‌کند.
func (s *Store) ReplaceNavaids(navaids []model.Navaid) error {
	if len(navaids) == 0 {
		return nil
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM navaids").Error; err != nil {
			return err
		}
		return tx.CreateInBatches(&navaids, upsertBatch).Error
	})
}

// FindAirport یک فرودگاه را با کد ICAO برمی‌گرداند (nil اگر نبود).
func (s *Store) FindAirport(icao string) (*model.Airport, error) {
	var a model.Airport
	err := s.db.Where("icao = ?", strings.ToUpper(strings.TrimSpace(icao))).First(&a).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// AirportSuggestion یک ردیف پیشنهاد autocomplete.
type AirportSuggestion struct {
	ICAO    string `json:"icao"`
	IATA    string `json:"iata,omitempty"`
	Name    string `json:"name"`
	Country string `json:"country,omitempty"`
}

// SearchAirports برای autocomplete: تطبیق روی ICAO/IATA/نام. نتایج با تطابق دقیق ICAO ابتدا.
func (s *Store) SearchAirports(q string, limit int) ([]AirportSuggestion, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []AirportSuggestion{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	up := strings.ToUpper(q)
	var out []AirportSuggestion
	// اولویت: تطابق پیشوندی ICAO/IATA اول (rank=0)، سپس تطابق در نام (rank=1). همه پارامتری.
	err := s.db.Model(&model.Airport{}).
		Select("icao, iata, name, country, CASE WHEN UPPER(icao) LIKE ? OR UPPER(iata) LIKE ? THEN 0 ELSE 1 END AS rank", up+"%", up+"%").
		Where("UPPER(icao) LIKE ? OR UPPER(iata) LIKE ? OR UPPER(name) LIKE ?", up+"%", up+"%", "%"+up+"%").
		Order("rank, icao").
		Limit(limit).
		Scan(&out).Error
	return out, err
}

// RunwaysFor باندهای یک فرودگاه را برمی‌گرداند (برای asset matching در امتیاز کانتکستی E5).
func (s *Store) RunwaysFor(icao string) ([]model.Runway, error) {
	var rws []model.Runway
	err := s.db.Where("airport_icao = ?", strings.ToUpper(strings.TrimSpace(icao))).Find(&rws).Error
	return rws, err
}

// RunwayCounts تعداد باندهای فعال (غیربسته) هر فرودگاه از فهرست داده‌شده را برمی‌گرداند (E5.5).
// برای فاکتور بستر «تعداد باند»: بستن باند در فرودگاه تک‌باند بحرانی‌تر از چندباند است.
func (s *Store) RunwayCounts(icaos []string) (map[string]int, error) {
	out := make(map[string]int, len(icaos))
	if len(icaos) == 0 {
		return out, nil
	}
	up := make([]string, 0, len(icaos))
	for _, c := range icaos {
		if v := strings.ToUpper(strings.TrimSpace(c)); v != "" {
			up = append(up, v)
		}
	}
	type row struct {
		AirportICAO string
		Cnt         int
	}
	var rows []row
	err := s.db.Model(&model.Runway{}).
		Select("airport_icao, COUNT(*) AS cnt").
		Where("airport_icao IN ? AND closed = ?", up, false).
		Group("airport_icao").
		Scan(&rows).Error
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out[r.AirportICAO] = r.Cnt
	}
	return out, nil
}
