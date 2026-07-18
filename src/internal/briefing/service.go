package briefing

import (
	"strings"

	"github.com/hossein-repo/BaseProject/data/db"
	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/reference"
	"gorm.io/gorm"
)

// Service ساخت بریفینگ از دیتابیس.
type Service struct {
	db  *gorm.DB
	ref *reference.Store
}

func NewService() *Service { return &Service{db: db.GetDb(), ref: reference.NewStore()} }

// maxNotamsPerBriefing سقف ایمنی برای جلوگیری از پاسخ‌های عظیم.
const maxNotamsPerBriefing = 500

// Build بریفینگ یک پرواز را می‌سازد: تطبیق مکانی + زمانی، سپس امتیازدهی و رتبه‌بندی.
func (s *Service) Build(fp model.FlightPlan) (Briefing, error) {
	notams, err := s.matchNotams(fp)
	if err != nil {
		return Briefing{}, err
	}
	ctx := s.flightContext(fp)
	ctx.RouteKnown, ctx.RouteIntersects = s.routeIntersections(fp, notams)
	return Build(fp, notams, ctx), nil
}

// flightContext بستر پرواز (تعداد باند + نوع هواپیما + قوانین + ارتفاع سِیر) را می‌سازد (E5.5/E5.6).
func (s *Service) flightContext(fp model.FlightPlan) FlightContext {
	counts, err := s.ref.RunwayCounts(fp.Airports())
	if err != nil || counts == nil {
		counts = map[string]int{}
	}
	rules := strings.ToUpper(strings.TrimSpace(fp.FlightRules))
	if rules == "" {
		rules = model.RulesIFR
	}
	acft := strings.ToUpper(strings.TrimSpace(fp.AircraftCategory))
	if acft == "" {
		acft = model.AircraftJet
	}
	from, to := fp.Window()
	return FlightContext{
		AircraftCategory: acft,
		FlightRules:      rules,
		RunwayCounts:     counts,
		CruiseAltitudeFt: fp.CruiseAltitudeFt,
		WindowFrom:       from,
		WindowTo:         to,
	}
}

// routeCorridorNM نصف‌عرضِ کریدور مسیر (NM). چون فعلاً مسیر تقریبِ دایره‌الوسط مبدأ→مقصد است
// (بدون waypoint واقعی)، عرضِ سخاوتمندانه انتخاب می‌شود تا تداخل واقعی از دست نرود (پرهیز از کم‌گویی).
const routeCorridorNM = 25.0

// routeIntersections برای NOTAMهای دارای هندسه تعیین می‌کند کدام‌ها با کریدور مسیر تداخل افقی دارند.
// از PostGIS و دادهٔ واقعی استفاده می‌کند؛ اگر مختصات مبدأ/مقصد نبود، routeKnown=false.
func (s *Service) routeIntersections(fp model.FlightPlan, notams []model.Notam) (bool, map[int]bool) {
	out := map[int]bool{}
	adep, _ := s.ref.FindAirport(fp.ADEP)
	ades, _ := s.ref.FindAirport(fp.ADES)
	if adep == nil || ades == nil || (adep.Lat == 0 && adep.Lon == 0) || (ades.Lat == 0 && ades.Lon == 0) {
		return false, out // مسیر نامعلوم
	}
	ids := make([]int, 0, len(notams))
	for _, n := range notams {
		if n.AreaRadiusNM > 0 {
			ids = append(ids, n.Id)
		}
	}
	if len(ids) == 0 {
		return true, out
	}
	// کریدور: بافرِ geodesic دور خط مبدأ→مقصد (geography → عبور از دایره‌الوسط).
	corridorMeters := routeCorridorNM * 1852.0
	var hitIDs []int
	err := s.db.Raw(`
		WITH corr AS (
		  SELECT ST_Buffer(
		    ST_SetSRID(ST_MakeLine(ST_MakePoint(?, ?), ST_MakePoint(?, ?)), 4326)::geography,
		    ?
		  )::geometry AS g
		)
		SELECT n.id FROM notams n, corr
		WHERE n.id IN ? AND n.area IS NOT NULL AND ST_Intersects(n.area, corr.g)`,
		adep.Lon, adep.Lat, ades.Lon, ades.Lat, corridorMeters, ids,
	).Scan(&hitIDs).Error
	if err != nil {
		// در صورت خطای فضایی، مسیر را «نامعلوم» اعلام کن (نه «بدون تداخل») تا کم‌گویی رخ ندهد.
		return false, out
	}
	for _, id := range hitIDs {
		out[id] = true
	}
	return true, out
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
