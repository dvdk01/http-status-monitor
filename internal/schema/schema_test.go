package schema

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestURLStats_AvgDuration(t *testing.T) {
	tests := []struct {
		name     string
		stats    *URLStats
		expected time.Duration
	}{
		// Test case for calculating average duration with no requests
		// Verifies that zero is returned when there are no requests
		{
			name: "zero requests",
			stats: &URLStats{
				TotalRequests: 0,
				TotalDuration: 0,
			},
			expected: 0,
		},
		// Test case for calculating average duration with a single request
		// Verifies that the duration of a single request is returned as is
		{
			name: "single request",
			stats: &URLStats{
				TotalRequests: 1,
				TotalDuration: time.Second,
			},
			expected: time.Second,
		},
		// Test case for calculating average duration with multiple requests
		// Verifies that the average is correctly calculated for multiple requests
		{
			name: "multiple requests",
			stats: &URLStats{
				TotalRequests: 3,
				TotalDuration: 3 * time.Second,
			},
			expected: time.Second,
		},
		// Test case for calculating average duration with fractional result
		// Verifies that the average is correctly calculated when the result is not a whole number
		{
			name: "fractional duration",
			stats: &URLStats{
				TotalRequests: 2,
				TotalDuration: 3 * time.Second,
			},
			expected: time.Second + 500*time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.stats.AvgDuration())
		})
	}
}

func TestURLStats_AvgPayload(t *testing.T) {
	tests := []struct {
		name     string
		stats    *URLStats
		expected int
	}{
		// Test case for calculating average payload with no requests
		// Verifies that zero is returned when there are no requests
		{
			name: "zero requests",
			stats: &URLStats{
				TotalRequests: 0,
				TotalPayload:  0,
			},
			expected: 0,
		},
		// Test case for calculating average payload with a single request
		// Verifies that the payload size of a single request is returned as is
		{
			name: "single request",
			stats: &URLStats{
				TotalRequests: 1,
				TotalPayload:  100,
			},
			expected: 100,
		},
		// Test case for calculating average payload with multiple requests
		// Verifies that the average is correctly calculated for multiple requests
		{
			name: "multiple requests",
			stats: &URLStats{
				TotalRequests: 3,
				TotalPayload:  300,
			},
			expected: 100,
		},
		// Test case for calculating average payload with fractional result
		// Verifies that the average is correctly calculated when the result is not a whole number
		{
			name: "fractional payload",
			stats: &URLStats{
				TotalRequests: 2,
				TotalPayload:  300,
			},
			expected: 150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.stats.AvgPayload())
		})
	}
}

func TestURLStats_SuccessPercentage(t *testing.T) {
	tests := []struct {
		name     string
		stats    *URLStats
		expected int
	}{
		// Test case for calculating success percentage with no requests
		// Verifies that zero is returned when there are no requests
		{
			name: "zero requests",
			stats: &URLStats{
				TotalRequests: 0,
				SuccessCount:  0,
			},
			expected: 0,
		},
		// Test case for calculating success percentage with all successful requests
		// Verifies that 100% is returned when all requests are successful
		{
			name: "all successful",
			stats: &URLStats{
				TotalRequests: 2,
				SuccessCount:  2,
			},
			expected: 100,
		},
		// Test case for calculating success percentage with half successful requests
		// Verifies that 50% is returned when half of the requests are successful
		{
			name: "half successful",
			stats: &URLStats{
				TotalRequests: 2,
				SuccessCount:  1,
			},
			expected: 50,
		},
		// Test case for calculating success percentage with no successful requests
		// Verifies that 0% is returned when no requests are successful
		{
			name: "none successful",
			stats: &URLStats{
				TotalRequests: 2,
				SuccessCount:  0,
			},
			expected: 0,
		},
		// Test case for calculating success percentage with partial success
		// Verifies that the percentage is correctly calculated when the result is not a whole number
		{
			name: "partial success",
			stats: &URLStats{
				TotalRequests: 3,
				SuccessCount:  2,
			},
			expected: 66, // 2/3 ≈ 66.67%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.stats.SuccessPercentage())
		})
	}
}
