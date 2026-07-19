package briefing

import (
	"fmt"
	"strings"

	"github.com/hossein-repo/BaseProject/data/db"
	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/pipeline/analysis"
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
	ctx.Route = s.buildRoute(fp, notams)
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

// buildRoute مسیر پرواز را می‌سازد: اولویت با waypointهای واقعی؛ سپس دایره‌الوسط مبدأ→مقصد؛ وگرنه UNKNOWN.
// سپس بازهٔ ارتفاعی هر segment و segmentهای متقاطع با هر NOTAM را تعیین می‌کند.
func (s *Service) buildRoute(fp model.FlightPlan, notams []model.Notam) RouteContext {
	legs, source, confidence := s.routeLegs(fp)
	rc := RouteContext{Source: source, Confidence: confidence}
	if source == RouteSourceUnknown || len(legs) == 0 {
		return rc
	}

	// بازهٔ ارتفاعی هر segment: از پروفایل، سپس ارتفاع سِیر، وگرنه نامعلوم.
	rc.Segments = make([]SegmentBand, len(legs))
	for i, lg := range legs {
		rc.Segments[i] = segmentBand(lg.fromSeq, lg.toSeq, fp)
	}

	// تقاطع افقی per-segment (یک کوئری PostGIS).
	rc.NotamSegments = s.segmentIntersections(legs, notams)
	return rc
}

// routeLegs بخش‌های هندسیِ مسیر + منبع + confidence افقی را برمی‌گرداند.
func (s *Service) routeLegs(fp model.FlightPlan) ([]routeLeg, string, string) {
	// ۱) waypointهای معتبر
	if legs := legsFromWaypoints(fp.RouteWaypoints); len(legs) > 0 {
		return legs, RouteSourceWaypoints, ConfHigh
	}
	// ۲) دایره‌الوسط مبدأ→مقصد (fallback)
	adep, _ := s.ref.FindAirport(fp.ADEP)
	ades, _ := s.ref.FindAirport(fp.ADES)
	if adep != nil && ades != nil && !(adep.Lat == 0 && adep.Lon == 0) && !(ades.Lat == 0 && ades.Lon == 0) {
		return []routeLeg{{fromSeq: 0, toSeq: 1, lon1: adep.Lon, lat1: adep.Lat, lon2: ades.Lon, lat2: ades.Lat}},
			RouteSourceGreatCircle, ConfMedium
	}
	// ۳) نامعلوم
	return nil, RouteSourceUnknown, ConfLow
}

// segmentBand بازهٔ ارتفاعی یک segment را از پروفایل (اولویت) یا ارتفاع سِیر تعیین می‌کند.
func segmentBand(fromSeq, toSeq int, fp model.FlightPlan) SegmentBand {
	b := SegmentBand{FromSeq: fromSeq, ToSeq: toSeq, Phase: model.PhaseUnknown}
	for _, p := range fp.RouteAltitudeProfile {
		if p.FromSequence <= fromSeq && p.ToSequence >= toSeq {
			b.LowerFt, b.UpperFt, b.AltKnown = p.LowerFt, p.UpperFt, true
			b.AltSource = AltSourceSegmentProfile
			if p.Phase != "" {
				b.Phase = p.Phase
			}
			return b
		}
	}
	if fp.CruiseAltitudeFt > 0 {
		b.LowerFt, b.UpperFt, b.AltKnown = fp.CruiseAltitudeFt, fp.CruiseAltitudeFt, true
		b.AltSource = AltSourceCruiseFixed
		b.Phase = model.PhaseCruise
		return b
	}
	b.AltSource = AltSourceNone
	return b
}

// segmentIntersections با یک کوئری PostGIS تعیین می‌کند هر NOTAM با کدام segmentها تداخل افقی دارد.
func (s *Service) segmentIntersections(legs []routeLeg, notams []model.Notam) map[int][]int {
	out := map[int][]int{}
	ids := make([]int, 0, len(notams))
	for _, n := range notams {
		if n.AreaRadiusNM > 0 {
			ids = append(ids, n.Id)
		}
	}
	if len(ids) == 0 || len(legs) == 0 {
		return out
	}
	// VALUES از مختصاتِ کنترل‌شده (float) ساخته می‌شود؛ بدون رشتهٔ کاربر → بدون ریسک injection.
	vals := make([]string, len(legs))
	for i, lg := range legs {
		vals[i] = fmt.Sprintf("(%d,%g::float8,%g::float8,%g::float8,%g::float8)", i, lg.lon1, lg.lat1, lg.lon2, lg.lat2)
	}
	corridorMeters := analysis.Current.RouteCorridorNM * 1852.0
	q := `WITH seg(idx,lon1,lat1,lon2,lat2) AS (VALUES ` + strings.Join(vals, ",") + `)
	      SELECT n.id AS notam_id, seg.idx AS seg_idx
	      FROM notams n JOIN seg
	        ON ST_Intersects(n.area,
	           ST_Buffer(ST_SetSRID(ST_MakeLine(ST_MakePoint(seg.lon1,seg.lat1),ST_MakePoint(seg.lon2,seg.lat2)),4326)::geography, ?)::geometry)
	      WHERE n.id IN ? AND n.area IS NOT NULL`
	type row struct {
		NotamID int
		SegIdx  int
	}
	var rows []row
	if err := s.db.Raw(q, corridorMeters, ids).Scan(&rows).Error; err != nil {
		return out // خطای فضایی → بدون تقاطع (assessAirspace آن را UNKNOWN می‌بیند اگر مسیر خالی شود)
	}
	for _, r := range rows {
		out[r.NotamID] = append(out[r.NotamID], r.SegIdx)
	}
	return out
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
