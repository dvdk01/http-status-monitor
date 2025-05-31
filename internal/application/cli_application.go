package application

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/dvdk01/http-status-monitor/internal/schema"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

type cliApplication struct {
	statsChan chan map[string]*schema.URLStats
}

func NewCLIApplication(statsChan chan map[string]*schema.URLStats) *cliApplication {
	return &cliApplication{statsChan: statsChan}
}

func (ca *cliApplication) renderStats(stats map[string]*schema.URLStats) {
	ca.clear()
	dumpTable(stats)
}

func (ca *cliApplication) clear() {
	fmt.Print("\033[H\033[2J")
}

func (ca *cliApplication) Render(stats map[string]*schema.URLStats) {
	ca.clear()
	ca.renderStats(stats)
}

func (ca *cliApplication) Start(ctx context.Context) error {

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case stats := <-ca.statsChan:
				ca.Render(stats)

			}
		}

	}()

	return nil
}

func colorizeStatus(successRate int, str string) string {
	switch {
	case successRate >= 90:
		return text.FgGreen.Sprint(str)
	case successRate >= 50:
		return text.FgYellow.Sprint(str)
	default:
		return text.FgRed.Sprint(str)
	}
}

func colorizeStatusCode(code int, txt string) string {
	switch {
	case code >= 200 && code < 300:
		return text.FgGreen.Sprint(txt)
	case code >= 300 && code < 400:
		return text.FgBlue.Sprint(txt)
	case code >= 400 && code < 500:
		return text.FgYellow.Sprint(txt)
	case code >= 500:
		return text.FgRed.Sprint(txt)
	default:
		return txt
	}
}

func dumpTable(stats map[string]*schema.URLStats) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)

	t.AppendHeader(table.Row{
		"URL", "Status",
		"Min Duration", "Max Duration", "Avg Duration",
		"Min Payload", "Max Payload", "Avg Payload",
		"Status Codes",
	})

	// Sort URLs alphabetically
	urls := make([]string, 0, len(stats))
	for url := range stats {
		urls = append(urls, url)
	}
	sort.Strings(urls)

	for _, url := range urls {
		stat := stats[url]
		successRate := stat.SuccessPercentage()
		status := fmt.Sprintf("%d/%d %d%%", stat.SuccessCount, stat.TotalRequests, successRate)
		status = colorizeStatus(successRate, status)

		statusCodes := ""
		for code, count := range stat.StatusCodes {
			codeText := fmt.Sprintf("%d:%d", code, count)
			statusCodes += colorizeStatusCode(code, codeText) + " "
		}
		if statusCodes == "" {
			statusCodes = "NO STATUS CODE"
		}

		t.AppendRow(table.Row{
			url,
			status,
			stat.MinDuration.Round(time.Millisecond),
			stat.MaxDuration.Round(time.Millisecond),
			stat.AvgDuration().Round(time.Millisecond),
			stat.MinPayload,
			stat.MaxPayload,
			stat.AvgPayload(),
			statusCodes,
		})
	}

	t.Render()
}
