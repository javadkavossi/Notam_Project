package messaging

import (
	"strings"
	"testing"
)

func TestBuildHumanReadableText_New(t *testing.T) {
	ev := NotamEvent{
		Series:         "A",
		Number:         46,
		Year:           "2026",
		EventType:      "N",
		AffectedFIR:    "OIIX",
		ICAOLocation:   "OIII",
		EffectiveStart: "2026021506",
		EffectiveEnd:   "2026021523",
		Text:           "RWY 11L/29R CLSD DUE WIP",
	}
	got := BuildHumanReadableText(ev)

	wantContains := []string{
		"A0046/26 NOTAMN",             // خط اول: سریال + نوع
		"Q) OIIX/QWMLW/IV/BO/W/000/999/", // بند Q
		"A) OIII",                     // مکان
		"B) 26021506",                 // پیشوند قرن حذف شده
		"C) 26021523",
		"E) RWY 11L/29R CLSD DUE WIP", // متن اصلی
	}
	for _, w := range wantContains {
		if !strings.Contains(got, w) {
			t.Errorf("خروجی شامل %q نیست.\n--- got ---\n%s", w, got)
		}
	}
}

func TestBuildHumanReadableText_Cancel(t *testing.T) {
	ev := NotamEvent{
		Series:      "A",
		Number:      50,
		Year:        "2026",
		EventType:   "C",
		AffectedFIR: "OIIX",
		ICAOLocation: "OIII",
		Text:        "A0046/26",
	}
	got := BuildHumanReadableText(ev)

	if !strings.Contains(got, "A0050/26 NOTAMC A0046/26") {
		t.Errorf("خط کنسل باید شامل ارجاع NOTAM لغوشده باشد.\ngot:\n%s", got)
	}
	if strings.Contains(got, "Q)") {
		t.Errorf("بند Q نباید برای NOTAMC نمایش داده شود.\ngot:\n%s", got)
	}
	if !strings.Contains(got, "E) NOTAM CANCELLED") {
		t.Errorf("متن NOTAMC باید NOTAM CANCELLED باشد.\ngot:\n%s", got)
	}
}

func TestEnsureHumanReadableText_UsesExistingWhenComplete(t *testing.T) {
	complete := "A1234/26 NOTAMN\nA) OIII\nE) SOMETHING"
	ev := NotamEvent{HumanReadableText: complete, Text: "ignored"}
	if got := EnsureHumanReadableText(ev); got != complete {
		t.Errorf("متن کامل موجود باید حفظ شود.\nwant:\n%s\ngot:\n%s", complete, got)
	}
}

func TestEnsureHumanReadableText_BuildsWhenShort(t *testing.T) {
	ev := NotamEvent{
		HumanReadableText: "just one line",
		Series:            "B",
		Number:            7,
		Year:              "2026",
		EventType:         "N",
		ICAOLocation:      "OISS",
		Text:              "TWY A CLSD",
	}
	got := EnsureHumanReadableText(ev)
	if !strings.Contains(got, "B0007/26 NOTAMN") || !strings.Contains(got, "E) TWY A CLSD") {
		t.Errorf("وقتی متن ناقص است باید ساخته شود.\ngot:\n%s", got)
	}
}

func TestParseNotamXML_SimpleFields(t *testing.T) {
	// XML مینیمال با مسیر بدون namespace برای فیلدهای ساده
	payload := `<root><hasMember><Event><timeSlice><EventTimeSlice><textNOTAM><NOTAM>
		<series>A</series><number>46</number><year>2026</year><type>N</type>
		<affectedFIR>OIIX</affectedFIR><location>OIII</location>
		<text>RWY 11L CLSD</text>
	</NOTAM></textNOTAM></EventTimeSlice></timeSlice></Event></hasMember></root>`

	ev, err := ParseNotamXML(payload)
	if err != nil {
		t.Fatalf("ParseNotamXML خطا داد: %v", err)
	}
	if ev.Series != "A" || ev.Number != 46 || ev.Year != "2026" {
		t.Errorf("فیلدهای سریال نادرست: %+v", ev)
	}
	if ev.AffectedFIR != "OIIX" || ev.Location != "OIII" {
		t.Errorf("فیلدهای مکان نادرست: %+v", ev)
	}
	if ev.Text != "RWY 11L CLSD" {
		t.Errorf("متن نادرست: %q", ev.Text)
	}
}

func TestParseNotamXML_Invalid(t *testing.T) {
	if _, err := ParseNotamXML("<not-closed>"); err == nil {
		t.Error("انتظار خطا برای XML نامعتبر بود")
	}
}
