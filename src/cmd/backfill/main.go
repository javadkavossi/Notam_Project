// Command backfill تحلیل E3 (دیکد Q-code، دسته‌بندی، امتیاز اهمیت) و کلید متعارف را
// روی NOTAMهایی که از قبل ذخیره شده‌اند دوباره محاسبه و به‌روزرسانی می‌کند.
//
// کاربرد: پس از تغییر parser یا جدول وزن‌دهی، رکوردهای موجود بدون نیاز به دریافت مجدد
// از منبع، با منطق جدید بازپردازش می‌شوند (raw_body به‌عنوان منبع حقیقت نگه داشته شده است).
//
// اجرا (از بیرون کانتینر، با اشاره به Postgres روی پورت میزبان):
//
//	POSTGRES_HOST=localhost POSTGRES_PORT=5434 POSTGRES_PASSWORD=... \
//	  go run ./cmd/backfill [-dry-run] [-limit N]
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/hossein-repo/BaseProject/config"
	"github.com/hossein-repo/BaseProject/data/db"
	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/messaging"
	"github.com/hossein-repo/BaseProject/internal/pipeline/analysis"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "فقط گزارش بده، چیزی ننویس")
	limit := flag.Int("limit", 0, "حداکثر تعداد رکورد (۰ = همه)")
	flag.Parse()

	cfg := config.GetConfig()
	if err := db.InitDb(cfg); err != nil {
		log.Fatalf("اتصال به دیتابیس ناموفق: %v", err)
	}
	defer db.CloseDb()
	gdb := db.GetDb()

	q := gdb.Model(&model.Notam{}).Where("raw_body IS NOT NULL AND raw_body <> ''")
	if *limit > 0 {
		q = q.Limit(*limit)
	}
	var notams []model.Notam
	if err := q.Find(&notams).Error; err != nil {
		log.Fatalf("خواندن NOTAMها ناموفق: %v", err)
	}
	log.Printf("🔎 %d رکورد برای بازپردازش", len(notams))

	var updated, parseFail, skipped int
	levels := map[string]int{}
	cats := map[string]int{}

	for _, n := range notams {
		ev, err := messaging.ParseNotamXML(n.RawBody)
		if err != nil {
			parseFail++
			continue
		}
		ev.HumanReadableText = messaging.EnsureHumanReadableText(ev)

		an := analysis.Analyze(ev)
		geo := messaging.ExtractGeo(ev)
		canonical := messaging.CanonicalKey(n.LocationICAO, n.SeriesNumber)
		if canonical == "" {
			canonical = "MSG:" + n.MessageID
		}

		levels[an.BaseLevel]++
		cats[an.Category]++

		if *dryRun {
			skipped++
			continue
		}
		// کلیدها باید دقیقاً نام ستون دیتابیس باشند (GORM فیلد QCode را q_code می‌نامد).
		err = gdb.Model(&model.Notam{}).Where("id = ?", n.Id).Updates(map[string]interface{}{
			"canonical_key": canonical,
			"source":        firstNonEmpty(n.Source, "FAA_SWIM"),
			"q_code":        an.QCode,
			"q_subject":     an.Subject,
			"q_condition":   an.Condition,
			"traffic":       strings.ToUpper(strings.TrimSpace(ev.Traffic)),
			"category":      an.Category,
			"flight_phases": model.StringSlice(an.Phases),
			"tags":          model.StringSlice(an.Tags),
			"base_score":     an.BaseScore,
			"base_level":     an.BaseLevel,
			"weights_ver":    analysis.WeightsVersion,
			"area_lat":       geo.Lat,
			"area_lon":       geo.Lon,
			"area_radius_nm": geo.RadiusNM,
			"lower_ft":       geo.LowerFt,
			"upper_ft":       geo.UpperFt,
			"vertical_known": geo.HasVertical,
		}).Error
		if err != nil {
			log.Printf("⚠️ به‌روزرسانی #%d ناموفق: %v", n.Id, err)
			continue
		}
		// ساخت ستون geometry از مرکز+شعاع (فقط برای موارد دارای هندسه)
		if geo.HasArea {
			if e := gdb.Exec(
				`UPDATE notams SET area = ST_Buffer(ST_SetSRID(ST_MakePoint(area_lon, area_lat),4326)::geography, area_radius_nm*1852.0)::geometry WHERE id = ?`,
				n.Id).Error; e != nil {
				log.Printf("⚠️ area #%d: %v", n.Id, e)
			}
		}
		updated++
	}

	fmt.Println("\n──────── نتیجهٔ backfill ────────")
	fmt.Printf("به‌روزشده: %d | خطای پارس: %d | dry-run: %d\n", updated, parseFail, skipped)
	fmt.Println("\nتوزیع سطح اهمیت:")
	for _, lv := range []string{analysis.LevelCritical, analysis.LevelHigh, analysis.LevelMedium, analysis.LevelLow, analysis.LevelInfo} {
		if levels[lv] > 0 {
			fmt.Printf("  %-9s %d\n", lv, levels[lv])
		}
	}
	fmt.Println("\nتوزیع دسته:")
	for c, n := range cats {
		fmt.Printf("  %-12s %d\n", c, n)
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
