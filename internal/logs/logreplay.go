package logs

import (
	"bufio"
	"context"
	"log"
	"os"
	"regexp"
	"time"
)

type ReplayerOptions struct {
	FilterRegex string
	TimeRegex string
	TimeFormat string
	Loop bool
}

type LogReplayer struct {
	options ReplayerOptions
	inputFile string
}

// NewLogReplayer creates a new LogReplayer object with the given input file and
// options. The options struct can be initialized with the following default
// values:
//
// - FilterRegex: ".*" (match all lines)
// - TimeRegex: "(\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}\\.\\d{3}).*" (match lines with
//   timestamps in the format 2006-01-02 15:04:05.000)
// - TimeFormat: "2006-01-02 15:04:05.000" (the format of the timestamps extracted
//   by TimeRegex)
//
// The returned LogReplayer object can be used to replay the log lines in the
// input file using the Start method.
func NewLogReplayer(inputFile string, options ReplayerOptions) *LogReplayer {
	return &LogReplayer{
		inputFile: inputFile,
		options: options,
	}
}

// Start replays the log lines in the input file according to the options given
// to NewLogReplayer. It will stop when the context is cancelled or when the
// end of the file is reached. The callback function is called on each log line
// that matches the filter regex and has a valid timestamp.
// mts defines the time the first log line is mapped to.
// This is usually time.Now, but can be different for testing.
func (lr *LogReplayer) Start(ctx context.Context, mst time.Time, callback func(string)) {
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

	start := time.Now()
	again := true
	for again {
		file.Seek(0, 0)
		lr.processFile(ctx, file, mst, frx, trx, callback)
		again = lr.options.Loop && ctx.Err() == nil
		mst = mst.Add(time.Since(start))
	}
}

// processFile reads a file line by line, applies a filter regex to each line and
// extracts a timestamp from each line that matches the filter regex. It then
// schedules a timer that will emit the lines at a time that ensures that the
// overall rate of the log replay is consistent with the timestamps in the
// log. This means that if the log has a gap of 10 seconds between two log
// lines, the timer will wait 10 seconds before emitting the second line.
// mts defines the time the first log line is mapped to.
// This is usually time.Now, but can be different for testing.
// The method returns when the context is cancelled or when the end of the
// file is reached.
func (lr *LogReplayer) processFile(ctx context.Context, file *os.File, mst time.Time, frx, trx *regexp.Regexp,
	callback func(string)) {
	scanner := bufio.NewScanner(file)
	rst := time.Now() // Real start time, i.e. when we started processing the file
	var lst time.Time // log start time (when the first line was logged)
	var ctime time.Time // time of the first line of the current batch

	// Channel for synchronization, used to wait for the timer to fire
	notify := make(chan struct{})
	defer close(notify)

	buffer := []string{}

	// TODO: optionally, resize scanner's capacity for lines over 64K
	for scanner.Scan() {
		line := scanner.Text()
		
		// Check if the line matches the filter regex
		if !frx.MatchString(line) {
			continue
		}

		// Find the timestamp
		t, line, ok := lr.extractAndReplaceTimestamp(line, mst, lst, trx)
		if !ok {
			// If timestamp could not be extracted, use first time of current batch.
			// If current batch is empty, ignore.
			if ctime.IsZero() {
				continue
			}
			t = ctime
		}

		// Check we have a logging start time and if yes, if this is before it
		if !lst.IsZero() && t.Before(lst) {
			continue
		}

		// If we have no ctime, we have an empty buffer
		if ctime.IsZero() {
			ctime = t
		}
		if lst.IsZero() {
			lst = ctime
		}

		// If the difference between first line in buffer and new line is 
		if (t.Sub(ctime) > time.Duration(500) * time.Millisecond) {
			timer, _ := lr.handleBufferedLines(buffer, notify, ctime, lst, rst, callback)
			lr.wait(ctx, notify, timer)
			// Reset buffer and current time
			buffer = []string{}
			ctime = time.Time{}
		}
		buffer = append(buffer, line)
	}
	// Last lines, flush buffer
	if len(buffer) > 0 {
		timer, _ := lr.handleBufferedLines(buffer, notify, ctime, lst, rst, callback)
		lr.wait(ctx, notify, timer)
	}

	// Handle errors during scanning of file
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

// wait pauses the execution until either the context is done or a notification
// is received on the provided channel. It is used to synchronize the log replay
// with the timing of the log entries, allowing for graceful cancellation using
// the context. When the context is cancelled, the passed timer is stopped.
func (lr *LogReplayer) wait(ctx context.Context, notify chan struct{}, timer *time.Timer) {
	select {
	case <-ctx.Done():
		timer.Stop()
		return
	case <- notify:
	}
}

// emitLines iterates over a slice of log lines and invokes the provided callback
// function on each line. It is used to output or process each log line individually
// after it has been buffered and is ready to be emitted.
func (lr *LogReplayer) emitLines(lines []string, callback func(string)) {
	for _, l := range lines {
		callback(l)
	}
}

// extractAndReplaceTimestamp extracts a timestamp from a log line using a given regular expression.
// It then replaces the extracted timestamp with a new, current, timestamp that ensures that the
// overall rate of the log replay is consistent with the timestamps in the log.
// It returns the extracted timestamp as a time.Time object, the modified log line,
// and a boolean indicating whether the extraction was successful.
// If the timestamp cannot be extracted or parsed, it returns a zero time, empty string, and false.
func (lr *LogReplayer) extractAndReplaceTimestamp(l string, mst, lst time.Time, trx *regexp.Regexp) (time.Time, string, bool) {
	matches := trx.FindStringSubmatchIndex(l)
	if matches == nil || len(matches) < 4 {
		return time.Time{}, "",false
	}
	tstr := l[matches[2]:matches[3]]

	ts, err := time.Parse(lr.options.TimeFormat, tstr)
	if err != nil {
		return time.Time{}, "", false
	}

	var nts time.Time
	if lst.IsZero() {
		nts = mst
	} else {
		// Replace timestamp with current timestamp
		nts = mst.Add(ts.Sub(lst))
	}

	return ts, (l[:matches[2]] + nts.Format(lr.options.TimeFormat) + l[matches[3]:]), true
}

// handleBufferedLines schedules a timer that will emit the given lines
// at a time that ensures that the overall rate of the log replay is
// consistent with the timestamps in the log. It returns a channel that
// will send a single value when the timer fires. The channel is closed afterwards.
func (lr *LogReplayer) handleBufferedLines(lines []string, notify chan struct{}, t, lst, rst time.Time,
	callback func(string)) (*time.Timer, error) {
	diff := t.Sub(lst)
	ndiff := time.Since(rst)
	dur := diff - ndiff
	if dur < 0 {
		dur = time.Duration(0)
	}
	timer := time.AfterFunc(dur, func() {
		lr.emitLines(lines, callback)
		notify <- struct{}{}
	})
	return timer, nil
}