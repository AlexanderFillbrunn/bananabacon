package main

import (
	logs "bananabacon/internal/logs"
	metrics "bananabacon/internal/metrics"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
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
	loop := getenv("LOOP", "true")

	lr := logs.NewLogReplayer(file, logs.ReplayerOptions{
		FilterRegex: filterRegex,
		TimeRegex: timeRegex,
		TimeFormat: timeFormat,
		Loop: loop == "true",
	})

	// Wrapper function for printing to stdout
	print := func(s string) {
        fmt.Println(s)
    }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine := createMetricsEngine()
	port := getPort()

	server := metrics.NewMetricsServer(engine, port)


	// Capture SIGTERM and SIGINT
	// Cancel is called when a signal is received, we do not need it
	ctx, _ = signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	
	// Start serving metrics
	go server.Run(ctx)

	// Start replaying the log
	go lr.Start(ctx, time.Now(), print)

	<-ctx.Done()
}

func createMetricsEngine() *metrics.MetricsEngine {
	builder, err := metrics.NewMetricsEngineBuilderFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	// Read metrics from env vars and expose them via http
	return builder.Build()
}

func getPort() int {
	portStr := getenv("METRICS_PORT", "8080")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid metrics port: %s, err: %s", portStr, err)
	}
	return port
}