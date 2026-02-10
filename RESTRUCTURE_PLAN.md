# برنامه ساختاربندی مجدد پروژه BaseProject (NOTAM Consumer)

## ✅ انجام شد

---

## وضعیت قبلی
پروژه ترکیبی از دو پروژه است:
1. **پروژه اصلی (NOTAM)**: `src/main.go` - مصرف‌کننده Solace، ذخیره NOTAM
2. **پروژه فرعی (Car Sale API)**: `src/cmd/main_after.go` - API فروش خودرو با Gin، Redis، Postgres

---

## مرحله ۱: حذف سرویس‌های Docker غیرضروری
**حذف:**
- `docker/elk/` (Elasticsearch, Kibana, Filebeat, setup)
- `docker/grafana/`
- `docker/prometheus/`
- `docker/alertmanager/`
- `docker/node-exporter` (در docker-compose)

**نگه‌داری:**
- `docker/redis/` (فایل redis.conf)

---

## مرحله ۲: یکپارچه‌سازی Docker Compose
**فایل اصلی:** `docker-compose.yml` در **ریشه پروژه** (جایگزین src/docker-compose.yml و docker/docker-compose.yml)

**سرویس‌ها:**
- `notam-consumer` (build از src/)
- `postgres`
- `pgadmin`
- `redis`

---

## مرحله ۳: حذف کد مربوط به Car Sale
**حذف کامل:**
- `src/cmd/` (main_after.go)
- `src/api/handlers/` — همه به جز health.go
- `src/api/routers/` — همه به جز health.go و basic.go (ساده)
- `src/api/dto/` — car, property, user, filter
- `src/services/` — کل پوشه
- `src/data/db/model/` — car.go, property.go, user.go (نگه‌داری: notam, airport, runway, base)
- `src/data/db/migrations/default_values.go` (car properties)
- `src/constants/` — اگر فقط برای car
- `src/common/` — اگر فقط برای car (persian.go, strings.go ممکن است استفاده شود)

**ساده‌سازی API:**
- نگه‌داری: health, swagger, middleware پایه (cors, logger, recovery)
- حذف: Prometheus middleware، auth، car routers
- ایجاد api مینیمال برای آینده NOTAM

---

## مرحله ۴: ساختار نهایی پروژه
```
BaseProject/
├── docker-compose.yml          # اصلی
├── .dockerignore
├── .gitignore
├── Readme.md
├── RESTRUCTURE_PLAN.md
├── docker/
│   └── redis/
│       └── redis.conf
└── src/
    ├── main.go                 # نقطه ورود NOTAM consumer
    ├── Dockerfile
    ├── go.mod
    ├── go.sum
    ├── config/
    │   ├── config.go
    │   ├── config-development.yml
    │   ├── config-docker.yml
    │   └── config-production.yml
    ├── internal/
    │   ├── app/
    │   ├── messaging/
    │   └── storage/
    ├── data/
    │   ├── cache/ (redis)
    │   └── db/
    │       ├── postgres.go
    │       ├── model/ (notam, airport, runway, base)
    │       └── migrations/
    ├── pkg/
    │   ├── logging/
    │   ├── limiter/
    │   └── service_errors/
    ├── api/                    # مینیمال
    │   ├── api.go
    │   ├── handlers/health.go
    │   ├── routers/health.go
    │   ├── middleware/ (cors, logger, recovery)
    │   └── helper/
    ├── docs/ (swagger)
    ├── third_party/
    └── logs/
```

---

## مرحله ۵: تغییرات Config
- نام دیتابیس: `car_sale_db` → `notam_db`
- hostهای Docker: `postgres_container` → `postgres`, `redis_container` → `redis`

---

## مرحله ۶: Migrations
- حذف seedهای car/country/city
- نگه‌داری فقط: notam, airport, runway
- ساده‌سازی 1_Init.go

---

## وابستگی‌های Go (go.mod)
- نگه‌داری: gin, redis, gorm, postgres, swaggo, solace, zap, viper
- حذف optional: prometheus (اگر از api حذف شود)
