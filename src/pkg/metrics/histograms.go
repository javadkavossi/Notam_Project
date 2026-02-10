package metrics

import "github.com/prometheus/client_golang/prometheus"

var HttpDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_time",
		Help:    "Duration of HTTP requests",
		Buckets: []float64{1, 2, 5, 10, 50, 100, 200, 500, 1000, 2000, 5000, 10000},
		// Buckets: prometheus.DefBuckets, // بینه‌های پیش‌فرض
	},
	[]string{"path", "method", "status_code"}, // برچسب‌ها
)
