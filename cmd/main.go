package main

import (
	bblogs "bananabacon/internal/logs"
	bbmetrics "bananabacon/internal/metrics"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

// getenv returns the value of the environment variable with the given key.
// If the key is not set, it returns the fallback value.
func getenv(key, fallback string) string {
    value := os.Getenv(key)
    if len(value) == 0 {
        return fallback
    }
    return value
}

// main runs the log replayer and prints the replayed log lines to stdout.
// Additionally, it reads metrics configuration from environment variables and
// exposes them via http.
// It stops when it receives a SIGTERM or SIGINT signal.
//
// It uses the following environment variables to configure the log replayer:
//
// - INPUT_FILE: the file to read the log from
// - FILTER_REGEX: a regex to filter out log lines that don't match
// - TIME_REGEX: a regex to extract timestamps from log lines
// - TIME_FORMAT: the format of the timestamps extracted by TIME_REGEX,
//     as understood by the time.Parse function.
func main() {
	file := getenv("INPUT_FILE", "/logs/test.log")
	filterRegex := getenv("FILTER_REGEX", ".*")
	timeRegex := getenv("TIME_REGEX", "(\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}\\.\\d{3}).*")
	timeFormat := getenv("TIME_FORMAT", "2006-01-02 15:04:05.000")
	lr := bblogs.NewLogReplayer(file, bblogs.ReplayerOptions{
		FilterRegex: filterRegex,
		TimeRegex: timeRegex,
		TimeFormat: timeFormat,
	})

	// Wrapper function for printing to stdout
	print := func(s string) {
        fmt.Println(s)
    }
	var ctx context.Context
	ctx, cancel := context.WithCancel(context.Background())

	// Read metrics from env vars and expose them via http
	engine := bbmetrics.NewMetricsEngineBuilderFromEnv().Build()
	portStr := getenv("METRICS_PORT", "3333")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid metrics port: %s, err: %s", portStr, err)
	}
	server := createMetricsServer(engine, port)

	// Capture SIGTERM and SIGINT
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-signalChan // Block until a signal is received
		cancel() // Cancel the context
		// Stop the server
		if err := server.Close(); err != nil {
            log.Fatalf("HTTP close error: %v", err)
        }
	}()

	// Start replaying the log
	go lr.Start(ctx, print)

	// Start the http server
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
        log.Fatalf("HTTP server error: %v", err)
    }
}

// createMetricsServer initializes and returns an HTTP server that will listen on the provided port
// and serves metrics at the "/metrics" endpoint. It evaluates each metric in the provided
// MetricsEngine and writes the results to the HTTP response. If an error occurs during
// evaluation of a metric, it is skipped.
func createMetricsServer(engine *bbmetrics.MetricsEngine, port int) (*http.Server) {
	server := &http.Server{
        Addr: ":" + strconv.Itoa(port),
    }
	http.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sb strings.Builder

		for _, m := range engine.Metrics {
			val, err := engine.Eval(m)
			if err == nil {
				sb.WriteString(val.String())
				sb.WriteString("\n")
			}
		}
		io.WriteString(w, sb.String())
	}))
	return server
}