package stream

import (
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v7"
)

// testClient به Redis آزمایشی وصل می‌شود؛ اگر در دسترس نباشد تست skip می‌شود.
// آدرس از REDIS_TEST_ADDR یا پیش‌فرض localhost:6379.
func testClient(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("REDIS_TEST_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	if err := rdb.Ping().Err(); err != nil {
		t.Skipf("Redis در %s در دسترس نیست؛ تست رد شد: %v", addr, err)
	}
	return rdb
}

func TestStreamRoundTrip(t *testing.T) {
	rdb := testClient(t)
	defer rdb.Close()

	stream := "test:notam:raw:" + time.Now().Format("150405.000000")
	group := "test-pipeline"
	consumer := "c1"
	defer rdb.Del(stream)

	c := New(rdb)

	// ساخت group (و استریم)
	if err := c.EnsureGroup(stream, group, "0"); err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}
	// idempotent: بار دوم هم نباید خطا بدهد
	if err := c.EnsureGroup(stream, group, "0"); err != nil {
		t.Fatalf("EnsureGroup (دوم): %v", err)
	}

	// انتشار
	id, err := c.Publish(stream, map[string]interface{}{"source": "FAA_SWIM", "body": "hello"}, 1000)
	if err != nil || id == "" {
		t.Fatalf("Publish: id=%q err=%v", id, err)
	}

	// خواندن
	msgs, err := c.Read(stream, group, consumer, 10, 0)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("انتظار ۱ پیام، دریافت %d", len(msgs))
	}
	if msgs[0].Values["source"] != "FAA_SWIM" || msgs[0].Values["body"] != "hello" {
		t.Fatalf("مقادیر نادرست: %+v", msgs[0].Values)
	}

	// قبل از Ack باید در pending باشد
	if n, _ := c.PendingCount(stream, group); n != 1 {
		t.Fatalf("انتظار ۱ pending، دریافت %d", n)
	}

	// Ack
	if err := c.Ack(stream, group, msgs[0].ID); err != nil {
		t.Fatalf("Ack: %v", err)
	}
	if n, _ := c.PendingCount(stream, group); n != 0 {
		t.Fatalf("انتظار ۰ pending پس از Ack، دریافت %d", n)
	}
}

func TestClaimStale(t *testing.T) {
	rdb := testClient(t)
	defer rdb.Close()

	stream := "test:notam:claim:" + time.Now().Format("150405.000000")
	group := "test-pipeline"
	defer rdb.Del(stream)

	c := New(rdb)
	if err := c.EnsureGroup(stream, group, "0"); err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}
	if _, err := c.Publish(stream, map[string]interface{}{"body": "x"}, 0); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// consumer اول می‌خواند ولی Ack نمی‌کند (شبیه‌سازی crash)
	if _, err := c.Read(stream, group, "crashed", 10, 0); err != nil {
		t.Fatalf("Read: %v", err)
	}

	// با minIdle=0 پیام معلق باید به consumer دوم منتقل شود
	claimed, err := c.ClaimStale(stream, group, "rescuer", 0, 10)
	if err != nil {
		t.Fatalf("ClaimStale: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("انتظار ۱ پیام claim‌شده، دریافت %d", len(claimed))
	}
}
