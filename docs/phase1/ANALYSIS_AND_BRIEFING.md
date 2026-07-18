# تحلیل NOTAM و موتور بریفینگ

> این سند «مغز» سیستم است: چطور هر NOTAM را دقیق می‌فهمیم (دیکد Q-code)، دسته‌بندی و امتیازدهی می‌کنیم، و چطور NOTAMهای یک پرواز را می‌سازیم و مرتب می‌کنیم.

فهرست:
1. [ساختار استاندارد NOTAM](#۱-ساختار-استاندارد-notam)
2. [دیکد Q-code — پایهٔ دقت](#۲-دیکد-q-code)
3. [دسته‌بندی (Classification)](#۳-دستهبندی)
4. [امتیاز اهمیت (Importance Scoring)](#۴-امتیاز-اهمیت)
5. [مدل داده](#۵-مدل-داده)
6. [موتور بریفینگ — تطبیق پرواز](#۶-موتور-بریفینگ)
7. [رتبه‌بندی و ارائه](#۷-رتبهبندی-و-ارائه)
8. [دادهٔ مرجع](#۸-دادهٔ-مرجع)
9. [تجربهٔ کاربری بریفینگ](#۹-تجربهٔ-کاربری)

---

## ۱. ساختار استاندارد NOTAM

هر NOTAM ICAO از فیلدهای Q و A تا G تشکیل شده:

```
(A1234/26 NOTAMN
 Q) OIIX/QMRLC/IV/NBO/A/000/999/3540N05111E005
 A) OIII  B) 2607150600  C) 2607152359
 D) 0600-1200
 E) RWY 11L/29R CLSD DUE WIP
 F) SFC  G) 999FT AMSL)
```

| بند | معنی | استفادهٔ ما |
|-----|------|-------------|
| **Q)** | خط ماشین‌خوان: FIR / Q-code / ترافیک / هدف / دامنه / حد پایین / حد بالا / مختصات+شعاع | **کلید دسته‌بندی و امتیازدهی دقیق** |
| **A)** | مکان (ICAO فرودگاه/FIR) | تطبیق مکانی |
| **B) / C)** | شروع/پایان اعتبار | تطبیق زمانی با پنجرهٔ پرواز |
| **D)** | زمان‌بندی (schedule) | فعال‌بودن در ساعت پرواز |
| **E)** | متن انسانی | نمایش + استخراج کلیدواژه |
| **F) / G)** | حد پایین/بالای ارتفاع | ارتباط با فاز/ارتفاع پرواز |

---

## ۲. دیکد Q-code

Q-code قلب تحلیل دقیق است. ساختار: `Q` + **۲ حرف موضوع** + **۲ حرف وضعیت**.

```
Q M R L C
  └┬┘ └┬┘
موضوع  وضعیت
(MR=Runway) (LC=Closed)  →  «باند بسته است»
```

### گروه‌های موضوع (حرف دوم Q-code)

| حرف | دامنه موضوع | مثال |
|-----|-------------|------|
| `A` | سازمان فضای هوایی | ساختار مسیرها |
| `C` | ارتباطات/نظارت | رادار، CPDLC |
| `F` | تسهیلات و خدمات | سوخت، آتش‌نشانی |
| `G` | خدمات GNSS | GPS، RAIM، WAAS |
| `I` | ILS / MLS | گلایدسلوپ، لوکالایزر |
| `L` | روشنایی | چراغ باند، PAPI |
| `M` | سطح حرکت و فرود | **باند، تاکسی‌وی، اپرون** |
| `N` | ناوبری ترمینال/enroute | VOR, DME, NDB |
| `O` | سایر | متفرقه |
| `P` | رویه‌های ATC | SID, STAR, رویهٔ نزدیکی |
| `R` | محدودیت فضای هوایی | ممنوعه، خطرناک، محدود |
| `S` | خدمات ATS/VOLMET | برج، اطلاعات پرواز |
| `W` | هشدارها | مانع، تیراندازی، پهپاد |

### گروه‌های وضعیت (حرف چهارم Q-code)

| حرف | معنی | شدت نسبی |
|-----|------|----------|
| `A` | Availability (AS=غیرقابل‌استفاده، AU=در دسترس نیست) | بالا |
| `C` | Changes (CN=کنسل، CA=فعال شد، CC=تکمیل شد) | متغیر |
| `H` | Hazard conditions | بالا |
| `L` | Limitations (LC=بسته، LP=ممنوع، LT=محدود) | بالا (LC بحرانی) |
| `T` | Trigger NOTAM | اطلاعی |
| `X` | XX = متن آزاد (زبان طبیعی) | نامشخص → fallback |

**راهبرد:** جدول کامل Q-code (ICAO Doc 8126 / Annex 15) در `pipeline/qcode` به‌صورت داده‌محور (map/جدول) پیاده می‌شود؛ خروجی: `{subjectCategory, conditionType, humanSubject, humanCondition}`. این deterministic است → **دقت بالا بدون LLM**.

```mermaid
flowchart TD
    Q["Q-code: QMRLC"] --> S[subject = MR → Runway]
    Q --> C[condition = LC → Closed]
    S --> CAT["category = RUNWAY"]
    C --> SEV["conditionSeverity = CLOSED"]
    CAT & SEV --> R["نتیجه ساختاریافته:<br/>«باند بسته»"]
    Q -.اگر QXXXX.-> FB["fallback: تحلیل متن E)<br/>+ کلیدواژه"]
```

> اگر Q-code موجود نبود یا `XX` بود (متن آزاد)، به تحلیل متن `E)` با کلیدواژه‌ها و الگوها fallback می‌کنیم (مثل FICON, RWY CLSD, ILS U/S).

---

## ۳. دسته‌بندی

هر NOTAM به یک **دستهٔ اصلی** و **زیرمشخصه‌ها** نگاشت می‌شود:

```mermaid
mindmap
  root((دسته‌بندی NOTAM))
    باند و سطح فرود
      RWY بسته
      FICON «وضعیت سطح»
      روشنایی باند
    تاکسی‌وی و اپرون
    ناوبری
      ILS
      VOR/DME
      GPS/RAIM
    فضای هوایی
      ممنوعه/محدود/خطرناک
      TFR
    رویه‌ها
      SID/STAR
      رویهٔ نزدیکی
    خدمات و ارتباطات
    هشدارها و موانع
    سایر/اداری
```

خروجی هر NOTAM شامل:
- `category` (enum اصلی)
- `subject` / `condition` (از Q-code)
- `flightPhase[]` — به کدام فاز مربوط است: `DEPARTURE`, `ENROUTE`, `APPROACH`, `LANDING`, `GROUND`
- `tags[]` — پرچم‌های ویژه: `FICON`, `RWY_CLOSED`, `AD_CLOSED`, `ILS_OUT`, `GPS_OUT`, `OBSTACLE`

---

## ۴. امتیاز اهمیت

هدف: به هر NOTAM یک **سطح** و **امتیاز عددی (۰–۱۰۰)** بدهیم تا خلبان سریع مورد بحرانی را ببیند.

### تابع امتیاز (قاعده‌محور، شفاف و قابل‌ممیزی)

```
score = base(subject, condition)                     ← وزن پایه از جدول Q-code
      + phaseRelevance(flightPhase, flightContext)   ← آیا فاز مرتبط با این پرواز است؟
      + assetMatch(notam, flightAssets)              ← آیا دقیقاً باند/رویه/ناوید موردِ استفادهٔ پرواز را می‌زند؟
      + specialFlags(FICON, AD_CLOSED, ...)          ← پرچم‌های بحرانی
      + verticalOverlap(F/G, flightLevels)           ← هم‌پوشانی ارتفاعی
      - staleness/expiry adjustments
```

خروجی به **سطوح** نگاشت می‌شود:

| سطح | امتیاز | نمونه | نمایش |
|-----|--------|-------|--------|
| 🔴 **CRITICAL** | ۸۰–۱۰۰ | باند مقصد بسته، فرودگاه بسته، ILSِ باندِ فرود out | **بولد، بالای لیست، اخطار** |
| 🟠 **HIGH** | ۶۰–۷۹ | FICON، بستن تاکسی‌وی اصلی، GPS out در approach | برجسته |
| 🟡 **MEDIUM** | ۳۵–۵۹ | محدودیت جزئی، روشنایی ثانویه | عادی |
| 🟢 **LOW** | ۱۵–۳۴ | تغییرات اداری، ناوید کم‌اهمیت | جمع‌شده |
| ⚪ **INFO** | ۰–۱۴ | اطلاعی/trigger | جمع‌شده |

### مثال محاسبه

```mermaid
flowchart LR
    N["A1234/26<br/>QMRLC RWY 29R CLSD<br/>@ OIII (مقصد)"] --> B["base(MR,LC)=70"]
    N --> P["approach/landing مرتبط +15"]
    N --> A["پرواز روی 29R فرود می‌آید +10"]
    N --> F["RWY_CLOSED flag +5"]
    B & P & A & F --> SUM["score=100 → CRITICAL 🔴"]
```

**اصول مهم امتیازدهی:**
- کاملاً **قاعده‌محور و قابل‌توضیح** است — برای safety-critical، هر امتیاز باید قابل ردیابی باشد (چرا CRITICAL شد؟). جدول وزن‌ها نسخه‌دار و قابل‌بازبینی توسط کارشناس هوانوردی.
- `assetMatch` نیازمند دادهٔ مرجع دقیق است (باندها/رویه‌های هر فرودگاه) — به همین دلیل دادهٔ مرجع حیاتی است.
- امتیاز **وابسته به کانتکست پرواز** است: یک NOTAM ممکن است برای یک پرواز CRITICAL و برای دیگری LOW باشد (مثلاً باندی که این پرواز از آن استفاده نمی‌کند). پس امتیاز نهایی در زمان ساخت بریفینگ محاسبه می‌شود، ضمن اینکه یک «امتیاز پایهٔ مستقل از پرواز» هم برای فهرست عمومی ذخیره می‌شود.
- فاز بعد: LLM می‌تواند **توضیح** و **خلاصه** بیفزاید، اما **تصمیم سطح‌بندی همچنان قاعده‌محور** می‌ماند (اعتماد).

---

## ۵. مدل داده

توسعهٔ مدل فعلی `Notam` + جداول جدید:

```mermaid
erDiagram
    NOTAM ||--o{ NOTAM_SIGHTING : provenance
    NOTAM }o--|| AIRPORT : "location (اختیاری)"
    FLIGHT_PLAN ||--o{ FLIGHT_AIRPORT : "فرودگاه‌ها"
    FLIGHT_PLAN ||--o{ BRIEFING_ITEM : "خروجی بریفینگ"
    BRIEFING_ITEM }o--|| NOTAM : "ارجاع"
    AIRPORT ||--o{ RUNWAY : has
    FIR ||--o{ NOTAM : "enroute (geo)"

    NOTAM {
        int id PK
        string canonical_key UK "location+series+year"
        string series_number
        string event_type "N/R/C"
        string location_icao
        string affected_fir
        geometry area "PostGIS point+radius/polygon"
        timestamp effective_start
        timestamp effective_end
        string schedule
        text plain_text
        text formatted_text
        string qcode "QMRLC"
        string q_subject "MR"
        string q_condition "LC"
        string category "RUNWAY/NAV/..."
        string flight_phases "آرایه"
        string tags "FICON,RWY_CLOSED"
        int base_score
        string base_level
        string status "active/cancelled/expired"
        string confidence "single/corroborated/conflicting"
        text raw_body
    }
    NOTAM_SIGHTING {
        int id PK
        int notam_id FK
        string source
        timestamp seen_at
        string raw_hash
    }
    FLIGHT_PLAN {
        int id PK
        string username
        string adep
        string ades
        timestamp etd
        timestamp eta
        int buffer_minutes
        geometry route_corridor "PostGIS"
    }
    FLIGHT_AIRPORT {
        int id PK
        int flight_id FK
        string icao
        string role "ADEP/ADES/ALTN"
    }
    BRIEFING_ITEM {
        int id PK
        int flight_id FK
        int notam_id FK
        int contextual_score
        string contextual_level
        string match_reason "چرا انتخاب شد"
    }
    FIR {
        string icao PK
        string name
        geometry boundary "PostGIS polygon"
    }
```

نکات:
- `NOTAM.area` (PostGIS) از مختصات+شعاع خط Q ساخته می‌شود → تطبیق جغرافیایی روت.
- `canonical_key` جایگزین `message_id` به‌عنوان کلید یکتا (برای اجماع چندمنبعی).
- `base_score`/`base_level` مستقل از پرواز؛ `contextual_score`/`contextual_level` در `BRIEFING_ITEM` وابسته به پرواز.

---

## ۶. موتور بریفینگ

ورودی: `FlightPlan{ ADEP, ADES, ALTN[], route(FIRs/waypoints), ETD, ETA, buffer }`.
خروجی: مجموعهٔ NOTAMهای مرتبط، دسته‌بندی و رتبه‌بندی‌شده.

```mermaid
flowchart TD
    FP["FlightPlan"] --> W["پنجرهٔ زمانی:<br/>[ETD - buffer , ETA + buffer]"]
    FP --> AP["مجموعه فرودگاه‌ها:<br/>ADEP+ADES+ALTN"]
    FP --> RT["کریدور روت (PostGIS)<br/>از waypointها/FIRها"]

    AP --> Q1["NOTAMهای aerodrome:<br/>location_icao ∈ فرودگاه‌ها"]
    RT --> Q2["NOTAMهای enroute:<br/>area ∩ کریدور روت<br/>یا affected_fir ∈ FIRهای مسیر"]
    W --> TF["فیلتر زمانی:<br/>اعتبار NOTAM ∩ پنجره<br/>+ چک schedule بند D"]

    Q1 & Q2 --> TF
    TF --> ST["حذف کنسل‌شده/منقضی<br/>+ اعمال NOTAMR/NOTAMC"]
    ST --> SC["امتیاز کانتکستی هر NOTAM"]
    SC --> GRP["گروه‌بندی:<br/>به تفکیک فرودگاه/فاز/دسته"]
    GRP --> RANK["مرتب‌سازی درون‌گروه بر اساس سطح"]
    RANK --> OUT["Briefing خروجی"]
```

### تطبیق زمانی دقیق
- بازهٔ اعتبار NOTAM `[B, C]` باید با پنجرهٔ پرواز `[ETD-buffer, ETA+buffer]` **هم‌پوشانی** داشته باشد.
- اگر بند `D)` (schedule) وجود دارد، باید بررسی شود که در ساعات واقعی عبور/حضور پرواز فعال است (مثلاً NOTAM فقط ۰۶۰۰–۱۲۰۰ فعال است).
- NOTAMهای دائمی/بدون پایان (`PERM`, `EST`) هم پوشش داده می‌شوند.

### تطبیق مکانی enroute
- کریدور روت به‌صورت buffer دور خط مسیر (مثلاً ±X نات‌مایل) در PostGIS ساخته می‌شود.
- `ST_Intersects(notam.area, route_corridor)` برای NOTAMهای ناوبری/فضای هوایی.
- fallback بر اساس FIR وقتی مختصات دقیق نیست.

### چابکی و سرعت
- ایندکس‌های GiST روی geometry + ایندکس روی `effective_start/end`, `location_icao`, `status`.
- کش نتیجهٔ بریفینگ در Redis (کلید = هش FlightPlan + نسخهٔ دادهٔ NOTAM) با invalidation هنگام رسیدن NOTAM جدید مرتبط.
- هدف کارایی: ساخت بریفینگ برای ۵ فرودگاه + روت < ~۳۰۰ms در حالت گرم.

---

## ۷. رتبه‌بندی و ارائه

خروجی بریفینگ برای هر پرواز به این شکل گروه‌بندی می‌شود:

```mermaid
flowchart TB
    B[Briefing] --> C1["🔴 CRITICAL «بالای همه، بولد»"]
    B --> G1["✈ ADEP — مبدأ"]
    B --> G2["✈ ADES — مقصد"]
    B --> G3["✈ ALTN — الترنت‌ها"]
    B --> G4["🌐 ENROUTE — مسیر/FIRها"]
    G1 & G2 & G3 & G4 --> SUB["درون هر گروه: به تفکیک دسته<br/>«باند/ناوبری/فضای هوایی/…»<br/>مرتب بر اساس سطح اهمیت"]
```

- ردیف بالای بریفینگ: **خلاصهٔ بحرانی** (همهٔ CRITICALها یک‌جا، بولد، با آیکن).
- هر NOTAM: سطح (رنگ/آیکن)، دستهٔ ترجمه‌شده، خلاصهٔ ماشینی («باند 29R بستهٔ مقصد»)، متن کامل قابل‌بازشدن، دلیل انتخاب (`match_reason`).
- برچسب اطمینان منبع (از [RELIABILITY.md](RELIABILITY.md)).
- خروجی قابل export (PIB) برای فازهای بعد.

---

## ۸. دادهٔ مرجع

دقت بریفینگ مستقیماً به دقت دادهٔ مرجع وابسته است.

| دیتاست | منبع پیشنهادی | کاربرد |
|--------|----------------|--------|
| فرودگاه‌ها (ICAO, نام, مختصات, ارتفاع) | OurAirports (باز) / FAA NASR | تطبیق مکانی، نمایش |
| باندها (نام، جهت، طول) | OurAirports / NASR | `assetMatch` امتیازدهی |
| ناویدها (VOR/DME/ILS/NDB) | NASR / منابع باز | تطبیق ناوبری |
| مرزهای FIR (polygon) | GeoJSON مرجع هوانوردی | تطبیق enroute |

**ETL نسخه‌دار + تشخیص تغییر:**

```mermaid
flowchart LR
    SRC[منبع مرجع] --> ETL[ETL job]
    ETL --> DIFF{مقایسه با نسخهٔ فعلی}
    DIFF -->|بدون تغییر| SKIP[هیچ]
    DIFF -->|تغییر| VER[نسخهٔ جدید + changelog]
    VER --> ALERT["اطلاع: «باند فرودگاه X تغییر کرد»"]
```

- هر دیتاست نسخه‌بندی می‌شود؛ diff بین نسخه‌ها ثبت و در صورت تغییرِ فرودگاه‌های پرکاربرد، هشدار داده می‌شود.
- اجرای دوره‌ای (مثلاً روزانه/هفتگی) + امکان اجرای دستی.

---

## ۹. تجربهٔ کاربری بریفینگ

جریان کاربر:

```mermaid
flowchart LR
    L[ورود] --> NF[«پرواز جدید»]
    NF --> F1[انتخاب ADEP/ADES]
    F1 --> F2["افزودن ALTN‌ها + روت<br/>«autocomplete از دادهٔ مرجع»"]
    F2 --> F3[زمان پرواز ETD/ETA + buffer]
    F3 --> GEN[ساخت بریفینگ]
    GEN --> VIEW["نمای بریفینگ:<br/>خلاصهٔ بحرانی + گروه‌ها + نقشه"]
    VIEW --> LIVE["پایش زنده «WS»:<br/>NOTAM جدید → بنر/اخطار"]
```

الزامات UX (طبق تأکید کارفرما بر سرعت و جذابیت):
- **سرعت:** autocomplete فرودگاه/waypoint فوری از کش؛ بریفینگ با اسکلتون و بارگذاری تدریجی.
- **جذابیت و وضوح:** رنگ‌بندی سطوح، آیکن دسته‌ها، بولد بحرانی‌ها، نقشهٔ روت با نقاط NOTAM.
- **اعتماد:** نوار وضعیت منابع همیشه دیده شود؛ زمان تازگی داده مشخص.
- **زنده:** پرواز فعال subscribe می‌شود؛ NOTAM جدید مرتبط → toast/بنر؛ بحرانی → اخطار صوتی+بصری اجباری.
- **RTL فارسی** با ترجمهٔ اصطلاحات فنی + نگه‌داشتن کدهای استاندارد (ICAO) به‌صورت لاتین.

---

## ۱۰. لایهٔ Operational Impact (E5.5 — بازخورد کارشناس دیسپچ)

> این بخش پس از بازبینی کارشناس هوانوردی (دیسپچر) روی نسخهٔ `1.0.0` افزوده شد.
> بازخورد کامل: [EXPERT_REVIEW.md](EXPERT_REVIEW.md) · جمع‌بندی امتیازها: [QCODE_WEIGHTS.md](QCODE_WEIGHTS.md).

### ۱۰.۱ مسئله‌ای که کارشناس درست تشخیص داد

امتیاز پایهٔ قاعده‌محور (بخش ۴) برای بسیاری از NOTAMها **یک عدد ثابت نمی‌تواند باشد**، چون اهمیت واقعی به **بستر پرواز** وابسته است:

| NOTAM | چرا عدد ثابت غلط است |
|-------|----------------------|
| باند بسته | فرودگاه تک‌باند = فاجعه؛ فرودگاه سه‌باند = قابل‌مدیریت |
| فضای هوایی محدود | آیا ارتفاع/مسیر پرواز اصلاً با آن تداخل دارد؟ Prohibited یا Danger؟ |
| سوخت موجود نیست | اگر نوع سوخت (مثل 100LL) اصلاً برای هواپیمای ما نیست → بی‌ربط |
| بسته به VFR | برای پرواز IFR بی‌اثر است |
| RVR موجود نیست | برای عملیات کم‌دید بحرانی؛ در دید خوب کم‌اهمیت |
| RFFS کاهش‌یافته | شرطِ **پیش از پرواز** است، نه رویدادِ حین پرواز |

**نتیجه:** امتیاز پایه، امتیاز **نهایی** نیست. امتیاز نهایی از ارزیابی «اثر عملیاتی این NOTAM روی همین پرواز» به‌دست می‌آید.

### ۱۰.۲ معماری دولایه (تثبیت‌شده)

```mermaid
flowchart LR
    N[NOTAM] --> EV["Evidence<br/>Q-code · متن E · هندسه · زمان"]
    EV --> IMP["Operational Effect<br/>نگاشت به اثر استاندارد"]
    IMP --> BASE["امتیاز پایه (مستقل از پرواز)<br/>= شدت عمومی اثر"]
    BASE --> CTX["Flight Context<br/>runway count · aircraft · rules · route/alt · fuel · time"]
    FP[FlightPlan] --> CTX
    CTX --> FINAL["امتیاز نهایی + اولویت + اقدام پیشنهادی"]
```

- **لایهٔ ۱ — Base (پیاده‌شده، E3):** «این NOTAM چه اثری دارد و شدت عمومی‌اش چقدر است؟» مستقل از پرواز، قابل‌کش، قابل‌ممیزی.
- **لایهٔ ۲ — Operational Impact (این طرح، E5.5):** «این اثر برای *این* پرواز چقدر مهم است؟» base را با بستر پرواز تعدیل می‌کند.

> این همان دولایه‌ای است که از ابتدا در معماری بود (`base_score` + `contextual_score`)، اما لایهٔ ۲ در پیاده‌سازی فعلی ضعیف است (فقط bonus نقش فرودگاه). بازخورد کارشناس آن را به یک ارزیابیِ **اثر عملیاتیِ درجه‌یک** ارتقا می‌دهد.

### ۱۰.۳ تاکسونومی اثر عملیاتی (Operational Effects)

هر NOTAM به یک یا چند اثر استاندارد نگاشت می‌شود (نه صرفاً «دسته»):

| اثر | نمونه Q-code/متن | فاکتور بستر تعیین‌کننده |
|-----|------------------|-------------------------|
| `RUNWAY_UNAVAILABLE` | QMRLC | **تعداد باندهای فرودگاه**، باندِ موردِ استفاده |
| `APPROACH_DEGRADED` | ILS U/S، PAPI، RVR | نوع نزدیکیِ برنامه‌ریزی‌شده، حداقل‌های عملیاتی |
| `NAV_DEGRADATION` | VOR/DME/GNSS out | آیا مسیر به آن ناوید متکی است؟ RNAV/RNP |
| `ROUTE_RESTRICTION` | فضای هوایی محدود/خطر | **تداخل مسیر و ارتفاع** (PostGIS + F/G) |
| `SURFACE_CONDITION` | FICON، برف/یخ/آب | نوع/وزن هواپیما، اصطکاک |
| `SERVICE_LIMITED` | سوخت، گمرک، اکسیژن | **آیا این سرویس برای این پرواز لازم است؟** |
| `ATS_REDUCED` | برج/نزدیکی بسته | ساعت عملیات، جایگزین |
| `OBSTACLE_HAZARD` | جرثقیل، مانع | فاز تقرب/برخاست، ارتفاع |
| `AERODROME_UNUSABLE` | AD CLSD، RFFS ناکافی | نوع هواپیما، شرطِ پیش‌پرواز |
| `INFORMATIONAL_ONLY` | Trigger، اطلاعیه | — (وزن پایین، هیچ NOTAMی بی‌صاحب نماند) |

### ۱۰.۴ فاکتورهای بستر پرواز (Context Factors)

آنچه لایهٔ ۲ باید بداند تا امتیاز نهایی بدهد:

| فاکتور | منبع داده | وضعیت |
|--------|-----------|-------|
| تعداد باند فرودگاه | جدول `runways` (داریم — ۳۷٬۲۳۲ رکورد) | ✅ آماده |
| قوانین پرواز IFR/VFR | خط Q فیلد traffic (IV/I/V) + مدل پرواز | 🟡 traffic داریم؛ باید به FlightPlan افزوده شود |
| تداخل مسیر/ارتفاع | PostGIS + مختصات/شعاع + minimumFL/maximumFL خط Q | ✅ پیاده شد (E5.6) |
| نوع/دستهٔ هواپیما | باید به FlightPlan افزوده شود | ⬜ |
| نوع سوخت موردنیاز | نوع هواپیما → JetA1/AVGAS | ⬜ |
| ساعت واقعی عبور/حضور | پنجرهٔ زمانی + schedule بند D | 🟡 پنجره داریم؛ schedule پارس نشده |

### ۱۰.۵ خروجی نهایی هر آیتم بریفینگ

به‌جای فقط یک عدد، لایهٔ ۲ تولید می‌کند:
- **امتیاز نهایی + سطح** (وابسته به پرواز)
- **اثر عملیاتی** (برچسب استاندارد)
- **اقدام پیشنهادی** (مثلاً «باند جایگزین را بررسی کن»، «حداقل‌های LVP را چک کن»، «بی‌ربط به این هواپیما»)
- **دلیل** (`match_reason`) — برای شفافیت و ممیزی

### ۱۰.۶ اصول پیاده‌سازی (طبق تأکید کارشناس)

- **چهارچوب باز:** افزودن اثر/فاکتور جدید بدون بازنویسی هسته؛ جریان ثابت `Evidence → Impact → Context → Priority/Action`.
- **هیچ NOTAMی بی‌صاحب نماند:** موارد بدون اثر → `INFORMATIONAL_ONLY` با وزن پایین (نه حذف).
- **NOTAM چندموضوعی → چند اثر:** یک NOTAM می‌تواند به چند سطر اثر نگاشت شود و در تجمیع، بیشینه/جمعِ وزن‌دار گرفته شود.
- **قابل‌ممیزی می‌ماند:** حتی با منطق شرطی، هر تصمیم قابل ردیابی باشد (برای safety-critical). LLM (فاز بعد) فقط توضیح می‌افزاید، نه تصمیمِ سطح.
