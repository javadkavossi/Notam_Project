package reference

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"

	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/reference/etl"
)

// LoadResult خلاصهٔ یک بارگذاری.
type LoadResult struct {
	Dataset  string
	Changed  bool
	RowCount int
}

// LoadAirports فایل airports.csv را می‌خواند و در صورت تغییر (checksum) upsert می‌کند (E7-2/E7-4).
func (s *Store) LoadAirports(path string) (LoadResult, error) {
	return s.loadDataset("airports", path, func(data []byte) (int, error) {
		airports, err := etl.ParseAirports(bytes.NewReader(data))
		if err != nil {
			return 0, err
		}
		return len(airports), s.UpsertAirports(airports)
	})
}

// LoadRunways فایل runways.csv را می‌خواند و جایگزین می‌کند.
func (s *Store) LoadRunways(path string) (LoadResult, error) {
	return s.loadDataset("runways", path, func(data []byte) (int, error) {
		rws, err := etl.ParseRunways(bytes.NewReader(data))
		if err != nil {
			return 0, err
		}
		return len(rws), s.ReplaceRunways(rws)
	})
}

// LoadNavaids فایل navaids.csv را می‌خواند و جایگزین می‌کند.
func (s *Store) LoadNavaids(path string) (LoadResult, error) {
	return s.loadDataset("navaids", path, func(data []byte) (int, error) {
		nv, err := etl.ParseNavaids(bytes.NewReader(data))
		if err != nil {
			return 0, err
		}
		return len(nv), s.ReplaceNavaids(nv)
	})
}

// loadDataset الگوی مشترک: خواندن فایل، بررسی تغییر با checksum، اعمال، و ثبت نسخه.
func (s *Store) loadDataset(name, path string, apply func([]byte) (int, error)) (LoadResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LoadResult{Dataset: name}, err
	}
	sum := checksum(data)

	// تشخیص تغییر: اگر checksum با آخرین نسخه یکسان بود، کاری نکن (E7-4).
	if last := s.lastVersion(name); last != nil && last.Checksum == sum {
		log.Printf("📚 %s بدون تغییر (checksum یکسان)؛ بارگذاری رد شد", name)
		return LoadResult{Dataset: name, Changed: false, RowCount: last.RowCount}, nil
	}

	count, err := apply(data)
	if err != nil {
		return LoadResult{Dataset: name}, err
	}
	s.recordVersion(name, sum, count)
	log.Printf("📚 %s بارگذاری شد: %d ردیف (checksum %s…)", name, count, sum[:8])
	return LoadResult{Dataset: name, Changed: true, RowCount: count}, nil
}

func (s *Store) lastVersion(dataset string) *model.RefDatasetVersion {
	var v model.RefDatasetVersion
	if err := s.db.Where("dataset = ?", dataset).Order("id DESC").First(&v).Error; err != nil {
		return nil
	}
	return &v
}

func (s *Store) recordVersion(dataset, sum string, count int) {
	_ = s.db.Create(&model.RefDatasetVersion{Dataset: dataset, Checksum: sum, RowCount: count}).Error
}

func checksum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
