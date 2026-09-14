# AGENTS.md

Guidance for coding agents (and human contributors) working on
`mosquitto_exporter`, a Prometheus exporter for the Mosquitto MQTT broker.

## Project overview

`mosquitto_exporter` is a standalone Go binary that connects to a Mosquitto
broker over MQTT (v3/3.1.1), subscribes to topics under the `$SYS` tree, and
exposes the values as Prometheus metrics on an HTTP endpoint. The HTTP server
runs independently of the MQTT connection: if the broker is unreachable the
exporter keeps serving `/metrics` with `mosquitto_up=0` and zero-valued
collectors (graceful degradation). Tested against Mosquitto v2.0.x.

- Module: `github.com/qaoru/mosquitto_exporter`
- Go floor: `1.25.0` (no `toolchain` directive; Dockerfile and CI use `golang:1.25` — keep these in sync)
- Default listen address: `:9344`; metrics path `/metrics`; health endpoint `/healthz`
- MQTT client: `github.com/eclipse/paho.mqtt.golang` (v3 only; MQTTv5 is **not** supported by the library version in use)

## Repository layout

```
mosquitto_exporter.go          # package main: flag parsing, MQTT client setup,
                              # collector registration, HTTP server, lifecycle
mosquitto_exporter_test.go    # trivial smoke test (does NOT exercise main())
internal/                     # all collector logic (package internal)
  exporter.go                 # shared `metric` struct (desc + valueType)
  subscription_errors.go      # NewSubscriptionErrors constructor (broker const label; registered in main())
  up_collector.go             # mosquitto_up gauge, driven by OnConnect/ConnectionLost
  default_collector.go        # uptime, version_info, subscriptions, shared_subscriptions
  clients_collector.go        # client counts (opt-in)
  messages_collector.go       # message stats + stored messages (opt-in)
  load_collector.go           # 1/5/15-min moving averages (opt-in, uses [3]metric)
  *_test.go, mock_test.go     # unit tests + mock mqtt.Client/Token/Message
Dockerfile                    # multi-stage build (golang:1.25 -> distroless/static)
Dockerfile.release            # distroless image copying goreleaser-built binaries
.goreleaser.yml               # v2 config: cross-build + dockers_v2 + checksums
.github/workflows/            # go.yml (CI), release.yml (goreleaser), docker-publish.yml
.github/dependabot.yml        # gomod + github-actions, weekly
grafana-dashboard.json        # bundled dashboard (not referenced from README)
README.md                     # user-facing docs (flags, metrics, examples)
coverage.out                  # gitignored build artifact
```

## Build, test, lint

```sh
go build                                 # build the exporter
go vet ./...                             # static checks (run in CI)
go test ./...                            # unit tests
go test -race ./...                      # NOTE: -race is NOT currently in CI; run it locally
go test ./... -coverprofile=coverage.out -covermode=atomic   # coverage
golangci-lint run                        # CI pins golangci-lint v2.4.0 (first release supporting Go 1.25)
go mod tidy && git diff --exit-code go.mod go.sum            # CI verifies no unused deps
```

CI (`.github/workflows/go.yml`) runs three jobs in order: **test** (unit tests,
coverage -> Codecov, `go vet`, golangci-lint) -> **integration-test** (installs
real Mosquitto via `apt-get` + `systemctl`, runs `go run .` and scrapes
`/metrics`) -> **build** (plain + `-race` build, and `go mod tidy` diff check).
The race detector is built but **not** executed against tests in CI.

Integration tests use an `apt-get`-installed Mosquitto service, **not** GitHub
Actions service containers (despite what the README says — a known minor doc
inaccuracy).

## Architecture & key invariants (preserve these)

### Collector pattern
Each collector is a struct implementing `prometheus.Collector` (`Describe` +
`Collect`), holding its own `sync.RWMutex`, a `Metrics` store (either
`map[string]float64` or a typed struct), and a `descriptions` map. It also
exposes a `Subscribe(client mqtt.Client)` method that wires MQTT topic
subscriptions to per-topic handler methods.

