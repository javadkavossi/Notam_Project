# FAA NOTAM Consumer

پروژه مصرف‌کننده NOTAM از FAA Solace و ذخیره در PostgreSQL.

## ساختار پروژه

```
BaseProject/
├── docker-compose.yml    # Docker اصلی (notam-consumer, postgres, pgadmin, redis)
├── docker/
│   └── redis/
│       └── redis.conf
└── src/
    ├── main.go           # نقطه ورود (Consumer + API)
    ├── Dockerfile
    ├── internal/         # messaging, storage, app
    ├── data/             # db, cache
    ├── api/              # health, swagger
    └── config/
```

## اجرا با Docker

```bash
docker compose up -d --build
```

سرویس‌ها:
- **notam-consumer** (API + Consumer): پورت 5005
- **postgres**: پورت 5432
- **pgadmin**: پورت 8090
- **redis**: پورت 6379

## API & Swagger

- Health: `GET /api/v1/health/`
- Swagger: `GET /swagger/index.html`

## متغیرهای محیطی

| متغیر | توضیح |
|-------|-------|
| SOLACE_HOST | آدرس Solace |
| SOLACE_VPN | نام VPN |
| SOLACE_USERNAME | نام کاربری |
| SOLACE_PASSWORD | رمز عبور |
| SOLACE_QUEUE | نام صف |
| APP_ENV | development / docker / production |
# Notam_Project
