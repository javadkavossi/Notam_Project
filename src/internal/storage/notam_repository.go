package storage

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hossein-repo/BaseProject/data/db"
	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/messaging"
	"gorm.io/gorm"
)

type NotamRepository struct {
	db *gorm.DB
}

func NewNotamRepository() *NotamRepository {
	return &NotamRepository{db: db.GetDb()}
}

// Save ذخیره NOTAM دریافتی از FAA در PostgreSQL با ساختار ICAO
func (r *NotamRepository) Save(msg messaging.Message) {
	notamMsg, ok := msg.(messaging.NotamMessage)
	if !ok {
		log.Println("❌ Invalid message type for save")
		return
	}

	ev := notamMsg.Event()

	// جلوگیری از ذخیره تکراری
	var existing model.Notam
	if err := r.db.Where("message_id = ?", notamMsg.ID()).First(&existing).Error; err == nil {
		// به‌روزرسانی در صورت وجود (برای NOTAMR/NOTAMC)
		updated := r.eventToNotam(ev, notamMsg)
		updated.Id = existing.Id
		updated.MessageID = existing.MessageID
		if err := r.db.Model(&existing).Updates(updated).Error; err != nil {
			log.Printf("❌ Update NOTAM failed: %v", err)
		}
		return
	}

	n := r.eventToNotam(ev, notamMsg)
	n.MessageID = notamMsg.ID()

	if err := r.db.Create(&n).Error; err != nil {
		log.Printf("❌ Save NOTAM failed: %v", err)
		return
	}

	r.evaluateAlertDeliveries(&n)
	log.Printf("💾 Saved NOTAM ID: %s | Location: %s | Series: %s",
		n.MessageID, n.LocationICAO, n.SeriesNumber)
}

// evaluateAlertDeliveries تنظیمات اعلان همهٔ کاربران را از DB می‌خواند و در صورت تطابق NOTAM، تحویل اعلان ثبت می‌کند
func (r *NotamRepository) evaluateAlertDeliveries(n *model.Notam) {
	var settingsList []model.NotamAlertSettings
	if err := r.db.Find(&settingsList).Error; err != nil {
		log.Printf("⚠️ Failed to load alert settings: %v", err)
		return
	}
	for i := range settingsList {
		s := &settingsList[i]
		if !NotamMatchesAlertSettings(n, s) {
			continue
		}
		d := model.NotamAlertDelivery{
			Username:  s.Username,
			NotamId:   n.Id,
			CreatedAt: time.Now().UTC(),
		}
		if err := r.db.Create(&d).Error; err != nil {
			log.Printf("⚠️ Failed to create alert delivery for user %s: %v", s.Username, err)
		}
	}
}

