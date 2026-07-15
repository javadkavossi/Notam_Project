package messaging

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

// notam_parser.go — پارس XML و ساخت متن استاندارد ICAO، جدا از آداپتور Solace (E2-1).
// این توابع خالص (pure) و مستقل از اتصال شبکه‌اند تا واحد‌تست‌پذیر باشند.

var refNumberRe = regexp.MustCompile(`\b([A-Z0-9]{1,6}/\d{2})\b`)

// ParseNotamXML یک payload خام XML را به NotamEvent تبدیل می‌کند.
func ParseNotamXML(payload string) (NotamEvent, error) {
	var event NotamEvent
	if err := xml.Unmarshal([]byte(payload), &event); err != nil {
		return NotamEvent{}, err
	}
	return event, nil
}

// EnsureHumanReadableText متن انسانی رویداد را تضمین می‌کند: اگر متن formatted موجود و کامل
// نباشد (کمتر از ۳ خط)، متن استاندارد ICAO را از فیلدها می‌سازد.
func EnsureHumanReadableText(event NotamEvent) string {
	humanText := strings.TrimSpace(event.HumanReadableText)
	if humanText == "" || len(strings.Split(humanText, "\n")) < 3 {
		humanText = BuildHumanReadableText(event)
	}
	return humanText
}

// BuildHumanReadableText متن استاندارد ICAO (بندهای NOTAM / Q / A / B-C / D / E / F / G)
// را از فیلدهای ساختاریافتهٔ رویداد می‌سازد.
func BuildHumanReadableText(event NotamEvent) string {
	var sb strings.Builder

	yr := "26"
	if len(event.Year) >= 2 {
		yr = event.Year[len(event.Year)-2:]
	}
	currentNumber := fmt.Sprintf("%s%04d/%s", event.Series, event.Number, yr)

	refNumber := ""
	refType := ""
	if strings.Contains(strings.ToUpper(event.EventType), "R") {
		refType = "NOTAMR"
	} else if strings.Contains(strings.ToUpper(event.EventType), "C") {
		refType = "NOTAMC"
	}

	if refType != "" {
		for _, m := range refNumberRe.FindAllStringSubmatch(event.Text, -1) {
			if len(m) >= 2 {
				cand := strings.TrimSpace(m[1])
				if cand != "" && cand != currentNumber {
					refNumber = cand
					break
				}
			}
		}
	}

	if refType != "" && refNumber != "" {
		sb.WriteString(fmt.Sprintf("%s %s %s\n", currentNumber, refType, refNumber))
	} else {
		sb.WriteString(fmt.Sprintf("%s NOTAM%s\n", currentNumber, event.EventType))
	}

	// بند Q برای NOTAMC طبق استاندارد نمایش داده نمی‌شود
	if refType != "NOTAMC" && event.AffectedFIR != "" {
		sb.WriteString("Q) " + event.AffectedFIR + "/QWMLW/IV/BO/W/000/999/\n")
	}

	aLine := event.ICAOLocation
	if aLine == "" {
		aLine = event.Location
	}
	if aLine == "" {
		aLine = event.AffectedFIR
	}
	sb.WriteString("A) " + aLine + "\n")

	bcLine := ""
	if event.EffectiveStart != "" {
		bcLine += "B) " + trimYearPrefix(event.EffectiveStart)
	}
	if event.EffectiveEnd != "" {
		if bcLine != "" {
			bcLine += "   "
		}
		bcLine += "C) " + trimYearPrefix(event.EffectiveEnd)
	}
	if bcLine != "" {
		sb.WriteString(bcLine + "\n")
	}

	if event.Schedule != "" {
		sb.WriteString("D) " + event.Schedule + "\n")
	}

	eText := strings.TrimSpace(event.Text)
	if refType == "NOTAMC" {
		eText = "NOTAM CANCELLED"
	}
	sb.WriteString("E) " + eText + "\n")

	if event.LowerLimit != "" {
		sb.WriteString("F) " + event.LowerLimit + "\n")
	}
	if event.UpperLimit != "" {
		sb.WriteString("G) " + event.UpperLimit + "\n")
	}

	return sb.String()
}

// trimYearPrefix پیشوند دو رقمی قرن را از timestamp ICAO حذف می‌کند (مثل 2026... → 26...).
// معادل رفتار قبلی event.EffectiveStart[2:] اما امن در برابر رشتهٔ کوتاه.
func trimYearPrefix(s string) string {
	if len(s) >= 2 {
		return s[2:]
	}
	return s
}
