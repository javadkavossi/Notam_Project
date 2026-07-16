// Package briefing موتور ساخت بریفینگ پرواز است (E5).
//
// ورودی: FlightPlan (مبدأ/مقصد/الترنت‌ها/FIRهای مسیر + پنجرهٔ زمانی).
// خروجی: NOTAMهای مرتبط، با امتیاز کانتکستیِ وابسته به پرواز، گروه‌بندی و رتبه‌بندی‌شده.
//
// منطق این فایل خالص است (روی []model.Notam کار می‌کند) تا مستقل از DB واحد‌تست شود؛
// کوئری دیتابیس در service.go است.
package briefing

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/pipeline/analysis"
	"github.com/hossein-repo/BaseProject/internal/pipeline/qcode"
)

// ItemNotam نمای NOTAM در خروجی بریفینگ.
type ItemNotam struct {
	ID             int        `json:"id"`
	SeriesNumber   string     `json:"seriesNumber"`
	LocationICAO   string     `json:"locationIcao"`
	AirportName    string     `json:"airportName,omitempty"`
	AffectedFIR    string     `json:"affectedFir,omitempty"`
	QCode          string     `json:"qcode,omitempty"`
	Category       string     `json:"category,omitempty"`
	Tags           []string   `json:"tags,omitempty"`
	EffectiveStart time.Time  `json:"effectiveStart"`
	EffectiveEnd   *time.Time `json:"effectiveEnd,omitempty"`
	PlainText      string     `json:"plainText"`
	FormattedText  string     `json:"formattedText,omitempty"`
	BaseScore      int        `json:"baseScore"`
	BaseLevel      string     `json:"baseLevel,omitempty"`
}

// Item یک ردیف بریفینگ: NOTAM + نقش آن در پرواز + امتیاز کانتکستی.
type Item struct {
	Notam           ItemNotam `json:"notam"`
	Role            string    `json:"role"`            // ADEP/ADES/ALTN/ENROUTE
	RoleICAO        string    `json:"roleIcao"`        // فرودگاه/FIR مربوطه
	ContextualScore int       `json:"contextualScore"` // امتیاز وابسته به این پرواز
	ContextualLevel string    `json:"contextualLevel"`
	MatchReason     string    `json:"matchReason"` // چرا این NOTAM انتخاب شد
}

// Group گروهی از ردیف‌ها (بر اساس نقش).
type Group struct {
	Role  string `json:"role"`
	ICAO  string `json:"icao,omitempty"`
	Items []Item `json:"items"`
}

// Briefing خروجی نهایی.
type Briefing struct {
	FlightID   int      `json:"flightId,omitempty"`
	ADEP       string   `json:"adep"`
	ADES       string   `json:"ades"`
	Alternates []string `json:"alternates,omitempty"`
	WindowFrom time.Time `json:"windowFrom"`
	WindowTo   time.Time `json:"windowTo"`

	// خلاصهٔ بحرانی: مهم‌ترین موارد یک‌جا و بالای بریفینگ
	Critical []Item `json:"critical"`

	Groups     []Group        `json:"groups"`
	TotalCount int            `json:"totalCount"`
	CountsByLevel map[string]int `json:"countsByLevel"`
	GeneratedAt time.Time     `json:"generatedAt"`
}

// دسته‌هایی که به فاز ورود/فرود مربوط‌اند.
var arrivalCategories = map[string]bool{
	qcode.CatRunway: true, qcode.CatILS: true, qcode.CatLighting: true,
	qcode.CatNavigation: true, qcode.CatAerodrome: true, qcode.CatProcedure: true,
	qcode.CatGNSS: true, qcode.CatObstacle: true,
}

// دسته‌هایی که به فاز خروج مربوط‌اند.
var departureCategories = map[string]bool{
	qcode.CatRunway: true, qcode.CatAerodrome: true, qcode.CatProcedure: true,
	qcode.CatTaxiway: true, qcode.CatLighting: true, qcode.CatObstacle: true,
}

// Build بریفینگ را از NOTAMهای از‌پیش‌فیلترشده و پرواز می‌سازد (منطق خالص).
func Build(fp model.FlightPlan, notams []model.Notam) Briefing {
	from, to := fp.Window()
	b := Briefing{
		FlightID:      fp.Id,
		ADEP:          up(fp.ADEP),
		ADES:          up(fp.ADES),
		Alternates:    upAll(fp.Alternates),
		WindowFrom:    from,
		WindowTo:      to,
		CountsByLevel: map[string]int{},
		GeneratedAt:   time.Now().UTC(),
	}

	byRole := map[string][]Item{}
	for _, n := range notams {
		role, roleICAO := classifyRole(fp, n)
		item := Item{
			Notam:    toItemNotam(n),
			Role:     role,
			RoleICAO: roleICAO,
		}
		item.ContextualScore = contextualScore(n, role)
		item.ContextualLevel = analysis.LevelFor(item.ContextualScore)
		item.MatchReason = matchReason(n, role, roleICAO)

		byRole[role] = append(byRole[role], item)
		b.CountsByLevel[item.ContextualLevel]++
		b.TotalCount++

		if item.ContextualLevel == analysis.LevelCritical {
			b.Critical = append(b.Critical, item)
		}
	}

	// ترتیب ثابت گروه‌ها: مبدأ → مقصد → الترنت → مسیر
	for _, role := range []string{model.RoleADEP, model.RoleADES, model.RoleALTN, model.RoleEnroute} {
		items := byRole[role]
		if len(items) == 0 {
			continue
		}
		sortItems(items)
		g := Group{Role: role, Items: items}
		switch role {
		case model.RoleADEP:
			g.ICAO = b.ADEP
		case model.RoleADES:
			g.ICAO = b.ADES
		}
		b.Groups = append(b.Groups, g)
	}
	sortItems(b.Critical)
	return b
}

