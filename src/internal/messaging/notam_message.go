package messaging

import "strings"

// notam_message.go — تایپ‌های دامنهٔ NOTAM (مستقل از منبع/اتصال).
// این تایپ‌ها قبلاً در solace_queue_consumer.go بودند و برای تفکیک ماژول‌ها (E1/E2) جدا شدند.

// NotamEvent فیلدهای ساختاریافتهٔ یک NOTAM که از XMLِ FAA پارس می‌شوند.
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
	// Q-code واقعی ICAO. در فید FAA/AIXM نام این عنصر selectionCode است (تگ qcode وجود ندارد).
	QCode        string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>selectionCode"`
	Traffic      string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>traffic"` // IV/I/V
	Purpose      string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>purpose"` // NBO/BO/M...
	Scope        string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>scope"`   // A/E/W
	Schedule     string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>schedule"`
	ICAOLocation string `xml:"hasMember>Event>timeSlice>EventTimeSlice>extension>EventExtension>icaoLocation"`
	AirportName  string `xml:"hasMember>Event>timeSlice>EventTimeSlice>extension>EventExtension>airportname"`

	// هندسه و ارتفاع ساختاریافتهٔ فضای هوایی (خط Q) — برای تداخل مسیر/ارتفاع (E5.6).
	Coordinates string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>coordinates"` // مرکز مثل 5609N04020E
	Radius      string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>radius"`      // شعاع بر حسب NM (مثل 002)
	MinimumFL   string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>minimumFL"`   // FL پایین (000=زمین)
	MaximumFL   string `xml:"hasMember>Event>timeSlice>EventTimeSlice>textNOTAM>NOTAM>maximumFL"`   // FL بالا (999=نامحدود)
}

// NotamMessage پیام پردازش‌شدهٔ NOTAM که به لایهٔ storage تحویل می‌شود (interface Message را برآورده می‌کند).
type NotamMessage struct {
	msgID   string
	body    string
	event   NotamEvent
	msgType string
	source  string
}

// NewNotamMessage یک NotamMessage می‌سازد (فیلدها unexported هستند تا از پکیج‌های دیگر ساخته شوند).
func NewNotamMessage(msgID, body, msgType, source string, event NotamEvent) NotamMessage {
	return NotamMessage{msgID: msgID, body: body, msgType: msgType, source: source, event: event}
}

func (m NotamMessage) ID() string        { return m.msgID }
func (m NotamMessage) Body() string      { return m.body }
func (m NotamMessage) Type() string      { return m.msgType }
func (m NotamMessage) Source() string    { return m.source }
func (m NotamMessage) Event() NotamEvent { return m.event }

// CanonicalKey هویت متعارف یک NOTAM را می‌سازد تا در بین منابع مختلف (SWIM/AFTN/…) یکتا باشد (E2-3).
// فرمت: "LOCATION|SERIES" با حروف بزرگ. اگر series خالی باشد رشتهٔ خالی برمی‌گرداند و
// فراخوان باید به کلید جایگزین (مثل message_id) fallback کند.
func CanonicalKey(locationICAO, seriesNumber string) string {
	ser := strings.ToUpper(strings.TrimSpace(seriesNumber))
	if ser == "" {
		return ""
	}
	loc := strings.ToUpper(strings.TrimSpace(locationICAO))
	return loc + "|" + ser
}
