package pipeline

import (
	"testing"
	"time"

	"github.com/hossein-repo/BaseProject/internal/ingest"
)

func xmlWith(fir, location, text string) string {
	return `<root><hasMember><Event><timeSlice><EventTimeSlice><textNOTAM><NOTAM>` +
		`<series>A</series><number>46</number><year>2026</year><type>N</type>` +
		`<affectedFIR>` + fir + `</affectedFIR><location>` + location + `</location>` +
		`<text>` + text + `</text>` +
		`</NOTAM></textNOTAM></EventTimeSlice></timeSlice></Event></hasMember></root>`
}

func rawWith(payload string) ingest.RawNotamMessage {
	return ingest.RawNotamMessage{
		Source:          "FAA_SWIM",
		SourceMessageID: "msg-1",
		Payload:         payload,
		ReceivedAt:      time.Now().UTC(),
	}
}

func TestProcess_AllowedByAirport(t *testing.T) {
	p := NewProcessor()
	// OIII در AllowedAirports هست
	res, err := p.Process(rawWith(xmlWith("XXXX", "OIII", "RWY 29R CLSD")))
	if err != nil {
		t.Fatalf("Process خطا داد: %v", err)
	}
	if res.Skip {
		t.Fatalf("نباید skip شود (فرودگاه OIII مجاز است): %s", res.Reason)
	}
	if res.Message.Source() != "FAA_SWIM" {
		t.Errorf("source نادرست: %q", res.Message.Source())
	}
	if res.Message.ID() != "msg-1" {
		t.Errorf("id نادرست: %q", res.Message.ID())
	}
}

func TestProcess_AllowedByFIR(t *testing.T) {
	p := NewProcessor()
	// OIIX در AllowedFIRs هست
	res, err := p.Process(rawWith(xmlWith("OIIX", "ZZZZ", "SOME ENROUTE NOTAM")))
	if err != nil {
		t.Fatalf("Process خطا داد: %v", err)
	}
	if res.Skip {
		t.Fatalf("نباید skip شود (FIR OIIX مجاز است): %s", res.Reason)
	}
}

func TestProcess_SkipOutsideRegion(t *testing.T) {
	p := NewProcessor()
	res, err := p.Process(rawWith(xmlWith("ZZZZ", "YYYY", "IRRELEVANT")))
	if err != nil {
		t.Fatalf("Process خطا داد: %v", err)
	}
	if !res.Skip {
		t.Fatal("باید skip شود (خارج از منطقهٔ مجاز)")
	}
}

func TestProcess_ParseError(t *testing.T) {
	p := NewProcessor()
	if _, err := p.Process(rawWith("<broken")); err == nil {
		t.Error("انتظار خطای پارس بود")
	}
}

func TestStreamValuesRoundTrip(t *testing.T) {
	raw := rawWith(xmlWith("OIIX", "OIII", "X"))
	vals := StreamValues(raw)
	// شبیه‌سازی خواندن از استریم (Redis مقادیر را رشته‌ای برمی‌گرداند)
	strVals := map[string]string{}
	for k, v := range vals {
		strVals[k] = v.(string)
	}
	back := RawFromValues(strVals)
	if back.Source != raw.Source || back.SourceMessageID != raw.SourceMessageID || back.Payload != raw.Payload {
		t.Errorf("round-trip ناسازگار: %+v", back)
	}
}
