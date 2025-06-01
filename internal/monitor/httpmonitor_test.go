package monitor

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dvdk01/http-status-monitor/internal/schema"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestHTTPMonitor_makeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		url          string
		mockResponse *http.Response
		mockError    error
		expected     schema.RequestResult
	}{
		// Test case for successful HTTP request with 200 status code
		// Verifies that the monitor correctly processes a successful response
		// and sets the appropriate success flag and status code
		{
			name: "successful request",
			url:  "http://example.com/success",
			mockResponse: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("test response")),
			},
			mockError: nil,
			expected: schema.RequestResult{
				URL:     "http://example.com/success",
				Status:  200,
				Success: true,
			},
		},
		// Test case for failed HTTP request with 404 status code
		// Verifies that the monitor correctly handles a not found response
		// and sets the success flag to false while preserving the status code
		{
			name: "404 response",
			url:  "http://example.com/notfound",
			mockResponse: &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("not found")),
			},
			mockError: nil,
			expected: schema.RequestResult{
				URL:     "http://example.com/notfound",
				Status:  404,
				Success: false,
			},
		},
		// Test case for network error during HTTP request
		// Verifies that the monitor correctly handles connection errors
		// and sets appropriate error state and success flag
		{
			name:         "request error",
			url:          "http://error.com",
			mockResponse: nil,
			mockError:    errors.New("connection refused"),
			expected: schema.RequestResult{
				URL:     "http://error.com",
				Success: false,
				Error:   errors.New("connection refused"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a new transport for each test
			transport := httpmock.NewMockTransport()
			client := &http.Client{Transport: transport}
			defer transport.Reset()

			// Register responder for this test
			if tt.mockError != nil {
				transport.RegisterResponder("GET", tt.url,
					func(req *http.Request) (*http.Response, error) {
						return nil, tt.mockError
					},
				)
			} else {
				transport.RegisterResponder("GET", tt.url,
					func(req *http.Request) (*http.Response, error) {
						return tt.mockResponse, nil
					},
				)
			}

			monitor := &httpMonitor{
				client: client,
				stats:  make(map[string]*schema.URLStats),
			}

			result := monitor.makeRequest(tt.url, time.Second)

			// Ověření základních vlastností
			assert.Equal(t, tt.expected.URL, result.URL)
			if tt.mockError != nil {
				assert.False(t, result.Success)
				assert.Equal(t, 0, result.Status)
				assert.Error(t, result.Error)
			} else {
				assert.Equal(t, tt.expected.Success, result.Success)
				assert.Equal(t, tt.expected.Status, result.Status)
				assert.NoError(t, result.Error)
			}
			assert.NotZero(t, result.Duration)
		})
	}
}

