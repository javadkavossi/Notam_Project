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
| APRON | 25 | اپرون/پارکینگ |
| OTHER | 20 | سایر/اداری |

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

## نکات برای بازبینی کارشناس

- آیا وزن‌های پایهٔ دسته‌ها با اولویت عملیاتی خلبان همخوان است؟
- آیا آستانه‌های سطوح (۸۰/۶۰/۳۵/۱۵) مناسب‌اند؟
- کدهای Q-code پوشش‌دادهنشده در `qcode/qcode.go` (به fallback حرف اول می‌افتند) — آیا موردی حیاتی جا مانده؟
- امتیاز کانتکستی (E5) این پایه را با تطبیق باند/فاز موردِ استفادهٔ پرواز تعدیل می‌کند.
