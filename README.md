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
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run tests with verbose output
go test -v ./...
```

### Building

```sh
# Build the exporter
go build

# Build with race detector
go build -race
```

### Test Coverage

The project uses Codecov for test coverage tracking. Tests are automatically run on every push and pull request.

Current coverage: 15.8% (core business logic)

### Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Commit your changes: `git commit -am 'Add some feature'`
4. Push to the branch: `git push origin feature-name`
5. Submit a pull request