func TestHTTPMonitor_updateStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		initialStats  map[string]*schema.URLStats
		result        schema.RequestResult
		expectedStats map[string]*schema.URLStats
	}{
		// Test case for first request to a URL
		// Verifies that initial statistics are correctly set up
		// including min/max values and counters
		{
			name: "first request",
			initialStats: map[string]*schema.URLStats{
				"http://example.com": {
					URL:         "http://example.com",
					StatusCodes: make(map[int]int),
					MinDuration: time.Duration(^uint64(0) >> 1), // math.MaxInt64
					MinPayload:  int(^uint(0) >> 1),             // math.MaxInt
				},
			},
			result: schema.RequestResult{
				URL:         "http://example.com",
				Duration:    time.Second,
				PayloadSize: 100,
				Status:      200,
				Success:     true,
			},
			expectedStats: map[string]*schema.URLStats{
				"http://example.com": {
					URL:           "http://example.com",
					TotalRequests: 1,
					SuccessCount:  1,
					MinDuration:   time.Second,
					MaxDuration:   time.Second,
					TotalDuration: time.Second,
					MinPayload:    100,
					MaxPayload:    100,
					TotalPayload:  100,
					StatusCodes:   map[int]int{200: 1},
				},
			},
		},
		// Test case for updating existing statistics
		// Verifies that subsequent requests correctly update
		// min/max values, counters, and status code distribution
		{
			name: "update existing stats",
			initialStats: map[string]*schema.URLStats{
				"http://example.com": {
					URL:           "http://example.com",
					TotalRequests: 1,
					SuccessCount:  1,
					MinDuration:   time.Second,
					MaxDuration:   time.Second,
					TotalDuration: time.Second,
					MinPayload:    100,
					MaxPayload:    100,
					TotalPayload:  100,
					StatusCodes:   map[int]int{200: 1},
				},
			},
			result: schema.RequestResult{
				URL:         "http://example.com",
				Duration:    2 * time.Second,
				PayloadSize: 200,
				Status:      404,
				Success:     false,
			},
			expectedStats: map[string]*schema.URLStats{
				"http://example.com": {
					URL:           "http://example.com",
					TotalRequests: 2,
					SuccessCount:  1,
					MinDuration:   time.Second,
					MaxDuration:   2 * time.Second,
					TotalDuration: 3 * time.Second,
					MinPayload:    100,
					MaxPayload:    200,
					TotalPayload:  300,
					StatusCodes:   map[int]int{200: 1, 404: 1},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			monitor := &httpMonitor{
				stats: tt.initialStats,
			}

			monitor.updateStats(tt.result)

			// Ověření statistik
			stats := monitor.stats[tt.result.URL]
			expected := tt.expectedStats[tt.result.URL]

			assert.Equal(t, expected.TotalRequests, stats.TotalRequests)
			assert.Equal(t, expected.SuccessCount, stats.SuccessCount)
			assert.Equal(t, expected.MinDuration, stats.MinDuration)
			assert.Equal(t, expected.MaxDuration, stats.MaxDuration)
			assert.Equal(t, expected.TotalDuration, stats.TotalDuration)
			assert.Equal(t, expected.MinPayload, stats.MinPayload)
			assert.Equal(t, expected.MaxPayload, stats.MaxPayload)
			assert.Equal(t, expected.TotalPayload, stats.TotalPayload)
			assert.Equal(t, expected.StatusCodes, stats.StatusCodes)
		})
	}
}

func TestHTTPMonitor_GetStats(t *testing.T) {
	t.Parallel()

	// Test case for retrieving statistics
	// Verifies that GetStats returns a deep copy of the statistics
	// and modifications to the returned copy don't affect the original data
	initialStats := map[string]*schema.URLStats{
		"http://example.com": {
			URL:           "http://example.com",
			TotalRequests: 1,
			SuccessCount:  1,
			MinDuration:   time.Second,
			MaxDuration:   time.Second,
			TotalDuration: time.Second,
			MinPayload:    100,
			MaxPayload:    100,
			TotalPayload:  100,
			StatusCodes:   map[int]int{200: 1},
		},
	}

	monitor := &httpMonitor{
		stats: initialStats,
	}

	stats := monitor.GetStats()

	// Ověření, že jsme dostali kopii statistik
	assert.Equal(t, initialStats["http://example.com"].TotalRequests, stats["http://example.com"].TotalRequests)
	assert.Equal(t, initialStats["http://example.com"].SuccessCount, stats["http://example.com"].SuccessCount)
	assert.Equal(t, initialStats["http://example.com"].MinDuration, stats["http://example.com"].MinDuration)
	assert.Equal(t, initialStats["http://example.com"].MaxDuration, stats["http://example.com"].MaxDuration)
	assert.Equal(t, initialStats["http://example.com"].TotalDuration, stats["http://example.com"].TotalDuration)
	assert.Equal(t, initialStats["http://example.com"].MinPayload, stats["http://example.com"].MinPayload)
	assert.Equal(t, initialStats["http://example.com"].MaxPayload, stats["http://example.com"].MaxPayload)
	assert.Equal(t, initialStats["http://example.com"].StatusCodes, stats["http://example.com"].StatusCodes)

	// Ověření, že změna kopie neovlivní originál
	stats["http://example.com"].TotalRequests = 2
	assert.Equal(t, 1, initialStats["http://example.com"].TotalRequests)
}
