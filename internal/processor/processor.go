package processor

import (
	"context"
	"net/http"
	"os"
	"sync"

	"github.com/dvdk01/http-status-monitor/internal/application"
	"github.com/dvdk01/http-status-monitor/internal/monitor"
	"github.com/dvdk01/http-status-monitor/internal/schema"
	log "github.com/sirupsen/logrus"
)

type processor struct {
	monitors    []monitor.Monitor
	application application.Application
	wg          *sync.WaitGroup
	statsChan   chan *schema.URLStats
}

func New(wg *sync.WaitGroup, client *http.Client, urls []string, statsChan chan *schema.URLStats, display application.Application) *processor {
	monitors := make([]monitor.Monitor, len(urls))
	for i, url := range urls {
		monitors[i] = monitor.NewMonitor(client, url, statsChan)
	}
	return &processor{
		monitors:    monitors,
		application: display,
		wg:          wg,
		statsChan:   statsChan,
	}
}

func (m *processor) Start(ctx context.Context) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		for _, mon := range m.monitors {
			m.wg.Add(1)
			go func(monitor monitor.Monitor) {
				defer m.wg.Done()
				if err := monitor.Start(ctx); err != nil {
					log.WithError(err).Error("failed to start monitor")
					os.Exit(1)
				}
			}(mon)
		}
	}()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		if err := m.application.Start(ctx); err != nil {
			log.WithError(err).Error("failed to start application")
			os.Exit(1)
		}
	}()

	m.wg.Wait()
	stats := make(map[string]*schema.URLStats)
	for _, mon := range m.monitors {
		stats[mon.GetURL()] = mon.GetStats()
	}
	m.application.Render(stats)
}
