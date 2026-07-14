# 📡 مستند بصری پروژه — FAA NOTAM Consumer

> سیستم دریافت، ذخیره، فیلتر و نمایش بلادرنگ **NOTAM** (اطلاعیه‌های هوانوردی) از سرویس **FAA SWIM / AIM FNS**
>
> تاریخ مستند: ۱۴۰۵/۰۴/۲۳ (2026-07-14) — نسخه ۱.۰

---

## 📑 فهرست

1. [نگاه کلی](#۱-نگاه-کلی)
2. [معماری سیستم](#۲-معماری-سیستم)
3. [جریان داده (Data Flow)](#۳-جریان-داده-data-flow)
4. [ساختار پوشه‌ها](#۴-ساختار-پوشهها)
5. [مدل داده (Database)](#۵-مدل-داده-database)
6. [API Endpoints](#۶-api-endpoints)
7. [منطق Consumer و فیلتر](#۷-منطق-consumer-و-فیلتر)
8. [منطق اعلان (Alert Matching)](#۸-منطق-اعلان-alert-matching)
9. [Frontend](#۹-frontend)
10. [زیرساخت و اجرا](#۱۰-زیرساخت-و-اجرا)
11. [تکنولوژی‌ها](#۱۱-تکنولوژیها)
12. [نقاط ضعف و بدهی فنی](#۱۲-نقاط-ضعف-و-بدهی-فنی)
13. [نقشه راه پیشنهادی](#۱۳-نقشه-راه-پیشنهادی)

---

## ۱. نگاه کلی

**NOTAM** = *Notice to Airmen*؛ اطلاعیه‌هایی درباره وضعیت باند، تجهیزات ناوبری، محدودیت‌های فضای هوایی و... که خلبانان باید بدانند.

این پروژه به‌صورت زنده به broker پیام **Solace** فرودگاه FAA وصل می‌شود، پیام‌های **XML** استاندارد را می‌گیرد، آن‌ها را به فرمت استاندارد **ICAO / Jeppesen** تبدیل می‌کند، در **PostgreSQL** ذخیره می‌کند و از طریق یک **REST API** و **داشبورد React** با قابلیت **اعلان بلادرنگ صوتی/تصویری** به کاربر نمایش می‌دهد.

```mermaid
mindmap
  root((NOTAM System))
    دریافت
      Solace Broker
      XML Parsing
      فیلتر FIR/Airport
    پردازش
      تبدیل به ICAO/Jeppesen
      استخراج Series Number
      مدیریت New/Replace/Cancel
    ذخیره
      PostgreSQL
      جلوگیری از تکرار
    نمایش
      REST API + Gin
      React Dashboard
      اعلان بلادرنگ + صدا
```

> ⚠️ **نکته تاریخی:** این پروژه از یک «BaseProject فروش خودرو» مشتق شده و طبق `RESTRUCTURE_PLAN.md` به NOTAM تبدیل شده. بقایای آن ساختار (pkg/logging، limiter، service_errors، metrics) هنوز موجود است.

---

## ۲. معماری سیستم

```mermaid
flowchart LR
    subgraph FAA["☁️ FAA SWIM"]
        S[("Solace Broker<br/>ems2.swim.faa.gov")]
    end

    subgraph App["🐹 Go Application (پورت 5005)"]
        direction TB
        C["Consumer<br/>(goroutine)"]
        R["NotamRepository"]
        AM["Alert Matcher"]
        API["Gin REST API"]
        C --> R
        R --> AM
    end

    subgraph Data["💾 Data Layer"]
        PG[("PostgreSQL<br/>notam_db")]
        RD[("Redis")]
    end

    subgraph Front["⚛️ React SPA (پورت 3000)"]
        L["Login"]
        D["Dashboard"]
        T["Toast + صدا"]
    end

    S -- "XML over TLS" --> C
    R --> PG
    AM --> PG
    API --> PG
    API -.-> RD
    D -- "REST /api/v1" --> API
    D -- "polling هر ۲ ثانیه" --> API
    API --> T
```

**دو بخش هم‌زمان در یک پروسه Go اجرا می‌شوند** ([main.go](src/main.go)):

| بخش | نوع اجرا | وظیفه |
|------|---------|--------|
| **Consumer** | goroutine پس‌زمینه | اتصال دائم به Solace، دریافت و ذخیره NOTAM |
| **API Server** | main thread | پاسخ به درخواست‌های REST فرانت‌اند |

---

## ۳. جریان داده (Data Flow)

```mermaid
sequenceDiagram
    participant FAA as Solace (FAA)
    participant CON as Consumer
    participant REPO as Repository
    participant DB as PostgreSQL
    participant API as REST API
    participant FE as React Dashboard

    FAA->>CON: پیام XML (NOTAM)
    CON->>CON: xml.Unmarshal → NotamEvent
    CON->>CON: ساخت متن ICAO فرمت‌شده
    CON->>CON: فیلتر: آیا FIR/فرودگاه مجاز است؟
    alt غیرمجاز
        CON-->>FAA: نادیده گرفتن (return)
    else مجاز
        CON->>REPO: Save(NotamMessage)
        REPO->>DB: بررسی تکراری بودن (message_id)
        alt جدید
            REPO->>DB: INSERT notam
            REPO->>DB: بررسی تنظیمات همه کاربران
            REPO->>DB: INSERT alert_delivery (اگر مطابقت)
        else موجود (R/C)
            REPO->>DB: UPDATE notam
        end
    end

    loop هر ۲ ثانیه
        FE->>API: GET /notams/recent
        API->>DB: alert_deliveries اخیر کاربر
        API-->>FE: NOTAMهای جدید
        FE->>FE: نمایش Toast + پخش صدا
    end

    loop هر ۸ ثانیه
        FE->>API: GET /notams (لیست با فیلتر)
        API->>DB: SELECT با pagination
        API-->>FE: لیست NOTAM
    end
```

---

## ۴. ساختار پوشه‌ها

```
BaseProject/
│
├── 📄 docker-compose.yml          # ارکستراسیون: consumer + postgres + pgadmin + redis + frontend
├── 📄 Readme.md
├── 📄 RESTRUCTURE_PLAN.md         # تاریخچه تبدیل از car-sale به NOTAM
│
├── 📁 src/                        # ── Backend (Go) ──
│   ├── main.go                    # نقطه ورود: Consumer + API
│   ├── Dockerfile
│   ├── go.mod / go.sum
│   │
│   ├── 📁 config/                 # Viper، سه محیط (dev/docker/prod)
│   │   ├── config.go
│   │   └── config-*.yml
│   │
│   ├── 📁 internal/               # منطق اصلی دامنه
│   │   ├── app/application.go     # ساختار Application (Consumer + Repo)
│   │   ├── messaging/
│   │   │   ├── solace_queue_consumer.go  # ⭐ اتصال Solace + پارس XML
│   │   │   ├── notam_filter.go           # لیست FIR/Airport مجاز
│   │   │   ├── message.go / consumer.go  # interfaceها
│   │   │   └── solace_message.go
│   │   └── storage/
│   │       ├── notam_repository.go       # ⭐ تبدیل event→model، regex series
│   │       ├── alert_match.go            # ⭐ منطق تطابق اعلان
│   │       └── fake_repository.go
│   │
│   ├── 📁 data/                   # لایه دسترسی به داده
│   │   ├── cache/redis.go
│   │   └── db/
│   │       ├── postgres.go
│   │       ├── model/             # Notam, Airport, Runway, AlertSettings, AlertDelivery
│   │       └── migrations/1_Init.go
│   │
│   ├── 📁 api/                    # لایه HTTP (Gin)
│   │   ├── api.go                 # راه‌اندازی سرور + روت‌ها + Swagger
│   │   ├── handlers/              # notam, auth, health
│   │   ├── routers/               # notam, auth, health
│   │   ├── middleware/            # cors, auth, logger, recovery, limiter
│   │   ├── helper/                # BaseResponse، status code mapping
│   │   └── validation/            # mobile, password, custom
│   │
│   ├── 📁 pkg/                    # ابزار مشترک (میراث BaseProject)
│   │   ├── logging/               # zap / zerolog
│   │   ├── limiter/               # rate limiter (IP)
│   │   ├── metrics/               # Prometheus (بلااستفاده)
│   │   └── service_errors/
│   │
│   └── 📁 docs/                   # Swagger تولیدشده
│
├── 📁 frontend/                   # ── Frontend (React + Vite + TS) ──
│   ├── package.json
│   ├── vite.config.ts             # proxy /api → :5005
│   └── src/
│       ├── App.tsx                # روتینگ + ToastContainer
│       ├── contexts/AuthContext.tsx
│       ├── api/client.ts          # ⭐ همه فراخوانی‌های API
│       ├── pages/                 # Login, Dashboard
│       ├── components/            # NotamList, FiltersForm, AlertSettings, AlertPopup...
│       └── utils/                 # alertSound, notamCancel
│
└── 📁 docker/
    └── redis/redis.conf
```

⭐ = فایل‌های هسته‌ی منطق کسب‌وکار

---

## ۵. مدل داده (Database)

```mermaid
erDiagram
    NOTAMS {
        int id PK
        string message_id UK "Solace Message ID"
        string series_number "A3910/25"
        string event_type "N / R / C"
        string q_line "Q) FIR/type/..."
        string location_icao "کد ICAO محل"
        timestamp effective_start
        timestamp effective_end
        text plain_text "متن اصلی E)"
        text formatted_text "خروجی ICAO کامل"
        text raw_body "XML خام"
        string airport_icao
        string airport_name
        string affected_fir
        string lower_limit "F)"
        string upper_limit "G)"
        int runway_id FK
    }
    AIRPORTS {
        int id PK
        string icao UK
        string name
        float lat
        float lon
    }
    RUNWAYS {
        int id PK
        string airport_icao FK
        string name "13R/31L"
        string le_ident
        string he_ident
    }
    NOTAM_ALERT_SETTINGS {
        int id PK
        string username UK
        json selected_firs
        json selected_airports
        json selected_keywords
        json custom_keywords
    }
    NOTAM_ALERT_DELIVERIES {
        int id PK
        string username "idx"
        int notam_id "idx"
        timestamp created_at
    }

    AIRPORTS ||--o{ RUNWAYS : "دارد"
    RUNWAYS ||--o{ NOTAMS : "ارجاع اختیاری"
    NOTAM_ALERT_SETTINGS ||..o{ NOTAM_ALERT_DELIVERIES : "username (بدون FK)"
    NOTAMS ||..o{ NOTAM_ALERT_DELIVERIES : "notam_id (بدون FK)"
```

**نکات کلیدی مدل:**
- مدل `Notam` دقیقاً منطبق بر فیلدهای استاندارد **ICAO** است (بندهای Q, A, B, C, D, E, F, G).
- `event_type`: `N`=جدید، `R`=جایگزین (Replace)، `C`=لغو (Cancel).
- بین NOTAM و Airport **کلید خارجی نیست** — چون NOTAMها از هزاران فرودگاه FAA می‌آیند که لزوماً در جدول airports نیستند.
- تنظیمات و تحویل اعلان بر اساس **username رشته‌ای** است (بدون جدول users واقعی).

---

## ۶. API Endpoints

**Base URL:** `http://localhost:5005/api/v1`
**فرمت پاسخ استاندارد:** `BaseHttpResponse { success, result, error, resultCode }`

| متد | مسیر | Auth | توضیح |
|-----|------|:----:|-------|
| `GET`  | `/health/` | ❌ | health check |
| `POST` | `/auth/login` | ❌ | لاگین → توکن |
| `GET`  | `/notams` | ❌ | لیست با فیلتر و صفحه‌بندی |
| `GET`  | `/notams/:id` | ❌ | یک NOTAM با ID |
| `GET`  | `/notams/by-series` | ❌ | یک NOTAM با شماره سریال |
| `GET`  | `/notams/alert-options` | ❌ | لیست FIR/فرودگاه مجاز |
| `GET`  | `/notams/alert-settings` | ✅ | خواندن تنظیمات اعلان کاربر |
| `PUT`  | `/notams/alert-settings` | ✅ | ذخیره تنظیمات اعلان کاربر |
| `GET`  | `/notams/recent` | ✅ | NOTAMهای تحویل‌شده اخیر (برای polling) |

**فیلترهای `GET /notams`:** `seriesNumber`, `eventType`, `locationIcao`, `airportIcao`, `airportName`, `affectedFir`, `plainText`, `from`, `to`, `limit`, `offset` (هم camelCase هم snake_case پشتیبانی می‌شود).

📖 مستندات تعاملی: `http://localhost:5005/swagger/index.html`

```mermaid
flowchart TD
    A[درخواست] --> B{مسیر محافظت‌شده؟}
    B -->|خیر| H[Handler]
    B -->|بله| M[NotamAuth middleware]
    M --> C{"Bearer notam-token-*<br/>معتبر؟"}
    C -->|خیر| E[401 Unauthorized]
    C -->|بله| U[استخراج username → context]
    U --> H
    H --> R["BaseHttpResponse (JSON)"]
```

---

## ۷. منطق Consumer و فیلتر

فایل: [solace_queue_consumer.go](src/internal/messaging/solace_queue_consumer.go)

```mermaid
flowchart TD
    START([پیام XML از Solace]) --> P[xml.Unmarshal → NotamEvent]
    P --> H{"متن انسانی<br/>کامل است؟"}
    H -->|خیر| BUILD["ساخت دستی متن ICAO<br/>(Q, A, B/C, D, E, F, G)"]
    H -->|بله| USE[استفاده از formattedText]
    BUILD --> FILTER
    USE --> FILTER{"FIR یا فرودگاه<br/>در لیست مجاز؟"}
    FILTER -->|خیر| DROP([نادیده گرفتن])
    FILTER -->|بله| TYPE["تعیین نوع:<br/>Airport / FIR-level NOTAM"]
    TYPE --> FICON{"شامل FICON؟"}
    FICON --> LOG[لاگ کامل NOTAM]
    LOG --> HANDLER["handler() → Repo.Save()"]
    HANDLER --> DONE([ذخیره در DB])
```

**فیلتر مجاز** ([notam_filter.go](src/internal/messaging/notam_filter.go)):
- `AllowedFIRs` — حدود ۸۰ کد FIR (خاورمیانه، آسیا، آفریقا، اروپا، اروپای شرقی)
- `AllowedAirports` — حدود ۳۰۰+ کد ICAO فرودگاه
- فقط NOTAMهایی که به این مناطق مربوط‌اند ذخیره می‌شوند.

**پیچیدگی استخراج Series Number** ([notam_repository.go](src/internal/storage/notam_repository.go)) — سه لایه fallback:
1. از فیلدهای XML ساختاریافته (`series` + `number` + `year`)
2. با regex از متن (`A1477/26`, `0046/26`, `M0137/26`)
3. از `xovernotamID` در XML خام (برای NOTAMهای R/C)

---

## ۸. منطق اعلان (Alert Matching)

فایل: [alert_match.go](src/internal/storage/alert_match.go)

هنگام ذخیره‌ی هر NOTAM جدید، تنظیمات **همه‌ی کاربران** بررسی و در صورت تطابق، یک رکورد `alert_delivery` ثبت می‌شود.

```mermaid
flowchart TD
    N([NOTAM جدید]) --> LOAD[خواندن تنظیمات همه کاربران]
    LOAD --> LOOP{برای هر کاربر}
    LOOP --> HL{"مکان انتخاب شده؟<br/>(FIR/فرودگاه)"}

    HL -->|بله| LM{"NOTAM با مکان<br/>مطابقت دارد؟"}
    LM -->|خیر| SKIP[رد شو]
    LM -->|بله| HK1{"کلیدواژه دارد؟"}
    HK1 -->|خیر| DELIVER[✅ ثبت تحویل]
    HK1 -->|بله| KW{"کلیدواژه در<br/>متن NOTAM هست؟"}
    KW -->|بله| DELIVER
    KW -->|خیر| SKIP

    HL -->|خیر| HK2{"فقط کلیدواژه؟"}
    HK2 -->|خیر| SKIP
    HK2 -->|بله| SKIP2["رد (منطق فعلی نیاز به<br/>مکان دارد)"]

    DELIVER --> LOOP
    SKIP --> LOOP
```

**قانون:** برای اعلان، NOTAM باید **هم** در محدوده‌ی FIR/فرودگاه انتخابی باشد **و** (در صورت انتخاب کلیدواژه) حداقل یک کلیدواژه در متنش وجود داشته باشد.

**کلیدواژه‌های پیش‌فرض:** `AD CLSD`, `RWY`, `ILS`, `GPS`, `SID`, `STAR`, `VOR`, `DME`, `SECTOR` + کلیدواژه‌های سفارشی کاربر.

---

## ۹. Frontend

**استک:** React 18 + Vite 5 + TypeScript + React Router 6 + react-toastify — بدون کتابخانه state management (فقط Context + localStorage).

```mermaid
flowchart TD
    APP[App.tsx] --> AUTH{AuthContext}
    AUTH -->|بدون توکن| LOGIN[صفحه Login]
    AUTH -->|با توکن| DASH[Dashboard]

    DASH --> HEAD["Header<br/>(کاربر، تست نوتیف، تنظیمات، خروج)"]
    DASH --> FILT["پنل فیلترها<br/>NotamFiltersForm"]
    DASH --> MAIN["پنل اصلی"]
    MAIN --> PAGIN[PaginationBar]
    MAIN --> LIST["NotamList<br/>(کارت‌های NOTAM)"]
    DASH --> MODAL["AlertSettingsModal<br/>(انتخاب FIR/فرودگاه/کلیدواژه)"]
    DASH --> POPUP["NotamAlertPopup<br/>(جزئیات NOTAM لغو/جایگزین)"]

    DASH -.->|"polling ۲ثانیه"| TOAST["Toast اعلان + صدا"]
    DASH -.->|"refresh ۸ثانیه"| LIST
```

**قابلیت‌های کلیدی داشبورد** ([Dashboard.tsx](frontend/src/pages/Dashboard.tsx)):
- ✅ لیست NOTAM با فیلتر و صفحه‌بندی
- 🔄 به‌روزرسانی خودکار لیست هر **۸ ثانیه**
- 🔔 polling اعلان از `/recent` هر **۲ ثانیه** → نمایش Toast
- 🔊 پخش **صدای هشدار** (نیاز به فعال‌سازی کاربر به‌دلیل محدودیت مرورگر)
- 🎨 رنگ Toast بر اساس نوع: سبز=جدید، زرد=جایگزین، قرمز=لغو
- 🔍 باز کردن NOTAM لغو/جایگزین‌شده در پاپ‌آپ
- 🌐 رابط کاربری **راست‌به‌چپ (فارسی)**

---

## ۱۰. زیرساخت و اجرا

```mermaid
flowchart LR
    subgraph compose["docker-compose.yml"]
        FE["frontend<br/>nginx :3000"]
        BE["notam-consumer<br/>:5005"]
        PG["postgres :5434→5432"]
        PGA["pgadmin :8091"]
        RD["redis :6379"]
    end
    FE --> BE
    BE --> PG
    BE --> RD
    PGA --> PG
```

**اجرا:**
```bash
docker compose up -d --build
```

| سرویس | پورت | آدرس |
|-------|------|------|
| Frontend | 3000 | http://localhost:3000 |
| Backend API | 5005 | http://localhost:5005/api/v1 |
| Swagger | 5005 | http://localhost:5005/swagger/index.html |
| PostgreSQL | 5434 | — |
| pgAdmin | 8091 | http://localhost:8091 |
| Redis | 6379 | — |

**اجرای dev فرانت:** `cd frontend && npm run dev` (پورت 3000، proxy به 5005)

**متغیرهای محیطی مهم:** `APP_ENV`, `SOLACE_HOST/VPN/USERNAME/PASSWORD/QUEUE`, `AUTH_USER`, `AUTH_PASS`

---

## ۱۱. تکنولوژی‌ها

```mermaid
flowchart TB
    subgraph BE["Backend — Go 1.24"]
        GIN[Gin — HTTP]
        GORM[GORM — ORM]
        SOL[solace.dev/go/messaging]
        VIP[Viper — config]
        ZAP[zap / zerolog — log]
        SWAG[Swaggo — Swagger]
        JWT["golang-jwt (بلااستفاده)"]
        PROM["Prometheus (بلااستفاده)"]
    end
    subgraph FE["Frontend"]
        REACT[React 18]
        VITE[Vite 5]
        TS[TypeScript]
        RR[React Router 6]
        TOAST[react-toastify]
    end
    subgraph INFRA["Infrastructure"]
        PGX[PostgreSQL 16]
        REDIS[Redis 7]
        DOCK[Docker Compose]
        NGINX[Nginx]
    end
```

---

## ۱۲. نقاط ضعف و بدهی فنی

| # | شدت | موضوع | توضیح |
|---|:---:|-------|-------|
| 1 | 🔴 | **احراز هویت ضعیف** | رمز `admin/admin` هاردکد، توکن ثابت `notam-token-<user>` بدون JWT/انقضا؛ هرکس می‌تواند توکن جعل کند |
| 2 | 🔴 | **افشای credential** | یوزر/پسورد Solace و پسورد DB مستقیم در `main.go` و `docker-compose.yml` کامیت شده |
| 3 | 🟡 | **اعلان مبتنی بر polling** | فرانت هر ۲ ثانیه `/recent` را صدا می‌زند؛ به‌جای WebSocket/SSE |
| 4 | 🟡 | **بار روی DB** | `evaluateAlertDeliveries` برای هر NOTAM، تنظیمات همه کاربران را می‌خواند |
| 5 | 🟡 | **بلااستفاده‌ها** | Redis، limiter، metrics تعریف شده‌اند اما عملاً استفاده نمی‌شوند |
| 6 | 🟡 | **پارس XML شکننده** | استخراج series با چند regex و fallback، بدون تست کافی |
| 7 | 🟢 | **تک‌کاربره عملی** | نبود جدول users واقعی؛ تنظیمات بر اساس رشته‌ی username |
| 8 | 🟢 | **نبود تست** | تست واحد/یکپارچه دیده نشد |

---

## ۱۳. نقشه راه پیشنهادی

```mermaid
flowchart LR
    subgraph P1["فاز ۱ — امن‌سازی"]
        A1[جدول Users + هش رمز]
        A2[JWT با انقضا]
        A3[انتقال secrets به env/vault]
    end
    subgraph P2["فاز ۲ — بلادرنگ"]
        B1[WebSocket/SSE برای اعلان]
        B2[حذف polling]
    end
    subgraph P3["فاز ۳ — مدیریت"]
        C1[پنل مدیریت FIR/فرودگاه]
        C2[نقش‌ها و دسترسی]
    end
    subgraph P4["فاز ۴ — بلوغ"]
        D1[تست واحد/یکپارچه]
        D2[نقشه و آمار]
        D3[بهینه‌سازی Alert Matching]
    end
    P1 --> P2 --> P3 --> P4
```

**اولویت پیشنهادی برای شروع توسعه:**
1. 🔐 **امن‌سازی** (Users واقعی + JWT + خارج‌کردن secrets)
2. ⚡ **WebSocket** برای اعلان بلادرنگ به‌جای polling
3. 🛠️ **پنل مدیریت** فرودگاه/FIR (به‌جای لیست هاردکد در کد)
4. 🧪 **تست و مانیتورینگ**

---

> این مستند از بررسی مستقیم کد تولید شده و ممکن است با تغییر پروژه نیاز به به‌روزرسانی داشته باشد.
