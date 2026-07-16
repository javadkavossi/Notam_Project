package etl

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/hossein-repo/BaseProject/data/db/model"
)

// FIRFeature یک FIR استخراج‌شده از GeoJSON به‌همراه هندسهٔ خام آن.
type FIRFeature struct {
	ICAO         string
	Name         string
	GeometryJSON string // آبجکت geometry به‌صورت JSON (ورودی ST_GeomFromGeoJSON)
}

// geojson ساختار حداقلی FeatureCollection.
type geojson struct {
	Type     string `json:"type"`
	Features []struct {
		Properties map[string]any  `json:"properties"`
		Geometry   json.RawMessage `json:"geometry"`
	} `json:"features"`
}

// ParseFIRs یک FeatureCollection از مرزهای FIR را می‌خواند.
// کد ICAO و نام از properties استخراج می‌شوند (کلیدهای رایج پشتیبانی می‌شوند).
func ParseFIRs(r io.Reader) ([]FIRFeature, error) {
	var fc geojson
	if err := json.NewDecoder(r).Decode(&fc); err != nil {
		return nil, err
	}
	if !strings.EqualFold(fc.Type, "FeatureCollection") {
		return nil, errors.New("geojson: expected FeatureCollection")
	}
	out := make([]FIRFeature, 0, len(fc.Features))
	for _, f := range fc.Features {
		icao := strings.ToUpper(strings.TrimSpace(firstProp(f.Properties, "icao", "ICAO", "id", "IDENT", "ident")))
		if icao == "" || len(f.Geometry) == 0 {
			continue
		}
		out = append(out, FIRFeature{
			ICAO:         icao,
			Name:         firstProp(f.Properties, "name", "NAME", "label"),
			GeometryJSON: string(f.Geometry),
		})
	}
	return out, nil
}

func firstProp(props map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := props[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

// ToFIRModels FIRFeatureها را به مدل FIR (بدون geometry؛ geometry جداگانه ست می‌شود) تبدیل می‌کند.
func ToFIRModels(feats []FIRFeature) []model.FIR {
	out := make([]model.FIR, 0, len(feats))
	for _, f := range feats {
		out = append(out, model.FIR{ICAO: f.ICAO, Name: f.Name, BoundaryGeoJSON: f.GeometryJSON})
	}
	return out
}
