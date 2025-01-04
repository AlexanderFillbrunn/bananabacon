package metrics

import (
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
)

const (
	UntypedType = iota
	CounterType = iota
	GaugeType = iota
	HistogramType = iota
	SummaryType = iota
)

const (
	MetricExpressionFuncTemplate = "function %s(t) { return %s }"
)

type Metric struct {
	name string
	script string
	typ int
	labels map[string]string
	description string
}

// NewMetric constructs a new Metric instance with the specified name, type,
// script, labels, and description. It returns a pointer to the Metric struct
// initialized with the provided values.
func NewMetric(name string, typ int, script string, labels map[string]string, description string) *Metric {
	return &Metric{
		name: name,
		typ: typ,
		script: script,
		labels: labels,
		description: description,
	}
}

// Name returns the name of the metric.
func (m *Metric) Name() string {
	return m.name
}

// Type returns the type of the metric as an integer value. Valid values are
// bbmetrics.UntypedType, bbmetrics.CounterType, bbmetrics.GaugeType,
// bbmetrics.HistogramType, and bbmetrics.SummaryType.
func (m *Metric) Type() int {
	return m.typ
}

// Script returns the script that defines the metric. The script is a Go
// expression that is executed in a context where the "t" variable is the
// elapsed time since the metric was created. The script should return a
// value of the appropriate type for the metric type.
func (m *Metric) Script() string {
	return m.script
}

// Labels returns the labels associated with the metric. The labels are a
// map of key-value pairs representing dimensions of the metric.
func (m *Metric) Labels() map[string]string {
	return m.labels
}

// Description returns the description of the metric, providing additional
// context or information about the metric. It is a string that can describe
// the purpose, usage, or other relevant details of the metric.
func (m *Metric) Description() string {
	return m.description
}

// String returns the name of the metric as a string.
func (m *Metric) String() string {
	return m.Name()
}

// Eval evaluates the given metric and returns its result and any error that occurred.
// The timestamp given to the metric is the time elapsed since instantiation or the
// last call to Reset.
func (m *Metric) Eval(vm *goja.Runtime, t time.Duration) (MetricValue, error) {
	fnScript := m.Script()
	if !strings.HasPrefix(m.Script(), "function") {
		fnScript = fmt.Sprintf(MetricExpressionFuncTemplate, m.Name(), m.Script())
	}
	_, err := vm.RunString(fnScript)
	if err != nil {
		return MetricValue{}, err
	}
	fn, ok := goja.AssertFunction(vm.Get(m.Name()))
	if !ok {
		return MetricValue{}, fmt.Errorf("metric %s is not a function", m.Name())
	}
	res, err := fn(goja.Undefined(), vm.ToValue(t.Milliseconds()))
	if err != nil {
		return MetricValue{}, err
	}
	// TODO: parse result based on metric type (as-is only works for gauge and counter)
	return NewMetricValue(m, res.Export()), nil;
}

type MetricValue struct {
	metric *Metric
	value any
}