package bbmetrics

import (
	"time"

	"github.com/dop251/goja"
)

type MetricsEngine struct {
	Metrics []*Metric
	vm *goja.Runtime
	startTime time.Time
}

func NewMetricsEngine(metrics []*Metric) *MetricsEngine {
	return &MetricsEngine{
		Metrics: metrics,
		vm: goja.New(),
		startTime: time.Now(),
	}
}

func (me *MetricsEngine) Reset() {
	me.startTime = time.Now()
}

// Eval evaluates the given metric and returns its result and any error that occurred.
// The timestamp given to the metric is the time elapsed since instantiation or the
// last call to Reset.
func (me *MetricsEngine) Eval(metric *Metric) (MetricValue, error) {
	return metric.Eval(me.vm, time.Since(me.startTime))
}