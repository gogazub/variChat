package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
	httpInFlight prometheus.Gauge

	dbQueryDuration *prometheus.HistogramVec
	dbQueryErrors   *prometheus.CounterVec

	redisCmdDuration *prometheus.HistogramVec
	redisCmdErrors   *prometheus.CounterVec

	businessDuration *prometheus.HistogramVec
	businessErrors   *prometheus.CounterVec
	messagesProcessed prometheus.Counter
)


func Init(serviceName string) {
	// HTTP
	httpRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: serviceName,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"path", "method", "status"},
	)

	httpDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: serviceName,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)

	httpInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: serviceName,
			Name:      "http_in_flight_requests",
			Help:      "Current number of in-flight HTTP requests",
		},
	)

	// DB
	dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: serviceName,
			Name:      "db_query_duration_seconds",
			Help:      "DB query duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"query"},
	)
	dbQueryErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: serviceName,
			Name:      "db_query_errors_total",
			Help:      "DB query errors",
		},
		[]string{"query"},
	)

	// Redis
	redisCmdDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: serviceName,
			Name:      "redis_command_duration_seconds",
			Help:      "Redis command duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"cmd"},
	)
	redisCmdErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: serviceName,
			Name:      "redis_command_errors_total",
			Help:      "Redis command errors",
		},
		[]string{"cmd"},
	)

	// Business-level
	businessDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: serviceName,
			Name:      "business_operation_duration_seconds",
			Help:      "Business operation duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
	businessErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: serviceName,
			Name:      "business_errors_total",
			Help:      "Business logic errors",
		},
		[]string{"operation"},
	)

	messagesProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: serviceName,
		Name:      "messages_processed_total",
		Help:      "Total processed messages",
	})
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}


func InstrumentHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		method := r.Method
		httpInFlight.Inc()
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)
		duration := time.Since(start).Seconds()
		status := "0"
		if rw.status != 0 {
			status = http.StatusText(rw.status)
		}
		httpRequests.WithLabelValues(path, method, status).Inc()
		httpDuration.WithLabelValues(path, method).Observe(duration)
		httpInFlight.Dec()
	})
}

func ObserveDB(query string, start time.Time, err error) {
	d := time.Since(start).Seconds()
	if dbQueryDuration != nil {
		dbQueryDuration.WithLabelValues(query).Observe(d)
	}
	if err != nil {
		if dbQueryErrors != nil {
			dbQueryErrors.WithLabelValues(query).Inc()
		}
	}
}

func ObserveRedis(cmd string, start time.Time, err error) {
	d := time.Since(start).Seconds()
	if redisCmdDuration != nil {
		redisCmdDuration.WithLabelValues(cmd).Observe(d)
	}
	if err != nil {
		if redisCmdErrors != nil {
			redisCmdErrors.WithLabelValues(cmd).Inc()
		}
	}
}

func ObserveBusiness(operation string, start time.Time, err error) {
	d := time.Since(start).Seconds()
	if businessDuration != nil {
		businessDuration.WithLabelValues(operation).Observe(d)
	}
	if err != nil {
		if businessErrors != nil {
			businessErrors.WithLabelValues(operation).Inc()
		}
	}
}

func IncMessagesProcessed() {
	if messagesProcessed != nil {
		messagesProcessed.Inc()
	}
}
