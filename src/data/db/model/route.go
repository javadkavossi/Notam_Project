package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// route.go — مسیر واقعی (waypoint) و پروفایل ارتفاعی segment-based برای دقت تداخل (E5.6+).
// این‌ها به‌صورت JSON در ستون TEXT ذخیره می‌شوند (backward-compatible؛ FlightPlanهای قدیمی nil دارند).

// فازهای پرواز روی هر segment.
const (
	PhaseClimb   = "CLIMB"
	PhaseCruise  = "CRUISE"
	PhaseDescent = "DESCENT"
	PhaseUnknown = "UNKNOWN"
)

// Waypoint یک نقطهٔ مسیر با ترتیب و مختصات.
type Waypoint struct {
	Sequence   int     `json:"sequence"`
	Identifier string  `json:"identifier,omitempty"`
	Lat        float64 `json:"latitude"`
	Lon        float64 `json:"longitude"`
}

// Valid مختصات را اعتبارسنجی می‌کند (بدون جعل: نقطهٔ نامعتبر باید علامت بخورد، نه نادیده گرفته شود).
func (w Waypoint) Valid() bool {
	return w.Lat >= -90 && w.Lat <= 90 && w.Lon >= -180 && w.Lon <= 180 &&
		!(w.Lat == 0 && w.Lon == 0)
}

// AltSegment بازهٔ ارتفاعی یک بخش از مسیر (بر اساس ترتیب waypointها).
type AltSegment struct {
	FromSequence int    `json:"fromSequence"`
	ToSequence   int    `json:"toSequence"`
	LowerFt      int    `json:"lowerFt"`
	UpperFt      int    `json:"upperFt"`
	Phase        string `json:"phase,omitempty"`
}

// Waypoints آرایهٔ waypoint با ذخیرهٔ JSON.
type Waypoints []Waypoint

func (s Waypoints) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	return json.Marshal(s)
}

func (s *Waypoints) Scan(value interface{}) error { return scanJSON(value, s) }

// AltProfile آرایهٔ بازه‌های ارتفاعی با ذخیرهٔ JSON.
type AltProfile []AltSegment

func (s AltProfile) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	return json.Marshal(s)
}

func (s *AltProfile) Scan(value interface{}) error { return scanJSON(value, s) }

func scanJSON(value, dest interface{}) error {
	if value == nil {
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("invalid type for JSON scan")
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, dest)
}
