{{/*
Expand the name of the chart.
*/}}
{{- define "mosquitto_exporter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this
(by the DNS naming spec). If the release name contains the chart name it is
used as the full name.
*/}}
{{- define "mosquitto_exporter.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "mosquitto_exporter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "mosquitto_exporter.labels" -}}
helm.sh/chart: {{ include "mosquitto_exporter.chart" . }}
{{ include "mosquitto_exporter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "mosquitto_exporter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "mosquitto_exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
The name of the ServiceAccount to use.
*/}}
{{- define "mosquitto_exporter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "mosquitto_exporter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
The MQTT credentials Secret name. Uses an existing Secret when
`mqtt.auth.existingSecret` is set, otherwise the chart-managed Secret
(`fullname`).
*/}}
{{- define "mosquitto_exporter.mqttSecretName" -}}
{{- if .Values.mqtt.auth.existingSecret }}
{{- .Values.mqtt.auth.existingSecret }}
{{- else }}
{{- include "mosquitto_exporter.fullname" . }}
{{- end }}
{{- end }}

{{/*
The Grafana dashboard ConfigMap name. Defaults to `<fullname>-dashboard`.
*/}}
{{- define "mosquitto_exporter.dashboardConfigMapName" -}}
{{- default (printf "%s-dashboard" (include "mosquitto_exporter.fullname" .)) .Values.grafana.dashboard.configMapName }}
{{- end }}

{{/*
The exporter container args derived from values (broker, client id, listen
address, telemetry path, opt-in collectors, and extra args).
*/}}
{{- define "mosquitto_exporter.args" -}}
{{- $args := list
  (printf "--mqtt.broker=%s" .Values.mqtt.broker)
  (printf "--mqtt.client-id=%s" .Values.mqtt.clientId)
  (printf "--web.listen-address=%s" .Values.web.listenAddress)
  (printf "--web.telemetry-path=%s" .Values.web.telemetryPath)
-}}
{{- if .Values.collectors.clients }}{{- $args = append $args "--collector.clients" }}{{- end }}
{{- if .Values.collectors.messages }}{{- $args = append $args "--collector.messages" }}{{- end }}
{{- if .Values.collectors.load }}{{- $args = append $args "--collector.load" }}{{- end }}
{{- if .Values.extraArgs }}{{- $args = concat $args .Values.extraArgs }}{{- end }}
{{- toYaml $args -}}
{{- end }}