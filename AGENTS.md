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
grafana-dashboard.json            # canonical Grafana dashboard (single source of truth; chart copy is CI-regenerated)
README.md                     # user-facing docs (flags, metrics, examples)
charts/mosquitto-exporter/    # Helm chart (Chart.yaml, values.yaml + schema, README.md.gotmpl,
                              #   templates: deployment/service/serviceaccount/servicemonitor/
                              #   secret/poddisruptionbudget/networkpolicy/ciliumnetworkpolicy)
.github/workflows/chart.yml        # chart CI: helm lint + kubeconform validation (quality gate)
.github/workflows/chart-release.yml  # on main push / tag: lint + package + helm push OCI + cosign + GitHub Release
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

## Helm chart (`charts/mosquitto-exporter`)

The Helm chart is **co-located and self-contained in this repo**: its source
lives under `charts/mosquitto-exporter/` (next to the binary/Dockerfile) and it
is **published from this same repo** as a cosign-signed OCI artifact to the
shared `qaoru/helm-charts` GHCR namespace (same path as the charts the
`qaoru/helm-charts` collection manages — `unifi`, `open-terminal`):
`ghcr.io/qaoru/helm-charts/mosquitto-exporter:<version>`. Only the OCI package
lives in that namespace; the chart source, release tag, and GitHub Release stay
in THIS repo and never touch the `qaoru/helm-charts` git repo.

There is **no classic HTTP Helm repository and no Artifact Hub entry** for this
chart (that unified-collection model would require the `qaoru/helm-charts`
release pipeline). Consumers install it via OCI:
`helm install ... oci://ghcr.io/qaoru/helm-charts/mosquitto-exporter`.

Workflows:
- `.github/workflows/chart.yml` — **quality gate** (helm lint + kubeconform)
  on PRs and pushes; `contents: read` only; never publishes.
- `.github/workflows/chart-release.yml` — **publisher**. On push to `main`, it
  detects a `Chart.yaml` `version`/`appVersion` change vs the previous commit;
  when only `appVersion` changed it auto-bumps the chart patch version (so image
  updates cut a release), regenerates `README.md` with helm-docs, commits, pushes
  a `mosquitto-exporter-<version>` tag, then lints + packages + `helm push`
  (OCI) + cosign-signs (keyless OIDC) + creates a GitHub Release with the
  `.tgz` attached. Also supports manual `mosquitto-exporter-<semver>` tag pushes
  and `workflow_dispatch` recovery.

Auth: everything uses this repo's built-in `GITHUB_TOKEN` (`contents: write`
for git ops, `packages: write` for the OCI push, `id-token: write` for cosign
keyless signing). `ghcr.io/qaoru/helm-charts/mosquitto-exporter` is just a GHCR
package name (a flat string with a slash, like the collection's
`helm-charts/open-terminal`); the `GITHUB_TOKEN` creates it on first push and
GHCR links it to THIS repo. No PAT, no per-package grant. Do NOT manually
re-link the package to `qaoru/helm-charts` — that would revoke this repo's
write access and force a PAT/grant.

Chart release tags use the `mosquitto-exporter-<version>` prefix so they never
collide with the exporter binary's `v*` tags (goreleaser / `release.yml`). A
`<chart>-<version>` tag pushed by the workflow via `GITHUB_TOKEN` does not
re-trigger the workflow, so publish runs in the same job as the tag push.

`appVersion` is owned by THIS repo (the exporter image is released here by
goreleaser): bump `appVersion` + the `artifacthub.io/images` annotation in
`Chart.yaml` and merge to main; `chart-release.yml` auto-bumps the chart patch
version and releases. After a `v*` tag release, `release.yml`'s
`chart-appversion-pr` job opens that bump as a PR automatically (merging it
triggers the chart release); you can still bump manually for control.
`grafana-dashboard.json` (root) is the canonical dashboard — the chart's
`dashboard/mosquitto.json` is regenerated from it by `chart.yml` and
`chart-release.yml`.

