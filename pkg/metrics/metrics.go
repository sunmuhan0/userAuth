package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "ttuser"
)

var (
	// HTTPServerMetrics 记录HTTP服务端请求指标
	HTTPServerMetrics *httpServerCollectors
	// HTTPClientMetrics 记录HTTP客户端请求指标（第三方API调用）
	HTTPClientMetrics *httpClientCollectors
	// GRPCServerMetrics 记录gRPC调用指标
	GRPCServerMetrics *grpcServerCollectors
)

type httpServerCollectors struct {
	RequestCount   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ActiveRequests prometheus.Gauge
}

type httpClientCollectors struct {
	RequestCount    *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ActiveRequests  prometheus.Gauge
}

type grpcServerCollectors struct {
	CallCount   *prometheus.CounterVec
	CallDuration *prometheus.HistogramVec
}

func init() {
	HTTPServerMetrics = &httpServerCollectors{
		RequestCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"method", "path"},
		),
		ActiveRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_requests_active",
				Help:      "Number of active HTTP requests",
			},
		),
	}

	HTTPClientMetrics = &httpClientCollectors{
		RequestCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_client_requests_total",
				Help:      "Total number of HTTP client requests",
			},
			[]string{"method", "host", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_client_request_duration_seconds",
				Help:      "HTTP client request duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"method", "host"},
		),
		ActiveRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_client_requests_active",
				Help:      "Number of active HTTP client requests",
			},
		),
	}

	GRPCServerMetrics = &grpcServerCollectors{
		CallCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "grpc_calls_total",
				Help:      "Total number of gRPC calls",
			},
			[]string{"method", "status"},
		),
		CallDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "grpc_call_duration_seconds",
				Help:      "gRPC call duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"method"},
		),
	}

	prometheus.MustRegister(HTTPServerMetrics.RequestCount)
	prometheus.MustRegister(HTTPServerMetrics.RequestDuration)
	prometheus.MustRegister(HTTPServerMetrics.ActiveRequests)
	prometheus.MustRegister(HTTPClientMetrics.RequestCount)
	prometheus.MustRegister(HTTPClientMetrics.RequestDuration)
	prometheus.MustRegister(HTTPClientMetrics.ActiveRequests)
	prometheus.MustRegister(GRPCServerMetrics.CallCount)
	prometheus.MustRegister(GRPCServerMetrics.CallDuration)
}

// MetricsHandler 返回Prometheus metrics HTTP handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// HTTPMiddleware 返回HTTP请求指标中间件
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		HTTPServerMetrics.ActiveRequests.Inc()
		start := time.Now()

		sw := statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(&sw, r)

		duration := time.Since(start).Seconds()
		HTTPServerMetrics.ActiveRequests.Dec()

		path := r.URL.Path
		HTTPServerMetrics.RequestCount.WithLabelValues(r.Method, path, fmt.Sprintf("%d", sw.statusCode)).Inc()
		HTTPServerMetrics.RequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

type grpcMetricsKey struct{}

// ContextWithGRPCMetrics 将gRPC指标收集器存入context
func ContextWithGRPCMetrics(ctx context.Context) context.Context {
	return context.WithValue(ctx, grpcMetricsKey{}, GRPCServerMetrics)
}

// RecordGRPCCall 记录gRPC调用指标（从拦截器调用）
func RecordGRPCCall(ctx context.Context, method string, status string, duration time.Duration) {
	GRPCServerMetrics.CallCount.WithLabelValues(method, status).Inc()
	GRPCServerMetrics.CallDuration.WithLabelValues(method).Observe(duration.Seconds())
}
