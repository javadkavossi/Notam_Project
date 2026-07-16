package reference

import (
	"bytes"
	"os"

	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/reference/etl"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LoadFIRs مرزهای FIR را از GeoJSON می‌خواند، upsert می‌کند و ستون geometry (PostGIS) را می‌سازد (E7-3).
func (s *Store) LoadFIRs(path string) (LoadResult, error) {
	return s.loadDataset("firs", path, func(data []byte) (int, error) {
		feats, err := etl.ParseFIRs(bytes.NewReader(data))
		if err != nil {
			return 0, err
		}
		firs := etl.ToFIRModels(feats)
		if err := s.upsertFIRs(firs); err != nil {
			return 0, err
		}
		return len(firs), nil
	})
}

func (s *Store) upsertFIRs(firs []model.FIR) error {
	if len(firs) == 0 {
		return nil
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "icao"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "boundary_geojson", "updated_at"}),
		}).CreateInBatches(&firs, upsertBatch).Error; err != nil {
			return err
		}
		// ساخت ستون geometry از GeoJSON (نیازمند PostGIS). SRID 4326.
		for _, f := range firs {
			if err := tx.Exec(
				"UPDATE firs SET boundary = ST_SetSRID(ST_GeomFromGeoJSON(?), 4326) WHERE icao = ?",
				f.BoundaryGeoJSON, f.ICAO,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// LoadReferenceData همهٔ دیتاست‌های مرجع را از مسیرهای داده‌شده بارگذاری می‌کند (مسیر خالی = رد شود).
// برای اجرای دستی/زمان‌بندی‌شده (E7). فقط دیتاست‌های تغییر‌کرده اعمال می‌شوند.
func (s *Store) LoadReferenceData(airportsPath, runwaysPath, navaidsPath, firsPath string) []LoadResult {
	var results []LoadResult
	load := func(path string, fn func(string) (LoadResult, error)) {
		if path == "" {
			return
		}
		if _, err := os.Stat(path); err != nil {
			return
		}
		if res, err := fn(path); err == nil {
			results = append(results, res)
		}
	}
	load(airportsPath, s.LoadAirports)
	load(runwaysPath, s.LoadRunways)
	load(navaidsPath, s.LoadNavaids)
	load(firsPath, s.LoadFIRs)
	return results
}
