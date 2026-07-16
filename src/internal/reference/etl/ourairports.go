// Package etl خواندن و تبدیل دادهٔ مرجع هوانوردی از منابع باز (OurAirports) است (E7-2).
//
// توابع پارس خالص‌اند (روی io.Reader کار می‌کنند) تا مستقل از شبکه/فایل واحد‌تست شوند.
// فرمت CSV: https://ourairports.com/data/
package etl

import (
	"bufio"
	"encoding/csv"
	"io"
	"strconv"
	"strings"

	"github.com/hossein-repo/BaseProject/data/db/model"
)

// ParseAirports فایل airports.csv را می‌خواند و فقط فرودگاه‌های دارای کد ICAO ۴حرفی معتبر را برمی‌گرداند.
func ParseAirports(r io.Reader) ([]model.Airport, error) {
	rows, idx, err := readCSV(r)
	if err != nil {
		return nil, err
	}
	out := make([]model.Airport, 0, len(rows))
	for _, rec := range rows {
		icao := strings.ToUpper(strings.TrimSpace(get(rec, idx, "ident")))
		// gps_code یا ident معمولاً ICAO است؛ اگر ident ۴حرفی نبود از gps_code استفاده کن
		if !isICAO(icao) {
			icao = strings.ToUpper(strings.TrimSpace(get(rec, idx, "gps_code")))
		}
		if !isICAO(icao) {
			continue
		}
		typ := get(rec, idx, "type")
		if typ == "closed" || strings.Contains(typ, "heliport") {
			// heliportها را نگه می‌داریم اما بسته‌ها را رد می‌کنیم
			if typ == "closed" {
				continue
			}
		}
		out = append(out, model.Airport{
			ICAO:         icao,
			IATA:         strings.ToUpper(strings.TrimSpace(get(rec, idx, "iata_code"))),
			Name:         get(rec, idx, "name"),
			Country:      strings.ToUpper(get(rec, idx, "iso_country")),
			Municipality: get(rec, idx, "municipality"),
			Type:         typ,
			Lat:          atof(get(rec, idx, "latitude_deg")),
			Lon:          atof(get(rec, idx, "longitude_deg")),
			ElevationFt:  atoi(get(rec, idx, "elevation_ft")),
		})
	}
	return out, nil
}

// ParseRunways فایل runways.csv را می‌خواند.
func ParseRunways(r io.Reader) ([]model.Runway, error) {
	rows, idx, err := readCSV(r)
	if err != nil {
		return nil, err
	}
	out := make([]model.Runway, 0, len(rows))
	for _, rec := range rows {
		icao := strings.ToUpper(strings.TrimSpace(get(rec, idx, "airport_ident")))
		if !isICAO(icao) {
			continue
		}
		le := strings.ToUpper(strings.TrimSpace(get(rec, idx, "le_ident")))
		he := strings.ToUpper(strings.TrimSpace(get(rec, idx, "he_ident")))
		name := le
		if he != "" {
			name = le + "/" + he
		}
		out = append(out, model.Runway{
			AirportICAO: icao,
			Name:        name,
			LEIdent:     le,
			HEIdent:     he,
			LengthFt:    atoi(get(rec, idx, "length_ft")),
			WidthFt:     atoi(get(rec, idx, "width_ft")),
			Surface:     get(rec, idx, "surface"),
			Lighted:     get(rec, idx, "lighted") == "1",
			Closed:      get(rec, idx, "closed") == "1",
		})
	}
	return out, nil
}

// ParseNavaids فایل navaids.csv را می‌خواند.
func ParseNavaids(r io.Reader) ([]model.Navaid, error) {
	rows, idx, err := readCSV(r)
	if err != nil {
		return nil, err
	}
	out := make([]model.Navaid, 0, len(rows))
	for _, rec := range rows {
		ident := strings.ToUpper(strings.TrimSpace(get(rec, idx, "ident")))
		if ident == "" {
			continue
		}
		out = append(out, model.Navaid{
			Ident:                 ident,
			Name:                  get(rec, idx, "name"),
			Type:                  strings.ToUpper(get(rec, idx, "type")),
			Frequency:             get(rec, idx, "frequency_khz"),
			Lat:                   atof(get(rec, idx, "latitude_deg")),
			Lon:                   atof(get(rec, idx, "longitude_deg")),
			AssociatedAirportICAO: strings.ToUpper(strings.TrimSpace(get(rec, idx, "associated_airport"))),
		})
	}
	return out, nil
}

// ---- helpers ----

// readCSV سرستون‌ها را می‌خواند و نگاشت نام→ایندکس + ردیف‌ها را برمی‌گرداند.
func readCSV(r io.Reader) ([][]string, map[string]int, error) {
	br := bufio.NewReader(r)
	// حذف BOM ابتدای فایل قبل از پارس CSV (وگرنه فیلد اولِ نقل‌قول‌شده خراب می‌شود)
	if b, err := br.Peek(3); err == nil && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		_, _ = br.Discard(3)
	}
	cr := csv.NewReader(br)
	cr.FieldsPerRecord = -1 // تحمل تعداد ستون متغیر
	header, err := cr.Read()
	if err != nil {
		return nil, nil, err
	}
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[strings.TrimSpace(h)] = i
	}
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	return rows, idx, nil
}

func get(rec []string, idx map[string]int, col string) string {
	if i, ok := idx[col]; ok && i < len(rec) {
		return strings.TrimSpace(rec[i])
	}
	return ""
}

func isICAO(s string) bool {
	if len(s) != 4 {
		return false
	}
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

func atof(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
