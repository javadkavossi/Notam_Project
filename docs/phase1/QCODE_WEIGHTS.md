# جدول وزن‌دهی امتیاز اهمیت NOTAM (E3-5)

> این سند برای **بازبینی کارشناس هوانوردی** است. امتیازدهی کاملاً قاعده‌محور و قابل‌ممیزی است.
> با هر تغییر در وزن‌ها، `WeightsVersion` در `internal/pipeline/analysis/analysis.go` را افزایش دهید.

**نسخهٔ فعلی: `1.0.0`**

---

## فرمول

```
BaseScore = clamp₀‑₁₀₀( categoryBase[Category] + conditionDelta[Condition] + tagBonus )
```

- **`categoryBase`** — وزن پایه بر اساس دستهٔ موضوع Q-code.
- **`conditionDelta`** — تعدیل بر اساس وضعیت (بستن/غیرقابل‌استفاده مثبت، لغو منفی).
- **`tagBonus`** — امتیاز اضافی برای پرچم‌های بحرانی.

> این «امتیاز پایهٔ مستقل از پرواز» است. امتیاز نهاییِ وابسته به پرواز (تطبیق باند/فاز موردِ استفاده) در موتور بریفینگ (E5) محاسبه می‌شود.

---

## سطوح (BaseLevel)

| سطح | بازهٔ امتیاز |
|-----|-------------|
| 🔴 CRITICAL | ۸۰–۱۰۰ |
| 🟠 HIGH | ۶۰–۷۹ |
| 🟡 MEDIUM | ۳۵–۵۹ |
| 🟢 LOW | ۱۵–۳۴ |
| ⚪ INFO | ۰–۱۴ |

---

## وزن پایهٔ دسته‌ها (`categoryBase`)

| دسته | وزن | توضیح |
|------|:---:|-------|
| AERODROME | 78 | کل فرودگاه |
| RUNWAY | 70 | باند، آستانه، فاصله‌های اعلامی |
| AIRSPACE | 55 | ساختار فضای هوایی، FIR، مسیر |
| RESTRICTION | 55 | ممنوعه/خطرناک/محدود |
| ILS | 50 | سامانهٔ فرود دقیق |
| PROCEDURE | 50 | SID/STAR/رویهٔ نزدیکی |
| GNSS | 50 | GPS/RAIM |
| WARNING | 45 | هشدارهای ناوبری |
| OBSTACLE | 45 | موانع |
| NAVIGATION | 45 | VOR/DME/NDB |
| LIGHTING | 40 | روشنایی باند/نزدیکی |
| COMMS | 40 | ارتباطات/رادار |
| TAXIWAY | 38 | تاکسی‌وی |
| MOVEMENT_AREA | 35 | گروه M با موضوع ناشناخته (عمداً < باند) |
| APRON | 25 | اپرون/پارکینگ |
| OTHER | 20 | سایر/اداری |

### ⚠️ اصل ایمنی: fallback هرگز تشدید نمی‌کند

اگر موضوع Q-code در جدول نباشد، بر اساس حرف اول گروه حدس زده می‌شود — اما **هرگز به پرخطرترین دستهٔ آن گروه** نگاشت نمی‌شود:

- گروه `M` (سطح حرکت) شامل باند، تاکسی‌وی و اپرون است. کد ناشناخته به `MOVEMENT_AREA` می‌رود، نه `RUNWAY`.
- برچسب‌های قاطع (`RWY_CLOSED`, `AD_CLOSED`, `ILS_OUT`, `GPS_OUT`) **فقط** وقتی صادر می‌شوند که موضوع دقیقاً شناسایی شده باشد (`Recognized=true`). برچسب متن‌محور `FICON` مستثناست.

> **پیشینه:** در آزمون با دادهٔ واقعی FAA، کد `QMYLC` («rapid exit taxiway بسته») به‌دلیل نبودِ `MY` در جدول با fallback به `RUNWAY` می‌رفت و با برچسب `RWY_CLOSED` امتیاز ۱۰۰/بحرانی می‌گرفت — یعنی **بستن یک تاکسی‌وی به‌عنوان بستن باند** به خلبان گزارش می‌شد. تست رگرسیون `TestRapidExitTaxiwayNotClassifiedAsRunway` این را می‌پاید.

## تعدیل وضعیت (`conditionDelta`) — منتخب

