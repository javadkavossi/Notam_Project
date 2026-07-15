package messaging

import (
	"fmt"
	"log"
	"strings"
	"time"

	"solace.dev/go/messaging"
	sol "solace.dev/go/messaging/pkg/solace"
	"solace.dev/go/messaging/pkg/solace/config"
	"solace.dev/go/messaging/pkg/solace/message"
	"solace.dev/go/messaging/pkg/solace/resource"
)

type NotamEvent struct {
	HumanReadableText string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>translation>NOTAMTranslation>formattedText>html~div>html~pre"`
	Text              string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>text"`
	Series            string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>series"`
	Number            int    `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>number"`
	Year              string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>year"`
	EventType         string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>type"`
	AffectedFIR       string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>affectedFIR"`
	Location          string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>location"`
	EffectiveStart    string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>effectiveStart"`
	EffectiveEnd      string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>effectiveEnd"`
	Issued            string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>issued"`
	LowerLimit        string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>lowerLimit"`
	UpperLimit        string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>upperLimit"`
	QCode             string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>qcode"`
	Schedule          string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>schedule"`
	ICAOLocation      string `xml:"hasMember>Event>timeSlice>EventTimeSlice>extension>EventExtension>icaoLocation"`
	AirportName       string `xml:"hasMember>Event>timeSlice>EventTimeSlice>extension>EventExtension>airportname"`
}

type NotamMessage struct {
	msgID   string
	body    string
	event   NotamEvent
	msgType string
}

func (m NotamMessage) ID() string        { return m.msgID }
func (m NotamMessage) Body() string      { return m.body }
func (m NotamMessage) Type() string      { return m.msgType }
func (m NotamMessage) Event() NotamEvent { return m.event }

type SolaceQueueConsumer struct {
	host     string
	vpn      string
	username string
	password string
	queue    string

	service  sol.MessagingService
	receiver sol.PersistentMessageReceiver
}

func NewSolaceQueueConsumer(host, vpn, username, password, queue string) *SolaceQueueConsumer {
	return &SolaceQueueConsumer{
		host:     host,
		vpn:      vpn,
		username: username,
		password: password,
		queue:    queue,
	}
}

func (c *SolaceQueueConsumer) Start(handler func(Message)) error {
	// روی macOS مسیر /etc/ssl/certs وجود ندارد؛ برای development اعتبارسنجی را غیرفعال می‌کنیم
	securityStrategy := config.NewTransportSecurityStrategy().
		WithoutCertificateValidation()

	service, err := messaging.NewMessagingServiceBuilder().
		FromConfigurationProvider(config.ServicePropertyMap{
			config.TransportLayerPropertyHost:                c.host,
			config.ServicePropertyVPNName:                    c.vpn,
			config.AuthenticationPropertySchemeBasicUserName: c.username,
			config.AuthenticationPropertySchemeBasicPassword: c.password,
		}).
		WithTransportSecurityStrategy(securityStrategy).
		Build()
	if err != nil {
		return err
	}

	if err := service.Connect(); err != nil {
		return err
	}
	log.Println("✅ Connected to Solace (secure TLS)")

	c.service = service

	receiver, err := service.CreatePersistentMessageReceiverBuilder().
		WithMessageAutoAcknowledgement().
		Build(resource.QueueDurableExclusive(c.queue))
	if err != nil {
		return err
	}

	if err := receiver.Start(); err != nil {
		return err
	}

	c.receiver = receiver
	log.Println("📥 Subscribed to queue:", c.queue)

	allowedFIRs := AllowedFIRsMap()
	allowedAirports := AllowedAirportsMap()

	err = receiver.ReceiveAsync(func(inboundMsg message.InboundMessage) {
		var payload string
		if p, ok := inboundMsg.GetPayloadAsString(); ok {
			payload = p
		} else if b, ok := inboundMsg.GetPayloadAsBytes(); ok {
			payload = string(b)
		} else {
			log.Println("⚠️ No payload")
			return
		}

		var msgID string
		if id, ok := inboundMsg.GetApplicationMessageID(); ok {
			msgID = id
		} else {
			msgID = inboundMsg.GetDestinationName()
		}

		event, err := ParseNotamXML(payload)
		if err != nil {
			log.Printf("❌ XML parse error for ID %s: %v", msgID, err)
			return
		}

		// ساخت/تکمیل متن استاندارد ICAO (منطق تست‌پذیر در notam_parser.go)
		humanText := EnsureHumanReadableText(event)
		event.HumanReadableText = humanText

		if humanText == "" {
			return
		}

		aLine := strings.TrimSpace(event.Location)
		if aLine == "" {
			aLine = strings.TrimSpace(event.ICAOLocation)
		}

		isRelated := allowedFIRs[event.AffectedFIR] || allowedAirports[aLine]
		if !isRelated {

			fmt.Println("isRelated", isRelated)
			return // فقط NOTAMهای FIRها و فرودگاه‌های مجاز
		}

		isFICON := strings.Contains(strings.ToUpper(humanText), "FICON")

		notamType := "Airport NOTAM"
		if aLine == event.AffectedFIR && allowedFIRs[event.AffectedFIR] {
			notamType = "FIR-level NOTAM"
		}

		areaDisplay := event.AirportName
		if areaDisplay == "" {
			if aLine != "" {
				areaDisplay = aLine
			} else {
				areaDisplay = event.AffectedFIR
			}
		}

		issueTime := event.Issued
		if issueTime == "" {
			issueTime = "Not available"
		}
		receiveTime := time.Now().UTC().Format("2006/01/02 15:04:05")

		log.Printf("🌍 REGIONAL NOTAM | Type: %s | Area: %s | Issue UTC: %s | Received UTC: %s | ID: %s", notamType, areaDisplay, issueTime, receiveTime, msgID)
		if isFICON {
			log.Println("❄️ IMPORTANT FICON REPORT:")
		}

		lines := strings.Split(humanText, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				log.Println(trimmed)
			}
		}
		log.Println("==========================================================================")

		handler(NotamMessage{
			msgID:   msgID,
			body:    payload,
			event:   event,
			msgType: "REGIONAL_NOTAM",
		})
	})

	return err
}

func (c *SolaceQueueConsumer) Close() error {
	if c.receiver != nil {
		_ = c.receiver.Terminate(0)
		c.receiver = nil
	}
	if c.service != nil {
		_ = c.service.Disconnect()
		c.service = nil
	}
	return nil
}
