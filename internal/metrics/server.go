package metrics

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
)

type MetricsServer struct {
	server *http.Server
}

func NewMetricsServer(engine *MetricsEngine, port int) *MetricsServer {
	return &MetricsServer{
		server: createMetricsServer(engine, port),
	}
}

func (ms *MetricsServer) Run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		ms.Stop(ctx, 5*time.Second)
	}()
	if err := ms.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
	}
}

func (ms *MetricsServer) Stop(ctx context.Context, timeout time.Duration) {
	sdctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := ms.server.Shutdown(sdctx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
}

// createMetricsServer initializes and returns an HTTP server that will listen on the provided port
// and serves metrics at the "/metrics" endpoint. It evaluates each metric in the provided
// MetricsEngine and writes the results to the HTTP response. If an error occurs during
// evaluation of a metric, it is skipped.
func createMetricsServer(engine *MetricsEngine, port int) (*http.Server) {
	server := &http.Server{
        Addr: ":" + strconv.Itoa(port),
    }
	http.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sb strings.Builder
		vm := goja.New()

		for _, m := range engine.Metrics {
			val, err := engine.Eval(m, vm)
			if err == nil {
				sb.WriteString(val.String())
				sb.WriteString("\n")
			}
		}
		io.WriteString(w, sb.String())
	}))
	return server
}