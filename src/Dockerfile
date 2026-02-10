# Solace go/messaging به glibc نیاز دارد (__strdup) - Alpine/musl سازگار نیست
FROM golang:1.24-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends git gcc libc6-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum ./

ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=off

RUN go mod download

# اول vendor رو پاک کن اگر هست
RUN rm -rf vendor
COPY . .

# Solace go/messaging به CGO نیاز دارد و کتابخانه‌های C خود را همراه دارد
RUN CGO_ENABLED=1 GOOS=linux go build -o notam-consumer ./main.go

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata openssl libc6 && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/notam-consumer .
COPY --from=builder /app/config ./config
RUN mkdir -p logs
ENTRYPOINT ["./notam-consumer"]