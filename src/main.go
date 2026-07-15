package main

import (
	"fmt"
	"log"

	"github.com/hossein-repo/BaseProject/api"
	"github.com/hossein-repo/BaseProject/config"
	"github.com/hossein-repo/BaseProject/data/cache"
	"github.com/hossein-repo/BaseProject/data/db"
	"github.com/hossein-repo/BaseProject/data/db/migrations"
	"github.com/hossein-repo/BaseProject/internal/app"
	"github.com/hossein-repo/BaseProject/internal/messaging"
	"github.com/hossein-repo/BaseProject/internal/storage"
	"github.com/hossein-repo/BaseProject/pkg/logging"
)

func main() {
	cfg := config.GetConfig()
	logger := logging.NewLogger(cfg)

	// Redis
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

	// Repository - ذخیره NOTAM در PostgreSQL با ساختار ICAO
	repo := storage.NewNotamRepository()

	// Solace consumer — همهٔ مقادیر از config/env (E0-1: بدون credential هاردکد)
	sc := cfg.Solace
	if sc.Username == "" || sc.Password == "" || sc.Queue == "" {
		logger.Fatal(logging.General, logging.Startup,
			"Solace credentials missing: set SOLACE_USERNAME, SOLACE_PASSWORD and SOLACE_QUEUE (see .env.example)", nil)
	}
	consumer := messaging.NewSolaceQueueConsumer(sc.Host, sc.VPN, sc.Username, sc.Password, sc.Queue)
	app := app.Application{
		Consumer: consumer,
		Repo:     repo,
	}

	// Start NOTAM consumer in background
	go func() {
		err := app.Consumer.Start(func(msg messaging.Message) {
			fmt.Printf("📨 %s | %s\n", msg.Type(), msg.ID())
			app.Repo.Save(msg)
		})
		if err != nil {
			log.Fatal(err)
		}
		defer consumer.Close()
		select {} // keep alive
	}()

	// Start API server (health + swagger)
	api.InitServer(cfg)
}
