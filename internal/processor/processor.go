package processor

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/dvdk01/http-status-monitor/internal/application"
	"github.com/dvdk01/http-status-monitor/internal/monitor"
	log "github.com/sirupsen/logrus"
)

type processor struct {
	monitor     monitor.Monitor
	application application.Application
}

func New(monitor monitor.Monitor, display application.Application) *processor {
	return &processor{monitor: monitor, application: display}
}

func (m *processor) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	go func() {
		defer wg.Done()
		wg.Add(1)
		if err := m.monitor.Start(ctx); err != nil {
			log.WithError(err).Error("failed to start monitor")
			os.Exit(1)
		}
	}()

	go func() {
		defer wg.Done()
		wg.Add(1)
		if err := m.application.Start(ctx); err != nil {
			log.WithError(err).Error("failed to start application")
			os.Exit(1)
		}
	}()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	<-sigterm
	cancel()
	wg.Wait()

	m.application.Render(m.monitor.GetStats())
}
