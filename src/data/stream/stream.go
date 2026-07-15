// Package stream یک لایهٔ نازک روی Redis Streams برای صف پایدار داخلی NOTAM است (E0-5).
//
// هدف: decoupling کامل ingest از pipeline با تضمین at-least-once.
//   - ingest با Publish پیام خام را می‌نویسد و پس از موفقیت به منبع ack می‌دهد.
//   - pipeline با Read (consumer group) مصرف می‌کند و پس از commit موفق DB، Ack می‌زند.
//   - پیام‌های معلقِ یک consumerِ crash‌شده با ClaimStale دوباره برداشته می‌شوند.
//
// جزئیات معماری: docs/phase1/ARCHITECTURE.md §۳ و docs/phase1/RELIABILITY.md §۲.
package stream

import (
	"errors"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
)

// Message یک پیام خوانده‌شده از استریم.
type Message struct {
	ID     string            // شناسهٔ استریم (مثل "1718000000000-0")
	Values map[string]string // فیلدهای پیام
}

// Client دسترسی به عملیات استریم را فراهم می‌کند.
type Client struct {
	rdb *redis.Client
}

// New یک Client جدید روی اتصال Redis موجود می‌سازد.
func New(rdb *redis.Client) *Client {
	return &Client{rdb: rdb}
}

// EnsureGroup یک consumer group را روی استریم می‌سازد (در صورت نبود، استریم هم ساخته می‌شود).
// اگر group از قبل باشد خطای BUSYGROUP نادیده گرفته می‌شود (idempotent).
// start معمولاً "0" (از ابتدا) یا "$" (فقط پیام‌های جدید) است.
func (c *Client) EnsureGroup(stream, group, start string) error {
	if start == "" {
		start = "0"
	}
	err := c.rdb.XGroupCreateMkStream(stream, group, start).Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

// Publish یک پیام را به انتهای استریم اضافه می‌کند و ID تولیدشده را برمی‌گرداند.
// maxLen اگر > 0 باشد، طول استریم را تقریبی نگه می‌دارد (trim) تا حافظه کنترل شود؛ 0 یعنی بدون trim.
func (c *Client) Publish(stream string, values map[string]interface{}, maxLen int64) (string, error) {
	if len(values) == 0 {
		return "", errors.New("stream: empty message values")
	}
	args := &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}
	if maxLen > 0 {
		args.MaxLenApprox = maxLen
	}
	return c.rdb.XAdd(args).Result()
}

// Read پیام‌های جدیدِ تحویل‌نشده به این group را می‌خواند (id ">").
// count حداکثر تعداد؛ block مدت انتظار برای پیام جدید (0 یعنی بدون انتظار).
// اگر پیامی نباشد، اسلایس خالی و خطای nil برمی‌گردد.
func (c *Client) Read(stream, group, consumer string, count int64, block time.Duration) ([]Message, error) {
	res, err := c.rdb.XReadGroup(&redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    count,
		Block:    block,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // بدون پیام جدید در بازهٔ block
		}
		return nil, err
	}
	return flatten(res), nil
}

// Ack پیام‌های پردازش‌شده را تأیید می‌کند تا از لیست pending خارج شوند.
// فقط پس از commit موفق در DB فراخوانی شود.
func (c *Client) Ack(stream, group string, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	return c.rdb.XAck(stream, group, ids...).Err()
}

// ClaimStale پیام‌هایی را که توسط یک consumer برداشته ولی برای مدت minIdle ack نشده‌اند
// (نشانهٔ crash آن consumer) به این consumer منتقل و برمی‌گرداند تا دوباره پردازش شوند.
// این تابع جایگزین XAUTOCLAIM (که در go-redis v7 نیست) با XPendingExt + XClaim است.
func (c *Client) ClaimStale(stream, group, consumer string, minIdle time.Duration, count int64) ([]Message, error) {
	pend, err := c.rdb.XPendingExt(&redis.XPendingExtArgs{
		Stream: stream,
		Group:  group,
		Start:  "-",
		End:    "+",
		Count:  count,
	}).Result()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(pend))
	for _, p := range pend {
		if p.Idle >= minIdle {
			ids = append(ids, p.ID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	msgs, err := c.rdb.XClaim(&redis.XClaimArgs{
		Stream:   stream,
		Group:    group,
		Consumer: consumer,
		MinIdle:  minIdle,
		Messages: ids,
	}).Result()
	if err != nil {
		return nil, err
	}
	return toMessages(msgs), nil
}

// PendingCount تعداد پیام‌های معلق (خوانده‌شده ولی ack‌نشده) یک group را برمی‌گرداند.
// برای متریک و تشخیص عقب‌افتادگی pipeline (RELIABILITY §۹).
func (c *Client) PendingCount(stream, group string) (int64, error) {
	res, err := c.rdb.XPending(stream, group).Result()
	if err != nil {
		return 0, err
	}
	return res.Count, nil
}

// Len تعداد کل پیام‌های استریم را برمی‌گرداند.
func (c *Client) Len(stream string) (int64, error) {
	return c.rdb.XLen(stream).Result()
}

// ---- helpers ----

func flatten(streams []redis.XStream) []Message {
	var out []Message
	for _, s := range streams {
		out = append(out, toMessages(s.Messages)...)
	}
	return out
}

func toMessages(xs []redis.XMessage) []Message {
	out := make([]Message, 0, len(xs))
	for _, x := range xs {
		vals := make(map[string]string, len(x.Values))
		for k, v := range x.Values {
			if s, ok := v.(string); ok {
				vals[k] = s
			}
		}
		out = append(out, Message{ID: x.ID, Values: vals})
	}
	return out
}
