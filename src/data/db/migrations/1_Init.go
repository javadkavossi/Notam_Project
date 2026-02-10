package migrations

import (
	"github.com/hossein-repo/BaseProject/config"
	"github.com/hossein-repo/BaseProject/data/db"
	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/pkg/logging"
	"gorm.io/gorm"
)

var logger = logging.NewLogger(config.GetConfig())

func Up_1() {
	database := db.GetDb()
	createTables(database)
}

func createTables(database *gorm.DB) {
	// حذف FKهای قبلی - NOTAMها از فرودگاه‌های مختلف FAA می‌آیند، نیازی به FK نیست
	_ = database.Exec("ALTER TABLE notams DROP CONSTRAINT IF EXISTS fk_notams_airport").Error
	_ = database.Exec("ALTER TABLE notams DROP CONSTRAINT IF EXISTS fk_notams_runway").Error

	tables := []interface{}{
		model.Airport{},
		model.Runway{},
		model.Notam{},
		model.NotamAlertSettings{},
		model.NotamAlertDelivery{},
	}
	if err := database.AutoMigrate(tables...); err != nil {
		logger.Error(logging.Postgres, logging.Migration, err.Error(), nil)
	} else {
		logger.Info(logging.Postgres, logging.Migration, "NOTAM tables migrated", nil)
	}
}
