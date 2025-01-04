# Bananabacon

## Why Bananabacon?

Did you know you can create [fake bacon from banana peel](https://gardengrubblog.com/how-to-make-the-best-vegan-banana-peel-bacon/)?
This project creates fake logs and metrics for testing observability solutions.

## What does it do?

It serves metrics on /metrics and replays a given log file by printing it to stdout.
The metrics are generated using JavaScript expressions that run in [Goja](https://github.com/dop251/goja). Expressions are evaluated
every time the /metrics endpoint is called.
The log output is generated from a reference file, where the first log line's timestamp is mapped to the current time and the following
log lines are output with a delay so that the replay is in-sync with the original output. That means when there was a 10 second gap between
two log lines in the original log, the lines will be output with a 10 seconds gap as well.

## How do I use it?

Control Bananabacon using the following general environment variables:

| Variable         | Description                                                                                                                         | Default        |
| ---------------- | ----------------------------------------------------------------------------------------------------------------------------------- | -------------- |
| **INPUT_FILE**   | The log file to replay.                                                                                                             | /logs/test.log |
| **FILTER_REGEX** | The regex for filtering log lines.                                                                                                  | `.*`           |
| **TIME_REGEX**   | The regex for extracting the timestamp from a log line. Must have exactly one subgroup for the timestamp.                           | (None)         |
| **TIME_FORMAT**  | The format of the timestamp in the logs, defined in the [Go time format](https://www.geeksforgeeks.org/time-formatting-in-golang/). | (None)         |
| **LOOP**         | Whether to loop the log output after the file has been replayed.                                                                    | `false`        |
| **METRICS_PORT** | Port the metrics server listens on.                                                                                                 | 8080           |

Add metrics to produce using the following environment variables (<name> stands for the exported metric name):

| Variable                  | Description                                                                                                                                                                                                                                                                          | Default   |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------- |
| **METRIC\_<name>\_EXPR**  | The expression generating the metric value. For counter this needs to return an int, for gauge any number. The variable `t` holds the passed milliseconds since the server was started. You can either provide a function: `function (t) { return t * 2 }` or an expression: `t * 2` | t         |
| **METRIC\_<name>\_TYPE**  | The metric type (counter, gauge, histogram, summary, untyped). Note: currently only gauge and counter are supported.                                                                                                                                                                 | `counter` |
| **METRIC\_<name>\_DESCR** | The description for the metric that will be printed in the HELP line                                                                                                                                                                                                                 | ""        |
| **METRIC\_<name>\_LABEL** | The labels for the metric in the format `key1=value1,key2=value2,key3=value3`.                                                                                                                                                                                                       | (None)    |

**Example:** Counter metric named `my_metric` sloping up and then becoming static.

```
METRIC_my_metric_EXPR = function(t) { if(t < 100000) return t / 1000; return 100; }
METRIC_my_metric_TYPE = counter
METRIC_my_metric_DESCR = Sloping up until 100, then staying there
METRIC_my_metric_LABEL = my_app=app
```

## TODOs

- Other metric types
- More tests
- Metric eval function to receive the last value as argument in addition to `t`
