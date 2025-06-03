package monitor

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/dvdk01/http-status-monitor/internal/schema"
)

type urlMonitor struct {
	client    *http.Client
	stats     *schema.URLStats
	url       string
	mutex     sync.RWMutex
	statsChan chan *schema.URLStats
}

func (m *urlMonitor) GetURL() string {
	return m.url
}

func (m *urlMonitor) Start(ctx context.Context) error {
	m.stats = &schema.URLStats{
		URL:         m.url,
		StatusCodes: make(map[int]int),
		MinDuration: time.Duration(^uint64(0) >> 1), // math.Maxint alternative (to avoid dependency on math package)
		MinPayload:  int(^uint(0) >> 1),             // math.Maxint alternative (to avoid dependency on math package)
	}
	m.monitorURL(ctx, 5*time.Second, 10*time.Second)
	return nil
}

func (m *urlMonitor) Stop() error {
	return nil
}

func (m *urlMonitor) GetStats() *schema.URLStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	//Generate deepcopy of stats
	stats := schema.URLStats{
		URL:           m.stats.URL,
		TotalRequests: m.stats.TotalRequests,
		SuccessCount:  m.stats.SuccessCount,
		MinDuration:   m.stats.MinDuration,
		MaxDuration:   m.stats.MaxDuration,
		TotalDuration: m.stats.TotalDuration,
		MinPayload:    m.stats.MinPayload,
		MaxPayload:    m.stats.MaxPayload,
		TotalPayload:  m.stats.TotalPayload,
		StatusCodes:   make(map[int]int),
	}

	for code, count := range m.stats.StatusCodes {
		stats.StatusCodes[code] = count
	}

	return &stats
}

func (m *urlMonitor) monitorURL(ctx context.Context, interval time.Duration, timeout time.Duration) {
	//defer wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Execute initial request immediately without waiting for the first tick
	result := m.makeRequest(ctx, m.url, timeout)
	m.updateStats(result)
	go func() {
		m.statsChan <- m.GetStats()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result := m.makeRequest(ctx, m.url, timeout)
			m.updateStats(result)
			go func() {
				m.statsChan <- m.GetStats()
			}()
		}
	}
}

func (m *urlMonitor) makeRequest(ctx context.Context, url string, timeout time.Duration) schema.RequestResult {
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

func (m *urlMonitor) updateStats(result schema.RequestResult) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	stats := m.stats
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

func NewMonitor(client *http.Client, url string, statsChan chan *schema.URLStats) *urlMonitor {
	return &urlMonitor{
		url:       url,
		client:    client,
		statsChan: statsChan,
	}
}
