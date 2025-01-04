package metrics

import (
	"errors"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type MetricBuilder struct {
	Name string
	Script string
	Type int
	Labels map[string]string
	Description string
}

// NewMetricBuilder initializes and returns a new MetricBuilder instance with the 
// specified name. The metric type is set to GaugeType by default, and an empty 
// map is created for labels. This function is used to start the process of building 
// a new metric with the given name.
func NewMetricBuilder(name string) *MetricBuilder {
	return &MetricBuilder{
		Name: name,
		Type: GaugeType,
		Labels: make(map[string]string),
	}
}

// WithType sets the type for the metric being built. The type is represented
// as an integer value corresponding to one of the predefined metric types
// (e.g., CounterType, GaugeType, etc.). Returns the MetricBuilder to allow
// for method chaining.
func (m *MetricBuilder) WithType(t int) (*MetricBuilder, error) {
	if t < 0 || t > SummaryType {
		return m, errors.New("invalid metric type: " + strconv.Itoa(t))
	}
	m.Type = t
	return m, nil
}

// WithScript sets the script for the metric being built. The script
// is a JavaScript expression that is executed in a context where the "t"
// variable is the elapsed time since the metric was created. The
// script should return a value of the appropriate type for the
// metric type. The script is evaluated with Goja.
// Returns the MetricBuilder to allow for method chaining.
func (m *MetricBuilder) WithScript(script string) *MetricBuilder {
	m.Script = script
	return m
}

// WithDescription sets the description for the metric being built.
// The description provides additional context or information about
// the metric. Returns the MetricBuilder to allow for method chaining.
func (m *MetricBuilder) WithDescription(description string) *MetricBuilder {
	m.Description = description
	return m
}

// WithLabel adds a label to the metric being built. The label is a string in the
// form "labelName=value". The labelName must be a valid Prometheus label name,
// i.e. it must match the regular expression [a-zA-Z_][a-zA-Z0-9_]*. The value can
// be any string.
func (mb *MetricBuilder) WithLabel(labelName, value string) (*MetricBuilder, error) {
	if !isValidLabelName(labelName) {
		return mb, errors.New("invalid label name")
	}
	mb.Labels[labelName] = value
	return mb, nil
}

func isValidLabelName(labelName string) bool {
	regexp := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	return regexp.MatchString(labelName)
}

// IsComplete returns true if the MetricBuilder is complete, i.e. it has a valid
// name and script. Otherwise, it returns false.
func (m *MetricBuilder) IsComplete() bool {
	return len(m.Script) > 0 && len(m.Name) > 0
}

// Build constructs a Metric instance from the MetricBuilder if it is complete,
// returning the Metric and true. If the MetricBuilder is not complete, it
// returns nil and false.
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


// NewMetricsEngineBuilderFromEnv creates a new MetricsEngineBuilder from environment
// variables. It iterates over all environment variables and calls
// MetricsEngineBuilder.AddFromEnv on each of them. If an error occurs during
// processing of an environment variable, that error is printed and the variable ignored.
func NewMetricsEngineBuilderFromEnv() (MetricsEngineBuilder, error) {
	mb := newMetricsEngineBuilder()
	for _, e := range os.Environ() {
        pair := strings.SplitN(e, "=", 2)
		_, err := mb.AddFromEnv(pair[0], pair[1])
		if err != nil {
			log.Println(err)
			continue
		}
    }
	return mb, nil
}

// addFromEnv adds a metric to the builder from a given environment variable. The
// variable name must start with METRIC_ and the value must be a valid metric
// expression. The metric type and labels can be specified separately using
// environment variables with the same name but different suffixes: _TYPE for
// the type and _LABEL for a label. The label name and value are separated by
// an equals sign.
func (mb MetricsEngineBuilder) AddFromEnv(varName, value string) (MetricsEngineBuilder, error) {
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
			t, ok := stringToMetricType(value)
			if !ok {
				return mb, errors.New("Invalid metrics type for metric " + name)
			}
			_, err := builder.WithType(t)
			if err != nil {
				return mb, err
			}
		} else if strings.HasSuffix(varName, MetricDescrEnvNameSuffix) {
			builder.WithDescription(value)
		} else if strings.HasSuffix(varName, MetricLabelEnvNameSuffix) {
			parts := strings.SplitN(value, "=", 2)
			_, err := builder.WithLabel(parts[0], parts[1])
			if err != nil {
				return mb, err
			}
		}
	}
	return mb, nil
}

// Build constructs a MetricsEngine instance from the MetricBuilders in the
// MetricsEngineBuilder. It iterates over each MetricBuilder, building a Metric
// if it is complete, and adds it to the list of metrics. Returns a new
// MetricsEngine initialized with the constructed metrics.
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

// stringToMetricType takes a string value and returns a corresponding metric type.
// It returns true as the second value if the string is a valid metric type, and
// false otherwise. Valid metric type strings are "counter", "gauge", "histogram",
// and "summary".
func stringToMetricType(s string) (int, bool) {
	switch strings.ToLower(s) {
	case "counter":
		return CounterType, true
	case "gauge":
		return GaugeType, true
	case "histogram":
		return HistogramType, true
	case "summary":
		return SummaryType, true
	default:
		return 0, false
	}
}