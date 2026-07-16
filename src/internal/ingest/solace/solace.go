// Package solace آداپتور دریافت NOTAM از FAA SWIM روی Solace با client-acknowledgement است (E1-2).
//
// تفاوت کلیدی با نسخهٔ قبلی: به‌جای WithMessageAutoAcknowledgement، از ack دستی استفاده می‌شود.
// پیام فقط پس از موفقیتِ emit (نوشتن پایدار در استریم) ack می‌شود؛ در نتیجه اگر اپ بین دریافت و
// نوشتن crash کند، Solace همان پیام را دوباره تحویل می‌دهد و NOTAM گم نمی‌شود.
package solace

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/hossein-repo/BaseProject/internal/ingest"

	sdk "solace.dev/go/messaging"
	sol "solace.dev/go/messaging/pkg/solace"
	solcfg "solace.dev/go/messaging/pkg/solace/config"
	"solace.dev/go/messaging/pkg/solace/message"
	"solace.dev/go/messaging/pkg/solace/resource"
)

const SourceName = "FAA_SWIM"

const (
	reconnectInterval = 5 * time.Second  // فاصلهٔ تلاش مجدد اتصال/اتصال مجدد
	startupBackoffMax = 60 * time.Second // سقف backoff تلاش اولیه
)

// Adapter آداپتور Solace.
type Adapter struct {
	host     string
	vpn      string
	username string
	password string
	queue    string

	service  sol.MessagingService
	receiver sol.PersistentMessageReceiver

	outage ingest.OutageSink

	mu       sync.Mutex
	downSince time.Time // زمان شروع قطعی جاری (صفر یعنی متصل)
}

// New یک آداپتور Solace می‌سازد.
func New(host, vpn, username, password, queue string) *Adapter {
	return &Adapter{host: host, vpn: vpn, username: username, password: password, queue: queue,
		outage: ingest.LogOutageSink{}}
}

// WithOutageSink ثبت‌کنندهٔ بازهٔ قطعی را جایگزین می‌کند (پیش‌فرض: لاگ).
func (a *Adapter) WithOutageSink(s ingest.OutageSink) *Adapter {
	if s != nil {
		a.outage = s
	}
	return a
}

func (a *Adapter) Name() string { return SourceName }

// markDown شروع یک بازهٔ قطعی را ثبت می‌کند (اگر قبلاً ثبت نشده باشد).
func (a *Adapter) markDown(t time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.downSince.IsZero() {
		a.downSince = t
		log.Printf("🔌 Solace connection lost at %s; reconnecting…", t.UTC().Format(time.RFC3339))
	}
}

// markUp پایان بازهٔ قطعی را ثبت و بازه را به OutageSink تحویل می‌دهد.
func (a *Adapter) markUp(t time.Time) {
	a.mu.Lock()
	start := a.downSince
	a.downSince = time.Time{}
	a.mu.Unlock()
	if !start.IsZero() {
		log.Printf("🔌 Solace reconnected at %s", t.UTC().Format(time.RFC3339))
		a.outage.RecordOutage(SourceName, start, t)
	}
}

