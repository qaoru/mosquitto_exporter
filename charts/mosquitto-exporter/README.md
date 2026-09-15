# mosquitto_exporter

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square)
![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)
![AppVersion: 1.0.1](https://img.shields.io/badge/AppVersion-1.0.1-informational?style=flat-square)

Prometheus exporter for the Mosquitto MQTT broker. Subscribes to the Mosquitto `$SYS` topic tree and exposes broker statistics as Prometheus metrics.

Deploys the [mosquitto_exporter](https://github.com/qaoru/mosquitto_exporter) container as a `Deployment` with a `Service` exposing the Prometheus metrics endpoint. Optional resources: `ServiceAccount`, `Secret` (MQTT credentials), `ServiceMonitor` (Prometheus Operator), `PodDisruptionBudget`, and an opt-in `NetworkPolicy` / `CiliumNetworkPolicy`.

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
  --version 0.1.0 \
  --set mqtt.broker=tcp://mosquitto:1883 \
  --set collectors.clients=true --set collectors.messages=true --set collectors.load=true
```

### With a values file

```bash
helm install mosquitto-exporter oci://ghcr.io/qaoru/helm-charts/mosquitto-exporter --version 0.1.0 -f values.yaml
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

### Hardening

The distroless `:nonroot` image runs as UID/GID 65532. The default pod and
container security contexts comply with the Pod Security Standards `restricted`
profile (`runAsNonRoot`, dropped capabilities, `readOnlyRootFilesystem`,
RuntimeDefault seccomp, no privilege escalation).

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `image.repository` | string | `ghcr.io/qaoru/mosquitto_exporter` | Container image repository. |
| `image.tag` | string | `""` | Container image tag (defaults to the chart `appVersion`). |
| `image.pullPolicy` | string | `IfNotPresent` | Image pull policy. |
| `imagePullSecrets` | list | `[]` | Image pull secrets for private registries. |
| `nameOverride` | string | `""` | Override the chart name used in resource names. |
| `fullnameOverride` | string | `""` | Override the fully qualified resource name. |
| `replicaCount` | int | `1` | Number of exporter replicas (stateless; safe to scale). |
| `revisionHistoryLimit` | int | `10` | Number of old ReplicaSets to retain. |
| `serviceAccount.create` | bool | `true` | Create a dedicated ServiceAccount. |
| `serviceAccount.name` | string | `""` | ServiceAccount name (defaults to release fullname). |
| `serviceAccount.annotations` | object | `{}` | Annotations on the ServiceAccount. |
| `serviceAccount.automount` | bool | `false` | Automount the ServiceAccount token (exporter doesn't need it). |
| `podSecurityContext` | object | (PSS `restricted`) | Pod security context. |
| `securityContext` | object | (PSS `restricted`) | Container security context. |
| `mqtt.broker` | string | `tcp://mosquitto:1883` | Broker URL. |
| `mqtt.clientId` | string | `mosquitto-exporter` | MQTT client id. |
| `mqtt.auth.existingSecret` | string | `""` | Existing Secret holding MQTT credentials. |
| `mqtt.auth.username` | string | `""` | MQTT username (chart-managed Secret). |
| `mqtt.auth.password` | string | `""` | MQTT password (chart-managed Secret). |
| `mqtt.auth.usernameKey` | string | `username` | Secret key holding the username. |
| `mqtt.auth.passwordKey` | string | `password` | Secret key holding the password. |
| `collectors.clients` | bool | `false` | Enable the clients collector. |
| `collectors.messages` | bool | `false` | Enable the messages collector. |
| `collectors.load` | bool | `false` | Enable the load collector. |
| `web.listenAddress` | string | `":9344"` | `--web.listen-address`. |
| `web.telemetryPath` | string | `/metrics` | `--web.telemetry-path`. |
| `extraArgs` | list | `[]` | Extra CLI args appended after generated flags. |
| `env` | list | `[]` | Extra environment variables. |
| `service.type` | string | `ClusterIP` | Service type. |
| `service.port` | int | `9344` | Service port. |
| `service.targetPort` | int | `9344` | Container port (must match `web.listenAddress`). |
| `service.annotations` | object | `{}` | Service annotations. |
| `prometheus.scrapeAnnotations` | bool | `false` | Add `prometheus.io/scrape` annotations to the Service. |
| `serviceMonitor.enabled` | bool | `false` | Create a ServiceMonitor (Prometheus Operator). |
| `serviceMonitor.labels` | object | `{}` | Labels for Prometheus to select the ServiceMonitor. |
| `serviceMonitor.annotations` | object | `{}` | ServiceMonitor annotations. |
| `serviceMonitor.interval` | string | `30s` | Scrape interval. |
| `serviceMonitor.scrapeTimeout` | string | `10s` | Scrape timeout. |
| `serviceMonitor.honorLabels` | bool | `true` | Honor the exporter's labels. |
| `serviceMonitor.metricRelabelings` | list | `[]` | Metric relabelings. |
| `serviceMonitor.relabelings` | list | `[]` | Target relabelings. |
| `serviceMonitor.namespaceSelector` | object | `{}` | Namespace selector (empty = same namespace). |
| `networkPolicy.enabled` | bool | `false` | Enable network policy creation. |
| `networkPolicy.flavor` | string | `kubernetes` | Policy flavor: `kubernetes` or `cilium`. |
| `networkPolicy.ingress` | list | (allow port 9344) | Kubernetes NetworkPolicy ingress rules. |
| `networkPolicy.egress` | list | (broker 1883 + DNS) | Kubernetes NetworkPolicy egress rules. |
| `networkPolicy.cilium.ingress` | list | (allow port 9344) | CiliumNetworkPolicy ingress rules. |
| `networkPolicy.cilium.egress` | list | (broker 1883 + DNS) | CiliumNetworkPolicy egress rules. |
| `podDisruptionBudget.enabled` | bool | `false` | Create a PodDisruptionBudget. |
| `podDisruptionBudget.apiVersion` | string | `policy/v1` | PDB API version. |
| `podDisruptionBudget.minAvailable` | int/string | `1` | Minimum available (exclusive with maxUnavailable). |
| `podDisruptionBudget.maxUnavailable` | int/string | `""` | Maximum unavailable (exclusive with minAvailable). |
| `livenessProbe` | object | `/healthz` | Liveness probe. |
| `readinessProbe` | object | `/healthz` | Readiness probe. |
| `resources.requests` | object | `{cpu: 10m, memory: 32Mi}` | Resource requests. |
| `resources.limits` | object | `{cpu: 100m, memory: 64Mi}` | Resource limits. |
| `podLabels` | object | `{}` | Additional pod labels. |
| `podAnnotations` | object | `{}` | Additional pod annotations. |
| `deploymentAnnotations` | object | `{}` | Additional Deployment annotations. |
| `nodeSelector` | object | `{}` | Node selector. |
| `affinity` | object | `{}` | Affinity rules. |
| `tolerations` | list | `[]` | Tolerations. |
| `topologySpreadConstraints` | list | `[]` | Topology spread constraints. |
| `volumes` | list | `[]` | Additional pod volumes. |
| `volumeMounts` | list | `[]` | Additional container volume mounts. |
| `initContainers` | list | `[]` | Additional init containers. |

<!-- README.md is generated from README.md.gotmpl by helm-docs. Regenerate after
editing values: `helm-docs charts/mosquitto_exporter/`. -->