func (r *NotamRepository) eventToNotam(ev messaging.NotamEvent, notamMsg messaging.NotamMessage) model.Notam {
	rawBody := notamMsg.Body()
	locInfo := extractLocationFromXML(rawBody)

	seriesNum := seriesNumber(ev)
	if seriesNum == "" {
		seriesNum = extractSeriesFromText(ev.HumanReadableText, ev.Text)
	}
	if seriesNum == "" {
		seriesNum = extractSeriesFromXML(rawBody, ev)
	}
	issuedAt := parseNOTAMTimePtr(ev.Issued)
	effectiveStart := parseNOTAMTime(ev.EffectiveStart)
	if effectiveStart.IsZero() && issuedAt != nil {
		effectiveStart = *issuedAt
	}
	if effectiveStart.IsZero() {
		effectiveStart = time.Now().UTC()
	}
	effectiveEnd := parseNOTAMTimePtr(ev.EffectiveEnd)

	// ICAO ۴ حرفی فرودگاه (مثل PACD, PAFA)
	airportICAO := ev.ICAOLocation
	if airportICAO == "" {
		airportICAO = locInfo.icaoLocation
	}
	if airportICAO == "" {
		airportICAO = locInfo.locationIndicatorICAO
	}

	// Location برای خط A) - کد فرودگاه یا designator
	locationICAO := airportICAO
	if locationICAO == "" {
		locationICAO = ev.Location
	}
	if locationICAO == "" {
		locationICAO = locInfo.location
	}
	if locationICAO == "" {
		locationICAO = locInfo.designator
	}
	if locationICAO == "" {
		locationICAO = ev.AffectedFIR
	}
	if locationICAO == "" {
		locationICAO = locInfo.affectedFIR
	}

	airportName := ev.AirportName
	if airportName == "" {
		airportName = locInfo.airportName
	}

	affectedFIR := ev.AffectedFIR
	if affectedFIR == "" {
		affectedFIR = locInfo.affectedFIR
	}

	plainText := ev.Text
	eventType := "N"
	if contains(ev.EventType, "R") {
		eventType = "R"
		// برای جایگزین (Replace)، آدرس NOTAM جایگزین‌شده را از متن یا XML (xovernotamID) استخراج و در plainText قرار می‌دهیم
		replacedRef := findRefNumber(ev, seriesNum)
		if replacedRef == "" {
			replacedRef = findXoverNotamFromXML(rawBody)
		}
		if replacedRef != "" {
			plainText = "REPLACES NOTAM: " + replacedRef + "\n\n" + ev.Text
		}
	} else if contains(ev.EventType, "C") {
		eventType = "C"
		// برای کنسل، در plainText هم مشخص کنیم کدام NOTAM لغو شده (برای نمایش در لیست/فرانت)
		cancelledRef := findRefNumber(ev, seriesNum)
		if cancelledRef == "" {
			cancelledRef = findXoverNotamFromXML(rawBody)
		}
		if cancelledRef != "" {
			plainText = "NOTAM CANCELLED - CANCELLED NOTAM: " + cancelledRef
		} else {
			plainText = "NOTAM CANCELLED"
		}
	}

	// Q-line استاندارد ICAO (برای رویداد کنسل طبق استاندارد نمایش داده نمی‌شود)
	qLine := ""
	if eventType != "C" && ev.AffectedFIR != "" {
		qLine = "Q) " + ev.AffectedFIR + "/QWMLW/IV/BO/W/000/999/"
	}

	// متن فرمت‌شده ICAO (برای خروجی Jeppesen)
	formatted := ev.HumanReadableText
	if formatted == "" {
		formatted = buildICAOFormattedText(ev, seriesNum, eventType, rawBody)
	}
	if eventType == "C" {
		formatted = normalizeFormattedTextForCancel(formatted, seriesNum, ev, rawBody)
	}

	return model.Notam{
		MessageID:      notamMsg.ID(),
		SeriesNumber:   seriesNum,
		EventType:      eventType,
		QLine:          qLine,
		LocationICAO:   locationICAO,
		EffectiveStart: effectiveStart,
		EffectiveEnd:   effectiveEnd,
		Schedule:       ev.Schedule,
		PlainText:      plainText,
		LowerLimit:     ev.LowerLimit,
		UpperLimit:     ev.UpperLimit,
		AirportName:    airportName,
		AffectedFIR:    affectedFIR,
		AirportICAO:    airportICAO,
		IssuedAt:       issuedAt,
		FormattedText:  formatted,
		RawBody:        rawBody,
	}
}

type locationFromXML struct {
	icaoLocation         string
	locationIndicatorICAO string
	airportName          string
	affectedFIR          string
	location             string
	designator           string
}

func extractLocationFromXML(raw string) locationFromXML {
	var loc locationFromXML
	if raw == "" {
		return loc
	}
	if m := xmlIcaoLocationRe.FindStringSubmatch(raw); len(m) >= 2 {
		loc.icaoLocation = strings.TrimSpace(m[1])
	}
	if m := xmlLocationIndicatorRe.FindStringSubmatch(raw); len(m) >= 2 {
		loc.locationIndicatorICAO = strings.TrimSpace(m[1])
	}
	if m := xmlAirportNameRe.FindStringSubmatch(raw); len(m) >= 2 {
		loc.airportName = strings.TrimSpace(m[1])
	}
	if loc.airportName == "" {
		if matches := xmlAirportName2Re.FindAllStringSubmatch(raw, 10); len(matches) > 0 {
			for _, m := range matches {
				if len(m) >= 2 {
					s := strings.TrimSpace(m[1])
					if len(s) >= 3 && s != "DESCRIPTION" && s != "REMARK" && s != "EAST RAMP" && s != "SOUTH TERMINAL RAMP" {
						loc.airportName = s
						break
					}
				}
			}
		}
	}
	if m := xmlAffectedFIRRe.FindStringSubmatch(raw); len(m) >= 2 {
		loc.affectedFIR = strings.TrimSpace(m[1])
	}
	if m := xmlLocationRe.FindStringSubmatch(raw); len(m) >= 2 {
		loc.location = strings.TrimSpace(m[1])
	}
	if m := xmlDesignatorRe.FindStringSubmatch(raw); len(m) >= 2 {
		loc.designator = strings.TrimSpace(m[1])
	}
	return loc
}

