package main

import (
	"fmt"
	"log"
	"os"

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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

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

	// Solace consumer
	host := getEnv("SOLACE_HOST", "tcps://ems2.swim.faa.gov:55443")
	vpn := getEnv("SOLACE_VPN", "AIM_FNS")
	username := getEnv("SOLACE_USERNAME", "hossein.kavosi2.gmail.com")
	password := getEnv("SOLACE_PASSWORD", "GK5F9tZFRnqhucFniZhoOw")
	queue := getEnv("SOLACE_QUEUE", "hossein.kavosi2.gmail.com.AIM_FNS.1696ec1b-7b8d-41e3-8e96-90f62d821170.OUT")

	consumer := messaging.NewSolaceQueueConsumer(host, vpn, username, password, queue)
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
