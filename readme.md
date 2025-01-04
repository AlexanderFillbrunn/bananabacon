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

Control Bananabacon using the following environment variables:

- **INPUT_FILE**: The log file to replay.
- **FILTER_REGEX**: The regex for filtering log lines. Defaults to `.*`.
- **TIME_REGEX**: The regex for extracting the timestamp from a log line. Must have exactly one subgroup for the timestamp.
- **TIME_FORMAT**: The format of the timestamp in the logs, defined in the [Go time format](https://www.geeksforgeeks.org/time-formatting-in-golang/).
- **LOOP**: Whether to loop the log output after the file has been replayed.
- **METRICS_PORT**: Port the metrics server listens on.
