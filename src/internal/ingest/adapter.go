// Package ingest قرارداد دریافت NOTAM از منابع مختلف را تعریف می‌کند (E1-1).
//
// هر منبع (FAA SWIM, AFTN, FAA REST, …) یک SourceAdapter پیاده‌سازی می‌کند که پیام خام را
// به RawNotamMessage نرمال کرده و از طریق EmitFunc تحویل می‌دهد. آداپتور فقط پس از موفقیتِ
// emit (یعنی نوشتن پایدار در استریم) پیام را به منبع ack می‌کند — اصل «عدم‌ازدست‌رفتن» (RELIABILITY §۲).
package ingest

import (
	"context"
	"time"
)

// RawNotamMessage پیام خامِ نرمال‌شده از یک منبع، مستقل از فرمت انتقال.
type RawNotamMessage struct {
	Source          string    // شناسهٔ منبع: "FAA_SWIM", "AFTN", "FAA_REST"
	SourceMessageID string    // شناسهٔ اصلی پیام در منبع (برای provenance و ردیابی)
	Payload         string    // بدنهٔ خام (برای FAA: XML)
	ReceivedAt      time.Time // زمان دریافت در سیستم ما (UTC)
}

// EmitFunc پیام خام را به لایهٔ بعد (استریم) تحویل می‌دهد.
// اگر nil برگرداند، آداپتور باید پیام را در منبع ack کند؛ در صورت خطا نباید ack کند
// تا منبع پیام را دوباره تحویل دهد (at-least-once).
type EmitFunc func(RawNotamMessage) error

// SourceAdapter قرارداد یک منبع NOTAM.
type SourceAdapter interface {
	// Name شناسهٔ منبع (مثل "FAA_SWIM").
	Name() string
	// Start دریافت را آغاز می‌کند و برای هر پیام emit را صدا می‌زند.
	// تا زمان لغو ctx یا خطای کشنده بلاک/فعال می‌ماند.
	Start(ctx context.Context, emit EmitFunc) error
	// Close اتصال را می‌بندد.
	Close() error
}
