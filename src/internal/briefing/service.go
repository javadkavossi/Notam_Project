package briefing

import (
	"github.com/hossein-repo/BaseProject/data/db"
	"github.com/hossein-repo/BaseProject/data/db/model"
	"gorm.io/gorm"
)

// Service ساخت بریفینگ از دیتابیس.
type Service struct {
	db *gorm.DB
}

func NewService() *Service { return &Service{db: db.GetDb()} }

// maxNotamsPerBriefing سقف ایمنی برای جلوگیری از پاسخ‌های عظیم.
const maxNotamsPerBriefing = 500

// Build بریفینگ یک پرواز را می‌سازد: تطبیق مکانی + زمانی، سپس امتیازدهی و رتبه‌بندی.
func (s *Service) Build(fp model.FlightPlan) (Briefing, error) {
	notams, err := s.matchNotams(fp)
	if err != nil {
		return Briefing{}, err
	}
	return Build(fp, notams), nil
}

// matchNotams تطبیق مکانی (فرودگاه‌های پرواز یا FIRهای مسیر) و زمانی (هم‌پوشانی با پنجرهٔ پرواز).
func (s *Service) matchNotams(fp model.FlightPlan) ([]model.Notam, error) {
	airports := upAll(fp.Airports())
	firs := upAll(fp.EnrouteFIRs)
	if len(airports) == 0 && len(firs) == 0 {
		return nil, nil
	}
	from, to := fp.Window()

	q := s.db.Model(&model.Notam{})

	// ---- تطبیق مکانی (E5-2): فرودگاه‌های پرواز یا FIRهای مسیر ----
	loc := s.db.Session(&gorm.Session{NewDB: true})
	switch {
	case len(airports) > 0 && len(firs) > 0:
		loc = loc.Where("UPPER(location_icao) IN ? OR UPPER(airport_icao) IN ?", airports, airports).
			Or("UPPER(affected_fir) IN ?", firs)
	case len(airports) > 0:
		loc = loc.Where("UPPER(location_icao) IN ? OR UPPER(airport_icao) IN ?", airports, airports)
	default:
		loc = loc.Where("UPPER(affected_fir) IN ?", firs)
	}
	q = q.Where(loc)

	// ---- فیلتر زمانی (E5-4): اعتبار NOTAM باید با پنجرهٔ پرواز هم‌پوشانی داشته باشد ----
	// شروعِ NOTAM قبل از پایان پنجره  و  پایانِ NOTAM (یا بی‌پایان) بعد از شروع پنجره
	q = q.Where("effective_start <= ?", to).
		Where("effective_end IS NULL OR effective_end >= ?", from)

	// NOTAMهای کنسل‌کننده (NOTAMC) خودشان اطلاعات عملیاتی ندارند؛ از بریفینگ حذف می‌شوند.
	q = q.Where("event_type <> ?", "C")

	var notams []model.Notam
	err := q.Order("base_score DESC, effective_start DESC").
		Limit(maxNotamsPerBriefing).
		Find(&notams).Error
	return notams, err
}

// FindFlight یک پرواز کاربر را برمی‌گرداند (nil اگر نبود/متعلق به کاربر دیگر).
func (s *Service) FindFlight(id int, username string) (*model.FlightPlan, error) {
	var fp model.FlightPlan
	err := s.db.Where("id = ? AND username = ?", id, username).First(&fp).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &fp, nil
}

// CreateFlight یک پرواز جدید ذخیره می‌کند.
func (s *Service) CreateFlight(fp *model.FlightPlan) error {
	return s.db.Create(fp).Error
}

// ListFlights پروازهای کاربر (جدیدترین اول).
func (s *Service) ListFlights(username string, limit int) ([]model.FlightPlan, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var out []model.FlightPlan
	err := s.db.Where("username = ?", username).Order("id DESC").Limit(limit).Find(&out).Error
	return out, err
}
