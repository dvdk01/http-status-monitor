package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dvdk01/http-status-monitor/internal/monitor"
	"github.com/dvdk01/http-status-monitor/internal/schema"
	"github.com/stretchr/testify/assert"
)

type staticResponder struct {
	status int
	body   string
}

func (s *staticResponder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: s.status,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
		Request:    req,
	}
	return resp, nil
}

type multiResponder struct {
	responders map[string]*staticResponder
}

func (m *multiResponder) RoundTrip(req *http.Request) (*http.Response, error) {
	if r, ok := m.responders[req.URL.String()]; ok {
		return r.RoundTrip(req)
	}
	return nil, fmt.Errorf("no responder for %s", req.URL.String())
}

type timeoutRoundTripper struct{}

func (t *timeoutRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, context.DeadlineExceeded
}

// Test case for monitoring multiple URLs simultaneously
// Verifies that the monitor can handle multiple URLs in parallel
// and correctly tracks statistics for each URL independently
func TestMonitor_MultipleURLs(t *testing.T) {
	t.Parallel()

	urls := []string{"https://test1.com", "https://test2.com"}
	responder := &multiResponder{responders: map[string]*staticResponder{
		"https://test1.com": {status: 200, body: "ok1"},
		"https://test2.com": {status: 404, body: "not found"},
	}}
	client := &http.Client{Transport: responder}

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
	client := &http.Client{Transport: &timeoutRoundTripper{}}

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
	responder := &multiResponder{responders: map[string]*staticResponder{
		"https://ok.com":       {status: 200, body: "ok"},
		"https://redirect.com": {status: 302, body: "redirect"},
		"https://fail.com":     {status: 500, body: "fail"},
	}}
	client := &http.Client{Transport: responder}

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
	client := &http.Client{Transport: &staticResponder{status: 200, body: body}}

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
	responders := map[string]*staticResponder{}
	for i := 0; i < 10; i++ {
		url := "https://multi" + string(rune('A'+i)) + ".com"
		urls = append(urls, url)
		responders[url] = &staticResponder{status: 200, body: "ok"}
	}
	client := &http.Client{Transport: &multiResponder{responders: responders}}

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