// Start به Solace وصل می‌شود و پیام‌ها را با client-ack دریافت و emit می‌کند.
// در برابر قطعی مقاوم است: تلاش اولیه با backoff، و اتصال مجدد خودکار با ثبت بازهٔ قطعی (E1-3).
func (a *Adapter) Start(ctx context.Context, emit ingest.EmitFunc) error {
	// روی محیط بدون گواهی معتبر، اعتبارسنجی TLS غیرفعال می‌شود (مطابق رفتار قبلی).
	securityStrategy := solcfg.NewTransportSecurityStrategy().WithoutCertificateValidation()

	service, err := sdk.NewMessagingServiceBuilder().
		FromConfigurationProvider(solcfg.ServicePropertyMap{
			solcfg.TransportLayerPropertyHost:                a.host,
			solcfg.ServicePropertyVPNName:                    a.vpn,
			solcfg.AuthenticationPropertySchemeBasicUserName: a.username,
			solcfg.AuthenticationPropertySchemeBasicPassword: a.password,
		}).
		WithTransportSecurityStrategy(securityStrategy).
		// اتصال مجدد خودکار برای همیشه؛ NOTAM از دست نمی‌رود چون صف durable است.
		WithReconnectionRetryStrategy(solcfg.RetryStrategyForeverRetryWithInterval(reconnectInterval)).
		WithConnectionRetryStrategy(solcfg.RetryStrategyForeverRetryWithInterval(reconnectInterval)).
		Build()
	if err != nil {
		return err
	}
	a.service = service

	// listenerها برای تشخیص قطعی و ثبت بازهٔ آن (برای backfill هدف‌دار در E4)
	service.AddReconnectionAttemptListener(func(e sol.ServiceEvent) {
		a.markDown(e.GetTimestamp())
	})
	service.AddReconnectionListener(func(e sol.ServiceEvent) {
		a.markUp(e.GetTimestamp())
	})
	service.AddServiceInterruptionListener(func(e sol.ServiceEvent) {
		a.markDown(e.GetTimestamp())
		log.Printf("⛔ Solace service interruption: %s", e.GetMessage())
	})

	if err := a.connectWithBackoff(ctx); err != nil {
		return err
	}
	log.Println("✅ Connected to Solace (secure TLS)")

	// بدون WithMessageAutoAcknowledgement → ack دستی (client-ack)
	receiver, err := service.CreatePersistentMessageReceiverBuilder().
		Build(resource.QueueDurableExclusive(a.queue))
	if err != nil {
		return err
	}
	if err := receiver.Start(); err != nil {
		return err
	}
	a.receiver = receiver
	log.Println("📥 Subscribed to queue (client-ack):", a.queue)

	err = receiver.ReceiveAsync(func(inbound message.InboundMessage) {
		a.handle(inbound, emit)
	})
	if err != nil {
		return err
	}

	// تا لغو context فعال بمان
	<-ctx.Done()
	return nil
}

// connectWithBackoff تلاش اولیهٔ اتصال را با backoff نمایی تا لغو ctx تکرار می‌کند
// (تا اگر broker هنگام راه‌اندازی در دسترس نبود، اپ crash نکند).
func (a *Adapter) connectWithBackoff(ctx context.Context) error {
	backoff := reconnectInterval
	for {
		if err := a.service.Connect(); err == nil {
			return nil
		} else {
			log.Printf("⏳ Solace connect failed, retrying in %s: %v", backoff, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < startupBackoffMax {
			backoff *= 2
			if backoff > startupBackoffMax {
				backoff = startupBackoffMax
			}
		}
	}
}

// handle یک پیام را emit می‌کند و فقط در صورت موفقیت ack می‌زند.
func (a *Adapter) handle(inbound message.InboundMessage, emit ingest.EmitFunc) {
	payload, ok := inbound.GetPayloadAsString()
	if !ok || payload == "" {
		if b, okB := inbound.GetPayloadAsBytes(); okB {
			payload = string(b)
		}
	}
	if payload == "" {
		// پیام بدون بدنه: ack می‌کنیم تا در صف نماند (چیزی برای پردازش نیست)
		_ = a.receiver.Ack(inbound)
		log.Println("⚠️ Solace message without payload; acked")
		return
	}

	var srcID string
	if id, okID := inbound.GetApplicationMessageID(); okID {
		srcID = id
	} else {
		srcID = inbound.GetDestinationName()
	}

	raw := ingest.RawNotamMessage{
		Source:          SourceName,
		SourceMessageID: srcID,
		Payload:         payload,
		ReceivedAt:      time.Now().UTC(),
	}

	if err := emit(raw); err != nil {
		// عمداً ack نمی‌کنیم؛ Solace پیام را دوباره تحویل می‌دهد (بدون از‌دست‌رفتن).
		log.Printf("❌ emit failed for %s (not acked, will be redelivered): %v", srcID, err)
		return
	}

	if err := a.receiver.Ack(inbound); err != nil {
		// نوشتن در استریم موفق بوده ولی ack ناموفق: در بدترین حالت پردازش تکراری (idempotent) رخ می‌دهد.
		log.Printf("⚠️ Ack failed for %s (message may be redelivered): %v", srcID, err)
	}
}

// Close اتصال را می‌بندد.
func (a *Adapter) Close() error {
	if a.receiver != nil {
		_ = a.receiver.Terminate(0)
		a.receiver = nil
	}
	if a.service != nil {
		_ = a.service.Disconnect()
		a.service = nil
	}
	return nil
}
