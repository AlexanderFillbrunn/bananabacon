package metrics

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestMetricsServer(t *testing.T) {
	// Mock metrics engine
	engine := NewMetricsEngine([]*Metric{
		NewMetric("test_one", CounterType, "99", nil, ""),
		NewMetric("test_two", GaugeType, "999", nil, "Test"),
	})

	port := 8081 // TODO: randomize port
	server := NewMetricsServer(engine, port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		server.Run(ctx)
	}()

	// Allow the server some time to start
	time.Sleep(100 * time.Millisecond)

	// Make an HTTP GET request to the /metrics endpoint
	resp, err := http.Get("http://localhost:" + strconv.Itoa(port) + "/metrics")
	if err != nil {
		t.Fatalf("Failed to connect to metrics endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.StatusCode)
	}

	// Read and verify the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	expectedMetrics := "# TYPE test_one counter\ntest_one {} 99\n# HELP test_two Test\n# TYPE test_two gauge\ntest_two {} 999\n"
	if string(body) != expectedMetrics {
		t.Errorf("Expected metrics:\n%s\nGot:\n%s", expectedMetrics, body)
	}

	// Stop the server gracefully
	cancel()
	time.Sleep(100 * time.Millisecond) // Allow some time for the server to shut down
}