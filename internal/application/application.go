package application

import (
	"context"
	"github.com/dvdk01/http-status-monitor/internal/schema"
)

type Application interface {
	Start(ctx context.Context) error
	Render(stats map[string]*schema.URLStats)
}
