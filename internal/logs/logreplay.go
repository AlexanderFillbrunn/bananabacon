package bblogs

import (
	"bufio"
	"context"
	"log"
	"os"
	"regexp"
	"sync"
	"time"
)

type ReplayerOptions struct {
	FilterRegex string
	TimeRegex string
	TimeFormat string
}

type LogReplayer struct {
	options ReplayerOptions
	inputFile string
}

func NewLogReplayer(inputFile string, options ReplayerOptions) *LogReplayer {
	return &LogReplayer{
		inputFile: inputFile,
		options: options,
	}
}

func (lr *LogReplayer) Start(ctx context.Context, callback func(string)) {
	frx, err := regexp.Compile(lr.options.FilterRegex)
	if err != nil {
		log.Fatalf("Invalid filter regex: %s, err: %s", lr.options.FilterRegex, err)
	}
	trx, err := regexp.Compile(lr.options.TimeRegex)
	if err != nil {
		log.Fatalf("Invalid time regex: %s, err: %s", lr.options.TimeRegex, err)
	}

	file, err := os.Open(lr.inputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	lr.processFile(ctx, file, frx, trx, callback)
}

func (lr *LogReplayer) processFile(ctx context.Context, file *os.File, frx, trx *regexp.Regexp,
	callback func(string)) {
	scanner := bufio.NewScanner(file)
	var rst time.Time // real start time (now)
	var lst time.Time // log start time (when the first line was logged)
	var wg sync.WaitGroup
	var ctime time.Time // time of the first line of the current batch
	buffer := []string{}

	// TODO: optionally, resize scanner's capacity for lines over 64K
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		line := scanner.Text()
		
		// Check if the line matches the filter regex
		if !frx.MatchString(line) {
			continue
		}

		// Find the timestamp
		t, ok := lr.extractTimestamp(line, trx)
		if !ok {
			continue
		}

		// Check we have a logging start time and if yes, if this is before it
		if !lst.IsZero() && t.Before(lst) {
			continue
		}

		// If we have no ctime, we have an empty buffer
		if ctime.IsZero() {
			ctime = t
		}

		// If the difference between first line in buffer and new line is 
		if (t.Sub(ctime) > time.Duration(1) * time.Second) {
			if lst.IsZero() {
				// First line, no delay
				lst, rst = ctime, time.Now()
				lr.emitLines(buffer, callback)
			} else {
				wg.Add(1)
				lr.handleBufferedLines(buffer, &wg, ctime, lst, rst, callback)
				wg.Wait()
			}
			// Reset buffer and current time
			buffer = []string{}
			ctime = time.Time{}
		}
		buffer = append(buffer, line)
	}
	// Last lines, flush buffer
	if len(buffer) > 0 {
		wg.Add(1)
		lr.handleBufferedLines(buffer, &wg, ctime, lst, rst, callback)
		wg.Wait()
	}

	// Handle errors during scanning of file
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func (lr *LogReplayer) emitLines(lines []string, callback func(string)) {
	for _, l := range lines {
		callback(l)
	}
}

func (lr *LogReplayer) extractTimestamp(line string, trx *regexp.Regexp) (time.Time, bool) {
	matches := trx.FindStringSubmatch(line)
	if matches == nil || len(matches) < 2 {
		return time.Time{}, false
	}

	timestamp, err := time.Parse(lr.options.TimeFormat, matches[1])
	if err != nil {
		return time.Time{}, false
	}
	return timestamp, true
}

func (lr *LogReplayer) handleBufferedLines(lines []string, wg *sync.WaitGroup,
	t, lst, rst time.Time, callback func(string)) {
	diff := t.Sub(lst)
	ndiff := time.Since(rst)
	time.AfterFunc(diff - ndiff, func() {
		lr.emitLines(lines, callback)
		wg.Done()
	})
}