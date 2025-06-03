package monitor

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/dvdk01/http-status-monitor/internal/schema"
)

type httpMonitor struct {
	urls      []string
	client    *http.Client
	stats     map[string]*schema.URLStats
	mutex     sync.RWMutex
	statsChan chan map[string]*schema.URLStats
}

func (m *httpMonitor) Start(ctx context.Context) error {

	for _, url := range m.urls {
		m.stats[url] = &schema.URLStats{
			URL:         url,
			StatusCodes: make(map[int]int),
			MinDuration: time.Duration(^uint64(0) >> 1), // math.Maxint alternative (to avoid dependency on math package)
			MinPayload:  int(^uint(0) >> 1),             // math.Maxint alternative (to avoid dependency on math package)
		}
	}

	var wg sync.WaitGroup
	for _, url := range m.urls {
		wg.Add(1)
		go m.monitorURL(ctx, url, &wg, 5*time.Second, 10*time.Second)
	}

	wg.Wait()
	return nil
}

func (m *httpMonitor) Stop() error {
	return nil
}

func (m *httpMonitor) GetStats() map[string]*schema.URLStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	//Generate deepcopy of stats
	stats := make(map[string]*schema.URLStats)
	for k, v := range m.stats {
		stats[k] = v
		stats[k] = &schema.URLStats{
			URL:           v.URL,
			TotalRequests: v.TotalRequests,
			SuccessCount:  v.SuccessCount,
			MinDuration:   v.MinDuration,
			MaxDuration:   v.MaxDuration,
			TotalDuration: v.TotalDuration,
			MinPayload:    v.MinPayload,
			MaxPayload:    v.MaxPayload,
			TotalPayload:  v.TotalPayload,
			StatusCodes:   make(map[int]int),
		}
		for code, count := range v.StatusCodes {
			stats[k].StatusCodes[code] = count
		}
	}
	return stats
}

func (m *httpMonitor) monitorURL(ctx context.Context, url string, wg *sync.WaitGroup, interval time.Duration, timeout time.Duration) {
	defer wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Execute initial request immediately without waiting for the first tick
	result := m.makeRequest(ctx, url, timeout)
	m.updateStats(result)
	go func() {
		m.statsChan <- m.GetStats()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result := m.makeRequest(ctx, url, timeout)
			m.updateStats(result)
			go func() {
				m.statsChan <- m.GetStats()
			}()

		}
	}
}

func (m *httpMonitor) makeRequest(ctx context.Context, url string, timeout time.Duration) schema.RequestResult {
	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		return schema.RequestResult{
			URL:     url,
			Error:   err,
			Success: false,
		}
	}

	start := time.Now()
	resp, err := m.client.Do(req)
	duration := time.Since(start)

	result := schema.RequestResult{
		URL:      url,
		Duration: duration,
	}

	if err != nil {
		result.Error = err
		result.Success = false
		return result
	}
	defer resp.Body.Close() //nolint

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.PayloadSize = 0
	}
	result.PayloadSize = len(body)

	result.Status = resp.StatusCode
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 400

	return result
}

func (m *httpMonitor) updateStats(result schema.RequestResult) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	stats := m.stats[result.URL]
	stats.TotalRequests++

	if result.Success {
		stats.SuccessCount++
	}

	if result.Duration < stats.MinDuration {
		stats.MinDuration = result.Duration
	}
	if result.Duration > stats.MaxDuration {
		stats.MaxDuration = result.Duration
	}

	stats.TotalDuration += result.Duration

	if result.PayloadSize < stats.MinPayload {
		stats.MinPayload = result.PayloadSize
	}
	if result.PayloadSize > stats.MaxPayload {
		stats.MaxPayload = result.PayloadSize
	}

	stats.TotalPayload += result.PayloadSize

	if result.Status > 0 {
		stats.StatusCodes[result.Status]++
	}
}

func NewMonitor(client *http.Client, urls []string, statsChan chan map[string]*schema.URLStats) Monitor {
	return &httpMonitor{
		client:    client,
		stats:     make(map[string]*schema.URLStats),
		urls:      urls,
		statsChan: statsChan,
	}
}