func seriesNumber(ev messaging.NotamEvent) string {
	if ev.Series == "" || ev.Year == "" {
		return ""
	}
	yr := ev.Year
	if len(yr) >= 2 {
		yr = yr[len(yr)-2:]
	}
	return ev.Series + formatNum(ev.Number) + "/" + yr
}

// الگوهای مختلف NOTAM series: A1477/26, 0046/26, M0137/26, F123/26, C1234/26
var (
	seriesRegex1 = regexp.MustCompile(`\b([A-Z]\d{3,4}/\d{2})\b`) // حرف + ۳ یا ۴ رقم
	seriesRegex2 = regexp.MustCompile(`\b(\d{4}/\d{2})\b`)        // ۴ رقم خالص
	seriesRegex3 = regexp.MustCompile(`\b([A-Z0-9]{4,6}/\d{2})\b`) // ترکیبی گسترده
	xmlSeriesRe    = regexp.MustCompile(`<[a-zA-Z:]*series[^>]*>([^<]+)</[a-zA-Z:]*series>`)
	xmlNumberRe    = regexp.MustCompile(`<event:number[^>]*>(\d+)</event:number>`) // دقیقاً event:number تا sequenceNumber اشتباه نشود
	xmlYearRe      = regexp.MustCompile(`<event:year[^>]*>(\d{4})</event:year>`)
	xmlXoverNotamRe = regexp.MustCompile(`<[a-zA-Z:]*xovernotamID[^>]*>([^<]+)</[a-zA-Z:]*xovernotamID>`)
	simpleTextRe   = regexp.MustCompile(`![A-Z0-9]+\s+(\d{2})/(\d+)`)
	// استخراج airport/ICAO از XML
	xmlIcaoLocationRe    = regexp.MustCompile(`<[a-zA-Z:]*icaoLocation[^>]*>([^<]+)</[a-zA-Z:]*icaoLocation>`)
	xmlLocationIndicatorRe = regexp.MustCompile(`<[a-zA-Z:]*locationIndicatorICAO[^>]*>([^<]+)</[a-zA-Z:]*locationIndicatorICAO>`)
	xmlAirportNameRe     = regexp.MustCompile(`<[a-zA-Z:]*airportname[^>]*>([^<]+)</[a-zA-Z:]*airportname>`)
	xmlAirportName2Re    = regexp.MustCompile(`<[a-zA-Z:]*name[^>]*>([^<]+)</[a-zA-Z:]*name>`) // aixm:name در AirportHeliport
	xmlAffectedFIRRe     = regexp.MustCompile(`<event:affectedFIR[^>]*>([^<]+)</event:affectedFIR>`)
	xmlLocationRe        = regexp.MustCompile(`<event:location[^>]*>([^<]+)</event:location>`)
	xmlDesignatorRe      = regexp.MustCompile(`<[a-zA-Z:]*designator[^>]*>([^<]+)</[a-zA-Z:]*designator>`)
)

func extractSeriesFromText(humanText, plainText string) string {
	texts := []string{humanText, plainText}
	regexes := []*regexp.Regexp{seriesRegex1, seriesRegex2, seriesRegex3}
	for _, s := range texts {
		if s == "" {
			continue
		}
		firstLine := strings.SplitN(strings.TrimSpace(s), "\n", 2)[0]
		for _, re := range regexes {
			if m := re.FindStringSubmatch(firstLine); len(m) >= 2 && m[1] != "0000/26" && m[1] != "0000/25" {
				return m[1]
			}
		}
		for _, re := range regexes {
			if m := re.FindStringSubmatch(s); len(m) >= 2 && m[1] != "0000/26" && m[1] != "0000/25" {
				return m[1]
			}
		}
	}
	return ""
}

