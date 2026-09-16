# mosquitto-exporter

![Version: 0.2.1](https://img.shields.io/badge/Version-0.2.1-informational?style=flat-square)
![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)
![AppVersion: 2.0.0](https://img.shields.io/badge/AppVersion-2.0.0-informational?style=flat-square)

Prometheus exporter for the Mosquitto MQTT broker. Subscribes to the Mosquitto `$SYS` topic tree and exposes broker statistics as Prometheus metrics.

Deploys the [mosquitto_exporter](https://github.com/qaoru/mosquitto_exporter) container as a `Deployment` with a `Service` exposing the Prometheus metrics endpoint. Optional resources: `ServiceAccount`, `Secret` (MQTT credentials), `ServiceMonitor` (Prometheus Operator), `PodDisruptionBudget`, an opt-in `NetworkPolicy` / `CiliumNetworkPolicy`, and a `ConfigMap` shipping the bundled Grafana dashboard.

## Prerequisites

- Kubernetes >= 1.23
- Helm >= 3.0
- A reachable Mosquitto broker (in-cluster or external) on MQTTv3 / MQTTv3.1.1
- (Optional) Prometheus Operator CRDs for `ServiceMonitor`
- (Optional) Cilium when using `networkPolicy.flavor: cilium`

## Installation

### OCI (GitHub Container Registry)

```bash
helm install mosquitto-exporter oci://ghcr.io/qaoru/helm-charts/mosquitto-exporter \
  --version 0.2.1 \
  --set mqtt.broker=tcp://mosquitto:1883 \
  --set collectors.clients=true --set collectors.messages=true --set collectors.load=true
```

### With a values file

```bash
helm install mosquitto-exporter oci://ghcr.io/qaoru/helm-charts/mosquitto-exporter --version 0.2.1 -f values.yaml
```

### MQTT credentials

Provide credentials either inline (a Secret is created for you) or via an existing Secret:

```bash
kubectl create secret generic mosquitto-exporter-mqtt \
  --from-literal=username=<user> --from-literal=password=<pass>
```

then set `mqtt.auth.existingSecret=mosquitto-exporter-mqtt`.

## Configuration

### Broker & collectors

The exporter connects to `mqtt.broker` (default `tcp://mosquitto:1883`) and
subscribes to the `$SYS` topic tree. Opt-in collectors (`clients`, `messages`,
`load`) mirror the exporter's `--collector.*` flags. If the broker is
unreachable the exporter keeps serving `/metrics` with `mosquitto_up=0`
(graceful degradation).

### Prometheus

Two opt-in scrape mechanisms (usable together):

- **ServiceMonitor** (`serviceMonitor.enabled: true`) for Prometheus Operator.
  Set `serviceMonitor.labels` so your Prometheus selects it.
- **Scrape annotations** (`prometheus.scrapeAnnotations: true`) for
  annotation-based discovery without an Operator.

### Network policies

Opt in with `networkPolicy.enabled: true` and choose a `flavor`:

- `kubernetes` — standard `NetworkPolicy` (rules from `networkPolicy.ingress` /
  `networkPolicy.egress`).
- `cilium` — `CiliumNetworkPolicy` (rules from `networkPolicy.cilium.ingress` /
  `networkPolicy.cilium.egress`).

Defaults assume an **in-cluster broker on port 1883** and allow the metrics port
from any in-cluster source. Tighten the egress peers to match your broker and
the ingress peers to match your Prometheus. For an external broker use an
`ipBlock` (Kubernetes) or `toEntities: [world]` / `toFQDNs` (Cilium).

### Grafana dashboard

Opt in with `grafana.dashboard.enabled: true` to ship the bundled
`mosquitto_exporter` dashboard as a `ConfigMap`. The ConfigMap is labeled
`grafana_dashboard: "1"` by default so the [Grafana sidecar](https://grafana.github.io/helm-charts/grafana)
discovers and imports it automatically; set `grafana.dashboard.labels` to match
your sidecar's selector (e.g. `release: kube-prometheus-stack`) and
`grafana.dashboard.annotations` for tooling that selects by annotation.

The repository-root `grafana-dashboard.json` is the canonical dashboard; the
chart's `dashboard/mosquitto.json` is regenerated from it by CI (`chart.yml`
and `chart-release.yml`) before linting and packaging, so editing the root file
is all that is needed to update the shipped dashboard.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity rules. |
| collectors | object | `{"clients":false,"load":false,"messages":false}` | Opt-in metric collectors. All default to `false` to match the exporter. |
| collectors.clients | bool | `false` | Enable the clients collector (`mosquitto_maximum_clients`, etc.). |
| collectors.load | bool | `false` | Enable the load collector (1/5/15-min moving averages). |
| collectors.messages | bool | `false` | Enable the messages collector (message stats + stored messages). |
| deploymentAnnotations | object | `{}` | Additional Deployment annotations. |
| env | list | `[]` | Extra environment variables (list of `{name, value}` / `{name, valueFrom}`). |
| extraArgs | list | `[]` | Extra CLI arguments appended after the generated flags (e.g. `--log.level=debug`). Each entry is a single arg. |
| fullnameOverride | string | `""` | Override the fully qualified resource name. |
| grafana | object | `{"dashboard":{"annotations":{},"configMapName":"","enabled":false,"labels":{}}}` | Grafana dashboard integration. The chart vendors the bundled `mosquitto_exporter` dashboard under `dashboard/mosquitto.json` and can ship it as a ConfigMap discoverable by the Grafana sidecar (or any tool that selects ConfigMaps by label). |
| grafana.dashboard.annotations | object | `{}` | Annotations added to the ConfigMap. |
| grafana.dashboard.configMapName | string | `""` | ConfigMap name. Defaults to `<release>-dashboard`. |
| grafana.dashboard.enabled | bool | `false` | Deploy a ConfigMap containing the bundled Grafana dashboard. |
| grafana.dashboard.labels | object | `{}` | Extra labels added to the ConfigMap. The chart already sets `grafana_dashboard: "1"` for sidecar discovery; extend or override here to match your Grafana's selector (e.g. `release: kube-prometheus-stack`). |
| image | object | `{"pullPolicy":"IfNotPresent","repository":"ghcr.io/qaoru/mosquitto_exporter","tag":""}` | Image configuration. |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy. |
| image.repository | string | `"ghcr.io/qaoru/mosquitto_exporter"` | Container image repository. |
| image.tag | string | `""` | Container image tag (defaults to the chart `appVersion`). |
| imagePullSecrets | list | `[]` | Image pull secrets for private registries. |
| initContainers | list | `[]` | Additional init containers. |
| livenessProbe | object | `{"failureThreshold":3,"httpGet":{"path":"/healthz","port":"http-metrics"},"initialDelaySeconds":0,"periodSeconds":10,"timeoutSeconds":1}` | Liveness probe. The exporter's `/healthz` returns 200 while the process is running (independent of broker connectivity — see graceful degradation). |
| mqtt | object | `{"auth":{"existingSecret":"","password":"","passwordKey":"password","username":"","usernameKey":"username"},"broker":"tcp://mosquitto:1883","clientId":"mosquitto-exporter"}` | MQTT broker connection. The exporter connects over MQTTv3 / MQTTv3.1.1. |
| mqtt.auth | object | `{"existingSecret":"","password":"","passwordKey":"password","username":"","usernameKey":"username"}` | MQTT credentials. Provide either inline `username`/`password` (a Secret is created for you) or `existingSecret` referencing your own Secret. |
| mqtt.auth.existingSecret | string | `""` | Existing Secret holding the MQTT credentials. When set, no Secret is created and `username`/`password` are ignored. |
| mqtt.auth.password | string | `""` | MQTT password (stored in the chart-managed Secret when no `existingSecret` is set). |
| mqtt.auth.passwordKey | string | `"password"` | Key inside the Secret that holds the password. |
| mqtt.auth.username | string | `""` | MQTT username (stored in the chart-managed Secret when no `existingSecret` is set). |
| mqtt.auth.usernameKey | string | `"username"` | Key inside the Secret that holds the username. |
| mqtt.broker | string | `"tcp://mosquitto:1883"` | Broker URL (`scheme://host:port`). Scheme is `tcp://` (MQTT), `ssl://` (MQTT over TLS), `ws://` or `wss://`. |
| mqtt.clientId | string | `"mosquitto-exporter"` | MQTT client id presented to the broker. |
| nameOverride | string | `""` | Override the chart name used in resource names. |
| networkPolicy | object | `{"cilium":{"egress":[{"toEndpoints":[],"toPorts":[{"ports":[{"port":"1883","protocol":"TCP"}]}]},{"toEndpoints":[{"matchLabels":{"k8s:io.kubernetes.pod.namespace":"kube-system","k8s:k8s-app":"kube-dns"}}],"toPorts":[{"ports":[{"port":"53","protocol":"UDP"},{"port":"53","protocol":"TCP"}],"rules":{"dns":[{"matchPattern":"*"}]}}]}],"ingress":[{"toPorts":[{"ports":[{"port":"9344","protocol":"TCP"}]}]}]},"egress":[{"ports":[{"port":1883,"protocol":"TCP"}],"to":[{"namespaceSelector":{}}]},{"ports":[{"port":53,"protocol":"UDP"},{"port":53,"protocol":"TCP"}],"to":[{"namespaceSelector":{"matchLabels":{"kubernetes.io/metadata.name":"kube-system"}},"podSelector":{"matchLabels":{"k8s-app":"kube-dns"}}}]}],"enabled":false,"flavor":"kubernetes","ingress":[{"ports":[{"port":9344,"protocol":"TCP"}]}]}` | Network policy for ingress/egress isolation. Disabled by default; opt in with `enabled: true`. Two mutually-exclusive flavors are supported via `flavor`: `kubernetes` (standard NetworkPolicy) or `cilium` (CiliumNetworkPolicy). Each flavor renders its native rules verbatim from the matching block below, so you can express any policy the underlying CRD supports. |
| networkPolicy.cilium | object | `{"egress":[{"toEndpoints":[],"toPorts":[{"ports":[{"port":"1883","protocol":"TCP"}]}]},{"toEndpoints":[{"matchLabels":{"k8s:io.kubernetes.pod.namespace":"kube-system","k8s:k8s-app":"kube-dns"}}],"toPorts":[{"ports":[{"port":"53","protocol":"UDP"},{"port":"53","protocol":"TCP"}],"rules":{"dns":[{"matchPattern":"*"}]}}]}],"ingress":[{"toPorts":[{"ports":[{"port":"9344","protocol":"TCP"}]}]}]}` | CiliumNetworkPolicy rules (used when `flavor: cilium`). The defaults allow the metrics port from any endpoint and the broker port to any in-cluster endpoint, plus DNS via the Cilium DNS proxy. Tighten `toEndpoints` `matchLabels` to match your broker and Prometheus pods. |
| networkPolicy.enabled | bool | `false` | Enable network policy creation. |
| networkPolicy.flavor | string | `"kubernetes"` | Policy flavor: `kubernetes` or `cilium`. |
| networkPolicy.ingress | list | `[{"ports":[{"port":9344,"protocol":"TCP"}]}]` | Standard Kubernetes NetworkPolicy rules (used when `flavor: kubernetes`). The defaults assume an in-cluster broker on port 1883 and allow the metrics port from any in-cluster source. Tighten the `ingress` `from` and `egress` `to` peers to your Prometheus and broker pods. |
| nodeSelector | object | `{}` | Node selector. |
| podAnnotations | object | `{}` | Additional pod annotations. |
| podDisruptionBudget | object | `{"apiVersion":"policy/v1","enabled":false,"maxUnavailable":"","minAvailable":1}` | Pod disruption budget. Recommended when `replicaCount > 1`. |
| podDisruptionBudget.apiVersion | string | `"policy/v1"` | API version of the PDB resource (`policy/v1` on clusters >= 1.21). |
| podDisruptionBudget.enabled | bool | `false` | Create a PodDisruptionBudget. |
| podDisruptionBudget.maxUnavailable | string | `""` | Maximum unavailable replicas. Mutually exclusive with `minAvailable`. |
| podDisruptionBudget.minAvailable | int | `1` | Minimum available replicas. Mutually exclusive with `maxUnavailable`. |
| podLabels | object | `{}` | Additional pod labels. |
| podSecurityContext | object | `{"fsGroup":65532,"runAsGroup":65532,"runAsNonRoot":true,"runAsUser":65532,"seccompProfile":{"type":"RuntimeDefault"}}` | Pod security context. The distroless `:nonroot` image runs as UID/GID 65532; these defaults comply with the Pod Security Standards `restricted` profile. |
| prometheus | object | `{"scrapeAnnotations":false}` | Prometheus scrape integration. Both mechanisms are opt-in and can be used together; the ServiceMonitor is preferred when a Prometheus Operator is installed. |
| prometheus.scrapeAnnotations | bool | `false` | Add `prometheus.io/scrape` annotations to the Service for annotation-based discovery (no Operator required). |
| readinessProbe | object | `{"failureThreshold":3,"httpGet":{"path":"/healthz","port":"http-metrics"},"initialDelaySeconds":0,"periodSeconds":10,"timeoutSeconds":1}` | Readiness probe. `/healthz` reflects process liveness; for broker connectivity scrape `mosquitto_up` instead. |
| replicaCount | int | `1` | Number of exporter replicas. The exporter is stateless and safe to scale; each pod maintains its own MQTT connection and subscriptions. |
| resources | object | `{"limits":{"cpu":"100m","memory":"64Mi"},"requests":{"cpu":"10m","memory":"32Mi"}}` | Container resource requests and limits. |
| revisionHistoryLimit | int | `10` | Number of old ReplicaSets to retain for rollback. |
| securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true}` | Container security context (PSS `restricted` compatible). |
| service | object | `{"annotations":{},"port":9344,"targetPort":9344,"type":"ClusterIP"}` | Service exposing the metrics endpoint. |
| service.annotations | object | `{}` | Annotations added to the Service. |
| service.port | int | `9344` | Service port. |
| service.targetPort | int | `9344` | Container port the Service targets. Must match the port in `web.listenAddress`. |
| service.type | string | `"ClusterIP"` | Service type. |
| serviceAccount | object | `{"annotations":{},"automount":false,"create":true,"name":""}` | Service account configuration. |
| serviceAccount.annotations | object | `{}` | Annotations added to the ServiceAccount. |
| serviceAccount.automount | bool | `false` | Automount the ServiceAccount token into the pod. The exporter does not talk to the Kubernetes API, so this can be disabled. |
| serviceAccount.create | bool | `true` | Create a dedicated ServiceAccount for the exporter. |
| serviceAccount.name | string | `""` | ServiceAccount name (defaults to the release fullname). |
| serviceMonitor | object | `{"annotations":{},"enabled":false,"honorLabels":true,"interval":"30s","labels":{},"metricRelabelings":[],"namespaceSelector":{},"relabelings":[],"scrapeTimeout":"10s"}` | Prometheus Operator ServiceMonitor. Requires the `monitoring.coreos.com/v1` CRD. |
| serviceMonitor.annotations | object | `{}` | Annotations added to the ServiceMonitor. |
| serviceMonitor.enabled | bool | `false` | Create a ServiceMonitor resource. |
| serviceMonitor.honorLabels | bool | `true` | Honor the exporter's own labels. |
| serviceMonitor.interval | string | `"30s"` | Scrape interval. |
| serviceMonitor.labels | object | `{}` | Labels for Prometheus to select this ServiceMonitor (e.g. `release: kube-prometheus-stack`). |
| serviceMonitor.metricRelabelings | list | `[]` | Metric relabelings applied to scraped samples. |
| serviceMonitor.namespaceSelector | object | `{}` | Namespace selector for the ServiceMonitor (empty = same namespace). |
| serviceMonitor.relabelings | list | `[]` | Target relabelings applied to the scrape target. |
| serviceMonitor.scrapeTimeout | string | `"10s"` | Scrape timeout. |
| tolerations | list | `[]` | Tolerations. |
| topologySpreadConstraints | list | `[]` | Topology spread constraints. |
| volumeMounts | list | `[]` | Additional volume mounts for the container. |
| volumes | list | `[]` | Additional volumes for the pod. |
| web | object | `{"listenAddress":":9344","telemetryPath":"/metrics"}` | HTTP server configuration. |
| web.listenAddress | string | `":9344"` | Listen address passed to `--web.listen-address`. |
| web.telemetryPath | string | `"/metrics"` | Metrics path passed to `--web.telemetry-path`. |