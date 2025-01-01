package bbmetrics

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


// Value returns the value of the metric as an arbitrary Go type.
func (mv MetricValue) Value() any {
	return mv.value
}


// Metric returns the metric that this MetricValue is a value for.
func (mv MetricValue) Metric() *Metric {
	return mv.metric
}

// NewMetricValue returns a new MetricValue instance with the given metric and value.
// The value argument can be any Go type, but it must be a type that can be serialized
// to a string that can be used as a value in a Prometheus metric line.
func NewMetricValue(metric *Metric, value any) MetricValue {
	return MetricValue{
		metric: metric,
		value: value,
	}
}

// String returns a string representation of the MetricValue, in the format
// expected by Prometheus. The format is as follows:
//
//   # HELP <metric_name> <metric_description>
//   # TYPE <metric_name> <metric_type>
//   <metric_name>{<label_name>="<label_value>",<label_name>="<label_value>",...} <value>
//
// If the metric has no description, the help part is omitted.
//
// The metric description and type are only included if the metric has a
// description and type, respectively.
func (mv MetricValue) String() string {
	var sb strings.Builder

	cnt := 0
    for k, v := range mv.Metric().Labels() {
		sb.WriteString(fmt.Sprintf("%s=\"%s\"", k, v))
		cnt++
		if cnt < len(mv.Metric().Labels()) {
			sb.WriteString(",")
		}
	}
	helpLine := ""
	if len(mv.Metric().Description()) > 0 {
		helpLine = fmt.Sprintf("# HELP %s %s\n", mv.Metric().Name(), mv.Metric().Description())
	}
	typeLine := fmt.Sprintf("# TYPE %s %s\n", mv.Metric().Name(), MetricTypeToString(mv.Metric().Type()))
	valueLine := fmt.Sprintf("%s {%s} %v", mv.Metric().Name(), sb.String(), mv.Value())
	return fmt.Sprintf("%s%s%s", helpLine, typeLine, valueLine)
}

// MetricTypeToString takes a metric type as an integer (as returned by
// Metric.Type()) and returns a string representation of the type suitable for
// use in a Prometheus metric definition.
func MetricTypeToString(t int) string {
	switch t {
	case CounterType:
		return "counter"
	case GaugeType:
		return "gauge"
	case HistogramType:
		return "histogram"
	case SummaryType:
		return "summary"
	default:
		return "gauge"
	}
}