// extractSeriesFromXML استخراج از XML خام وقتی مسیر پارس متفاوت است
func extractSeriesFromXML(rawBody string, ev messaging.NotamEvent) string {
	if rawBody == "" {
		return ""
	}
	// ۱. fnse:xovernotamID - FAA ICAO NOTAM ID مثل A4455/26
	if m := xmlXoverNotamRe.FindStringSubmatch(rawBody); len(m) >= 2 {
		s := strings.TrimSpace(m[1])
		if s != "" && strings.Contains(s, "/") {
			return s
		}
	}
	series := ""
	if m := xmlSeriesRe.FindStringSubmatch(rawBody); len(m) >= 2 {
		series = strings.TrimSpace(m[1])
	}
	number := 0
	if m := xmlNumberRe.FindStringSubmatch(rawBody); len(m) >= 2 {
		number, _ = strconv.Atoi(m[1])
	}
	year := ""
	if m := xmlYearRe.FindStringSubmatch(rawBody); len(m) >= 2 {
		year = m[1]
	}
	if year == "" && ev.Issued != "" {
		if t := parseNOTAMTimePtr(ev.Issued); t != nil {
			year = fmt.Sprintf("%d", t.Year())
		}
	}
	if series != "" || number > 0 {
		if year == "" {
			year = fmt.Sprintf("%d", time.Now().Year())
		}
		yr := year
		if len(yr) >= 2 {
			yr = yr[len(yr)-2:]
		}
		if series == "" {
			// ۲. از simpleText مثل !CDB 02/120 استخراج کن
			if m := simpleTextRe.FindStringSubmatch(rawBody); len(m) >= 3 {
				series = m[1]
				if n, e := strconv.Atoi(m[2]); e == nil {
					number = n
				}
			}
			if series == "" {
				series = "A"
			}
		}
		return series + formatNum(number) + "/" + yr
	}
	// آخرین تلاش: جستجوی الگوی NOTAM در کل XML
	if m := seriesRegex1.FindStringSubmatch(rawBody); len(m) >= 2 {
		return m[1]
	}
	if m := seriesRegex2.FindStringSubmatch(rawBody); len(m) >= 2 {
		return m[1]
	}
	return ""
}

func formatNum(n int) string {
	return fmt.Sprintf("%04d", n)
}

func parseNOTAMTime(s string) time.Time {
	t := parseNOTAMTimePtr(s)
	if t == nil {
		return time.Time{}
	}
	return *t
}

func parseNOTAMTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	s = strings.TrimSpace(s)
	// FAA RFC3339: 2026-02-08T15:50:00.000Z
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000Z", s)
	}
	if err != nil && len(s) >= 10 {
		// ICAO format: YYMMDDHHMM (e.g. 2602081550)
		if n, e := strconv.Atoi(s[:10]); e == nil {
			yy := n / 100000000
			n %= 100000000
			mm := n / 1000000
			n %= 1000000
			dd := n / 10000
			n %= 10000
			hh := n / 100
			mn := n % 100
			t = time.Date(2000+yy, time.Month(mm), dd, hh, mn, 0, 0, time.UTC)
			return &t
		}
	}
	if err != nil {
		return nil
	}
	return &t
}

func contains(s, sub string) bool {
	return strings.Contains(strings.ToUpper(s), sub)
}

