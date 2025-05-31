package schema

import "time"

type RequestResult struct {
	URL         string
	Duration    time.Duration
	PayloadSize int
	Status      int
	Success     bool
	Error       error
}

type URLStats struct {
	URL           string
	TotalRequests int
	SuccessCount  int
	MinDuration   time.Duration
	MaxDuration   time.Duration
	TotalDuration time.Duration
	MinPayload    int
	MaxPayload    int
	TotalPayload  int
	StatusCodes   map[int]int
}

func (stats *URLStats) AvgDuration() time.Duration {
	if stats.TotalRequests == 0 {
		return 0
	}
	return stats.TotalDuration / time.Duration(stats.TotalRequests)
}
func (stats *URLStats) AvgPayload() int {
	if stats.TotalRequests == 0 {
		return 0
	}
	return stats.TotalPayload / stats.TotalRequests
}
func (stats *URLStats) SuccessPercentage() int {
	if stats.TotalRequests == 0 {
		return 0
	}
	return int(100 * float32(stats.SuccessCount) / float32(stats.TotalRequests))
}
