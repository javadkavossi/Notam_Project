package main

import (
	"context"
	"os"

	"github.com/hossein-repo/BaseProject/api"
	"github.com/hossein-repo/BaseProject/config"
	"github.com/hossein-repo/BaseProject/data/cache"
	"github.com/hossein-repo/BaseProject/data/db"
	"github.com/hossein-repo/BaseProject/data/db/migrations"
	"github.com/hossein-repo/BaseProject/data/stream"
	"github.com/hossein-repo/BaseProject/internal/ingest"
	solaceadapter "github.com/hossein-repo/BaseProject/internal/ingest/solace"
	"github.com/hossein-repo/BaseProject/internal/pipeline"
	"github.com/hossein-repo/BaseProject/internal/storage"
	"github.com/hossein-repo/BaseProject/pkg/logging"
)

func main() {
	cfg := config.GetConfig()
	logger := logging.NewLogger(cfg)

	// Redis (کش + استریم داخلی)
	if err := cache.InitRedis(cfg, logger); err != nil {
		logger.Fatal(logging.Redis, logging.Startup, err.Error(), nil)
	}
	defer cache.CloseRedis(logger)

	// Postgres
	if err := db.InitDb(cfg); err != nil {
		logger.Fatal(logging.Postgres, logging.Startup, err.Error(), nil)
	}
	defer db.CloseDb()
	migrations.Up_1()

	repo := storage.NewNotamRepository()

	// استریم داخلی روی Redis (E0-5)
	streamClient := stream.New(cache.GetRedis())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ---- Pipeline: مصرف استریم → پردازش → ذخیرهٔ idempotent (E2-2/E2-3) ----
	consumerName := "pipeline-" + hostnameOr("1")
	runner := pipeline.NewRunner(streamClient, repo, consumerName)
	go func() {
		if err := runner.Run(ctx); err != nil {
			logger.Error(logging.General, logging.Startup, "pipeline runner stopped: "+err.Error(), nil)
		}
	}()

	// ---- Ingest: دریافت از منبع → نوشتن در استریم با client-ack (E1-2) ----
	sc := cfg.Solace
	if sc.Username == "" || sc.Password == "" || sc.Queue == "" {
		logger.Fatal(logging.General, logging.Startup,
			"Solace credentials missing: set SOLACE_USERNAME, SOLACE_PASSWORD and SOLACE_QUEUE (see .env.example)", nil)
	}
	adapter := solaceadapter.New(sc.Host, sc.VPN, sc.Username, sc.Password, sc.Queue)

	// emit: پیام خام را در استریم می‌نویسد؛ فقط در صورت موفقیت آداپتور به منبع ack می‌دهد.
	emit := func(raw ingest.RawNotamMessage) error {
		_, err := streamClient.Publish(pipeline.StreamNotamRaw, pipeline.StreamValues(raw), 100000)
		return err
	}
	go func() {
		if err := adapter.Start(ctx, emit); err != nil {
			logger.Fatal(logging.General, logging.Startup, "ingest adapter failed: "+err.Error(), nil)
		}
	}()
	defer adapter.Close()

	// Start API server (health + swagger) — بلاک‌کننده
	api.InitServer(cfg)
}

func hostnameOr(fallback string) string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return fallback
}
