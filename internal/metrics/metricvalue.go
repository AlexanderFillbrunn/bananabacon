package metrics

import (
	"fmt"
	"strings"
)

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
	var valueLines string
	if mv.Metric().Type() == GaugeType || mv.Metric().Type() == CounterType || mv.Metric().Type() == UntypedType {
		valueLines = fmt.Sprintf("%s {%s} %v", mv.Metric().Name(), sb.String(), mv.Value())
	} else if mv.Metric().Type() == HistogramType {
		valueLines = createHistogramLines(mv, sb.String())
	} else if mv.Metric().Type() == SummaryType {
		valueLines = createSummaryLines(mv, sb.String())
	}
	return fmt.Sprintf("%s%s%s", helpLine, typeLine, valueLines)
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

func createSummaryLines(mv MetricValue, labels string) string {
	var msb strings.Builder
	for k, v := range mv.Value().(map[string]any) {
		labels := fmt.Sprintf("%s,quantile=\"%s\"", labels, k)
		msb.WriteString(fmt.Sprintf("%s {%s} %v\n", mv.Metric().Name(), labels, v))
	}
	return msb.String()
}

func createHistogramLines(mv MetricValue, labels string) string {
	var msb strings.Builder
	var suffix string
	for k, v := range mv.Value().(map[string]any) {
		if k == "sum" {
			suffix = "_sum"
		} else if k == "count" {
			suffix = "_count"
		} else {
			suffix = "_bucket"
			labels = fmt.Sprintf("%s,le=\"%s\"", labels, k)
		}
		msb.WriteString(fmt.Sprintf("%s%s {%s} %v\n", mv.Metric().Name(), suffix, labels, v))
	}
	return msb.String()
}