package e2e

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"

	"github.com/dvdk01/http-status-monitor/internal/monitor"
	"github.com/dvdk01/http-status-monitor/internal/schema"
	"github.com/stretchr/testify/assert"
)

// Test case for monitoring multiple URLs simultaneously
// Verifies that the monitor can handle multiple URLs in parallel
// and correctly tracks statistics for each URL independently
func TestMonitor_MultipleURLs(t *testing.T) {
	t.Parallel()

	urls := []string{"https://test1.com", "https://test2.com"}
	transport := httpmock.NewMockTransport()
	client := &http.Client{Transport: transport}
	defer transport.Reset()

	transport.RegisterResponder("GET", "https://test1.com",
		func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("ok1")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		},
	)
	transport.RegisterResponder("GET", "https://test2.com",
		func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		},
	)

	statsChan := make(chan map[string]*schema.URLStats, 10)
	mon := monitor.NewMonitor(client, urls, statsChan)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	go mon.Start(ctx) //nolint:errcheck

	timeout := time.After(2 * time.Second)
	var stats map[string]*schema.URLStats

WAIT:
	for {
		select {
		case s := <-statsChan:
			stats = s
			allReady := true
			for _, url := range urls {
				if stats[url].TotalRequests < 1 {
					allReady = false
					break
				}
			}
			if allReady {
				break WAIT
			}
		case <-timeout:
			t.Fatal("Timeout waiting for all URLs to have at least one request")
		}
	}

	assert.Contains(t, stats, "https://test1.com")
	assert.Contains(t, stats, "https://test2.com")

	t1 := stats["https://test1.com"]
	t2 := stats["https://test2.com"]

	assert.GreaterOrEqual(t, t1.TotalRequests, 1)
	assert.GreaterOrEqual(t, t2.TotalRequests, 1)
	assert.Equal(t, t1.SuccessCount, t1.TotalRequests)
	assert.Equal(t, 0, t2.SuccessCount)
	assert.Equal(t, 200, findStatus(t1.StatusCodes))
	assert.Equal(t, 404, findStatus(t2.StatusCodes))
}

func findStatus(m map[int]int) int {
	for k := range m {
		return k
	}
	return 0
}

// Test case for handling request timeouts
// Verifies that the monitor correctly handles and reports
// timeout errors when requests exceed their deadline
func TestMonitor_Timeout(t *testing.T) {
	t.Parallel()

	url := "https://timeout.com"
	transport := httpmock.NewMockTransport()
	client := &http.Client{Transport: transport}
	defer transport.Reset()

	transport.RegisterResponder("GET", url,
		func(req *http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		},
	)

	statsChan := make(chan map[string]*schema.URLStats, 5)
	mon := monitor.NewMonitor(client, []string{url}, statsChan)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go mon.Start(ctx) //nolint:errcheck

	timeout := time.After(2 * time.Second)
	var stats map[string]*schema.URLStats

WAIT:
	for {
		select {
		case s := <-statsChan:
			stats = s
			allReady := true
			for _, url := range []string{url} {
				if stats[url].TotalRequests < 1 {
					allReady = false
					break
				}
			}
			if allReady {
				break WAIT
			}
		case <-timeout:
			t.Fatal("Timeout waiting for all URLs to have at least one request")
		}
	}

	result := stats[url]
	assert.Equal(t, 0, result.SuccessCount)
	assert.GreaterOrEqual(t, result.TotalRequests, 1)
}

// Test case for monitoring different HTTP status codes
// Verifies that the monitor correctly tracks and reports
// various HTTP status codes (200, 302, 500) and their success/failure states
func TestMonitor_HTTPStatusCodes(t *testing.T) {
	t.Parallel()

	urls := []string{"https://ok.com", "https://redirect.com", "https://fail.com"}
	transport := httpmock.NewMockTransport()
	client := &http.Client{Transport: transport}
	defer transport.Reset()

	transport.RegisterResponder("GET", "https://ok.com",
		func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		},
	)
	transport.RegisterResponder("GET", "https://redirect.com",
		func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 302,
				Body:       io.NopCloser(strings.NewReader("redirect")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		},
	)
	transport.RegisterResponder("GET", "https://fail.com",
		func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(strings.NewReader("fail")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		},
	)

	statsChan := make(chan map[string]*schema.URLStats, 10)
	mon := monitor.NewMonitor(client, urls, statsChan)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	go mon.Start(ctx) //nolint:errcheck

	timeout := time.After(2 * time.Second)
	var stats map[string]*schema.URLStats

