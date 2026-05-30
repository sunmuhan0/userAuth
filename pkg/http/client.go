package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ttuser/pkg/log"
	"ttuser/pkg/metrics"
)

type Client struct {
	httpClient *http.Client
}

type ClientOption struct {
	Timeout time.Duration
}

type Response struct {
	StatusCode int
	Body       []byte
}

func (r *Response) JSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

func NewClient(opts ...ClientOption) *Client {
	timeout := 30 * time.Second
	if len(opts) > 0 && opts[0].Timeout > 0 {
		timeout = opts[0].Timeout
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
	}
}

type RequestOption func(*http.Request)

func WithHeader(key, value string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

func WithBearerToken(token string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func (c *Client) Get(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodGet, url, nil, opts...)
}

func (c *Client) Post(ctx context.Context, url string, body interface{}, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodPost, url, body, opts...)
}

func (c *Client) Put(ctx context.Context, url string, body interface{}, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodPut, url, body, opts...)
}

func (c *Client) Delete(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodDelete, url, nil, opts...)
}

func (c *Client) Patch(ctx context.Context, url string, body interface{}, opts ...RequestOption) (*Response, error) {
	return c.do(ctx, http.MethodPatch, url, body, opts...)
}

func (c *Client) do(ctx context.Context, method, rawURL string, body interface{}, opts ...RequestOption) (*Response, error) {
	start := time.Now()
	metrics.HTTPClientMetrics.ActiveRequests.Inc()
	defer metrics.HTTPClientMetrics.ActiveRequests.Dec()

	var reqBodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body failed: %w", err)
		}
		reqBodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, reqBodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for _, opt := range opts {
		opt(req)
	}

	resp, err := c.httpClient.Do(req)
	cost := time.Since(start)
	host := req.URL.Host

	if err != nil {
		metrics.HTTPClientMetrics.RequestCount.WithLabelValues(method, host, "error").Inc()
		metrics.HTTPClientMetrics.RequestDuration.WithLabelValues(method, host).Observe(cost.Seconds())

		log.Error(ctx, "http_client request failed",
			"method", method,
			"url", rawURL,
			"cost", fmt.Sprintf("%dms", cost.Milliseconds()),
			"error", err,
		)
		return nil, fmt.Errorf("%s %s failed: %w", method, rawURL, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	metrics.HTTPClientMetrics.RequestCount.WithLabelValues(method, host, fmt.Sprintf("%d", resp.StatusCode)).Inc()
	metrics.HTTPClientMetrics.RequestDuration.WithLabelValues(method, host).Observe(cost.Seconds())

	log.Info(ctx, "http_client",
		"method", method,
		"url", rawURL,
		"status", resp.StatusCode,
		"cost", fmt.Sprintf("%dms", cost.Milliseconds()),
		"ret", tryParseJSON(respBody),
	)

	return &Response{StatusCode: resp.StatusCode, Body: respBody}, nil
}

func tryParseJSON(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return string(b)
	}
	return v
}
