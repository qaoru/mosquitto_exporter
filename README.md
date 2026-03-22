# mosquitto_exporter
Prometheus exporter for Mosquitto MQTT broker.

[![Go](https://github.com/qaoru/mosquitto_exporter/actions/workflows/go.yml/badge.svg)](https://github.com/qaoru/mosquitto_exporter/actions/workflows/go.yml)
[![Codecov](https://codecov.io/gh/qaoru/mosquitto_exporter/branch/main/graph/badge.svg)](https://codecov.io/gh/qaoru/mosquitto_exporter)

This exporter subscribes to topics under the `$SYS` tree and exposes values as prometheus metrics.

Tested against Mosquitto v2.0.20.

Due to limitations of the client library, this exporter can only connect using MQTTv3 / MQTTv3.1.

## Running the exporter

```sh
docker run -d -p 9344:9344 ghcr.io/qaoru/mosquitto_exporter \
    --mqtt.broker=tcp://127.0.0.1:1883 \
    --collector.clients \
    --collector.messages \
    --collector.load
```

## Development

### Running tests

```sh
# Run all unit tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run tests with verbose output
go test -v ./...
```

**Note:** Integration tests are automatically run in CI using GitHub Actions service containers. They test against a real Mosquitto broker in a containerized environment.

### Building

```sh
# Build the exporter
go build

# Build with race detector
go build -race
```