WAIT:
	for {
		select {
		case s := <-statsChan:
			stats = s
			allReady := true
			for _, url := range urls {
				if stats[url].TotalRequests < 1 {
					allReady = false
					break
				}
			}
			if allReady {
				break WAIT
			}
		case <-timeout:
			t.Fatal("Timeout waiting for all URLs to have at least one request")
		}
	}

	assert.Equal(t, stats[urls[0]].TotalRequests, stats[urls[0]].SuccessCount)
	assert.Equal(t, stats[urls[1]].TotalRequests, stats[urls[1]].SuccessCount)
	assert.Equal(t, 0, stats[urls[2]].SuccessCount)
}

// Test case for monitoring payload size statistics
// Verifies that the monitor correctly tracks and reports
// minimum, maximum, and total payload sizes for responses
func TestMonitor_PayloadSizeStats(t *testing.T) {
	t.Parallel()

	url := "https://payload.com"
	body := "1234567890"
	transport := httpmock.NewMockTransport()
	client := &http.Client{Transport: transport}
	defer transport.Reset()

	transport.RegisterResponder("GET", url,
		func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		},
	)

	statsChan := make(chan map[string]*schema.URLStats, 5)
	mon := monitor.NewMonitor(client, []string{url}, statsChan)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	go mon.Start(ctx) //nolint:errcheck

	timeout := time.After(2 * time.Second)
	var stats map[string]*schema.URLStats

WAIT:
	for {
		select {
		case s := <-statsChan:
			stats = s
			allReady := true
			for _, url := range []string{url} {
				if stats[url].TotalRequests < 1 {
					allReady = false
					break
				}
			}
			if allReady {
				break WAIT
			}
		case <-timeout:
			t.Fatal("Timeout waiting for all URLs to have at least one request")
		}
	}

	result := stats[url]
	assert.GreaterOrEqual(t, result.MinPayload, len(body))
	assert.GreaterOrEqual(t, result.MaxPayload, len(body))
	assert.GreaterOrEqual(t, result.TotalPayload, len(body))
}

// Test case for monitoring many URLs in parallel
// Verifies that the monitor can handle a large number of URLs
// simultaneously without any issues or race conditions
func TestMonitor_ManyParallelURLs(t *testing.T) {
	t.Parallel()

	urls := []string{}
	transport := httpmock.NewMockTransport()
	client := &http.Client{Transport: transport}
	defer transport.Reset()

	for i := 0; i < 10; i++ {
		url := "https://multi" + string(rune('A'+i)) + ".com"
		urls = append(urls, url)
		transport.RegisterResponder("GET", url,
			func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("ok")),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			},
		)
	}

	statsChan := make(chan map[string]*schema.URLStats, 20)
	mon := monitor.NewMonitor(client, urls, statsChan)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	go mon.Start(ctx) //nolint:errcheck

	timeout := time.After(2 * time.Second)
	var stats map[string]*schema.URLStats

WAIT:
	for {
		select {
		case s := <-statsChan:
			stats = s
			allReady := true
			for _, url := range urls {
				if stats[url].TotalRequests < 1 {
					allReady = false
					break
				}
			}
			if allReady {
				break WAIT
			}
		case <-timeout:
			t.Fatal("Timeout waiting for all URLs to have at least one request")
		}
	}

	for _, url := range urls {
		assert.Contains(t, stats, url)
		assert.GreaterOrEqual(t, stats[url].TotalRequests, 1)
		assert.Equal(t, stats[url].TotalRequests, stats[url].SuccessCount)
	}
}