Chart conventions (follow the `qaoru/helm-charts` charts `unifi` / `open-terminal`):
- Chart name uses hyphens (`mosquitto-exporter`) even though the Go module /
  repo uses an underscore — Kubernetes resource names cannot contain `_`.
- `Chart.yaml` carries Artifact Hub annotations (`changes`, `links`,
  `maintainers`, `images`); `appVersion` tracks the exporter image tag and is
  kept in sync with goreleaser releases.
- `values.yaml` is documented with `# --` comments consumed by `helm-docs`;
  `README.md` is generated from `README.md.gotmpl`. Regenerate after editing
  values with `helm-docs charts/mosquitto-exporter/` (`chart-release.yml`
  regenerates it on release too).
- `values.schema.json` validates inputs (`additionalProperties: false` where
  practical, enums for `image.pullPolicy`, `service.type`,
  `networkPolicy.flavor`).
- Network isolation is opt-in via `networkPolicy.enabled` + `networkPolicy.flavor`
  (`kubernetes` → `NetworkPolicy`, `cilium` → `CiliumNetworkPolicy`); the two
  templates render native rules verbatim from `networkPolicy.{ingress,egress}`
  and `networkPolicy.cilium.{ingress,egress}` respectively. Defaults assume an
  in-cluster broker on port 1883.
- Prometheus scraping is opt-in two ways: `serviceMonitor.enabled` (Prometheus
  Operator) and `prometheus.scrapeAnnotations` (annotation-based). Both can be
  on simultaneously.
- The chart-managed MQTT `Secret` is created **only** when inline
  `mqtt.auth.username`/`password` are set and no `mqtt.auth.existingSecret` is
  referenced. Do not render a Secret with empty credentials.
- The default pod/container security contexts comply with PSS `restricted`
  (UID 65532, dropped caps, `readOnlyRootFilesystem`, RuntimeDefault seccomp).

When adding chart templates, add a corresponding render case to
`.github/workflows/chart.yml`'s "Render and validate manifests" matrix and keep
`values.schema.json` in sync.

## Release process

- Releases are driven by **goreleaser** on tag push (`v*`) via
  `.github/workflows/release.yml`. It cross-builds (linux/darwin/windows,
  amd64/arm64), archives, checksums, and pushes images via `dockers_v2` to
  ghcr.io and Docker Hub. After a successful release, the same workflow's
  `chart-appversion-pr` job opens a PR bumping the Helm chart's `appVersion`
  (and the Artifact Hub image annotation) to the released version. Merging that
  PR is what triggers `chart-release.yml` to auto-bump the chart patch and
  publish the OCI chart — so the manual "bump `appVersion` and merge" step is
  now automated via a human-gated PR (the chart quality gate `chart.yml` runs
  on it). `appVersion` is only bumped after the image is confirmed published,
  so the chart never references a missing image.
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
  load values. Load parsing rejects non-finite values (`NaN`/`Inf`) after
  parsing; keep that guard for any new `ParseFloat`-based handler.
- Keep metric names unique across collectors and update `README.md`'s metrics
  tables and `grafana-dashboard.json` when adding/renaming metrics. The chart's
  `charts/mosquitto-exporter/dashboard/mosquitto.json` is a CI-regenerated copy
  of the canonical root `grafana-dashboard.json` — edit the root file only;
  `chart.yml` and `chart-release.yml` sync the chart copy before linting/packaging.
- Tests: add a `NewXCollector`, `Describe`, `Collect`, handler integration,
  `Subscribe`, `Subscribe_Error`, and handler parse-error test for each new
  collector, mirroring the existing ones. Use `testutil.ToFloat64` against
  `SubscriptionErrors` for the error path.

## Do not

- Do not register collectors lazily or against a custom registry without also
  moving the shared `subscriptionErrors` counter (constructed in `main()`).
- Do not add MQTTv5 support assumptions — the paho v1.5.0 client is v3-only here.
- Do not change metric names in isolation — update README, the Grafana
  dashboard, and treat renames as breaking.
- Do not block `main()` on broker connection; the non-blocking `Connect()` is
  intentional for graceful degradation.
