// Package pipeline پردازش NOTAM از استریم داخلی تا ذخیره را انجام می‌دهد (E2-2).
//
// جریان: مصرف از Redis Stream → parse (XML) → ساخت متن ICAO → فیلتر منطقه‌ای →
// ذخیرهٔ idempotent (canonical_key) → XACK پس از commit موفق.
package pipeline

import (
	"strings"

	"github.com/hossein-repo/BaseProject/internal/ingest"
	"github.com/hossein-repo/BaseProject/internal/messaging"
)

// نام‌های استریم و گروه مصرف‌کننده.
const (
	StreamNotamRaw = "notam:raw"
	GroupPipeline  = "pipeline"
)

// کلیدهای فیلد پیام استریم (بین publisher و consumer مشترک).
const (
	fieldSource      = "source"
	fieldSourceMsgID = "source_msg_id"
	fieldPayload     = "payload"
	fieldReceivedAt  = "received_at"
)

// ProcessResult نتیجهٔ پردازش یک پیام خام.
type ProcessResult struct {
	Message messaging.NotamMessage
	Skip    bool   // اگر true، NOTAM ذخیره نمی‌شود (خارج از منطقه/متن خالی)
	Reason  string // دلیل skip
}

// Processor منطق خالصِ پردازش یک NOTAM را نگه می‌دارد (مستقل از استریم و DB → تست‌پذیر).
type Processor struct {
	allowedFIRs     map[string]bool
	allowedAirports map[string]bool
}

// NewProcessor نقشه‌های منطقهٔ مجاز را یک‌بار می‌سازد.
func NewProcessor() *Processor {
	return &Processor{
		allowedFIRs:     messaging.AllowedFIRsMap(),
		allowedAirports: messaging.AllowedAirportsMap(),
	}
}

// Process یک پیام خام را پارس، فیلتر و به NotamMessage تبدیل می‌کند.
// خطا فقط برای شکست پارس برمی‌گردد (که باید به DLQ برود)؛ skip برای پیام‌های نامرتبط است.
func (p *Processor) Process(raw ingest.RawNotamMessage) (ProcessResult, error) {
	event, err := messaging.ParseNotamXML(raw.Payload)
	if err != nil {
		return ProcessResult{}, err
	}

	human := messaging.EnsureHumanReadableText(event)
	event.HumanReadableText = human
	if human == "" {
		return ProcessResult{Skip: true, Reason: "empty human-readable text"}, nil
	}

	aLine := strings.TrimSpace(event.Location)
	if aLine == "" {
		aLine = strings.TrimSpace(event.ICAOLocation)
	}

	if !(p.allowedFIRs[event.AffectedFIR] || p.allowedAirports[aLine]) {
		return ProcessResult{Skip: true, Reason: "outside allowed FIR/airport region"}, nil
	}

	msg := messaging.NewNotamMessage(raw.SourceMessageID, raw.Payload, "REGIONAL_NOTAM", raw.Source, event)
	return ProcessResult{Message: msg}, nil
}

// StreamValues فیلدهای یک RawNotamMessage را برای نوشتن در استریم آماده می‌کند (توسط publisher/ingest).
func StreamValues(raw ingest.RawNotamMessage) map[string]interface{} {
	return map[string]interface{}{
		fieldSource:      raw.Source,
		fieldSourceMsgID: raw.SourceMessageID,
		fieldPayload:     raw.Payload,
		fieldReceivedAt:  raw.ReceivedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

// RawFromValues یک RawNotamMessage را از فیلدهای پیام استریم بازسازی می‌کند (توسط consumer).
func RawFromValues(values map[string]string) ingest.RawNotamMessage {
	return ingest.RawNotamMessage{
		Source:          values[fieldSource],
		SourceMessageID: values[fieldSourceMsgID],
		Payload:         values[fieldPayload],
	}
}
