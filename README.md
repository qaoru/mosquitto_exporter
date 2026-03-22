# mosquitto_exporter
Prometheus exporter for Mosquitto MQTT broker.

[![Go](https://github.com/qaoru/mosquitto_exporter/actions/workflows/go.yml/badge.svg)](https://github.com/qaoru/mosquitto_exporter/actions/workflows/go.yml)
[![Codecov](https://codecov.io/gh/qaoru/mosquitto_exporter/branch/main/graph/badge.svg)](https://codecov.io/gh/qaoru/mosquitto_exporter)

This exporter subscribes to topics under the `$SYS` tree and exposes values as Prometheus metrics.

Tested against Mosquitto v2.0.x.

Due to limitations of the client library, this exporter can only connect using MQTTv3 / MQTTv3.1.

## Features

- **Core metrics**: Broker uptime, version, subscription counts, client counts, message statistics, and load metrics.
- **Broker availability**: Exports `mosquitto_up` gauge (1 = connected, 0 = disconnected) for monitoring connectivity.
- **Graceful degradation**: If the broker becomes unavailable, the exporter continues running and sets `mosquitto_up=0`. Subscriptions are automatically restored when the broker returns.
- **Health endpoint**: HTTP `/healthz` endpoint for liveness probes (always returns 200 when the exporter is running).
- **Configurable collectors**: Enable/disable specific metric collectors to reduce load.
- **Environment variable support**: All important settings can be provided via environment variables.

## Installation

### Docker (recommended)

```sh
docker run -d -p 9344:9344 ghcr.io/qaoru/mosquitto_exporter \
    --mqtt.broker=tcp://127.0.0.1:1883 \
    --collector.clients \
    --collector.messages \
    --collector.load
```

### Binary release

Automated binary releases are created via [GoReleaser](https://goreleaser.com) when a new version tag is pushed. Download the latest binary for your platform from the [Releases](https://github.com/qaoru/mosquitto_exporter/releases) page.

The exporter provides a version flag:
```sh
./mosquitto_exporter --version
```

### From source

```sh
git clone https://github.com/qaoru/mosquitto_exporter.git
cd mosquitto_exporter
go build
./mosquitto_exporter --help
```

## Configuration

### Command-line flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--web.listen-address` | | `:9344` | Address on which the web server will listen. |
| `--web.telemetry-path` | | `/metrics` | Path on which metrics will be served. |
| `--mqtt.broker` | `-b` | `tcp://127.0.0.1:1883` | Broker connection string (e.g., `tcp://host:1883`). |
| `--mqtt.client-id` | | `mosquitto-exporter` | Client ID to use when connected to the broker. |
| `--mqtt.username` | `-u` | (none) | Broker username. |
| `--mqtt.password` | `-p` | (none) | Broker password. |
| `--collector.clients` | | `false` | Enable the clients collector (client counts). |
| `--collector.messages` | | `false` | Enable the messages collector (message statistics). |
| `--collector.load` | | `false` | Enable the load collector (broker load metrics). |

### Environment variables

All MQTT-related flags can also be set via environment variables:

| Environment variable | Corresponding flag |
|----------------------|-------------------|
| `MQTT_BROKER`        | `--mqtt.broker`   |
| `MQTT_CLIENT_ID`     | `--mqtt.client-id`|
| `MQTT_USERNAME`      | `--mqtt.username` |
| `MQTT_PASSWORD`      | `--mqtt.password` |

Environment variables take precedence over default flag values but are overridden by explicit command-line arguments.

### Collector selection

By default, only the basic collector (uptime, version, subscription counts) is enabled. To enable additional collectors, use the corresponding flags:

- `--collector.clients` – exposes client counts (active, connected, disconnected, expired, inactive, maximum, total).
- `--collector.messages` – exposes message statistics (received, sent, stored, dropped, etc.).
- `--collector.load` – exposes load metrics (messages, bytes, sockets, etc.).

## Metrics

### Always present

| Metric | Type | Description |
|--------|------|-------------|
| `mosquitto_up` | Gauge | Whether the exporter is connected to the broker (1 = up, 0 = down). |
| `mosquitto_subscription_errors_total` | Counter | Total number of subscription errors, labeled by topic and error. |
| `mosquitto_uptime_seconds` | Counter | Seconds since the broker was started. |
| `mosquitto_version_info` | Gauge | Mosquitto version (label `version`). |
| `mosquitto_subscriptions_total` | Gauge | Number of active subscriptions. |
| `mosquitto_shared_subscriptions_total` | Gauge | Number of active shared subscriptions. |

### Enabled with `--collector.clients`

| Metric | Type | Description |
|--------|------|-------------|
| `mosquitto_active_clients_count` | Gauge | Number of active clients. |
| `mosquitto_connected_clients_count` | Gauge | Number of connected clients. |
| `mosquitto_disconnected_clients_count` | Gauge | Number of disconnected clients. |
| `mosquitto_expired_clients_count` | Gauge | Number of expired clients. |
| `mosquitto_inactive_clients_count` | Gauge | Number of inactive clients. |
| `mosquitto_maximum_clients_count` | Gauge | Maximum number of simultaneously connected clients. |
| `mosquitto_total_clients_count` | Gauge | Total number of clients. |

### Enabled with `--collector.messages`

| Metric | Type | Description |
|--------|------|-------------|
| `mosquitto_received_messages_count` | Counter | Total number of messages received. |
| `mosquitto_sent_messages_count` | Counter | Total number of messages sent. |
| `mosquitto_stored_messages_count` | Gauge | Number of messages currently stored. |
| `mosquitto_stored_messages_bytes` | Gauge | Total size of stored messages in bytes. |
| `mosquitto_inflight_messages_gauge` | Gauge | Number of inflight messages. |

### Enabled with `--collector.load`

The load collector exposes moving averages over 1‑minute, 5‑minute and 15‑minute intervals for several broker metrics. Each metric has three variants with suffixes `_load1`, `_load5`, `_load15`.

| Metric | Type | Description |
|--------|------|-------------|
| `mosquitto_connections_load1`<br>`mosquitto_connections_load5`<br>`mosquitto_connections_load15` | Gauge | Moving average of connections opened per second. |
| `mosquitto_sockets_load1`<br>`mosquitto_sockets_load5`<br>`mosquitto_sockets_load15` | Gauge | Moving average of socket connections opened per second. |
| `mosquitto_bytes_received_load1`<br>`mosquitto_bytes_received_load5`<br>`mosquitto_bytes_received_load15` | Gauge | Moving average of bytes received per second. |
| `mosquitto_bytes_sent_load1`<br>`mosquitto_bytes_sent_load5`<br>`mosquitto_bytes_sent_load15` | Gauge | Moving average of bytes sent per second. |
| `mosquitto_messages_received_load1`<br>`mosquitto_messages_received_load5`<br>`mosquitto_messages_received_load15` | Gauge | Moving average of messages received per second. |
| `mosquitto_messages_sent_load1`<br>`mosquitto_messages_sent_load5`<br>`mosquitto_messages_sent_load15` | Gauge | Moving average of messages sent per second. |
| `mosquitto_publish_received_load1`<br>`mosquitto_publish_received_load5`<br>`mosquitto_publish_received_load15` | Gauge | Moving average of publish messages received per second. |
| `mosquitto_publish_sent_load1`<br>`mosquitto_publish_sent_load5`<br>`mosquitto_publish_sent_load15` | Gauge | Moving average of publish messages sent per second. |
| `mosquitto_publish_dropped_load1`<br>`mosquitto_publish_dropped_load5`<br>`mosquitto_publish_dropped_load15` | Gauge | Moving average of publish messages dropped per second. |

All metrics include a `broker` label containing the connection string.

## Health endpoint

The exporter provides a simple HTTP endpoint for liveness probes:

```
GET /healthz
```

Returns `200 OK` with body `"ok"` as long as the exporter’s HTTP server is running. This endpoint does not check broker connectivity (use `mosquitto_up` for that).

## Example usage

### Docker Compose

```yaml
version: '3'
services:
  mosquitto:
    image: eclipse-mosquitto:2
    ports:
      - "1883:1883"
    volumes:
      - ./mosquitto.conf:/mosquitto/config/mosquitto.conf

  mosquitto_exporter:
    image: ghcr.io/qaoru/mosquitto_exporter
    ports:
      - "9344:9344"
    environment:
      MQTT_BROKER: "tcp://mosquitto:1883"
      MQTT_CLIENT_ID: "exporter"
    command:
      - "--collector.clients"
      - "--collector.messages"
      - "--collector.load"
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

## License

MIT – see [LICENSE](LICENSE) file.
