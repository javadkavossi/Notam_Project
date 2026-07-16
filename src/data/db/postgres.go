package db

import (
	"fmt"
	"time"

	"github.com/hossein-repo/BaseProject/config"
	"github.com/hossein-repo/BaseProject/pkg/logging"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var dbClient *gorm.DB

func InitDb(cfg *config.Config) error {
	var err error
	cnn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Tehran",
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password,
		cfg.Postgres.DbName, cfg.Postgres.SSLMode)

	// FKها هنگام مهاجرت ساخته نمی‌شوند: دادهٔ مرجع/NOTAM از منابع بیرونی می‌آید و ممکن است
	// به رکوردهایی اشاره کند که ما فیلترشان کرده‌ایم؛ FK کل بارگذاری را می‌شکند.
	dbClient, err = gorm.Open(postgres.Open(cnn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return err
	}

	sqlDb, _ := dbClient.DB()
	err = sqlDb.Ping()
	if err != nil {
		return err
	}

	sqlDb.SetMaxIdleConns(cfg.Postgres.MaxIdleConns)
	sqlDb.SetMaxOpenConns(cfg.Postgres.MaxOpenConns)
	sqlDb.SetConnMaxLifetime(cfg.Postgres.ConnMaxLifetime * time.Minute)

	// لاگر اینجا ساخته می‌شود (نه در سطح پکیج) تا صرفِ import کردن این پکیج،
	// خواندن فایل config را اجباری نکند — در غیر این صورت تست‌های واحدِ
	// پکیج‌های وابسته هنگام init با Fatal شکست می‌خورند.
	logging.NewLogger(cfg).Info(logging.Postgres, logging.Startup, "Db connection established", nil)
	return nil
}

func GetDb() *gorm.DB {
	return dbClient
}

func CloseDb() {
	con, _ := dbClient.DB()
	con.Close()
}
