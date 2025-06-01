package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/dvdk01/http-status-monitor/internal/application"
	"github.com/dvdk01/http-status-monitor/internal/monitor"
	"github.com/dvdk01/http-status-monitor/internal/schema"

	"github.com/dvdk01/http-status-monitor/internal/processor"
	"github.com/dvdk01/http-status-monitor/internal/validator"
)

func printUsage(programName string) {
	fmt.Fprintf(os.Stderr, "Usage: %s <url1> <url2> ... <urlN>\n", programName)
}

func main() {
	args := os.Args[1:]
	args = removeDuplicates(args)

	if len(args) == 0 {
		printUsage(os.Args[0])
		os.Exit(1)
	}

	results := validator.NewURLValidator().ValidateURLs(args)

	if validator.HasInvalidURLs(results) {
		fmt.Fprintf(os.Stderr, "\nValidation failed: Some URLs are invalid\n")
		fmt.Fprintf(os.Stderr, "Invalid URLs: %v\n", results.GetInvalidURLs())

		printUsage(os.Args[0])
		os.Exit(1)
	}

	statsChan := make(chan map[string]*schema.URLStats)
	defer close(statsChan)

	display := application.NewCLIApplication(statsChan)
	monitor := monitor.NewMonitor(http.DefaultClient, args, statsChan)

	processor.New(monitor, display).Start()
}

func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