- **Registration is up front in `main()`** against the default Prometheus
  registry so metrics are always present (zero-valued) even before broker data
  arrives. Do not lazy-register.
- **Subscriptions are (re)established in `OnConnectHandler`**, not at
  construction. This keeps the HTTP server usable while the broker is down.
- `UpCollector` and `DefaultCollector` are always on; `clients`, `messages`,
  and `load` collectors are opt-in via `--collector.*` flags (default false).

### Concurrency
- Message handlers take `mu.Lock()`; `Collect` takes `mu.RLock()` with
  `defer`. Keep this discipline for any new collector.
- Handlers run in paho goroutines; `Collect` runs in the Prometheus scrape
  goroutine. Both touch the `Metrics` map — the mutex is mandatory, not
  decorative.
- Prefer adding `-race` coverage when touching concurrency code.

### MQTT connection lifecycle
`main()` configures: `SetAutoReconnect(true)`, `SetConnectRetry(true)`,
`SetResumeSubs(true)`, `SetCleanSession(false)`, max reconnect interval 30s,
connect timeout 5s. `OnConnect` sets `mosquitto_up=1` and (re)subscribes;
`ConnectionLost` sets `mosquitto_up=0`. `client.Connect()` is called
non-blocking and its token is currently ignored.

### Metric conventions used here
- Const label `broker` is attached to every metric (set from the broker URL).
- `mosquitto_subscription_errors_total` is built by `NewSubscriptionErrors()`
  with the `broker` const label and explicitly registered in `main()`, like
  the other collectors. Each collector receives the shared counter and
  increments it on subscribe failure.
- Load metrics use `[3]metric` per key and emit `_load1`/`_load5`/`_load15`
  variants. `LoadCollector.Collect` reads keys `<base>_1min`/`_5min`/`_15min`.

## Configuration surface (keep README in sync)

| Flag | Short | Env var | Default |
|------|-------|---------|---------|
| `--web.listen-address` | | | `:9344` |
| `--web.telemetry-path` | | | `/metrics` |
| `--mqtt.broker` | `-b` | `MQTT_BROKER` | `tcp://127.0.0.1:1883` |
| `--mqtt.client-id` | | `MQTT_CLIENT_ID` | `mosquitto-exporter` |
| `--mqtt.username` | `-u` | `MQTT_USERNAME` | (none) |
| `--mqtt.password` | `-p` | `MQTT_PASSWORD` | (none) |
| `--collector.clients` | | | `false` |
| `--collector.messages` | | | `false` |
| `--collector.load` | | | `false` |

`--version` prints `version (commit ..., built ... by ...)`; the `version`,
`commit`, `date`, `builtBy` vars are injected by goreleaser ldflags.

## Release process

- Releases are driven by **goreleaser** on tag push (`v*`) via
  `.github/workflows/release.yml`. It cross-builds (linux/darwin/windows,
  amd64/arm64), archives, checksums, and pushes images via `dockers_v2` to
  ghcr.io and Docker Hub.
- `docker-publish.yml` builds dev-branch/PR images (tagged by branch/sha) and
  cosign-signs non-PR images. It deliberately does **not** push `:latest` or
  `:vX.Y.Z` — those are owned by the release workflow to avoid races.
- When bumping the Go floor in `go.mod`, also update `Dockerfile`,
  `Dockerfile.release`, all `actions/setup-go` steps (`go-version: '1.25'`),
  and the pinned `golangci-lint` version if needed (v2.4.0 is the first to
  support Go 1.25).

## Coding conventions for this repo

