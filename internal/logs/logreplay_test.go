package logs

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLogReplayer_Start(t *testing.T) {
	// Create a temporary log file with sample log lines
	tempFile, err := os.CreateTemp("", "test-log-*.log")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %s", err)
	}
	defer os.Remove(tempFile.Name())

	logLines := `2023-01-01 00:00:01.000 Log line 1
2023-01-01 00:00:02.000 Log line 2
2023-01-01 00:00:03.000 Log line 3`
	if _, err := tempFile.WriteString(logLines); err != nil {
		t.Fatalf("Failed to write to temporary file: %s", err)
	}
	tempFile.Close()

	// Define options for the LogReplayer
	options := ReplayerOptions{
		FilterRegex: ".*",
		TimeRegex:   `(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}).*`,
		TimeFormat:  "2006-01-02 15:04:05.000",
		Loop:        false,
	}

	// Create a LogReplayer instance
	replayer := NewLogReplayer(tempFile.Name(), options)

	// Define the context and the callback function
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var processedLines []string
	callback := func(line string) {
		processedLines = append(processedLines, line)
	}

	// Start the LogReplayer
	startTime := time.Now().In(time.UTC)
	replayer.Start(ctx, startTime, callback)

	// Validate the results
	expectedLines := []string{
		"2023-01-01 00:00:01.000 Log line 1",
		"2023-01-01 00:00:02.000 Log line 2",
		"2023-01-01 00:00:03.000 Log line 3",
	}

	if len(processedLines) != len(expectedLines) {
		t.Fatalf("Expected %d processed lines, got %d", len(expectedLines), len(processedLines))
	}

	expected := startTime
	for i, line := range processedLines {
		tstr := line[:strings.Index(line, "Log line") - 1]
		ts, err := time.Parse(options.TimeFormat, tstr)
		if err != nil {
			t.Errorf("Failed to parse timestamp: %s", err)
		}

		if ts.Sub(expected).Abs() > time.Millisecond*200 {
			t.Errorf("Expected timestamp %s, got %s", expected, ts)
		}
		expected = expected.Add(time.Second)
		if !strings.Contains(line, "Log line " + strconv.Itoa(i+1)) {
			t.Errorf("Expected line %d to contain 'Log line %d', got %q", i+1, i+1, line)
		}
	}
}