package bbmetrics

import (
	"os"
	"strings"
)

type MetricBuilder struct {
	Name string
	Script string
	Type int
	Labels map[string]string
	Description string
}

func NewMetricBuilder(name string) *MetricBuilder {
	return &MetricBuilder{
		Name: name,
		Type: GaugeType,
		Labels: make(map[string]string),
	}
}

func (m *MetricBuilder) WithType(t int) *MetricBuilder {
	m.Type = t
	return m
}

func (m *MetricBuilder) WithScript(script string) *MetricBuilder {
	m.Script = script
	return m
}

func (m *MetricBuilder) WithDescription(description string) *MetricBuilder {
	m.Description = description
	return m
}

func (m *MetricBuilder) WithLabel(label string, value string) *MetricBuilder {
	m.Labels[label] = value
	return m
}

func (m *MetricBuilder) IsComplete() bool {
	return len(m.Script) > 0 && len(m.Name) > 0
}

func (mb *MetricBuilder) Build() (*Metric, bool) {
	if !mb.IsComplete() {
		return nil, false
	}
	return NewMetric(mb.Name, mb.Type, mb.Script, mb.Labels, mb.Description), true
}

const (
	MetricEnvNamePrefix = "METRIC_"
	MetricExprEnvNameSuffix = "_EXPR"
	MetricTypeEnvNameSuffix = "_TYPE"
	MetricDescrEnvNameSuffix = "_DESCR"
	MetricLabelEnvNameSuffix = "_LABEL"
)

type MetricsEngineBuilder map[string]*MetricBuilder

func newMetricsEngineBuilder() MetricsEngineBuilder {
	return make(map[string]*MetricBuilder)
}

func NewMetricsEngineBuilderFromEnv() MetricsEngineBuilder {
	mb := newMetricsEngineBuilder()
	for _, e := range os.Environ() {
        pair := strings.SplitN(e, "=", 2)
		mb.AddFromEnv(pair[0], pair[1])
    }
	return mb
}

// addFromEnv adds a metric to the builder from a given environment variable. The
// variable name must start with METRIC_ and the value must be a valid metric
// expression. The metric type and labels can be specified separately using
// environment variables with the same name but different suffixes: _TYPE for
// the type and _LABEL for a label. The label name and value are separated by
// an equals sign.
func (mb MetricsEngineBuilder) AddFromEnv(varName, value string) MetricsEngineBuilder {
	if strings.HasPrefix(varName, MetricEnvNamePrefix) {
		name := varName[len(MetricEnvNamePrefix):strings.LastIndex(varName, "_")]
		builder, ok := mb[name]
		if !ok {
			builder = NewMetricBuilder(name)
			mb[name] = builder
		}
		if strings.HasSuffix(varName, MetricExprEnvNameSuffix) {
			builder.WithScript(value)
		} else if strings.HasSuffix(varName, MetricTypeEnvNameSuffix) {
			builder.WithType(stringToMetricType(value))
		} else if strings.HasSuffix(varName, MetricDescrEnvNameSuffix) {
			builder.WithDescription(value)
		} else if strings.HasSuffix(varName, MetricLabelEnvNameSuffix) {
			parts := strings.SplitN(value, "=", 2)
			builder.WithLabel(parts[0], parts[1])
		}
	}
	return mb
}

func (m MetricsEngineBuilder) Build() *MetricsEngine {
	metrics := make([]*Metric, 0, len(m))
	for _, mb := range m {
		metric, ok := mb.Build()
		if ok {
			metrics = append(metrics, metric)
		}
	}
	return NewMetricsEngine(metrics)
}

func stringToMetricType(s string) int {
	switch strings.ToLower(s) {
	case "counter":
		return CounterType
	case "gauge":
		return GaugeType
	case "histogram":
		return HistogramType
	case "summary":
		return SummaryType
	default:
		return GaugeType
	}
}