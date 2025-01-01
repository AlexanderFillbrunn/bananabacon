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

/*************  ✨ Codeium Command ⭐  *************/
// NewMetric constructs a new Metric instance with the specified name, type,
// script, labels, and description. It returns a pointer to the Metric struct
// initialized with the provided values.
/******  6ec234c6-f965-4bda-936d-b8e7d8a7c1d4  *******/
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

func (m *Metric) Labels() map[string]string {
	return m.labels
}

func (m *Metric) Description() string {
	return m.description
}

func (m *Metric) String() string {
	return m.Name()
}

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

func (mv MetricValue) Value() any {
	return mv.value
}

func (mv MetricValue) Metric() *Metric {
	return mv.metric
}

func NewMetricValue(metric *Metric, value any) MetricValue {
	return MetricValue{
		metric: metric,
		value: value,
	}
}

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