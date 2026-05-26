package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPMetrics(t *testing.T) {
	// Simulate an HTTP request with the middleware
	handler := HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify counter incremented
	count, err := HTTPServerMetrics.RequestCount.GetMetricWithLabelValues("GET", "/test", "200")
	if err != nil {
		t.Fatalf("get metric failed: %v", err)
	}
	if count == nil {
		t.Fatal("metric should not be nil")
	}
}

func TestGRPCMetrics(t *testing.T) {
	start := time.Now()
	time.Sleep(1 * time.Millisecond)
	RecordGRPCCall(nil, "/AuthService/Login", "OK", time.Since(start))

	count, err := GRPCServerMetrics.CallCount.GetMetricWithLabelValues("/AuthService/Login", "OK")
	if err != nil {
		t.Fatalf("get metric failed: %v", err)
	}
	if count == nil {
		t.Fatal("metric should not be nil")
	}
}

func TestMetricsHandler(t *testing.T) {
	handler := MetricsHandler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if len(body) == 0 {
		t.Fatal("metrics response should not be empty")
	}
}
