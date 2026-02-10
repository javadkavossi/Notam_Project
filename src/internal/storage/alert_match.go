package storage

import (
	"strings"

	"github.com/hossein-repo/BaseProject/data/db/model"
)

// NotamMatchesAlertSettings بررسی تطابق NOTAM با تنظیمات اعلان کاربر.
// شرط اعلان: NOTAM باید هم در محدودهٔ FIR/فرودگاه انتخاب‌شده باشد، هم حداقل یکی از کلیدواژه‌ها در متن آن وجود داشته باشد.
// مثال: FIR ایران + فرودگاه مهرآباد (OIII) + کلیدواژه STAR → فقط وقتی اعلان که NOTAM مربوط به آن مکان باشد و در متنش STAR (یا هر کلیدواژهٔ انتخاب‌شده) باشد.
func NotamMatchesAlertSettings(n *model.Notam, s *model.NotamAlertSettings) bool {
	hasFir := len(s.SelectedFirs) > 0
	hasAirport := len(s.SelectedAirports) > 0
	hasLocationFilter := hasFir || hasAirport
	allKeywords := append(append([]string{}, s.SelectedKeywords...), s.CustomKeywords...)
	hasKeyword := len(allKeywords) > 0

	// تطابق مکان: NOTAM باید مربوط به یکی از FIRها یا فرودگاه‌های انتخاب‌شده باشد
	locationMatch := false
	if hasLocationFilter {
		if hasFir && n.AffectedFIR != "" && containsStr(s.SelectedFirs, n.AffectedFIR) {
			locationMatch = true
		}
		if !locationMatch && hasFir && n.LocationICAO != "" && containsStr(s.SelectedFirs, n.LocationICAO) {
			locationMatch = true
		}
		if !locationMatch && hasAirport && containsStr(s.SelectedAirports, n.LocationICAO) {
			locationMatch = true
		}
		if !locationMatch && hasAirport && n.AirportICAO != "" && containsStr(s.SelectedAirports, n.AirportICAO) {
			locationMatch = true
		}
	}

	// اگر کاربر مکان انتخاب نکرده، اعلان نمی‌دهیم
	if hasLocationFilter && !locationMatch {
		return false
	}
	// اگر کاربر هیچ مکان و هیچ کلیدواژه انتخاب نکرده، اعلان نمی‌دهیم
	if !hasLocationFilter && !hasKeyword {
		return false
	}
	// فقط مکان انتخاب شده (بدون کلیدواژه): با تطابق مکان کافی است
	if hasLocationFilter && !hasKeyword {
		return locationMatch
	}
	// مکان + کلیدواژه: هر دو باید مطابقت داشته باشند (مثلاً FIR ایران + مهرآباد + STAR)
	if hasLocationFilter && hasKeyword {
		text := strings.ToUpper(n.PlainText)
		for _, kw := range allKeywords {
			kw = strings.TrimSpace(kw)
			if kw == "" {
				continue
			}
			if strings.Contains(text, strings.ToUpper(kw)) {
				return true
			}
		}
		return false
	}
	return false
}

func containsStr(slice []string, v string) bool {
	for _, x := range slice {
		if x == v {
			return true
		}
	}
	return false
}