| وضعیت | Δ | معنی |
|-------|:--:|------|
| LC | +30 | Closed |
| LP | +27 | Prohibited |
| LD | +26 | Unsafe |
| AS / AU | +22 | Unserviceable / Not available |
| AW | +20 | Withdrawn |
| LI | +20 | Closed to IFR |
| HH | +20 | Hazard |
| CT | +15 | On test, do not use |
| CG | +12 | Downgraded |
| CA / CE | +12 | Activated / Erected |
| LN | +12 | Closed to night ops |
| CM / CL | +10 | Displaced / Realigned |
| LV | +10 | Closed to VFR |
| HW | +10 | Work in progress |
| LT / LL | +8 | Limited / Usable limited |
| **CC** | **−12** | Completed |
| **HV** | **−12** | Work completed |
| **AO / AK** | **−18** | Operational / Resumed |
| **CN** | **−50** | Cancelled |

## پرچم‌های بحرانی (`tagBonus`)

| پرچم | امتیاز | شرط |
|------|:------:|-----|
| AD_CLOSED | +12 | دستهٔ AERODROME + بسته |
| RWY_CLOSED | +10 | دستهٔ RUNWAY + بسته |
| FICON | +8 | متن شامل FICON |
| ILS_OUT / GPS_OUT | +6 | ناوبری غیرقابل‌استفاده |
| OBSTACLE | +3 | دستهٔ OBSTACLE |

---

## نمونه‌های تأییدشده (golden set — `analysis_test.go`)

| Q-code / متن | محاسبه | نتیجه |
|--------------|--------|-------|
| `QMRLC` RWY CLSD | 70 + 30 + 10 = 110→100 | 🔴 CRITICAL |
| `QFALC` AD CLSD | 78 + 30 + 12 →100 | 🔴 CRITICAL |
| `QICAS` ILS U/S | 50 + 22 + 6 = 78 | 🟠 HIGH |
| `QMXLC` TWY CLSD | 38 + 30 = 68 | 🟠 HIGH |
| `QRTCA` RESTRICTED ACT | 55 + 12 = 67 | 🟠 HIGH |
| `QOBCE` CRANE | 45 + 12 + 3 = 60 | 🟠 HIGH |
| `QMRCN` CANCELLED | 70 − 50 = 20 | 🟢 LOW |
| متن `RWY … CLOSED` (بدون Q) | 85 + 10 = 95 | 🔴 CRITICAL |
| متن `FICON …` (بدون Q) | 65 + 8 = 73 | 🟠 HIGH |

---

## منبع Q-code

Q-code از عنصر **`event:selectionCode`** در فید AIXM/FNS خوانده می‌شود (تگی به نام `qcode` در فید FAA وجود ندارد). اگر منبع آن را نداده باشد:

1. به تحلیل متنی (E3-2) fallback می‌شود، و
2. بند `Q)` در متن خروجی **حذف** می‌شود — هرگز Q-code ساختگی تولید نمی‌شود.

> **پیشینه:** کد اولیه مقدار ثابت `QWMLW` را در بند Q درج می‌کرد. این کد در ICAO یعنی «شلیک موشک/توپ، انجام خواهد شد». وقتی تحلیل E3 آن را می‌خواند، **۷۶۳ از ۹۴۳ NOTAM واقعی** به‌غلط در دستهٔ «هشدار/شلیک موشک» قرار گرفتند. تست `TestBuildHumanReadableText_NoFabricatedQCode` این را می‌پاید.

---

## نکات باز برای بازبینی کارشناس

- **`AERODROME` = 78 شاید بالا باشد:** با هر تعدیل مثبت (حتی `LT` = «محدود به») به CRITICAL می‌رسد. نمونهٔ واقعی: `QFALT` با متن «STAND-BY POWER IS NOT AVAILABLE» امتیاز ۸۶/بحرانی گرفت — احتمالاً باید HIGH باشد.
- **توزیع واقعی روی ۹۰۵ NOTAM:** CRITICAL ۹۷ · HIGH ۴۳۵ · MEDIUM ۲۷۲ · LOW ۵۸ · INFO ۴۳. سهم HIGH (~۴۸٪) احتمالاً زیاد است و آستانه‌ها یا وزن‌ها نیاز به تنظیم دارند.
- آیا آستانه‌های سطوح (۸۰/۶۰/۳۵/۱۵) مناسب‌اند؟
- کدهای پوشش‌داده‌نشده در `qcode/qcode.go` — آیا موردی حیاتی جا مانده؟
- امتیاز کانتکستی (E5) این پایه را با نقش فرودگاه در پرواز تعدیل می‌کند.
