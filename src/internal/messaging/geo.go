package messaging

import (
	"regexp"
	"strconv"
	"strings"
)

// geo.go — استخراج هندسه و ارتفاع فضای هوایی از فیلدهای ساختاریافتهٔ خط Q (E5.6).
// هیچ مقداری جعل نمی‌شود: اگر فیلد نبود یا نامعتبر بود، ok=false برمی‌گردد.

// FLUnlimitedFt سقفِ «نامحدود» (FL999) بر حسب فوت.
const FLUnlimitedFt = 99900

var coordRe = regexp.MustCompile(`^(\d{2})(\d{2})([NS])(\d{3})(\d{2})([EW])$`)

// ParseCoord مرکز فضای هوایی به فرمت DDMM[N/S]DDDMM[E/W] (مثل 5609N04020E) را به درجه تبدیل می‌کند.
func ParseCoord(s string) (lat, lon float64, ok bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	m := coordRe.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, false
	}
	latDeg, _ := strconv.Atoi(m[1])
	latMin, _ := strconv.Atoi(m[2])
	lonDeg, _ := strconv.Atoi(m[4])
	lonMin, _ := strconv.Atoi(m[5])
	lat = float64(latDeg) + float64(latMin)/60.0
	lon = float64(lonDeg) + float64(lonMin)/60.0
	if m[3] == "S" {
		lat = -lat
	}
	if m[6] == "W" {
		lon = -lon
	}
	// اعتبارسنجی محدوده
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return 0, 0, false
	}
	return lat, lon, true
}

// ParseRadiusNM شعاع را بر حسب مایل دریایی برمی‌گرداند (مثل "002" → 2).
func ParseRadiusNM(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, false
	}
	return float64(n), true
}

// ParseFLtoFt سطح پروازی (FL) را به فوت تبدیل می‌کند (FL100 → 10000، FL999 → نامحدود).
func ParseFLtoFt(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, false
	}
	if n >= 999 {
		return FLUnlimitedFt, true
	}
	return n * 100, true
}

// NotamGeo هندسه و ارتفاع استخراج‌شدهٔ یک NOTAM.
type NotamGeo struct {
	Lat, Lon, RadiusNM float64
	HasArea            bool
	LowerFt, UpperFt   int
	HasVertical        bool
}

// ExtractGeo هندسه و ارتفاع را از رویداد استخراج می‌کند (بدون جعل).
func ExtractGeo(ev NotamEvent) NotamGeo {
	var g NotamGeo
	if lat, lon, ok := ParseCoord(ev.Coordinates); ok {
		if r, okR := ParseRadiusNM(ev.Radius); okR {
			g.Lat, g.Lon, g.RadiusNM, g.HasArea = lat, lon, r, true
		}
	}
	lo, okLo := ParseFLtoFt(ev.MinimumFL)
	hi, okHi := ParseFLtoFt(ev.MaximumFL)
	if okLo && okHi && hi >= lo {
		g.LowerFt, g.UpperFt, g.HasVertical = lo, hi, true
	}
	return g
}
