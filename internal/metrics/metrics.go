package metrics

import (
	"time"

	"github.com/dop251/goja"
)

type MetricsEngine struct {
	Metrics []*Metric
	startTime time.Time
}

// NewMetricsEngine constructs a new MetricsEngine instance from the provided
// metrics. The timestamp passed to each metric's Eval method will be the
// elapsed time since the MetricsEngine was created.
func NewMetricsEngine(metrics []*Metric) *MetricsEngine {
	return &MetricsEngine{
		Metrics: metrics,
		startTime: time.Now(),
	}
}

// Reset sets the startTime of the MetricsEngine to the current time.
// This effectively resets the time elapsed since the engine's creation
// or the last reset, affecting timestamps passed to metric evaluations.
func (me *MetricsEngine) Reset() {
	me.startTime = time.Now()
}

// Eval evaluates the given metric using the given Goja runtime
// and returns its result and any error that occurred.
// The timestamp given to the metric is the time elapsed since instantiation or the
// last call to Reset.
func (me *MetricsEngine) Eval(metric *Metric, vm *goja.Runtime) (MetricValue, error) {
	return metric.Eval(vm, time.Since(me.startTime))
}