- New collector? Follow the existing struct pattern: `mu sync.RWMutex`,
  `Metrics` store, `descriptions map[string]metric`, `Describe`/`Collect`/
  `Subscribe`, and a per-topic handler. Register it in `main()` and gate it
  behind a `--collector.<name>` flag unless it is core (then it's always on).
- Handlers must parse defensively: a malformed payload must **log and return
  without overwriting the previously stored value**. Existing tests assert this
  (`*_Handler_ParseError` / `*_Handlers_ParseErrors`); preserve the behavior.
- Use `strconv.Atoi` for integer $SYS payloads and `strconv.ParseFloat` for
  load values. Note load parsing does not yet reject `NaN`/`Inf` (see Known
  issues).
- Keep metric names unique across collectors and update `README.md`'s metrics
  tables and `grafana-dashboard.json` when adding/renaming metrics.
- Tests: add a `NewXCollector`, `Describe`, `Collect`, handler integration,
  `Subscribe`, `Subscribe_Error`, and handler parse-error test for each new
  collector, mirroring the existing ones. Use `testutil.ToFloat64` against
  `SubscriptionErrors` for the error path.

## Known issues & risks (from the latest reviewer + oracle review)

Treat this as a backlog, not a to-do mandate, but be aware when making
related changes:

- **H1 — Credential leak in `broker` label.** The raw `--mqtt.broker` URL
  (which may embed `user:pass@`) is used verbatim as the `broker` const label
  on every metric and is logged at startup. Sanitize (strip userinfo) before
  using it as a label, and avoid logging the raw URL. (`mosquitto_exporter.go`,
  `constLabels["broker"] = *broker` and the "Connecting to broker" log line.)
- **ResumeSubs + OnConnect redundancy.** `SetResumeSubs(true)` makes paho
  auto-resubscribe on reconnect *and* `OnConnect` explicitly re-subscribes,
  so every reconnect sends duplicate SUBSCRIBE packets (idempotent at the
  broker, but redundant). Pick one mechanism; relying on OnConnect only is the
  more explicit/testable option.
- **`mosquitto_subscription_errors_total` previously had no `broker` label**
  and was registered via `init()` rather than explicitly in `main()` —
  inconsistent with every other metric and with the registration pattern. Made
  multi-broker attribution impossible and coupled tests to the default
  registry. **Fixed**: the counter is now built by `NewSubscriptionErrors()`
  with the `broker` const label, constructed and registered in `main()`, and
  passed into each collector. Tests use a per-test unregistered counter for
  isolation.
- **Metric naming violates Prometheus conventions** (`_count` on gauges,
  `_total` on gauges, `_count` instead of `_total` on counters, `_gauge`
  suffix on the inflight metric). Fixing this is **breaking** (dashboards,
  alerts, and the bundled `grafana-dashboard.json` reference current names) —
  coordinate with a major version bump.
- **Docker images previously ran as root** — neither Dockerfile set `USER`.
  The distroless/static image ships a `nonroot` user (UID 65534). **Fixed**:
  both Dockerfiles now set `USER nonroot:nonroot` before `ENTRYPOINT`; the
  container runs non-root and is compatible with Pod Security Standards
  `restricted`. Binding to a privileged port (<1024) now needs a
  `securityContext` override.
- **`strconv.ParseFloat` accepts `NaN`/`Inf`** in the load handler; a
  non-finite payload would later panic in `MustNewConstMetric` during a scrape.
  Reject non-finite values after parsing.
- **No HTTP server timeouts** (`ReadTimeout`/`WriteTimeout`/`IdleTimeout`).
- **`--web.telemetry-path=/healthz` panics** at startup (double registration
  on `DefaultServeMux`). Validate the path or guard the healthz registration.
- **No concurrency/format tests**: the mutex-protected paths are not exercised
  under `-race`, and no test uses `testutil.CollectAndCompare` to verify
  actual metric names/types/labels in text output. CI runs `go test` without
  `-race` (the `-race` build is compile-only).
- Minor: README says messages collector exposes "dropped" (it doesn't — that's
  the load collector's `publish_dropped`); README says integration tests use
  "service containers" (they use apt-get + systemd); `grafana-dashboard.json`
  is not mentioned in the README; `mosquitto_exporter_test.go::TestMainFunctionality`
  doesn't actually test `main()`.

## Do not

- Do not register collectors lazily or against a custom registry without also
  moving the shared `subscriptionErrors` counter (constructed in `main()`).
- Do not add MQTTv5 support assumptions — the paho v1.5.0 client is v3-only here.
- Do not change metric names in isolation — update README, the Grafana
  dashboard, and treat renames as breaking.
- Do not block `main()` on broker connection; the non-blocking `Connect()` is
  intentional for graceful degradation.