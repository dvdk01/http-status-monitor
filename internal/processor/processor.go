package processor

import (
	"context"
	"github.com/dvdk01/http-status-monitor/internal/application"
	"github.com/dvdk01/http-status-monitor/internal/monitor"
	log "github.com/sirupsen/logrus"
	"os"
	"sync"
)

type processor struct {
	monitor     monitor.Monitor
	application application.Application
	wg          *sync.WaitGroup
}

func New(wg *sync.WaitGroup, monitor monitor.Monitor, display application.Application) *processor {
	return &processor{wg: wg, monitor: monitor, application: display}
}

func (m *processor) Start(ctx context.Context) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		if err := m.monitor.Start(ctx); err != nil {
			log.WithError(err).Error("failed to start monitor")
			os.Exit(1)
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
	m.application.Render(m.monitor.GetStats())
}
