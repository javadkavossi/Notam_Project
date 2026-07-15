// Package solace آداپتور دریافت NOTAM از FAA SWIM روی Solace با client-acknowledgement است (E1-2).
//
// تفاوت کلیدی با نسخهٔ قبلی: به‌جای WithMessageAutoAcknowledgement، از ack دستی استفاده می‌شود.
// پیام فقط پس از موفقیتِ emit (نوشتن پایدار در استریم) ack می‌شود؛ در نتیجه اگر اپ بین دریافت و
// نوشتن crash کند، Solace همان پیام را دوباره تحویل می‌دهد و NOTAM گم نمی‌شود.
package solace

import (
	"context"
	"log"
	"time"

	"github.com/hossein-repo/BaseProject/internal/ingest"

	sdk "solace.dev/go/messaging"
	sol "solace.dev/go/messaging/pkg/solace"
	solcfg "solace.dev/go/messaging/pkg/solace/config"
	"solace.dev/go/messaging/pkg/solace/message"
	"solace.dev/go/messaging/pkg/solace/resource"
)

const SourceName = "FAA_SWIM"

// Adapter آداپتور Solace.
type Adapter struct {
	host     string
	vpn      string
	username string
	password string
	queue    string

	service  sol.MessagingService
	receiver sol.PersistentMessageReceiver
}

// New یک آداپتور Solace می‌سازد.
func New(host, vpn, username, password, queue string) *Adapter {
	return &Adapter{host: host, vpn: vpn, username: username, password: password, queue: queue}
}

func (a *Adapter) Name() string { return SourceName }

// Start به Solace وصل می‌شود و پیام‌ها را با client-ack دریافت و emit می‌کند.
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
		Build()
	if err != nil {
		return err
	}
	if err := service.Connect(); err != nil {
		return err
	}
	a.service = service
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