// classifyRole تعیین می‌کند این NOTAM به کدام بخش پرواز مربوط است.
// اولویت با تطابق فرودگاه است؛ سپس FIR مسیر.
func classifyRole(fp model.FlightPlan, n model.Notam) (role, icao string) {
	loc := up(n.LocationICAO)
	apt := up(n.AirportICAO)
	match := func(target string) bool {
		t := up(target)
		return t != "" && (loc == t || apt == t)
	}
	if match(fp.ADES) {
		return model.RoleADES, up(fp.ADES)
	}
	if match(fp.ADEP) {
		return model.RoleADEP, up(fp.ADEP)
	}
	for _, alt := range fp.Alternates {
		if match(alt) {
			return model.RoleALTN, up(alt)
		}
	}
	return model.RoleEnroute, up(n.AffectedFIR)
}

// contextualScore امتیاز پایه را بر اساس نقش این NOTAM در پرواز تعدیل می‌کند (E5-5).
// اصل: یک NOTAM ممکن است برای یک پرواز بحرانی و برای پرواز دیگر کم‌اهمیت باشد.
func contextualScore(n model.Notam, role string) int {
	s := n.BaseScore
	switch role {
	case model.RoleADES:
		// مقصد: باید آنجا فرود بیاییم → مرتبط با نزدیکی/فرود مهم‌تر است
		if arrivalCategories[n.Category] {
			s += 12
		} else {
			s += 4
		}
	case model.RoleADEP:
		if departureCategories[n.Category] {
			s += 8
		} else {
			s += 3
		}
	case model.RoleALTN:
		// الترنت ثانویه است اما اگر خودِ فرودگاه بسته باشد حیاتی است
		if n.Category == qcode.CatAerodrome || n.Category == qcode.CatRunway {
			s += 6
		} else {
			s += 2
		}
	case model.RoleEnroute:
		// مسیر: فضای هوایی/ناوبری مرتبط‌تر است
		if n.Category == qcode.CatAirspace || n.Category == qcode.CatRestriction || n.Category == qcode.CatNavigation {
			s += 5
		}
	}
	return analysis.Clamp(s)
}

// matchReason توضیح انسانی اینکه چرا این NOTAM در بریفینگ آمده (شفافیت/ممیزی).
func matchReason(n model.Notam, role, icao string) string {
	switch role {
	case model.RoleADES:
		return fmt.Sprintf("مقصد %s — %s", icao, categoryFa(n.Category))
	case model.RoleADEP:
		return fmt.Sprintf("مبدأ %s — %s", icao, categoryFa(n.Category))
	case model.RoleALTN:
		return fmt.Sprintf("الترنت %s — %s", icao, categoryFa(n.Category))
	default:
		if icao != "" {
			return fmt.Sprintf("مسیر — FIR %s — %s", icao, categoryFa(n.Category))
		}
		return "مسیر — " + categoryFa(n.Category)
	}
}

// sortItems مرتب‌سازی: امتیاز کانتکستی نزولی، سپس شروع اعتبار.
func sortItems(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].ContextualScore != items[j].ContextualScore {
			return items[i].ContextualScore > items[j].ContextualScore
		}
		return items[i].Notam.EffectiveStart.After(items[j].Notam.EffectiveStart)
	})
}

func toItemNotam(n model.Notam) ItemNotam {
	return ItemNotam{
		ID:             n.Id,
		SeriesNumber:   n.SeriesNumber,
		LocationICAO:   n.LocationICAO,
		AirportName:    n.AirportName,
		AffectedFIR:    n.AffectedFIR,
		QCode:          n.QCode,
		Category:       n.Category,
		Tags:           []string(n.Tags),
		EffectiveStart: n.EffectiveStart,
		EffectiveEnd:   n.EffectiveEnd,
		PlainText:      n.PlainText,
		FormattedText:  n.FormattedText,
		BaseScore:      n.BaseScore,
		BaseLevel:      n.BaseLevel,
	}
}

// categoryFa برچسب فارسی دسته برای نمایش.
func categoryFa(c string) string {
	switch c {
	case qcode.CatAerodrome:
		return "فرودگاه"
	case qcode.CatRunway:
		return "باند"
	case qcode.CatTaxiway:
		return "تاکسی‌وی"
	case qcode.CatApron:
		return "اپرون"
	case qcode.CatLighting:
		return "روشنایی"
	case qcode.CatILS:
		return "ILS"
	case qcode.CatNavigation:
		return "ناوبری"
	case qcode.CatGNSS:
		return "GPS/GNSS"
	case qcode.CatComms:
		return "ارتباطات"
	case qcode.CatAirspace:
		return "فضای هوایی"
	case qcode.CatRestriction:
		return "محدودیت فضای هوایی"
	case qcode.CatProcedure:
		return "رویه"
	case qcode.CatWarning:
		return "هشدار"
	case qcode.CatObstacle:
		return "مانع"
	default:
		return "سایر"
	}
}

func up(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

func upAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if v := up(s); v != "" {
			out = append(out, v)
		}
	}
	return out
}
