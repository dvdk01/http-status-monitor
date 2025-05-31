package monitor

import (
	"context"

	"github.com/dvdk01/http-status-monitor/internal/schema"
)

type Monitor interface {
	Start(ctx context.Context) error

	Stop() error

	GetStats() map[string]*schema.URLStats
}