// buildICAOFormattedText ساخت متن فرمت‌شده ICAO برای خروجی استاندارد خلبان
func buildICAOFormattedText(ev messaging.NotamEvent, seriesNum, eventType, rawBody string) string {
	var sb strings.Builder
	var cancelledRef string
	if seriesNum != "" {
		if eventType == "R" {
			refNum := findRefNumber(ev, seriesNum)
			if refNum == "" {
				refNum = findXoverNotamFromXML(rawBody)
			}
			if refNum != "" {
				sb.WriteString(fmt.Sprintf("%s NOTAMR %s\n", seriesNum, refNum))
			} else {
				sb.WriteString(fmt.Sprintf("%s NOTAMR\n", seriesNum))
			}
		} else if eventType == "C" {
			cancelledRef = findRefNumber(ev, seriesNum)
			if cancelledRef == "" {
				cancelledRef = findXoverNotamFromXML(rawBody)
			}
			if cancelledRef != "" {
				sb.WriteString(fmt.Sprintf("%s NOTAMC %s\n", seriesNum, cancelledRef))
			} else {
				sb.WriteString(fmt.Sprintf("%s NOTAMC\n", seriesNum))
			}
		} else {
			sb.WriteString(fmt.Sprintf("%s NOTAMN\n", seriesNum))
		}
	}
	// بند Q برای رویداد کنسل طبق استاندارد نمایش داده نمی‌شود
	if eventType != "C" && ev.AffectedFIR != "" {
		sb.WriteString("Q) " + ev.AffectedFIR + "/QWMLW/IV/BO/W/000/999/\n")
	}
	loc := ev.ICAOLocation
	if loc == "" {
		loc = ev.Location
	}
	if loc == "" {
		loc = ev.AffectedFIR
	}
	if loc != "" {
		sb.WriteString("A) " + loc + "\n")
	}
	if ev.EffectiveStart != "" || ev.EffectiveEnd != "" {
		bc := ""
		if ev.EffectiveStart != "" {
			bc = "B) " + strings.TrimPrefix(ev.EffectiveStart, "20")
		}
		if ev.EffectiveEnd != "" {
			if bc != "" {
				bc += "   "
			}
			bc += "C) " + strings.TrimPrefix(ev.EffectiveEnd, "20")
		}
		sb.WriteString(bc + "\n")
	}
	if ev.Schedule != "" {
		sb.WriteString("D) " + ev.Schedule + "\n")
	}
	txt := strings.TrimSpace(ev.Text)
	if eventType == "C" {
		if cancelledRef != "" {
			txt = "NOTAM CANCELLED - CANCELLED NOTAM: " + cancelledRef
		} else {
			txt = "NOTAM CANCELLED"
		}
	}
	sb.WriteString("E) " + txt + "\n")
	if ev.LowerLimit != "" {
		sb.WriteString("F) " + ev.LowerLimit + "\n")
	}
	if ev.UpperLimit != "" {
		sb.WriteString("G) " + ev.UpperLimit + "\n")
	}
	return sb.String()
}

// findRefNumber شماره NOTAM مرجع را از متن (برای جایگزین/لغو) استخراج می‌کند
func findRefNumber(ev messaging.NotamEvent, currentSeries string) string {
	// الگوی سریال NOTAM: حرف/اعداد + / + دو رقم سال مثل H2460/26, A1234/26
	refSeriesRe := regexp.MustCompile(`\b([A-Z0-9]{1,6}/\d{2})\b`)
	matches := refSeriesRe.FindAllStringSubmatch(ev.Text, -1)
	for _, m := range matches {
		if len(m) >= 2 {
			cand := strings.TrimSpace(m[1])
			if cand != "" && cand != currentSeries {
				return cand
			}
		}
	}
	// روش قبلی: فیلد با پیشوند سری فعلی
	fields := strings.Fields(ev.Text)
	for _, f := range fields {
		if strings.Contains(f, "/") && (ev.Series == "" || strings.HasPrefix(f, ev.Series)) {
			if refSeriesRe.MatchString(f) {
				sub := refSeriesRe.FindString(f)
				if sub != "" && sub != currentSeries {
					return sub
				}
			}
		}
	}
	return ""
}

// findXoverNotamFromXML از XML خام شناسه NOTAM مرجع را استخراج می‌کند (xovernotamID با __text مثل A1617/26 برای Replace/Cancel)
func findXoverNotamFromXML(rawBody string) string {
	if rawBody == "" {
		return ""
	}
	if m := xmlXoverNotamRe.FindStringSubmatch(rawBody); len(m) >= 2 {
		s := strings.TrimSpace(m[1])
		if s != "" && strings.Contains(s, "/") {
			return s
		}
	}
	// جستجوی هر الگوی سریال NOTAM در XML
	refSeriesRe := regexp.MustCompile(`\b([A-Z0-9]{1,6}/\d{2})\b`)
	if m := refSeriesRe.FindStringSubmatch(rawBody); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// normalizeFormattedTextForCancel برای NOTAM کنسل: خط اول را به صورت «سریال NOTAMC ref» درمی‌آورد و بند Q را حذف می‌کند
func normalizeFormattedTextForCancel(formatted, seriesNum string, ev messaging.NotamEvent, rawBody string) string {
	cancelledRef := findRefNumber(ev, seriesNum)
	if cancelledRef == "" {
		cancelledRef = findXoverNotamFromXML(rawBody)
	}
	firstLine := seriesNum + " NOTAMC"
	if cancelledRef != "" {
		firstLine = seriesNum + " NOTAMC " + cancelledRef
	}
	lines := strings.Split(formatted, "\n")
	var out []string
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(trimmed, "Q)") {
			continue
		}
		if i == 0 && strings.Contains(strings.ToUpper(trimmed), "NOTAMC") {
			line = firstLine
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
