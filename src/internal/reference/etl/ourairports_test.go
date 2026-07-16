package etl

import (
	"strings"
	"testing"
)

const airportsCSV = `"id","ident","type","name","latitude_deg","longitude_deg","elevation_ft","continent","iso_country","iso_region","municipality","scheduled_service","gps_code","iata_code","local_code","home_link","wikipedia_link","keywords"
1,"OIII","large_airport","Mehrabad International Airport",35.6892,51.3134,3962,"AS","IR","IR-07","Tehran","yes","OIII","THR","",,,
2,"OIIE","large_airport","Imam Khomeini International Airport",35.416099,51.152199,3305,"AS","IR","IR-07","Tehran","yes","OIIE","IKA","",,,
3,"XX","small_airport","No ICAO here",10.0,10.0,100,"AS","IR","IR-07","Nowhere","no","","","",,,
4,"CLZ1","closed","Closed Field",1.0,1.0,0,"NA","US","US-CA","Gone","no","CLZ1","","",,,
`

const runwaysCSV = `"id","airport_ref","airport_ident","length_ft","width_ft","surface","lighted","closed","le_ident","he_ident"
1,1,"OIII",13071,197,"ASP",1,0,"11L","29R"
2,1,"OIII",12008,148,"ASP",1,0,"11R","29L"
3,5,"XXXX",3000,50,"GRS",0,1,"09","27"
`

func TestParseAirports(t *testing.T) {
	airports, err := ParseAirports(strings.NewReader(airportsCSV))
	if err != nil {
		t.Fatalf("ParseAirports: %v", err)
	}
	// XX (بدون ICAO) و CLZ1 (بسته) باید رد شوند → فقط OIII و OIIE
	if len(airports) != 2 {
		t.Fatalf("انتظار ۲ فرودگاه، دریافت %d: %+v", len(airports), airports)
	}
	a := airports[0]
	if a.ICAO != "OIII" || a.IATA != "THR" || a.Country != "IR" {
		t.Errorf("فیلدهای فرودگاه نادرست: %+v", a)
	}
	if a.ElevationFt != 3962 {
		t.Errorf("ارتفاع نادرست: %d", a.ElevationFt)
	}
	if a.Lat < 35.6 || a.Lat > 35.7 {
		t.Errorf("عرض جغرافیایی نادرست: %f", a.Lat)
	}
}

func TestParseRunways(t *testing.T) {
	rws, err := ParseRunways(strings.NewReader(runwaysCSV))
	if err != nil {
		t.Fatalf("ParseRunways: %v", err)
	}
	// XXXX معتبر ICAO است (۴ حرف)، پس ۳ باند برمی‌گردد
	if len(rws) != 3 {
		t.Fatalf("انتظار ۳ باند، دریافت %d", len(rws))
	}
	if rws[0].AirportICAO != "OIII" || rws[0].Name != "11L/29R" {
		t.Errorf("باند نادرست: %+v", rws[0])
	}
	if rws[0].LengthFt != 13071 || !rws[0].Lighted {
		t.Errorf("طول/روشنایی نادرست: %+v", rws[0])
	}
	if !rws[2].Closed {
		t.Errorf("باند بسته باید Closed باشد: %+v", rws[2])
	}
}

func TestParseAirportsBOM(t *testing.T) {
	// سرستون با BOM ابتدای فایل نباید ستون اول را خراب کند
	withBOM := "\ufeff" + airportsCSV
	airports, err := ParseAirports(strings.NewReader(withBOM))
	if err != nil {
		t.Fatalf("ParseAirports(BOM): %v", err)
	}
	if len(airports) != 2 || airports[0].ICAO != "OIII" {
		t.Errorf("BOM باعث خرابی پارس شد: %+v", airports)
	}
}
