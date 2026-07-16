package pipeline

import (
	"context"
	"log"
	"time"

	"github.com/hossein-repo/BaseProject/data/stream"
	"github.com/hossein-repo/BaseProject/internal/ingest"
	"github.com/hossein-repo/BaseProject/internal/messaging"
)

// Repo قرارداد ذخیره‌سازی موردنیاز pipeline (توسط storage.NotamRepository برآورده می‌شود).
type Repo interface {
	Save(messaging.Message)
}

// Runner استریم را مصرف می‌کند و NOTAMها را پردازش و ذخیره می‌کند.
type Runner struct {
	stream    *stream.Client
	proc      *Processor
	repo      Repo
	consumer  string        // شناسهٔ یکتای این مصرف‌کننده (برای claim)
	batch     int64         // حداکثر پیام در هر خواندن
	block     time.Duration // مدت انتظار برای پیام جدید
	claimIdle time.Duration // آستانهٔ idle برای بازبرداشت پیام‌های معلق
}

// NewRunner یک Runner می‌سازد.
func NewRunner(s *stream.Client, repo Repo, consumer string) *Runner {
	return &Runner{
		stream:    s,
		proc:      NewProcessor(),
		repo:      repo,
		consumer:  consumer,
		batch:     32,
		block:     5 * time.Second,
		claimIdle: 60 * time.Second,
	}
}

// Run حلقهٔ مصرف را تا لغو ctx اجرا می‌کند.
func (r *Runner) Run(ctx context.Context) error {
	if err := r.stream.EnsureGroup(StreamNotamRaw, GroupPipeline, "0"); err != nil {
		return err
	}
	log.Printf("🛠️  Pipeline consumer '%s' started on stream '%s'", r.consumer, StreamNotamRaw)

	var sinceClaim time.Duration
	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Pipeline runner stopped")
			return nil
		default:
		}

		// هر ~۳۰ ثانیه، پیام‌های معلقِ مصرف‌کننده‌های crash‌شده را بازبرداشت کن
		if sinceClaim >= 30*time.Second {
			sinceClaim = 0
			if claimed, err := r.stream.ClaimStale(StreamNotamRaw, GroupPipeline, r.consumer, r.claimIdle, r.batch); err != nil {
				log.Printf("⚠️ ClaimStale error: %v", err)
			} else if len(claimed) > 0 {
				log.Printf("♻️  Reclaimed %d stale message(s)", len(claimed))
				r.handleBatch(claimed)
			}
		}

		msgs, err := r.stream.Read(StreamNotamRaw, GroupPipeline, r.consumer, r.batch, r.block)
		if err != nil {
			log.Printf("⚠️ Stream read error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if len(msgs) == 0 {
			sinceClaim += r.block
			continue
		}
		r.handleBatch(msgs)
	}
}

// handleBatch یک دسته پیام را پردازش، ذخیره و ack می‌کند.
// در صورت خطای پارس، پیام به DLQ منتقل و سپس ack می‌شود تا پیامِ خرابْ حلقهٔ بی‌پایان نسازد (E2-6).
func (r *Runner) handleBatch(msgs []stream.Message) {
	for _, m := range msgs {
		raw := RawFromValues(m.Values)
		res, err := r.proc.Process(raw)
		if err != nil {
			// شکست پارس: به DLQ منتقل کن (گم نمی‌شود) و سپس ack کن تا poison-loop رخ ندهد.
			if !r.toDLQ(raw, err, m.ID) {
				// اگر نوشتن در DLQ هم شکست خورد، ack نکن تا پیام حفظ شود و بعداً دوباره تلاش شود.
				continue
			}
		} else if !res.Skip {
			// ذخیرهٔ idempotent (canonical_key). Save داخلی خطا را لاگ می‌کند.
			r.repo.Save(res.Message)
		}
		// پیام پردازش شد (ذخیره / skip آگاهانه / منتقل به DLQ) → ack
		if err := r.stream.Ack(StreamNotamRaw, GroupPipeline, m.ID); err != nil {
			log.Printf("⚠️ XACK failed for %s: %v", m.ID, err)
		}
	}
}

// toDLQ پیام شکست‌خورده را در استریم DLQ می‌نویسد. در صورت موفقیت true برمی‌گرداند.
func (r *Runner) toDLQ(raw ingest.RawNotamMessage, procErr error, streamID string) bool {
	if _, err := r.stream.Publish(StreamNotamDLQ, DLQValues(raw, procErr.Error()), 50000); err != nil {
		log.Printf("❌ DLQ publish failed for %s (kept for retry): %v", streamID, err)
		return false
	}
	log.Printf("📥 Moved to DLQ id=%s src=%s: %v", streamID, raw.Source, procErr)
	return true
}
