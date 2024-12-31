package main

import (
	bblogs "bananabacon/internal/logs"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func getenv(key, fallback string) string {
    value := os.Getenv(key)
    if len(value) == 0 {
        return fallback
    }
    return value
}

func main() {
	file := getenv("INPUT_FILE", "/Users/alexander/Development/go/BananaBacon/test.log")
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

	// Capture SIGTERM and SIGINT
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-signalChan // Block until a signal is received
		cancel() // Cancel the context
	}()

	// Start replaying the log
	lr.Start(ctx, print)
}