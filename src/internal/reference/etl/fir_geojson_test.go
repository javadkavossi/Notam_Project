package etl

import (
	"strings"
	"testing"
)

const firGeoJSON = `{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "properties": {"icao": "OIIX", "name": "Tehran FIR"},
      "geometry": {"type": "Polygon", "coordinates": [[[50,35],[52,35],[52,37],[50,37],[50,35]]]}
    },
    {
      "type": "Feature",
      "properties": {"name": "No ICAO"},
      "geometry": {"type": "Polygon", "coordinates": [[[0,0],[1,0],[1,1],[0,0]]]}
    }
  ]
}`

func TestParseFIRs(t *testing.T) {
	feats, err := ParseFIRs(strings.NewReader(firGeoJSON))
	if err != nil {
		t.Fatalf("ParseFIRs: %v", err)
	}
	// فیچر بدون ICAO باید رد شود
	if len(feats) != 1 {
		t.Fatalf("انتظار ۱ FIR، دریافت %d", len(feats))
	}
	f := feats[0]
	if f.ICAO != "OIIX" || f.Name != "Tehran FIR" {
		t.Errorf("فیلدها نادرست: %+v", f)
	}
	if !strings.Contains(f.GeometryJSON, "Polygon") {
		t.Errorf("geometry حفظ نشد: %s", f.GeometryJSON)
	}

	firs := ToFIRModels(feats)
	if len(firs) != 1 || firs[0].BoundaryGeoJSON == "" {
		t.Errorf("تبدیل به مدل نادرست: %+v", firs)
	}
}

func TestParseFIRs_Invalid(t *testing.T) {
	if _, err := ParseFIRs(strings.NewReader(`{"type":"Point"}`)); err == nil {
		t.Error("انتظار خطا برای غیر-FeatureCollection")
	}
}
